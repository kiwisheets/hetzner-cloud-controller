package model

type PrivateKey struct {
	Model
	Name      string `gorm:"UNIQUE_INDEX:idx_private_key_name"`
	PublicKey string
	Key       string
}
