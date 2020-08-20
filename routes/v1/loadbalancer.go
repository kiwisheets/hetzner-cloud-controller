package v1

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/jinzhu/gorm"
	"github.com/kiwisheets/hetzner-cloud-controller/key"
	"github.com/kiwisheets/hetzner-cloud-controller/model"
)

func loadbalancersEndpoint(hClient *hcloud.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		lbs, err := hClient.LoadBalancer.All(c.Request.Context())

		for i, lb := range lbs {
			for j, s := range lb.Services {
				for k, cert := range s.HTTP.Certificates {
					cert, _, err = hClient.Certificate.GetByID(c.Request.Context(), cert.ID)
					if err != nil {
						log.Println(err)
					} else {
						s.HTTP.Certificates[k] = cert
					}
				}
				lb.Services[j] = s
			}
			lbs[i] = lb
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
		}
		if lbs != nil {
			c.JSON(http.StatusOK, lbs)
		} else {
			c.JSON(http.StatusInternalServerError, nil)
		}
	}
}

func loadbalancerCreateEndpoint(hClient *hcloud.Client, db *gorm.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		name := c.PostForm("name")
		typeString := c.PostForm("type")
		domains := c.PostFormArray("domain_names")

		if name == "" {
			c.JSON(http.StatusBadRequest, jsonError{
				Error: "name not specified",
			})
			return
		}

		if typeString == "" {
			c.JSON(http.StatusBadRequest, jsonError{
				Error: "type not specified",
			})
			return
		}

		typeID, err := strconv.ParseInt(typeString, 10, strconv.IntSize)

		if err != nil {
			c.JSON(http.StatusBadRequest, jsonError{
				Error: "type not an int",
			})
			return
		}

		// enable, err := strconv.ParseBool(enableString)

		// if err != nil {
		// 	c.JSON(http.StatusBadRequest, jsonError{
		// 		Error: "enable not a bool",
		// 	})
		// 	return
		// }

		// lb, _, err := hClient.LoadBalancer.GetByID(c.Request.Context(), int(id))

		createCertConfig := createDummyCert(domains)

		createCertConfig.Name = name + "_dummy_cert"

		dummyCert, _, err := hClient.Certificate.Create(c.Request.Context(), *createCertConfig)

		if dummyCert == nil {
			log.Println("no cert")
			log.Println(err)
		}

		lb, _, err := hClient.LoadBalancer.Create(c.Request.Context(), hcloud.LoadBalancerCreateOpts{
			Name: name,
			Algorithm: &hcloud.LoadBalancerAlgorithm{
				Type: hcloud.LoadBalancerAlgorithmTypeRoundRobin,
			},
			LoadBalancerType: &hcloud.LoadBalancerType{
				ID: int(typeID),
			},
			Services: []hcloud.LoadBalancerCreateOptsService{
				{
					Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
					ListenPort:      hcloud.Int(443),
					DestinationPort: hcloud.Int(80),
					HTTP: &hcloud.LoadBalancerCreateOptsServiceHTTP{
						CookieName:     hcloud.String("HCLBSTICKY"),
						CookieLifetime: hcloud.Duration(300 * time.Second),
						Certificates: []*hcloud.Certificate{
							dummyCert,
						},
						RedirectHTTP:   hcloud.Bool(true),
						StickySessions: hcloud.Bool(false),
					},
					HealthCheck: &hcloud.LoadBalancerCreateOptsServiceHealthCheck{
						Protocol: hcloud.LoadBalancerServiceProtocolHTTP,
						HTTP: &hcloud.LoadBalancerCreateOptsServiceHealthCheckHTTP{
							Domain: hcloud.String(""),
							Path:   hcloud.String("/"),
						},
						Interval: hcloud.Duration(15 * time.Second),
						Port:     hcloud.Int(80),
						Retries:  hcloud.Int(3),
						Timeout:  hcloud.Duration(10 * time.Second),
					},
					Proxyprotocol: hcloud.Bool(false),
				},
			},
			PublicInterface: hcloud.Bool(true),
			Location: &hcloud.Location{
				Name: "nbg1",
			},
		})

		if err != nil {
			c.JSON(http.StatusBadRequest, jsonError{
				Error:       "load balancer not created",
				ErrorObject: err,
			})
			hClient.Certificate.Delete(c.Request.Context(), dummyCert)
			return
		}
		log.Println("creating loadbalancer")

		_, actionError := hClient.Action.WatchProgress(c.Request.Context(), lb.Action)

		// should wait for error to return
		err = <-actionError

		if err != nil {
			c.JSON(http.StatusBadRequest, jsonError{
				Error:       "error creating loadbalancer",
				ErrorObject: err,
			})
			hClient.Certificate.Delete(c.Request.Context(), dummyCert)
			return
		}

		// if enable {
		// create dummy certs

		// if dummyCert != nil {
		// 	for _, s := range lb.LoadBalancer.Services {
		// 		if s.Protocol == "https" {
		// 			s.HTTP.Certificates = append(s.HTTP.Certificates, dummyCert)
		// 		}
		// 		hClient.LoadBalancer.UpdateService(context.Background(), lb.LoadBalancer, s.ListenPort, hcloud.LoadBalancerUpdateServiceOpts{
		// 			Protocol:        s.Protocol,
		// 			DestinationPort: &s.DestinationPort,
		// 			HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
		// 				CookieName:     &s.HTTP.CookieName,
		// 				Certificates:   s.HTTP.Certificates,
		// 				CookieLifetime: &s.HTTP.CookieLifetime,
		// 				RedirectHTTP:   &s.HTTP.RedirectHTTP,
		// 				StickySessions: &s.HTTP.StickySessions,
		// 			},
		// 			HealthCheck: &hcloud.LoadBalancerUpdateServiceOptsHealthCheck{
		// 				Protocol: s.HealthCheck.Protocol,
		// 				HTTP: &hcloud.LoadBalancerUpdateServiceOptsHealthCheckHTTP{
		// 					Domain:      &s.HealthCheck.HTTP.Domain,
		// 					Path:        &s.HealthCheck.HTTP.Path,
		// 					Response:    &s.HealthCheck.HTTP.Response,
		// 					StatusCodes: s.HealthCheck.HTTP.StatusCodes,
		// 					TLS:         &s.HealthCheck.HTTP.TLS,
		// 				},
		// 				Interval: &s.HealthCheck.Interval,
		// 				Port:     &s.HealthCheck.Port,
		// 				Retries:  &s.HealthCheck.Retries,
		// 				Timeout:  &s.HealthCheck.Timeout,
		// 			},
		// 			Proxyprotocol: &s.Proxyprotocol,
		// 		})
		// 	}
		// }

		// attempt to create job
		err = db.Create(&model.CertRenewJob{
			LoadbalancerID: lb.LoadBalancer.ID,
			IntervalHours:  24,
			LastRun:        time.Now(),
			NextRun:        time.Now(),
		}).Error

		if err != nil {
			c.JSON(http.StatusBadRequest, jsonError{
				Error:       "Auto renew already enabled",
				ErrorObject: err,
			})
			return
		}

		// finally get loadbalancer details
		lb.LoadBalancer, _, err = hClient.LoadBalancer.GetByID(c.Request.Context(), lb.LoadBalancer.ID)

		if err != nil {
			log.Println("Created loadbalancer. But unable to retrieve updated info and return it. Returning incomplete loadbalancer data")
		}

		c.JSON(http.StatusOK, lb.LoadBalancer)
	}
}

func loadbalancerTypesEndpoint(hClient *hcloud.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		types, err := hClient.LoadBalancerType.All(c.Request.Context())

		if err != nil {
			c.JSON(http.StatusInternalServerError, jsonError{
				Error: "Unable to retrieve types",
			})
			return
		}

		c.JSON(http.StatusOK, types)
	}
}

func createDummyCert(domainNames []string) *hcloud.CertificateCreateOpts {
	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Hetzner Cloud Controller Dummy Certificate"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              domainNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, key.PublicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}
	out := &bytes.Buffer{}
	// cert
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certificate := out.String()
	out.Reset()

	// private key
	pem.Encode(out, key.PemBlockForKey(priv))
	privateKey := out.String()

	return &hcloud.CertificateCreateOpts{
		Certificate: certificate,
		PrivateKey:  privateKey,
	}

}
