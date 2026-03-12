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

// Create godoc
// @Summary Create trip offer
// @Description Creates a driver trip offer using one of current user's vehicles.
// @Tags Driver Trips
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateTripRequest true "Create trip request"
// @Success 201 {object} models.TripOffer
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /driver/trips [post]
func (h *TripHandler) Create(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		respondWithError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	driverID, ok := userIDValue.(uint)
	if !ok {
		respondWithError(c, http.StatusUnauthorized, "invalid user id in context")
		return
	}

	var req models.CreateTripRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	origin := strings.TrimSpace(req.Origin)
	destination := strings.TrimSpace(req.Destination)

	if origin == "" || destination == "" {
		respondWithError(c, http.StatusBadRequest, "origin and destination are required")
		return
	}

	if strings.EqualFold(origin, destination) {
		respondWithError(c, http.StatusBadRequest, "origin and destination must be different")
		return
	}

	if req.TripDate.Before(time.Now()) {
		respondWithError(c, http.StatusBadRequest, "trip_date must not be in the past")
		return
	}

	if req.AvailableSeats < 1 {
		respondWithError(c, http.StatusBadRequest, "available_seats must be at least 1")
		return
	}

	var vehicle models.Vehicle
	if err := h.db.First(&vehicle, req.VehicleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondWithError(c, http.StatusNotFound, "vehicle not found")
			return
		}

		respondWithError(c, http.StatusInternalServerError, "failed to fetch vehicle")
		return
	}

	if vehicle.UserID != driverID {
		respondWithError(c, http.StatusForbidden, "vehicle does not belong to current user")
		return
	}

	if req.AvailableSeats > vehicle.SeatsTotal {
		respondWithError(c, http.StatusBadRequest, "available_seats cannot exceed vehicle seats_total")
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
		respondWithError(c, http.StatusInternalServerError, "failed to create trip")
		return
	}

	c.JSON(http.StatusCreated, trip)
}

// ListMy godoc
// @Summary List my trip offers
// @Description Returns trip offers created by the authenticated driver.
// @Tags Driver Trips
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.TripOffer
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /driver/trips [get]
func (h *TripHandler) ListMy(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		respondWithError(c, http.StatusUnauthorized, "user not found in context")
		return
	}

	driverID, ok := userIDValue.(uint)
	if !ok {
		respondWithError(c, http.StatusUnauthorized, "invalid user id in context")
		return
	}

	var trips []models.TripOffer
	if err := h.db.Where("driver_id = ?", driverID).Order("created_at DESC").Find(&trips).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "failed to fetch trips")
		return
	}

	c.JSON(http.StatusOK, trips)
}
