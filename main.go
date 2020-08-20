package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-acme/lego/v3/lego"
	"github.com/go-acme/lego/v3/providers/dns/namecheap"
	"github.com/go-acme/lego/v3/registration"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/joho/godotenv"
	"github.com/kiwisheets/hetzner-cloud-controller/job"
	"github.com/kiwisheets/hetzner-cloud-controller/key"
	"github.com/kiwisheets/hetzner-cloud-controller/model"
	"github.com/kiwisheets/hetzner-cloud-controller/routes"
	"github.com/maxtroughear/goenv"
)

func main() {
	godotenv.Load()

	db := Init(DBConfig{
		Host:     goenv.MustGet("POSTGRES_HOST"),
		Database: goenv.MustGet("POSTGRES_DB"),
		User:     goenv.MustGet("POSTGRES_USER"),
		Password: goenv.MustGetSecretFromEnv("POSTGRES_PASSWORD"),
	})

	hClient := hcloud.NewClient(hcloud.WithToken(goenv.MustGet("HETZNER_API_KEY")))

	if hClient == nil {
		log.Fatalf("Failed to connect to Hetzner, check your API key")
	}

	var userKey model.PrivateKey
	var privateKey *ecdsa.PrivateKey

	keyName := "acme_key"

	// get private key from db
	if db.Where(model.PrivateKey{Name: keyName}).First(&userKey).RecordNotFound() {
		// Create a user. New accounts need an email and private key to start.
		keys, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			log.Println("Unable to generate private key")
			log.Fatal(err)
		}

		privateKey = keys

		userKey.Name = keyName
		userKey.Key, userKey.PublicKey = key.Encode(privateKey, &privateKey.PublicKey)

		// save key
		db.Save(&userKey)
	} else {
		log.Println("loaded ACME private key from DB")
		privateKey, _ = key.Decode(userKey.Key, userKey.PublicKey)
	}

	certUser := CertUser{
		Email: goenv.MustGet("EMAIL"),
		key:   privateKey,
	}

	certConfig := lego.NewConfig(&certUser)

	certConfig.CADirURL = goenv.MustGet("ACME_DIRECTORY")

	lClient, err := lego.NewClient(certConfig)

	if err != nil {
		log.Println("Failed to create lego client, check env variables")
		log.Fatal(err)
	}

	providerConfig, err := namecheap.NewDNSProvider()

	if err != nil {
		log.Println("Unable to configure namecheap DNS provider for lets encrypt")
		log.Fatal(err)
	}

	lClient.Challenge.SetDNS01Provider(providerConfig)

	reg, err := lClient.Registration.ResolveAccountByKey()
	if err != nil {
		reg, err = lClient.Registration.Register(registration.RegisterOptions{
			TermsOfServiceAgreed: true,
		})

		if err != nil {
			log.Print("Unable to register new Let's Encrypt user")
			log.Fatal(err)
		}
	}

	certUser.Registration = reg

	router := gin.Default()

	baseRoute := router.Group("/")

	routes.SetupRoutes(baseRoute, hClient, lClient, db)

	job.Init(hClient, lClient, db)

	router.Run()

	job.Shutdown()
}

type CertUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *CertUser) GetEmail() string {
	return u.Email
}
func (u CertUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *CertUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}
