// Package v1 holds the v1 api routes
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/go-acme/lego/v3/lego"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/jinzhu/gorm"
)

// SetupRoutes sets up the v1 API routes
func SetupRoutes(r *gin.RouterGroup, hClient *hcloud.Client, lClient *lego.Client, db *gorm.DB) {
	r.GET("/servers", serversEndpoint(hClient))
	r.DELETE("/server", deleteServerEndpoint(hClient))
	r.POST("/server", createServerEndpoint(hClient, db))

	r.GET("/servertypes", serverTypesEndpoint(hClient))

	r.GET("/certificates", certificatesEndpoint(hClient))

	r.GET("/loadbalancers", loadbalancersEndpoint(hClient))
	r.POST("/loadbalancer", loadbalancerCreateEndpoint(hClient, db))
	// r.POST("/assigncertificate", loadbalancerAssignCert(hClient, db))

	r.GET("/loadbalancertypes", loadbalancerTypesEndpoint(hClient))

	job := r.Group("/job")
	{
		job.GET("/certrenews", certRenewsEndpoint(db))
	}

	r.GET("/images", imagesEndpoint(hClient))
}
