package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/korgx9/safar-backend/internal/models"
)

type cleanupResponse struct {
	Message           string `json:"message"`
	ExpiredTripsCount int64  `json:"expired_trips_count"`
}

func TestCleanupMarksTripsOlderThanSevenDaysAsExpired(t *testing.T) {
	db, router := setupTestEnv(t)
	_, token := createTestUserAndToken(t, db, "+992900006001", models.UserRolePassenger)

	driver, _ := createTestUserAndToken(t, db, "+992900006002", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)

	oldTrip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		TripDate:       time.Now().UTC().AddDate(0, 0, -8),
		AvailableSeats: 2,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})
	createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		TripDate:       time.Now().UTC().AddDate(0, 0, -1),
		AvailableSeats: 2,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})

	w := performRequest(t, router, http.MethodPost, "/admin/cleanup/expired-trips", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeJSON[cleanupResponse](t, w)
	if resp.Message != "cleanup completed" {
		t.Fatalf("expected message 'cleanup completed', got %q", resp.Message)
	}
	if resp.ExpiredTripsCount != 1 {
		t.Fatalf("expected expired_trips_count=1, got %d", resp.ExpiredTripsCount)
	}

	var updated models.TripOffer
	if err := db.First(&updated, oldTrip.ID).Error; err != nil {
		t.Fatalf("failed to reload trip: %v", err)
	}
	if updated.Status != models.TripStatusExpired {
		t.Fatalf("expected status=%s, got %s", models.TripStatusExpired, updated.Status)
	}
}

func TestCleanupDoesNotReExpireAlreadyExpiredTrips(t *testing.T) {
	db, router := setupTestEnv(t)
	_, token := createTestUserAndToken(t, db, "+992900006003", models.UserRolePassenger)

	driver, _ := createTestUserAndToken(t, db, "+992900006004", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)

	createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		TripDate:       time.Now().UTC().AddDate(0, 0, -9),
		AvailableSeats: 2,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})
	createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		TripDate:       time.Now().UTC().AddDate(0, 0, -9),
		AvailableSeats: 2,
		TotalSeats:     4,
		Status:         models.TripStatusExpired,
	})

	firstRun := performRequest(t, router, http.MethodPost, "/admin/cleanup/expired-trips", nil, token)
	if firstRun.Code != http.StatusOK {
		t.Fatalf("expected first cleanup status 200, got %d, body=%s", firstRun.Code, firstRun.Body.String())
	}

	firstResp := decodeJSON[cleanupResponse](t, firstRun)
	if firstResp.ExpiredTripsCount != 1 {
		t.Fatalf("expected first expired_trips_count=1, got %d", firstResp.ExpiredTripsCount)
	}

	secondRun := performRequest(t, router, http.MethodPost, "/admin/cleanup/expired-trips", nil, token)
	if secondRun.Code != http.StatusOK {
		t.Fatalf("expected second cleanup status 200, got %d, body=%s", secondRun.Code, secondRun.Body.String())
	}

	secondResp := decodeJSON[cleanupResponse](t, secondRun)
	if secondResp.ExpiredTripsCount != 0 {
		t.Fatalf("expected second expired_trips_count=0, got %d", secondResp.ExpiredTripsCount)
	}
}
