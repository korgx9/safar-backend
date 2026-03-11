package models

import "time"

type OTPCode struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PhoneNumber string    `gorm:"index;not null" json:"phone_number"`
	Code        string    `gorm:"not null" json:"code"`
	ExpiresAt   time.Time `gorm:"not null" json:"expires_at"`
	Verified    bool      `gorm:"not null;default:false" json:"verified"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
