package job

import (
	"log"
	"time"

	"github.com/go-acme/lego/v3/lego"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/jinzhu/gorm"
	"github.com/kiwisheets/hetzner-cloud-controller/model"
)

func processCreateCertJobs(hClient *hcloud.Client, lClient *lego.Client, db *gorm.DB) {
	var jobs []model.CreateCertJob

	db.Where("next_run < ?", time.Now()).Find(&jobs)

	log.Println("running createcert jobs")

	for _, job := range jobs {
		log.Println(job.IDint)
	}

	log.Println("done running createcert jobs")
}
