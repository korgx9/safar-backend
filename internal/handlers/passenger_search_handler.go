package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/models"
)

const nextTripWarningMessage = "You selected not the first vehicle in queue. Waiting time may be longer."

type PassengerSearchHandler struct {
	db *gorm.DB
}

func NewPassengerSearchHandler(db *gorm.DB) *PassengerSearchHandler {
	return &PassengerSearchHandler{db: db}
}

// Search godoc
// @Summary Search trips queue
// @Description Creates a search session and returns the first matching trip in queue.
// @Tags Passenger Search
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.SearchTripsRequest true "Search trips request"
// @Success 200 {object} models.SearchTripResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /passenger/search [post]
func (h *PassengerSearchHandler) Search(c *gin.Context) {
	passengerIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	passengerID, ok := passengerIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return
	}

	var req models.SearchTripsRequest
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

	if req.RequestedSeats < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "requested_seats must be at least 1"})
		return
	}

	requestDayStart, _ := dayRangeUTC(req.TripDate)
	todayStart, _ := dayRangeUTC(time.Now())
	if requestDayStart.Before(todayStart) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trip_date must not be in the past"})
		return
	}

	session := models.SearchSession{
		PassengerID:    passengerID,
		Origin:         origin,
		Destination:    destination,
		TripDate:       requestDayStart,
		RequestedSeats: req.RequestedSeats,
		CurrentOffset:  0,
	}

	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create search session"})
		return
	}

	trip, err := h.findTripByOffset(session, 0)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no trips found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search trips"})
		return
	}

	c.JSON(http.StatusOK, models.SearchTripResponse{
		SearchSessionID: session.ID,
		Trip:            *trip,
		IsFirstInQueue:  true,
		Warning:         nil,
	})
}

// Next godoc
// @Summary Get next trip in queue
// @Description Returns the next matching trip from an existing search session queue.
// @Tags Passenger Search
// @Produce json
// @Security BearerAuth
// @Param sessionId path int true "Search session ID"
// @Success 200 {object} models.SearchTripResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /passenger/search/{sessionId}/next [post]
func (h *PassengerSearchHandler) Next(c *gin.Context) {
	passengerIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	passengerID, ok := passengerIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return
	}

	sessionIDValue := c.Param("sessionId")
	sessionID, err := strconv.ParseUint(sessionIDValue, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	var session models.SearchSession
	if err := h.db.First(&session, uint(sessionID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "search session not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch search session"})
		return
	}

	if session.PassengerID != passengerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "search session does not belong to current user"})
		return
	}

	nextOffset := session.CurrentOffset + 1
	trip, err := h.findTripByOffset(session, nextOffset)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no more trips in queue"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search next trip"})
		return
	}

	session.CurrentOffset = nextOffset
	if err := h.db.Save(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update search session"})
		return
	}

	warning := nextTripWarningMessage
	c.JSON(http.StatusOK, models.SearchTripResponse{
		SearchSessionID: session.ID,
		Trip:            *trip,
		IsFirstInQueue:  false,
		Warning:         &warning,
	})
}

func (h *PassengerSearchHandler) findTripByOffset(session models.SearchSession, offset int) (*models.TripOffer, error) {
	startOfDay, endOfDay := dayRangeUTC(session.TripDate)

	var trip models.TripOffer
	err := h.db.
		Where("status = ?", models.TripStatusActive).
		Where("origin = ?", session.Origin).
		Where("destination = ?", session.Destination).
		Where("trip_date >= ? AND trip_date < ?", startOfDay, endOfDay).
		Where("available_seats >= ?", session.RequestedSeats).
		Order("created_at ASC").
		Order("id ASC").
		Offset(offset).
		Limit(1).
		First(&trip).Error
	if err != nil {
		return nil, err
	}

	return &trip, nil
}

func dayRangeUTC(date time.Time) (time.Time, time.Time) {
	d := date.UTC()
	start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	return start, start.Add(24 * time.Hour)
}
