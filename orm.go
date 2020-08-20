package main

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/kiwisheets/hetzner-cloud-controller/auth"
	"github.com/kiwisheets/hetzner-cloud-controller/model"
	"github.com/maxtroughear/goenv"
)

type DBConfig struct {
	Host     string
	User     string
	Password string
	Database string
}

func Init(cfg DBConfig) *gorm.DB {
	connectionString := constructConnectionString(cfg)

	time.Sleep(5 * time.Second)

	db, err := gorm.Open("postgres", connectionString)
	db.BlockGlobalUpdate(true)

	if err != nil {
		log.Println("Failed to connect to db")
		log.Println(connectionString)
		panic(err)
	}

	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(20)
	db.DB().SetConnMaxLifetime(time.Hour * 1)

	if goenv.CanGet("ENVIRONMENT", "production") == "development" {
		// clear db
		// note: does not drop tables used for many2many relationships, please bare this in mind!
		dropAll(db)
	}

	migrateAll(db)

	seedDefaultUser(db)

	return db
}

func migrateAll(db *gorm.DB) {
	db.AutoMigrate(&model.User{})
	db.AutoMigrate(&model.CertRenewJob{})
	db.AutoMigrate(&model.PrivateKey{})
	db.AutoMigrate(&model.Certificate{})
}

func dropAll(db *gorm.DB) {
	db.DropTableIfExists(&model.User{})
	db.DropTableIfExists(&model.CertRenewJob{})
	db.DropTableIfExists(&model.PrivateKey{})
	db.DropTableIfExists(&model.Certificate{})
}

func seedDefaultUser(db *gorm.DB) {
	hash, err := auth.HashPassword(goenv.MustGetSecretFromEnv("DEFAULT_PASSWORD"))

	if err != nil {
		log.Fatalf("Failed to hash default password, please check it is at least 6 characters")
	}

	var user model.User

	err = db.Where(model.User{
		Username: goenv.MustGetSecretFromEnv("DEFAULT_USERNAME"),
	}).Attrs(model.User{
		Password: hash,
	}).FirstOrCreate(&user).Error

	if err != nil {
		log.Println("Error creating default user. May already exist")
	}
}

func constructConnectionString(cfg DBConfig) string {
	return "host=" + cfg.Host + " user=" + cfg.User + " password=" + cfg.Password + " dbname=" + cfg.Database + " sslmode=disable"
}
