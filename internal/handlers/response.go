package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/korgx9/safar-backend/internal/models"
)

func respondWithError(c *gin.Context, status int, message string) {
	c.JSON(status, models.ErrorResponse{Error: message})
}
