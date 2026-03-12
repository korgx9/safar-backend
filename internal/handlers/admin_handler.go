package handlers

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/korgx9/safar-backend/internal/services"
)

type AdminHandler struct {
	cleanupService *services.TripCleanupService
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		cleanupService: services.NewTripCleanupService(db),
	}
}

func (h *AdminHandler) CleanupExpiredTrips(c *gin.Context) {
	expiredTripsCount, err := h.cleanupService.ExpireOldTrips()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cleanup expired trips"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "cleanup completed",
		"expired_trips_count": expiredTripsCount,
	})
}
