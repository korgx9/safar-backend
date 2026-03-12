package services

import (
	"testing"
	"time"

	"github.com/korgx9/safar-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTripCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&models.TripOffer{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

func createCleanupTrip(t *testing.T, db *gorm.DB, status string, tripDate time.Time) models.TripOffer {
	t.Helper()

	trip := models.TripOffer{
		DriverID:       1,
		VehicleID:      1,
		Origin:         "Dushanbe",
		Destination:    "Khujand",
		TripDate:       tripDate,
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         status,
	}

	if err := db.Create(&trip).Error; err != nil {
		t.Fatalf("failed to create trip: %v", err)
	}

	return trip
}

func TestTripCleanupService_ExpireOldTrips(t *testing.T) {
	db := setupTripCleanupTestDB(t)
	service := NewTripCleanupService(db)

	oldDate := time.Now().AddDate(0, 0, -8)
	recentDate := time.Now().AddDate(0, 0, -6)

	oldActive := createCleanupTrip(t, db, models.TripStatusActive, oldDate)
	oldFull := createCleanupTrip(t, db, models.TripStatusFull, oldDate)
	oldCancelled := createCleanupTrip(t, db, models.TripStatusCancelled, oldDate)
	oldExpired := createCleanupTrip(t, db, models.TripStatusExpired, oldDate)
	recentActive := createCleanupTrip(t, db, models.TripStatusActive, recentDate)

	affectedRows, err := service.ExpireOldTrips()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if affectedRows != 3 {
		t.Fatalf("expected 3 affected rows, got %d", affectedRows)
	}

	var trips []models.TripOffer
	if err := db.Find(&trips).Error; err != nil {
		t.Fatalf("failed to load trips: %v", err)
	}

	tripStatusByID := make(map[uint]string)
	for _, trip := range trips {
		tripStatusByID[trip.ID] = trip.Status
	}

	if tripStatusByID[oldActive.ID] != models.TripStatusExpired {
		t.Fatalf("expected old active trip to be expired, got %s", tripStatusByID[oldActive.ID])
	}
	if tripStatusByID[oldFull.ID] != models.TripStatusExpired {
		t.Fatalf("expected old full trip to be expired, got %s", tripStatusByID[oldFull.ID])
	}
	if tripStatusByID[oldCancelled.ID] != models.TripStatusExpired {
		t.Fatalf("expected old cancelled trip to be expired, got %s", tripStatusByID[oldCancelled.ID])
	}
	if tripStatusByID[oldExpired.ID] != models.TripStatusExpired {
		t.Fatalf("expected old expired trip to remain expired, got %s", tripStatusByID[oldExpired.ID])
	}
	if tripStatusByID[recentActive.ID] != models.TripStatusActive {
		t.Fatalf("expected recent active trip to remain active, got %s", tripStatusByID[recentActive.ID])
	}

	affectedRows, err = service.ExpireOldTrips()
	if err != nil {
		t.Fatalf("expected no error on second run, got %v", err)
	}

	if affectedRows != 0 {
		t.Fatalf("expected 0 affected rows on second run, got %d", affectedRows)
	}
}
