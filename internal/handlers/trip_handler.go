package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/models"
)

type TripHandler struct {
	db *gorm.DB
}

func NewTripHandler(db *gorm.DB) *TripHandler {
	return &TripHandler{db: db}
}

func (h *TripHandler) Create(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	driverID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return
	}

	var req models.CreateTripRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	origin := strings.TrimSpace(req.Origin)
	destination := strings.TrimSpace(req.Destination)

	if origin == "" || destination == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "origin and destination are required"})
		return
	}

	if strings.EqualFold(origin, destination) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "origin and destination must be different"})
		return
	}

	if req.TripDate.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trip_date must not be in the past"})
		return
	}

	if req.AvailableSeats < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "available_seats must be at least 1"})
		return
	}

	var vehicle models.Vehicle
	if err := h.db.First(&vehicle, req.VehicleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch vehicle"})
		return
	}

	if vehicle.UserID != driverID {
		c.JSON(http.StatusForbidden, gin.H{"error": "vehicle does not belong to current user"})
		return
	}

	if req.AvailableSeats > vehicle.SeatsTotal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "available_seats cannot exceed vehicle seats_total"})
		return
	}

	trip := models.TripOffer{
		DriverID:       driverID,
		VehicleID:      vehicle.ID,
		Origin:         origin,
		Destination:    destination,
		TripDate:       req.TripDate,
		AvailableSeats: req.AvailableSeats,
		TotalSeats:     vehicle.SeatsTotal,
		Status:         models.TripStatusActive,
	}

	if err := h.db.Create(&trip).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create trip"})
		return
	}

	c.JSON(http.StatusCreated, trip)
}

func (h *TripHandler) ListMy(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	driverID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return
	}

	var trips []models.TripOffer
	if err := h.db.Where("driver_id = ?", driverID).Order("created_at DESC").Find(&trips).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trips"})
		return
	}

	c.JSON(http.StatusOK, trips)
}
