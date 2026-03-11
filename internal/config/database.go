package config

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(cfg Config) *gorm.DB {

	db, err := gorm.Open(postgres.Open(cfg.DBUrl), &gorm.Config{})

	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	log.Println("Database connected")

	return db
}
