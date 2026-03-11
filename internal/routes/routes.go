package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/handlers"
)

func SetupRouter(db *gorm.DB, jwtSecret string) *gin.Engine {

	r := gin.Default()
	r.GET("/health", handlers.HealthCheck)

	authHandler := handlers.NewAuthHandler(db, jwtSecret)

	auth := r.Group("/auth")
	{
		auth.POST("/send-otp", authHandler.SendOTP)
		auth.POST("/verify-otp", authHandler.VerifyOTP)
	}

	return r
}
