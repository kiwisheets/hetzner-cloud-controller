package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/kiwisheets/hetzner-cloud-controller/model"
)

func certRenewsEndpoint(db *gorm.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		var jobs []model.CertRenewJob
		err := db.Find(&jobs).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, jsonError{
				Error: "Error getting jobs",
			})
			return
		}

		c.JSON(http.StatusOK, jobs)
	}
}
