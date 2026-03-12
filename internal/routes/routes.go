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
	passengerSearchHandler := handlers.NewPassengerSearchHandler(db)
	bookingHandler := handlers.NewBookingHandler(db)

	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtSecret))
	{
		protected.GET("/me", userHandler.Me)

		vehicles := protected.Group("/vehicles")
		{
			vehicles.POST("", vehicleHandler.Create)
			vehicles.GET("", vehicleHandler.List)
		}

		driver := protected.Group("/driver")
		{
			driver.POST("/trips", tripHandler.Create)
			driver.GET("/trips", tripHandler.ListMy)
		}

		passenger := protected.Group("/passenger")
		{
			passenger.POST("/search", passengerSearchHandler.Search)
			passenger.POST("/search/:sessionId/next", passengerSearchHandler.Next)
		}

		bookings := protected.Group("/bookings")
		{
			bookings.POST("", bookingHandler.Create)
			bookings.GET("/my", bookingHandler.ListMy)
			bookings.PATCH("/:id/cancel", bookingHandler.Cancel)
		}
	}

	return r
}
