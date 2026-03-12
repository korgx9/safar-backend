package services

import (
	"time"

	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/models"
)

type TripCleanupService struct {
	db *gorm.DB
}

func NewTripCleanupService(db *gorm.DB) *TripCleanupService {
	return &TripCleanupService{db: db}
}

func (s *TripCleanupService) ExpireOldTrips() (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -7)

	result := s.db.Model(&models.TripOffer{}).
		Where("trip_date < ?", cutoffDate).
		Where("status IN ?", []string{
			models.TripStatusActive,
			models.TripStatusFull,
			models.TripStatusCancelled,
		}).
		Update("status", models.TripStatusExpired)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
