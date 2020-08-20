package v1

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/jinzhu/gorm"
	"github.com/kiwisheets/hetzner-cloud-controller/key"
	"github.com/kiwisheets/hetzner-cloud-controller/model"
)

func serversEndpoint(client *hcloud.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		servers, err := client.Server.All(c.Request.Context())

		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
		}
		if servers != nil {
			c.JSON(http.StatusOK, servers)
		} else {
			c.JSON(http.StatusInternalServerError, nil)
		}
	}
}

func serverTypesEndpoint(hClient *hcloud.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		types, err := hClient.ServerType.All(c.Request.Context())

		if err != nil {
			c.JSON(http.StatusInternalServerError, jsonError{
				Error: "Unable to retrieve types",
			})
			return
		}

		c.JSON(http.StatusOK, types)
	}
}

func deleteServerEndpoint(client *hcloud.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		id := c.PostForm("id")

		if id == "" {
			c.JSON(http.StatusBadRequest, jsonError{
				Error: "id not specified",
			})
			return
		}

		idInt, err := strconv.ParseInt(id, 10, strconv.IntSize)

		if err != nil {
			c.JSON(http.StatusBadRequest, jsonError{
				Error: "bad id",
			})
			return
		}

		_, err = client.Server.Delete(c.Request.Context(), &hcloud.Server{
			ID: int(idInt),
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, jsonError{
				Error:       "unable to delete server",
				ErrorObject: err,
			})
			return
		}

		c.JSON(http.StatusOK, nil)
	}
}

func createServerEndpoint(hClient *hcloud.Client, db *gorm.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		// params
		name := c.PostForm("name")
		typeString := c.PostForm("type")
		labelsStrings := c.PostFormArray("labels")
		userData := c.PostForm("user-data")

		// fmt.Println(userData)

		// c.JSON(http.StatusBadRequest, jsonError{
		// 	Error: "Invalid format for labels",
		// })

		// return

		labels := make(map[string]string)

		for _, l := range labelsStrings {
			label := strings.Split(l, "=")
			if len(label) != 2 {
				c.JSON(http.StatusBadRequest, jsonError{
					Error: "Invalid format for labels",
				})
				return
			}
			labels[label[0]] = label[1]
		}

		// get from database
		var dbSSHKey model.PrivateKey

		var sshKey *hcloud.SSHKey

		dbNameKey := "server_key_" + name

		if db.Where(&model.PrivateKey{
			Name: dbNameKey,
		}).First(&dbSSHKey).RecordNotFound() {
			// generate key
			// create ssh key
			privKeyString, pubKeyString, err := key.GenerateSSHkeyPair()
			if err != nil {
				c.JSON(http.StatusInternalServerError, jsonError{
					Error: "Unable to generate SSH keypair",
				})
				return
			}

			dbSSHKey.Name = dbNameKey
			dbSSHKey.Key = privKeyString
			dbSSHKey.PublicKey = pubKeyString

			// delete old key if exists
			sshKey, _, err = hClient.SSHKey.GetByName(c.Request.Context(), name)

			if err == nil && sshKey != nil {
				hClient.SSHKey.Delete(c.Request.Context(), sshKey)
			}

			// create new key
			sshKey, _, err = hClient.SSHKey.Create(c.Request.Context(), hcloud.SSHKeyCreateOpts{
				Name:      name,
				PublicKey: dbSSHKey.PublicKey,
			})

			if err != nil {
				c.JSON(http.StatusInternalServerError, jsonError{
					Error: "Unable to save SSH key to Hetzner",
				})
				return
			}

			// save to db

			if err := db.Create(&dbSSHKey).Error; err != nil {
				c.JSON(http.StatusInternalServerError, jsonError{
					Error: "Unable to save SSH key to database",
				})
				return
			}
		} else {
			// try get existing key
			var err error
			sshKey, _, err = hClient.SSHKey.GetByName(c.Request.Context(), name)

			if err != nil {
				// key does not exist
				// create key
				sshKey, _, err = hClient.SSHKey.Create(c.Request.Context(), hcloud.SSHKeyCreateOpts{
					Name:      name,
					PublicKey: sshKey.PublicKey,
				})

				if err != nil {
					c.JSON(http.StatusInternalServerError, jsonError{
						Error: "Unable to save SSH key to Hetzner",
					})
					return
				}
			}

			if sshKey.PublicKey != dbSSHKey.PublicKey {
				// error with key integrity

				log.Println("key integrity error. hetzner public key did not equal stored key")
				// potentially regenerate needs to be tested
			}
		}

		serverCreate, _, err := hClient.Server.Create(c.Request.Context(), hcloud.ServerCreateOpts{
			Name: name,
			Location: &hcloud.Location{
				Name: "nbg1",
			},
			Image: &hcloud.Image{
				Name: "ubuntu-20.04",
			},
			Labels: labels,
			SSHKeys: []*hcloud.SSHKey{
				sshKey,
			},
			ServerType: &hcloud.ServerType{
				Name: typeString,
			},
			UserData: userData,
		})

		if err != nil {
			c.JSON(http.StatusBadRequest, jsonError{
				Error:       "server not created",
				ErrorObject: err,
			})
			return
		}

		_, actionError := hClient.Action.WatchProgress(c.Request.Context(), serverCreate.Action)

		// should wait for error to return
		err = <-actionError

		if err != nil {
			c.JSON(http.StatusBadRequest, jsonError{
				Error:       "error creating server",
				ErrorObject: err,
			})
			return
		}

		// finally get loadbalancer details
		serverCreate.Server, _, err = hClient.Server.GetByID(c.Request.Context(), serverCreate.Server.ID)

		if err != nil {
			log.Println("Created server. But unable to retrieve updated info and return it. Returning incomplete server data")
		}

		c.JSON(http.StatusOK, serverCreate.Server)
	}
}
