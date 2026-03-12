package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/korgx9/safar-backend/internal/models"
)

// HealthCheck godoc
// @Summary Health check
// @Description Returns backend health status.
// @Tags Health
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{Status: "ok"})
}
