package handlers

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/korgx9/safar-backend/internal/models"
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

// CleanupExpiredTrips godoc
// @Summary Cleanup expired trips
// @Description Marks trips older than 7 days after trip_date as expired.
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.CleanupExpiredTripsResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /admin/cleanup/expired-trips [post]
func (h *AdminHandler) CleanupExpiredTrips(c *gin.Context) {
	expiredTripsCount, err := h.cleanupService.ExpireOldTrips()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cleanup expired trips"})
		return
	}

	c.JSON(http.StatusOK, models.CleanupExpiredTripsResponse{
		Message:           "cleanup completed",
		ExpiredTripsCount: expiredTripsCount,
	})
}
