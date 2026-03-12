package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/korgx9/safar-backend/internal/models"
)

type bookingHTTPError struct {
	Status  int
	Message string
}

func (e *bookingHTTPError) Error() string {
	return e.Message
}

type BookingHandler struct {
	db *gorm.DB
}

func NewBookingHandler(db *gorm.DB) *BookingHandler {
	return &BookingHandler{db: db}
}

func (h *BookingHandler) Create(c *gin.Context) {
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

	var req models.CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RequestedSeats < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "requested_seats must be at least 1"})
		return
	}

	var createdBooking models.Booking
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var trip models.TripOffer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&trip, req.TripOfferID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &bookingHTTPError{
					Status:  http.StatusNotFound,
					Message: "trip not found",
				}
			}
			return err
		}

		if trip.Status != models.TripStatusActive {
			return &bookingHTTPError{
				Status:  http.StatusBadRequest,
				Message: "trip is not active",
			}
		}

		if trip.AvailableSeats < req.RequestedSeats {
			return &bookingHTTPError{
				Status:  http.StatusBadRequest,
				Message: "not enough seats",
			}
		}

		booking := models.Booking{
			TripOfferID:         req.TripOfferID,
			PassengerID:         passengerID,
			RequestedSeats:      req.RequestedSeats,
			Status:              models.BookingStatusConfirmed,
			WarningAcknowledged: req.WarningAcknowledged,
		}

		if err := tx.Create(&booking).Error; err != nil {
			return err
		}

		trip.AvailableSeats -= req.RequestedSeats
		if trip.AvailableSeats == 0 {
			trip.Status = models.TripStatusFull
		}

		if err := tx.Save(&trip).Error; err != nil {
			return err
		}

		createdBooking = booking
		return nil
	})
	if err != nil {
		var httpErr *bookingHTTPError
		if errors.As(err, &httpErr) {
			c.JSON(httpErr.Status, gin.H{"error": httpErr.Message})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create booking"})
		return
	}

	c.JSON(http.StatusCreated, createdBooking)
}

func (h *BookingHandler) ListMy(c *gin.Context) {
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

	var bookings []models.Booking
	if err := h.db.Where("passenger_id = ?", passengerID).Order("created_at DESC").Find(&bookings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bookings"})
		return
	}

	c.JSON(http.StatusOK, bookings)
}

func (h *BookingHandler) Cancel(c *gin.Context) {
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

	bookingIDValue := c.Param("id")
	bookingID, err := strconv.ParseUint(bookingIDValue, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	var updatedBooking models.Booking
	err = h.db.Transaction(func(tx *gorm.DB) error {
		var booking models.Booking
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&booking, uint(bookingID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &bookingHTTPError{
					Status:  http.StatusNotFound,
					Message: "booking not found",
				}
			}
			return err
		}

		if booking.PassengerID != passengerID {
			return &bookingHTTPError{
				Status:  http.StatusForbidden,
				Message: "booking does not belong to current user",
			}
		}

		if booking.Status == models.BookingStatusCancelled {
			return &bookingHTTPError{
				Status:  http.StatusBadRequest,
				Message: "booking already cancelled",
			}
		}

		var trip models.TripOffer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&trip, booking.TripOfferID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &bookingHTTPError{
					Status:  http.StatusNotFound,
					Message: "trip not found",
				}
			}
			return err
		}

		booking.Status = models.BookingStatusCancelled
		if err := tx.Save(&booking).Error; err != nil {
			return err
		}

		trip.AvailableSeats += booking.RequestedSeats
		if trip.Status == models.TripStatusFull && trip.AvailableSeats > 0 {
			trip.Status = models.TripStatusActive
		}

		if err := tx.Save(&trip).Error; err != nil {
			return err
		}

		updatedBooking = booking
		return nil
	})
	if err != nil {
		var httpErr *bookingHTTPError
		if errors.As(err, &httpErr) {
			c.JSON(httpErr.Status, gin.H{"error": httpErr.Message})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel booking"})
		return
	}

	c.JSON(http.StatusOK, updatedBooking)
}
