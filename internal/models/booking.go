package models

import "time"

const (
	BookingStatusConfirmed = "confirmed"
	BookingStatusCancelled = "cancelled"
)

type Booking struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	TripOfferID         uint      `gorm:"not null;index" json:"trip_offer_id"`
	PassengerID         uint      `gorm:"not null;index" json:"passenger_id"`
	RequestedSeats      int       `gorm:"not null" json:"requested_seats"`
	Status              string    `gorm:"type:varchar(20);not null;default:'confirmed'" json:"status"`
	WarningAcknowledged bool      `gorm:"not null;default:false" json:"warning_acknowledged"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CreateBookingRequest struct {
	TripOfferID         uint `json:"trip_offer_id" binding:"required"`
	RequestedSeats      int  `json:"requested_seats" binding:"required,min=1"`
	WarningAcknowledged bool `json:"warning_acknowledged"`
}
