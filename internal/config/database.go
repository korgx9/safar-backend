package config

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/models"
)

func InitDB(cfg Config) *gorm.DB {
	db, err := gorm.Open(postgres.Open(cfg.DBUrl), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	log.Println("Database connected")

	err = db.AutoMigrate(
		&models.User{},
		&models.OTPCode{},
		&models.Vehicle{},
	)
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	log.Println("Database migrated")

	return db
}
