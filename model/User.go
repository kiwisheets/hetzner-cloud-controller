package model

// User model
type User struct {
	SoftDelete
	Username string `gorm:"UNIQUE_INDEX:idx_username"`
	Password string
}
