package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

func imagesEndpoint(hClient *hcloud.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		images, err := hClient.Image.All(c.Request.Context())

		if err != nil {
			c.JSON(http.StatusInternalServerError, jsonError{
				Error:       "Unable to retrieve images",
				ErrorObject: err,
			})
			return
		}

		c.JSON(http.StatusOK, images)
	}
}
