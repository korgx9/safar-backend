package tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/korgx9/safar-backend/internal/models"
)

func TestBookingPassengerCanCreateBooking(t *testing.T) {
	db, router := setupTestEnv(t)
	passenger, token := createTestUserAndToken(t, db, "+992900005001", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900005002", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)
	trip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/bookings",
		map[string]interface{}{
			"trip_offer_id":        trip.ID,
			"requested_seats":      1,
			"warning_acknowledged": true,
		},
		token,
	)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}

	booking := decodeJSON[models.Booking](t, w)
	if booking.PassengerID != passenger.ID {
		t.Fatalf("expected passenger_id=%d, got %d", passenger.ID, booking.PassengerID)
	}
	if booking.Status != models.BookingStatusConfirmed {
		t.Fatalf("expected status=%s, got %s", models.BookingStatusConfirmed, booking.Status)
	}
}

func TestBookingCreationReducesAvailableSeats(t *testing.T) {
	db, router := setupTestEnv(t)
	_, token := createTestUserAndToken(t, db, "+992900005003", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900005004", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)
	trip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/bookings",
		map[string]interface{}{
			"trip_offer_id":        trip.ID,
			"requested_seats":      2,
			"warning_acknowledged": false,
		},
		token,
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}

	var updatedTrip models.TripOffer
	if err := db.First(&updatedTrip, trip.ID).Error; err != nil {
		t.Fatalf("failed to reload trip: %v", err)
	}

	if updatedTrip.AvailableSeats != 1 {
		t.Fatalf("expected available_seats=1, got %d", updatedTrip.AvailableSeats)
	}
}

func TestBookingTripBecomesFullWhenSeatsReachZero(t *testing.T) {
	db, router := setupTestEnv(t)
	_, token := createTestUserAndToken(t, db, "+992900005005", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900005006", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)
	trip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		AvailableSeats: 2,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/bookings",
		map[string]interface{}{
			"trip_offer_id":        trip.ID,
			"requested_seats":      2,
			"warning_acknowledged": false,
		},
		token,
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}

	var updatedTrip models.TripOffer
	if err := db.First(&updatedTrip, trip.ID).Error; err != nil {
		t.Fatalf("failed to reload trip: %v", err)
	}

	if updatedTrip.Status != models.TripStatusFull {
		t.Fatalf("expected status=%s, got %s", models.TripStatusFull, updatedTrip.Status)
	}
	if updatedTrip.AvailableSeats != 0 {
		t.Fatalf("expected available_seats=0, got %d", updatedTrip.AvailableSeats)
	}
}

func TestBookingPassengerCanListOwnBookings(t *testing.T) {
	db, router := setupTestEnv(t)
	passenger, token := createTestUserAndToken(t, db, "+992900005007", models.UserRolePassenger)
	otherPassenger, _ := createTestUserAndToken(t, db, "+992900005008", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900005009", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)
	trip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		AvailableSeats: 4,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})

	baseTime := time.Now().UTC()
	createBooking(t, db, bookingSeed{
		TripOfferID:    trip.ID,
		PassengerID:    passenger.ID,
		RequestedSeats: 1,
		CreatedAt:      baseTime.Add(-2 * time.Hour),
	})
	newest := createBooking(t, db, bookingSeed{
		TripOfferID:    trip.ID,
		PassengerID:    passenger.ID,
		RequestedSeats: 1,
		CreatedAt:      baseTime.Add(-1 * time.Hour),
	})
	createBooking(t, db, bookingSeed{
		TripOfferID:    trip.ID,
		PassengerID:    otherPassenger.ID,
		RequestedSeats: 1,
		CreatedAt:      baseTime,
	})

	w := performRequest(t, router, http.MethodGet, "/bookings/my", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	bookings := decodeJSON[[]models.Booking](t, w)
	if len(bookings) != 2 {
		t.Fatalf("expected 2 bookings, got %d, body=%s", len(bookings), w.Body.String())
	}
	if bookings[0].ID != newest.ID {
		t.Fatalf("expected newest booking first id=%d, got %d", newest.ID, bookings[0].ID)
	}
	for _, booking := range bookings {
		if booking.PassengerID != passenger.ID {
			t.Fatalf("expected passenger_id=%d, got %d", passenger.ID, booking.PassengerID)
		}
	}
}

func TestBookingPassengerCanCancelOwnBooking(t *testing.T) {
	db, router := setupTestEnv(t)
	passenger, token := createTestUserAndToken(t, db, "+992900005010", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900005011", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)
	trip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})
	booking := createBooking(t, db, bookingSeed{
		TripOfferID:    trip.ID,
		PassengerID:    passenger.ID,
		RequestedSeats: 1,
		Status:         models.BookingStatusConfirmed,
	})

	cancelPath := "/bookings/" + strconv.FormatUint(uint64(booking.ID), 10) + "/cancel"
	w := performRequest(t, router, http.MethodPatch, cancelPath, nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	updated := decodeJSON[models.Booking](t, w)
	if updated.Status != models.BookingStatusCancelled {
		t.Fatalf("expected status=%s, got %s", models.BookingStatusCancelled, updated.Status)
	}
}

func TestBookingCancellationRestoresSeats(t *testing.T) {
	db, router := setupTestEnv(t)
	passenger, token := createTestUserAndToken(t, db, "+992900005012", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900005013", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)
	trip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		AvailableSeats: 0,
		TotalSeats:     4,
		Status:         models.TripStatusFull,
	})
	booking := createBooking(t, db, bookingSeed{
		TripOfferID:    trip.ID,
		PassengerID:    passenger.ID,
		RequestedSeats: 2,
		Status:         models.BookingStatusConfirmed,
	})

	cancelPath := "/bookings/" + strconv.FormatUint(uint64(booking.ID), 10) + "/cancel"
	w := performRequest(t, router, http.MethodPatch, cancelPath, nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var updatedTrip models.TripOffer
	if err := db.First(&updatedTrip, trip.ID).Error; err != nil {
		t.Fatalf("failed to reload trip: %v", err)
	}
	if updatedTrip.AvailableSeats != 2 {
		t.Fatalf("expected available_seats=2, got %d", updatedTrip.AvailableSeats)
	}
	if updatedTrip.Status != models.TripStatusActive {
		t.Fatalf("expected status=%s, got %s", models.TripStatusActive, updatedTrip.Status)
	}
}

func TestBookingCannotOverbook(t *testing.T) {
	db, router := setupTestEnv(t)
	_, token := createTestUserAndToken(t, db, "+992900005014", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900005015", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)
	trip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		AvailableSeats: 1,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/bookings",
		map[string]interface{}{
			"trip_offer_id":        trip.ID,
			"requested_seats":      2,
			"warning_acknowledged": false,
		},
		token,
	)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBookingCannotCancelAnotherUsersBooking(t *testing.T) {
	db, router := setupTestEnv(t)
	owner, _ := createTestUserAndToken(t, db, "+992900005016", models.UserRolePassenger)
	otherUser, otherToken := createTestUserAndToken(t, db, "+992900005017", models.UserRolePassenger)
	driver, _ := createTestUserAndToken(t, db, "+992900005018", models.UserRoleDriver)
	vehicle := createVehicle(t, db, driver.ID, 4)
	trip := createTrip(t, db, tripSeed{
		DriverID:       driver.ID,
		VehicleID:      vehicle.ID,
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
	})
	booking := createBooking(t, db, bookingSeed{
		TripOfferID:    trip.ID,
		PassengerID:    owner.ID,
		RequestedSeats: 1,
		Status:         models.BookingStatusConfirmed,
	})

	cancelPath := "/bookings/" + strconv.FormatUint(uint64(booking.ID), 10) + "/cancel"
	w := performRequest(t, router, http.MethodPatch, cancelPath, nil, otherToken)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d, body=%s", w.Code, w.Body.String())
	}
	if otherUser.ID == owner.ID {
		t.Fatal("unexpected equal fixture user ids")
	}
}
