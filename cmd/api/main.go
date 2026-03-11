package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/korgx9/safar-backend/internal/config"
	"github.com/korgx9/safar-backend/internal/routes"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	cfg := config.LoadConfig()

	db := config.InitDB(cfg)

	router := routes.SetupRouter(db)

	log.Println("Server starting on port", cfg.Port)

	router.Run(":" + cfg.Port)
}
