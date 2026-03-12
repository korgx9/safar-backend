package models

import "time"

type SearchSession struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	PassengerID    uint      `gorm:"not null;index" json:"passenger_id"`
	Origin         string    `gorm:"not null" json:"origin"`
	Destination    string    `gorm:"not null" json:"destination"`
	TripDate       time.Time `gorm:"not null;index" json:"trip_date"`
	RequestedSeats int       `gorm:"not null" json:"requested_seats"`
	CurrentOffset  int       `gorm:"not null;default:0" json:"current_offset"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SearchTripsRequest struct {
	Origin         string    `json:"origin" binding:"required"`
	Destination    string    `json:"destination" binding:"required"`
	TripDate       time.Time `json:"trip_date" binding:"required"`
	RequestedSeats int       `json:"requested_seats" binding:"required,min=1"`
}

type SearchTripResponse struct {
	SearchSessionID uint      `json:"search_session_id"`
	Trip            TripOffer `json:"trip"`
	IsFirstInQueue  bool      `json:"is_first_in_queue"`
	Warning         *string   `json:"warning"`
}
