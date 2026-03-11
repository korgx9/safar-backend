package models

import "time"

type UserRole string

const (
	UserRoleDriver    UserRole = "driver"
	UserRolePassenger UserRole = "passenger"
	UserRoleBoth      UserRole = "both"
)

type User struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PhoneNumber string    `gorm:"uniqueIndex;not null" json:"phone_number"`
	Role        UserRole  `gorm:"type:varchar(20);not null;default:'passenger'" json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
