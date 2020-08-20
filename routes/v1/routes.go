// Package v1 holds the v1 api routes
package v1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-acme/lego/v3/lego"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/jinzhu/gorm"
	"github.com/kiwisheets/hetzner-cloud-controller/auth"
	"github.com/kiwisheets/hetzner-cloud-controller/model"
	"github.com/maxtroughear/goenv"
)

// SetupRoutes sets up the v1 API routes
func SetupRoutes(r *gin.RouterGroup, hClient *hcloud.Client, lClient *lego.Client, db *gorm.DB) {
	r.POST("/login", loginEndpoint(db))

	authRoute := r.Group("/")

	authRoute.Use(verifyToken())

	authRoute.GET("/servers", serversEndpoint(hClient))
	authRoute.DELETE("/server", deleteServerEndpoint(hClient))
	authRoute.POST("/server", createServerEndpoint(hClient, db))

	authRoute.GET("/servertypes", serverTypesEndpoint(hClient))

	authRoute.GET("/certificates", certificatesEndpoint(hClient))

	authRoute.GET("/loadbalancers", loadbalancersEndpoint(hClient))
	authRoute.POST("/loadbalancer", loadbalancerCreateEndpoint(hClient, db))
	// r.POST("/assigncertificate", loadbalancerAssignCert(hClient, db))

	authRoute.GET("/loadbalancertypes", loadbalancerTypesEndpoint(hClient))

	job := authRoute.Group("/job")
	{
		job.GET("/certrenews", certRenewsEndpoint(db))
	}

	authRoute.GET("/images", imagesEndpoint(hClient))
}

func loginEndpoint(db *gorm.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		username, hasUsername := c.GetPostForm("username")
		password, hasPassword := c.GetPostForm("password")

		if !hasUsername || !hasPassword {
			c.JSON(http.StatusBadRequest, jsonError{
				Error: "no username or password specified",
			})
			return
		}

		var user model.User
		user.Username = username

		err := db.Where(&user).First(&user).Error

		if err != nil {
			c.JSON(http.StatusBadRequest, jsonError{
				Error: "no username or password specified",
			})
			return
		}

		if auth.VerifyPassword(&user, password) {
			token, err := auth.LoginUser(&user, goenv.MustGet("JWT_SECRET_KEY"))

			if err != nil {
				c.JSON(http.StatusBadRequest, jsonError{
					Error: "no username or password specified",
				})
			}

			c.JSON(http.StatusOK, token)
		} else {
			c.JSON(http.StatusBadRequest, jsonError{
				Error: "username or password incorrect",
			})
			return
		}
	}
}

func verifyToken() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		// split

		token, err := splitToken(authHeader)

		if err != nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// process and validate jwt token
		_, err = auth.ValidateTokenAndGetUserID(token, goenv.MustGet("JWT_SECRET_KEY"))
		if err != nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

func splitToken(header string) (string, error) {
	splitToken := strings.Split(header, "Bearer")

	if len(splitToken) != 2 || len(splitToken[1]) < 2 {
		return "", fmt.Errorf("bad token format")
	}

	return strings.TrimSpace(splitToken[1]), nil
}
