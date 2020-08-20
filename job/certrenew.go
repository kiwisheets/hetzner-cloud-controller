package job

import (
	"context"
	"log"
	"time"

	"github.com/go-acme/lego/v3/certificate"
	"github.com/go-acme/lego/v3/lego"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/jinzhu/gorm"
	"github.com/kiwisheets/hetzner-cloud-controller/model"
)

func processCertRenews(hClient *hcloud.Client, lClient *lego.Client, db *gorm.DB) {
	var jobs []model.CertRenewJob

	db.Where("next_run < ?", time.Now()).Find(&jobs)

	log.Println("running certrenew jobs")

	for _, job := range jobs {
		// get info for each

		lb, _, err := hClient.LoadBalancer.GetByID(context.Background(), job.LoadbalancerID)

		if err != nil {
			// load balancer probably does not exist
			log.Printf("error processing cert renew for load balancer: %d", job.LoadbalancerID)
			log.Print(err)
		}

		if lb == nil {
			// load balancer definitely does not exist
			// TODO: try to delete job
			db.Delete(&job)
			continue
		}

		oldCertToNewCert := make(map[int]*hcloud.Certificate)

		for _, s := range lb.Services {
			// ensure it is up to date

			if s.HTTP.Certificates == nil {
				// no certificates
				continue
			}

			var certsToDelete []hcloud.Certificate

			for i, oldHcert := range s.HTTP.Certificates {
				// lookup in map so we don't renew again if already renewed
				if oldCertToNewCert[oldHcert.ID] != nil {
					s.HTTP.Certificates[i] = oldCertToNewCert[oldHcert.ID]
					continue
				}

				// get cert data
				// also ensures that all
				oldHcert, _, err = hClient.Certificate.GetByID(context.Background(), oldHcert.ID)

				if err != nil {
					log.Println("failed to get certificate data")
					log.Println(err)
					continue
				}

				if time.Until(oldHcert.NotValidAfter) <= 45*24*time.Hour || oldHcert.Name == lb.Name+"_dummy_cert" {
					if oldHcert.Name == lb.Name+"_dummy_cert" {
						oldHcert.Name = lb.Name
					}

					// renew cert

					log.Printf("renewing certs for Loadbalancer: %d. Service port: %d\n", lb.ID, s.ListenPort)
					log.Printf("domains: %v\n", oldHcert.DomainNames)

					// renew cert logic

					var certificates *certificate.Resource

					// get cert from db
					var dbCertificate model.Certificate
					certdataNotFound := db.Where(model.Certificate{HetznerID: oldHcert.ID}).First(&dbCertificate).RecordNotFound()
					if certdataNotFound {
						// always true on first run, works well with setting up autorenew
						// no data found for this cert
						// will have to create a new cert

						// create certdata

						log.Println("requesting new certs")
						request := certificate.ObtainRequest{
							Domains: oldHcert.DomainNames,
							Bundle:  true,
						}

						certificates, err = lClient.Certificate.Obtain(request)
						if err != nil {
							log.Println("failed to obtain certs")
							log.Println(err)
							continue
						}
					} else {
						// else renew existing certs
						oldCerts := dbCertificate.Resource

						certificates, err = lClient.Certificate.Renew(oldCerts, true, false)

						if err != nil {
							log.Println("failed to renew certs")
							log.Println(err)
							continue
						}
					}
					// upload new cert to hetzner

					// rename old cert

					_, _, err = hClient.Certificate.Update(context.Background(), oldHcert, hcloud.CertificateUpdateOpts{
						Name: oldHcert.Name + "_old",
					})

					if err != nil {
						log.Println("failed to rename old cert")
						log.Println(err)
						continue
					}

					// create new cert

					newHcert, _, err := hClient.Certificate.Create(context.Background(), hcloud.CertificateCreateOpts{
						Name:        oldHcert.Name,
						Certificate: string(certificates.Certificate),
						PrivateKey:  string(certificates.PrivateKey),
					})

					if err != nil {
						log.Println("failed to create new cert on hetzner")
						log.Println(err)
						continue
					}

					// replace old cert in slice with new cert
					s.HTTP.Certificates[i] = newHcert

					// update map

					oldCertToNewCert[oldHcert.ID] = newHcert

					// mark cert for deletion
					certsToDelete = append(certsToDelete, *oldHcert)

					// update data in database

					dbCertificate.Resource = *certificates
					dbCertificate.HetznerID = newHcert.ID

					if !certdataNotFound {
						// update
						db.Save(&dbCertificate)
					} else {
						if db.NewRecord(&dbCertificate) {
							db.Create(&dbCertificate)
						} else {
							log.Println("Attempted to create certdata but already has primary key")
							log.Println("This is a bug")
							log.Printf("Certificate ID: %d\n", dbCertificate.IDint())
							log.Printf("Hetzner LB ID: %d\n", dbCertificate.HetznerID)
						}
					}
				}

			}
			// update load balancer

			hClient.LoadBalancer.UpdateService(context.Background(), lb, s.ListenPort, hcloud.LoadBalancerUpdateServiceOpts{
				Protocol:        s.Protocol,
				DestinationPort: &s.DestinationPort,
				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
					CookieName:     &s.HTTP.CookieName,
					Certificates:   s.HTTP.Certificates,
					CookieLifetime: &s.HTTP.CookieLifetime,
					RedirectHTTP:   &s.HTTP.RedirectHTTP,
					StickySessions: &s.HTTP.StickySessions,
				},
				HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
					Protocol: s.HealthCheck.Protocol,
					HTTP: &hcloud.LoadBalancerUpdateServiceOptsHealthCheckHTTP{
						Domain:      &s.HealthCheck.HTTP.Domain,
						Path:        &s.HealthCheck.HTTP.Path,
						Response:    &s.HealthCheck.HTTP.Response,
						StatusCodes: s.HealthCheck.HTTP.StatusCodes,
						TLS:         &s.HealthCheck.HTTP.TLS,
					},
					Interval: &s.HealthCheck.Interval,
					Port:     &s.HealthCheck.Port,
					Retries:  &s.HealthCheck.Retries,
					Timeout:  &s.HealthCheck.Timeout,
				},
				Proxyprotocol: &s.Proxyprotocol,
			})

			// delete old certs

			for _, cert := range certsToDelete {
				hClient.Certificate.Delete(context.Background(), &cert)
			}
		}

		job.LastRun = time.Now()
		job.NextRun = time.Now().Add(time.Duration(job.IntervalHours) * time.Hour)

		err = db.Save(&job).Error
		if err != nil {
			log.Printf("failed to update cert renew job: %d\n", job.IDint())
			log.Println(err)
		}
	}

	log.Println("done running certrenew jobs")
}
