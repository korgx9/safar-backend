package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/korgx9/safar-backend/internal/models"
)

func TestTripDriverCanCreateTrip(t *testing.T) {
	db, router := setupTestEnv(t)
	driver, token := createTestUserAndToken(t, db, "+992900003001", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/driver/trips",
		map[string]interface{}{
			"vehicle_id":      vehicle.ID,
			"origin":          "Dushanbe",
			"destination":     "Khujand",
			"trip_date":       time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
			"available_seats": 3,
		},
		token,
	)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}

	trip := decodeJSON[models.TripOffer](t, w)
	if trip.DriverID != driver.ID {
		t.Fatalf("expected driver_id=%d, got %d", driver.ID, trip.DriverID)
	}
	if trip.VehicleID != vehicle.ID {
		t.Fatalf("expected vehicle_id=%d, got %d", vehicle.ID, trip.VehicleID)
	}
	if trip.TotalSeats != vehicle.SeatsTotal {
		t.Fatalf("expected total_seats=%d, got %d", vehicle.SeatsTotal, trip.TotalSeats)
	}
}

func TestTripCannotCreateWithTooManySeats(t *testing.T) {
	db, router := setupTestEnv(t)
	driver, token := createTestUserAndToken(t, db, "+992900003002", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/driver/trips",
		map[string]interface{}{
			"vehicle_id":      vehicle.ID,
			"origin":          "Dushanbe",
			"destination":     "Khujand",
			"trip_date":       time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
			"available_seats": 5,
		},
		token,
	)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestTripCannotCreateWithAnotherUsersVehicle(t *testing.T) {
	db, router := setupTestEnv(t)
	owner, _ := createTestUserAndToken(t, db, "+992900003003", models.UserRoleDriver)
	otherDriver, token := createTestUserAndToken(t, db, "+992900003004", models.UserRoleDriver)
	vehicle := createVehicle(t, db, owner.ID, 4)

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/driver/trips",
		map[string]interface{}{
			"vehicle_id":      vehicle.ID,
			"origin":          "Dushanbe",
			"destination":     "Khujand",
			"trip_date":       time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
			"available_seats": 2,
		},
		token,
	)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d, body=%s", w.Code, w.Body.String())
	}

	if otherDriver.ID == owner.ID {
		t.Fatal("unexpected equal user ids in fixture")
	}
}

func TestTripListOwnTrips(t *testing.T) {
	db, router := setupTestEnv(t)
	driver1, token1 := createTestUserAndToken(t, db, "+992900003005", models.UserRoleDriver)
	driver2, _ := createTestUserAndToken(t, db, "+992900003006", models.UserRoleDriver)

	v1 := createVehicle(t, db, driver1.ID, 4)
	v2 := createVehicle(t, db, driver2.ID, 4)

	createTrip(t, db, tripSeed{
		DriverID:       driver1.ID,
		VehicleID:      v1.ID,
		AvailableSeats: 2,
		TotalSeats:     4,
	})
	createTrip(t, db, tripSeed{
		DriverID:       driver1.ID,
		VehicleID:      v1.ID,
		AvailableSeats: 3,
		TotalSeats:     4,
	})
	createTrip(t, db, tripSeed{
		DriverID:       driver2.ID,
		VehicleID:      v2.ID,
		AvailableSeats: 2,
		TotalSeats:     4,
	})

	w := performRequest(t, router, http.MethodGet, "/driver/trips", nil, token1)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	trips := decodeJSON[[]models.TripOffer](t, w)
	if len(trips) != 2 {
		t.Fatalf("expected 2 trips, got %d, body=%s", len(trips), w.Body.String())
	}

	for _, trip := range trips {
		if trip.DriverID != driver1.ID {
			t.Fatalf("expected driver_id=%d, got %d", driver1.ID, trip.DriverID)
		}
	}
}
