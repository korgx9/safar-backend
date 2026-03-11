package main

import (
	"log"

	"github.com/korgx9/safar-backend/internal/config"
	"github.com/korgx9/safar-backend/internal/routes"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	cfg := config.LoadConfig()

	router := routes.SetupRouter()

	log.Println("Server starting on port", cfg.Port)

	router.Run(":" + cfg.Port)
}
