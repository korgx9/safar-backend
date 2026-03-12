package models

import "time"

const (
	TripStatusActive    = "active"
	TripStatusFull      = "full"
	TripStatusCancelled = "cancelled"
	TripStatusExpired   = "expired"
)

type TripOffer struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	DriverID       uint      `gorm:"not null;index" json:"driver_id"`
	VehicleID      uint      `gorm:"not null;index" json:"vehicle_id"`
	Origin         string    `gorm:"not null" json:"origin"`
	Destination    string    `gorm:"not null" json:"destination"`
	TripDate       time.Time `gorm:"not null;index" json:"trip_date"`
	AvailableSeats int       `gorm:"not null" json:"available_seats"`
	TotalSeats     int       `gorm:"not null" json:"total_seats"`
	Status         string    `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateTripRequest struct {
	VehicleID      uint      `json:"vehicle_id" binding:"required"`
	Origin         string    `json:"origin" binding:"required"`
	Destination    string    `json:"destination" binding:"required"`
	TripDate       time.Time `json:"trip_date" binding:"required"`
	AvailableSeats int       `json:"available_seats" binding:"required,min=1"`
}
