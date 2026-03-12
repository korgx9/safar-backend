package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/middleware"
	"github.com/korgx9/safar-backend/internal/models"
	"github.com/korgx9/safar-backend/internal/services"
)

func setupBookingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Vehicle{},
		&models.TripOffer{},
		&models.Booking{},
	); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

func setupBookingRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	bookingHandler := NewBookingHandler(db)
	router := gin.Default()

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware("test-secret"))
	{
		protected.POST("/bookings", bookingHandler.Create)
		protected.GET("/bookings/my", bookingHandler.ListMy)
		protected.PATCH("/bookings/:id/cancel", bookingHandler.Cancel)
	}

	return router
}

func createBookingUser(t *testing.T, db *gorm.DB, phone string, role models.UserRole) models.User {
	t.Helper()

	user := models.User{
		PhoneNumber: phone,
		Role:        role,
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return user
}

func createBookingToken(t *testing.T, user models.User) string {
	t.Helper()

	jwtService := services.NewJWTService("test-secret")
	token, err := jwtService.Generate(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return token
}

func createBookingVehicle(t *testing.T, db *gorm.DB, driverID uint, seatsTotal int) models.Vehicle {
	t.Helper()

	vehicle := models.Vehicle{
		UserID:     driverID,
		SeatsTotal: seatsTotal,
	}

	if err := db.Create(&vehicle).Error; err != nil {
		t.Fatalf("failed to create vehicle: %v", err)
	}

	return vehicle
}

func createBookingTrip(
	t *testing.T,
	db *gorm.DB,
	driverID, vehicleID uint,
	availableSeats, totalSeats int,
	status string,
) models.TripOffer {
	t.Helper()

	trip := models.TripOffer{
		DriverID:       driverID,
		VehicleID:      vehicleID,
		Origin:         "Dushanbe",
		Destination:    "Khujand",
		TripDate:       time.Now().UTC().Add(24 * time.Hour),
		AvailableSeats: availableSeats,
		TotalSeats:     totalSeats,
		Status:         status,
	}

	if err := db.Create(&trip).Error; err != nil {
		t.Fatalf("failed to create trip: %v", err)
	}

	return trip
}

func TestBookingCreate_Unauthorized_WithoutToken(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	body := []byte(`{"trip_offer_id":1,"requested_seats":1,"warning_acknowledged":false}`)
	req, err := http.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBookingCreate_Success_ReducesSeatsAndMarksTripFull(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	passenger := createBookingUser(t, db, "+992900000051", models.UserRolePassenger)
	driver := createBookingUser(t, db, "+992900000052", models.UserRoleDriver)
	token := createBookingToken(t, passenger)

	vehicle := createBookingVehicle(t, db, driver.ID, 4)
	trip := createBookingTrip(t, db, driver.ID, vehicle.ID, 2, 4, models.TripStatusActive)

	body := []byte(`{"trip_offer_id":` + strconv.FormatUint(uint64(trip.ID), 10) + `,"requested_seats":2,"warning_acknowledged":true}`)
	req, err := http.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}

	var booking models.Booking
	if err := json.Unmarshal(w.Body.Bytes(), &booking); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}

	if booking.TripOfferID != trip.ID {
		t.Fatalf("expected trip_offer_id=%d, got %d", trip.ID, booking.TripOfferID)
	}

	if booking.PassengerID != passenger.ID {
		t.Fatalf("expected passenger_id=%d, got %d", passenger.ID, booking.PassengerID)
	}

	if booking.Status != models.BookingStatusConfirmed {
		t.Fatalf("expected status=%s, got %s", models.BookingStatusConfirmed, booking.Status)
	}

	if !booking.WarningAcknowledged {
		t.Fatal("expected warning_acknowledged=true")
	}

	var updatedTrip models.TripOffer
	if err := db.First(&updatedTrip, trip.ID).Error; err != nil {
		t.Fatalf("failed to reload trip: %v", err)
	}

	if updatedTrip.AvailableSeats != 0 {
		t.Fatalf("expected available_seats=0, got %d", updatedTrip.AvailableSeats)
	}

	if updatedTrip.Status != models.TripStatusFull {
		t.Fatalf("expected trip status=%s, got %s", models.TripStatusFull, updatedTrip.Status)
	}
}

func TestBookingCreate_NotFound_WhenTripMissing(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	passenger := createBookingUser(t, db, "+992900000053", models.UserRolePassenger)
	token := createBookingToken(t, passenger)

	body := []byte(`{"trip_offer_id":999999,"requested_seats":1,"warning_acknowledged":false}`)
	req, err := http.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBookingCreate_BadRequest_WhenTripNotActive(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	passenger := createBookingUser(t, db, "+992900000054", models.UserRolePassenger)
	driver := createBookingUser(t, db, "+992900000055", models.UserRoleDriver)
	token := createBookingToken(t, passenger)

	vehicle := createBookingVehicle(t, db, driver.ID, 4)
	trip := createBookingTrip(t, db, driver.ID, vehicle.ID, 3, 4, models.TripStatusCancelled)

	body := []byte(`{"trip_offer_id":` + strconv.FormatUint(uint64(trip.ID), 10) + `,"requested_seats":1,"warning_acknowledged":false}`)
	req, err := http.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBookingCreate_BadRequest_WhenNotEnoughSeats(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	passenger := createBookingUser(t, db, "+992900000056", models.UserRolePassenger)
	driver := createBookingUser(t, db, "+992900000057", models.UserRoleDriver)
	token := createBookingToken(t, passenger)

	vehicle := createBookingVehicle(t, db, driver.ID, 4)
	trip := createBookingTrip(t, db, driver.ID, vehicle.ID, 1, 4, models.TripStatusActive)

	body := []byte(`{"trip_offer_id":` + strconv.FormatUint(uint64(trip.ID), 10) + `,"requested_seats":2,"warning_acknowledged":false}`)
	req, err := http.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBookingListMy_Success_ReturnsOnlyOwnBookingsAndSorted(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	passenger := createBookingUser(t, db, "+992900000058", models.UserRolePassenger)
	otherPassenger := createBookingUser(t, db, "+992900000059", models.UserRolePassenger)
	driver := createBookingUser(t, db, "+992900000060", models.UserRoleDriver)
	token := createBookingToken(t, passenger)

	vehicle := createBookingVehicle(t, db, driver.ID, 4)
	trip := createBookingTrip(t, db, driver.ID, vehicle.ID, 4, 4, models.TripStatusActive)

	baseTime := time.Now().UTC()
	oldBooking := models.Booking{
		TripOfferID:         trip.ID,
		PassengerID:         passenger.ID,
		RequestedSeats:      1,
		Status:              models.BookingStatusConfirmed,
		WarningAcknowledged: false,
		CreatedAt:           baseTime.Add(-2 * time.Hour),
		UpdatedAt:           baseTime.Add(-2 * time.Hour),
	}
	newBooking := models.Booking{
		TripOfferID:         trip.ID,
		PassengerID:         passenger.ID,
		RequestedSeats:      1,
		Status:              models.BookingStatusConfirmed,
		WarningAcknowledged: true,
		CreatedAt:           baseTime.Add(-1 * time.Hour),
		UpdatedAt:           baseTime.Add(-1 * time.Hour),
	}
	otherUserBooking := models.Booking{
		TripOfferID:         trip.ID,
		PassengerID:         otherPassenger.ID,
		RequestedSeats:      1,
		Status:              models.BookingStatusConfirmed,
		WarningAcknowledged: false,
		CreatedAt:           baseTime,
		UpdatedAt:           baseTime,
	}

	if err := db.Create(&oldBooking).Error; err != nil {
		t.Fatalf("failed to seed old booking: %v", err)
	}
	if err := db.Create(&newBooking).Error; err != nil {
		t.Fatalf("failed to seed new booking: %v", err)
	}
	if err := db.Create(&otherUserBooking).Error; err != nil {
		t.Fatalf("failed to seed other user booking: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/bookings/my", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var bookings []models.Booking
	if err := json.Unmarshal(w.Body.Bytes(), &bookings); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}

	if len(bookings) != 2 {
		t.Fatalf("expected 2 bookings, got %d, body=%s", len(bookings), w.Body.String())
	}

	for _, booking := range bookings {
		if booking.PassengerID != passenger.ID {
			t.Fatalf("expected only passenger_id=%d, got %d", passenger.ID, booking.PassengerID)
		}
	}

	if bookings[0].ID != newBooking.ID {
		t.Fatalf("expected newest booking id=%d first, got %d", newBooking.ID, bookings[0].ID)
	}
}

func TestBookingCancel_Success_RestoresSeatsAndReactivatesTrip(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	passenger := createBookingUser(t, db, "+992900000061", models.UserRolePassenger)
	driver := createBookingUser(t, db, "+992900000062", models.UserRoleDriver)
	token := createBookingToken(t, passenger)

	vehicle := createBookingVehicle(t, db, driver.ID, 4)
	trip := createBookingTrip(t, db, driver.ID, vehicle.ID, 0, 4, models.TripStatusFull)

	booking := models.Booking{
		TripOfferID:         trip.ID,
		PassengerID:         passenger.ID,
		RequestedSeats:      2,
		Status:              models.BookingStatusConfirmed,
		WarningAcknowledged: true,
	}
	if err := db.Create(&booking).Error; err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPatch,
		"/bookings/"+strconv.FormatUint(uint64(booking.ID), 10)+"/cancel",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var updatedBooking models.Booking
	if err := json.Unmarshal(w.Body.Bytes(), &updatedBooking); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}

	if updatedBooking.Status != models.BookingStatusCancelled {
		t.Fatalf("expected booking status=%s, got %s", models.BookingStatusCancelled, updatedBooking.Status)
	}

	var updatedTrip models.TripOffer
	if err := db.First(&updatedTrip, trip.ID).Error; err != nil {
		t.Fatalf("failed to reload trip: %v", err)
	}

	if updatedTrip.AvailableSeats != 2 {
		t.Fatalf("expected available_seats=2, got %d", updatedTrip.AvailableSeats)
	}

	if updatedTrip.Status != models.TripStatusActive {
		t.Fatalf("expected trip status=%s, got %s", models.TripStatusActive, updatedTrip.Status)
	}
}

func TestBookingCancel_Forbidden_WhenBookingBelongsToAnotherUser(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	passenger1 := createBookingUser(t, db, "+992900000063", models.UserRolePassenger)
	passenger2 := createBookingUser(t, db, "+992900000064", models.UserRolePassenger)
	driver := createBookingUser(t, db, "+992900000065", models.UserRoleDriver)
	token2 := createBookingToken(t, passenger2)

	vehicle := createBookingVehicle(t, db, driver.ID, 4)
	trip := createBookingTrip(t, db, driver.ID, vehicle.ID, 2, 4, models.TripStatusActive)

	booking := models.Booking{
		TripOfferID:         trip.ID,
		PassengerID:         passenger1.ID,
		RequestedSeats:      1,
		Status:              models.BookingStatusConfirmed,
		WarningAcknowledged: false,
	}
	if err := db.Create(&booking).Error; err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPatch,
		"/bookings/"+strconv.FormatUint(uint64(booking.ID), 10)+"/cancel",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token2)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBookingCancel_BadRequest_WhenAlreadyCancelled(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	passenger := createBookingUser(t, db, "+992900000066", models.UserRolePassenger)
	driver := createBookingUser(t, db, "+992900000067", models.UserRoleDriver)
	token := createBookingToken(t, passenger)

	vehicle := createBookingVehicle(t, db, driver.ID, 4)
	trip := createBookingTrip(t, db, driver.ID, vehicle.ID, 2, 4, models.TripStatusActive)

	booking := models.Booking{
		TripOfferID:         trip.ID,
		PassengerID:         passenger.ID,
		RequestedSeats:      1,
		Status:              models.BookingStatusCancelled,
		WarningAcknowledged: false,
	}
	if err := db.Create(&booking).Error; err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	req, err := http.NewRequest(
		http.MethodPatch,
		"/bookings/"+strconv.FormatUint(uint64(booking.ID), 10)+"/cancel",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBookingCancel_NotFound_WhenBookingMissing(t *testing.T) {
	db := setupBookingTestDB(t)
	router := setupBookingRouter(t, db)

	passenger := createBookingUser(t, db, "+992900000068", models.UserRolePassenger)
	token := createBookingToken(t, passenger)

	req, err := http.NewRequest(http.MethodPatch, "/bookings/999999/cancel", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
}
