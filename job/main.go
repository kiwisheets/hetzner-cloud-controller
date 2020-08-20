package job

import (
	"log"
	"sync"
	"time"

	"github.com/go-acme/lego/v3/lego"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/jinzhu/gorm"
)

var wg sync.WaitGroup

var mux sync.Mutex
var run bool

func Init(hClient *hcloud.Client, lClient *lego.Client, db *gorm.DB) {
	// setup goroutine
	log.Println("starting job system")

	run = true
	wg.Add(1)
	go loop(hClient, lClient, db)
}

func Shutdown() {
	log.Println("shutting down job system")

	mux.Lock()
	run = false
	mux.Unlock()

	wg.Wait()
}

func loop(hClient *hcloud.Client, lClient *lego.Client, db *gorm.DB) {
	defer wg.Done()

	for {

		processCertRenews(hClient, lClient, db)
		// processCreateCertJobs(hClient, lClient, db)

		mux.Lock()
		if !run {
			mux.Unlock()
			log.Println("breaking")
			break
		}
		mux.Unlock()

		time.Sleep(1 * time.Minute)
	}
}
