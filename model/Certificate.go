package model

import "github.com/go-acme/lego/v3/certificate"

type Certificate struct {
	Model

	// The lego ACME certificate data
	certificate.Resource

	// The ID of the certificate in Hetzner
	HetznerID int `gorm:"UNIQUE_INDEX:idx_cert_h_id"`
}
