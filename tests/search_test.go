package tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/korgx9/safar-backend/internal/models"
)

func TestSearchReturnsFirstTrip(t *testing.T) {
	db, router := setupTestEnv(t)
	passenger, token := createTestUserAndToken(t, db, "+992900004001", models.UserRolePassenger)
	_ = passenger

	driver1, _ := createTestUserAndToken(t, db, "+992900004002", models.UserRoleDriver)
	driver2, _ := createTestUserAndToken(t, db, "+992900004003", models.UserRoleDriver)

	v1 := createVehicle(t, db, driver1.ID, 4)
	v2 := createVehicle(t, db, driver2.ID, 6)

	searchDayStart := time.Now().UTC().Add(48 * time.Hour).Truncate(24 * time.Hour)
	baseCreatedAt := time.Now().UTC().Add(-2 * time.Hour)

	firstTrip := createTrip(t, db, tripSeed{
		DriverID:       driver1.ID,
		VehicleID:      v1.ID,
		Origin:         "Dushanbe",
		Destination:    "Khujand",
		TripDate:       searchDayStart.Add(10 * time.Hour),
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
		CreatedAt:      baseCreatedAt,
	})
	createTrip(t, db, tripSeed{
		DriverID:       driver2.ID,
		VehicleID:      v2.ID,
		Origin:         "Dushanbe",
		Destination:    "Khujand",
		TripDate:       searchDayStart.Add(12 * time.Hour),
		AvailableSeats: 4,
		TotalSeats:     6,
		Status:         models.TripStatusActive,
		CreatedAt:      baseCreatedAt.Add(1 * time.Minute),
	})

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/passenger/search",
		map[string]interface{}{
			"origin":          "Dushanbe",
			"destination":     "Khujand",
			"trip_date":       searchDayStart.Format(time.RFC3339),
			"requested_seats": 2,
		},
		token,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeJSON[models.SearchTripResponse](t, w)
	if resp.Trip.ID != firstTrip.ID {
		t.Fatalf("expected first trip id=%d, got %d", firstTrip.ID, resp.Trip.ID)
	}
	if !resp.IsFirstInQueue {
		t.Fatal("expected is_first_in_queue=true")
	}
}

func TestSearchNextReturnsSecondTrip(t *testing.T) {
	db, router := setupTestEnv(t)
	passenger, token := createTestUserAndToken(t, db, "+992900004004", models.UserRolePassenger)
	_ = passenger
	driver, _ := createTestUserAndToken(t, db, "+992900004005", models.UserRoleDriver)
	v := createVehicle(t, db, driver.ID, 4)

	searchDayStart := time.Now().UTC().Add(48 * time.Hour).Truncate(24 * time.Hour)
	baseCreatedAt := time.Now().UTC().Add(-1 * time.Hour)

	createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      v.ID,
		TripDate:       searchDayStart.Add(9 * time.Hour),
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
		CreatedAt:      baseCreatedAt,
	})
	secondTrip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      v.ID,
		TripDate:       searchDayStart.Add(11 * time.Hour),
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
		CreatedAt:      baseCreatedAt.Add(1 * time.Minute),
	})

	searchRes := performRequest(
		t,
		router,
		http.MethodPost,
		"/passenger/search",
		map[string]interface{}{
			"origin":          "Dushanbe",
			"destination":     "Khujand",
			"trip_date":       searchDayStart.Format(time.RFC3339),
			"requested_seats": 2,
		},
		token,
	)
	if searchRes.Code != http.StatusOK {
		t.Fatalf("expected search status 200, got %d, body=%s", searchRes.Code, searchRes.Body.String())
	}

	searchResp := decodeJSON[models.SearchTripResponse](t, searchRes)
	nextPath := "/passenger/search/" + strconv.FormatUint(uint64(searchResp.SearchSessionID), 10) + "/next"
	nextRes := performRequest(t, router, http.MethodPost, nextPath, nil, token)

	if nextRes.Code != http.StatusOK {
		t.Fatalf("expected next status 200, got %d, body=%s", nextRes.Code, nextRes.Body.String())
	}

	nextResp := decodeJSON[models.SearchTripResponse](t, nextRes)
	if nextResp.Trip.ID != secondTrip.ID {
		t.Fatalf("expected second trip id=%d, got %d", secondTrip.ID, nextResp.Trip.ID)
	}
	if nextResp.IsFirstInQueue {
		t.Fatal("expected is_first_in_queue=false for next response")
	}
}

func TestSearchFiltersTripsWithInsufficientSeats(t *testing.T) {
	db, router := setupTestEnv(t)
	_, token := createTestUserAndToken(t, db, "+992900004006", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900004007", models.UserRoleDriver)
	v := createVehicle(t, db, driver.ID, 4)

	searchDayStart := time.Now().UTC().Add(48 * time.Hour).Truncate(24 * time.Hour)
	baseCreatedAt := time.Now().UTC().Add(-1 * time.Hour)

	createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      v.ID,
		TripDate:       searchDayStart.Add(10 * time.Hour),
		AvailableSeats: 1,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
		CreatedAt:      baseCreatedAt,
	})
	sufficientTrip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      v.ID,
		TripDate:       searchDayStart.Add(11 * time.Hour),
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
		CreatedAt:      baseCreatedAt.Add(1 * time.Minute),
	})

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/passenger/search",
		map[string]interface{}{
			"origin":          "Dushanbe",
			"destination":     "Khujand",
			"trip_date":       searchDayStart.Format(time.RFC3339),
			"requested_seats": 2,
		},
		token,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeJSON[models.SearchTripResponse](t, w)
	if resp.Trip.ID != sufficientTrip.ID {
		t.Fatalf("expected filtered trip id=%d, got %d", sufficientTrip.ID, resp.Trip.ID)
	}
}

func TestSearchIgnoresInactiveTrips(t *testing.T) {
	db, router := setupTestEnv(t)
	_, token := createTestUserAndToken(t, db, "+992900004008", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900004009", models.UserRoleDriver)
	v := createVehicle(t, db, driver.ID, 4)

	searchDayStart := time.Now().UTC().Add(48 * time.Hour).Truncate(24 * time.Hour)
	baseCreatedAt := time.Now().UTC().Add(-1 * time.Hour)

	createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      v.ID,
		TripDate:       searchDayStart.Add(10 * time.Hour),
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusCancelled,
		CreatedAt:      baseCreatedAt,
	})
	activeTrip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      v.ID,
		TripDate:       searchDayStart.Add(11 * time.Hour),
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
		CreatedAt:      baseCreatedAt.Add(1 * time.Minute),
	})

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/passenger/search",
		map[string]interface{}{
			"origin":          "Dushanbe",
			"destination":     "Khujand",
			"trip_date":       searchDayStart.Format(time.RFC3339),
			"requested_seats": 2,
		},
		token,
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeJSON[models.SearchTripResponse](t, w)
	if resp.Trip.ID != activeTrip.ID {
		t.Fatalf("expected active trip id=%d, got %d", activeTrip.ID, resp.Trip.ID)
	}
}

func TestSearchUserCannotUseAnotherUsersSearchSession(t *testing.T) {
	db, router := setupTestEnv(t)
	_, token1 := createTestUserAndToken(t, db, "+992900004010", models.UserRolePassenger)
	_, token2 := createTestUserAndToken(t, db, "+992900004011", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900004012", models.UserRoleDriver)
	v := createVehicle(t, db, driver.ID, 4)

	searchDayStart := time.Now().UTC().Add(48 * time.Hour).Truncate(24 * time.Hour)
	createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      v.ID,
		TripDate:       searchDayStart.Add(10 * time.Hour),
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})

	searchRes := performRequest(
		t,
		router,
		http.MethodPost,
		"/passenger/search",
		map[string]interface{}{
			"origin":          "Dushanbe",
			"destination":     "Khujand",
			"trip_date":       searchDayStart.Format(time.RFC3339),
			"requested_seats": 2,
		},
		token1,
	)
	if searchRes.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", searchRes.Code, searchRes.Body.String())
	}

	searchResp := decodeJSON[models.SearchTripResponse](t, searchRes)
	nextPath := "/passenger/search/" + strconv.FormatUint(uint64(searchResp.SearchSessionID), 10) + "/next"

	nextRes := performRequest(t, router, http.MethodPost, nextPath, nil, token2)
	if nextRes.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d, body=%s", nextRes.Code, nextRes.Body.String())
	}
}
