package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/handlers"
	"github.com/korgx9/safar-backend/internal/middleware"
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

	userHandler := handlers.NewUserHandler(db)
	vehicleHandler := handlers.NewVehicleHandler(db)
	tripHandler := handlers.NewTripHandler(db)

	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtSecret))
	{
		protected.GET("/me", userHandler.Me)
		protected.POST("/vehicles", vehicleHandler.Create)
		protected.GET("/vehicles", vehicleHandler.List)
		protected.POST("/driver/trips", tripHandler.Create)
		protected.GET("/driver/trips", tripHandler.ListMy)
	}

	return r
}
