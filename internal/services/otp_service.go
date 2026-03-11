package services

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/models"
)

type OTPService struct {
	db *gorm.DB
}

func NewOTPService(db *gorm.DB) *OTPService {
	return &OTPService{db: db}
}

func (s *OTPService) Generate(phoneNumber string) (*models.OTPCode, error) {
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	otp := models.OTPCode{
		PhoneNumber: phoneNumber,
		Code:        code,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
		Verified:    false,
	}

	if err := s.db.Create(&otp).Error; err != nil {
		return nil, err
	}

	return &otp, nil
}

func (s *OTPService) Verify(phoneNumber, code string) (*models.OTPCode, error) {
	var otp models.OTPCode

	err := s.db.
		Where("phone_number = ? AND code = ? AND verified = ?", phoneNumber, code, false).
		Order("created_at DESC").
		First(&otp).Error
	if err != nil {
		return nil, err
	}

	if time.Now().After(otp.ExpiresAt) {
		return nil, fmt.Errorf("otp expired")
	}

	otp.Verified = true
	if err := s.db.Save(&otp).Error; err != nil {
		return nil, err
	}

	return &otp, nil
}
