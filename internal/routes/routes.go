package routes

import (
	"github.com/korgx9/safar-backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {

	r := gin.Default()

	r.GET("/health", handlers.HealthCheck)

	return r
}
