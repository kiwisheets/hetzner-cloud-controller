package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/go-acme/lego/v3/lego"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/jinzhu/gorm"
	v1 "github.com/kiwisheets/hetzner-cloud-controller/routes/v1"
)

func SetupRoutes(r *gin.RouterGroup, hClient *hcloud.Client, lClient *lego.Client, db *gorm.DB) {
	rV1 := r.Group("/v1")
	{
		v1.SetupRoutes(rV1, hClient, lClient, db)
	}
}
