package models

import "time"

type Vehicle struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	SeatsTotal int       `gorm:"not null" json:"seats_total"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateVehicleRequest struct {
	SeatsTotal int `json:"seats_total" binding:"required,min=1,max=8"`
}
