package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

func certificatesEndpoint(client *hcloud.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		certs, err := client.Certificate.All(c.Request.Context())

		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
		}
		if certs != nil {
			c.JSON(http.StatusOK, certs)
		} else {
			c.JSON(http.StatusInternalServerError, nil)
		}
	}
}

func updateCertificateEndpoint(client *hcloud.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		// check if cert able to be updated

	}
}
