package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/models"
	"github.com/korgx9/safar-backend/internal/routes"
	"github.com/korgx9/safar-backend/internal/services"
)

const testJWTSecret = "test-secret"

type tripSeed struct {
	DriverID       uint
	VehicleID      uint
	Origin         string
	Destination    string
	TripDate       time.Time
	AvailableSeats int
	TotalSeats     int
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type bookingSeed struct {
	TripOfferID         uint
	PassengerID         uint
	RequestedSeats      int
	Status              string
	WarningAcknowledged bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func setupTestEnv(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db := initTestDB(t)
	truncateTables(t, db)

	router := routes.SetupRouter(db, testJWTSecret)
	return db, router
}

func initTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.OTPCode{},
		&models.Vehicle{},
		&models.TripOffer{},
		&models.SearchSession{},
		&models.Booking{},
	); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	return db
}

func truncateTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	// Keep delete order deterministic so cleanup remains stable as relations grow.
	tables := []string{
		"bookings",
		"search_sessions",
		"trip_offers",
		"vehicles",
		"otp_codes",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("failed to truncate table %s: %v", table, err)
		}
	}
}

func performRequest(
	t *testing.T,
	router *gin.Engine,
	method, path string,
	payload interface{},
	token string,
) *httptest.ResponseRecorder {
	t.Helper()

	var bodyBytes []byte
	if payload != nil {
		switch v := payload.(type) {
		case []byte:
			bodyBytes = v
		case string:
			bodyBytes = []byte(v)
		default:
			marshaled, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}
			bodyBytes = marshaled
		}
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode json: %v, body=%s", err, w.Body.String())
	}

	return out
}

func createTestUser(t *testing.T, db *gorm.DB, phone string, role models.UserRole) models.User {
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

func createTestUserAndToken(
	t *testing.T,
	db *gorm.DB,
	phone string,
	role models.UserRole,
) (models.User, string) {
	t.Helper()

	user := createTestUser(t, db, phone, role)
	jwtService := services.NewJWTService(testJWTSecret)
	token, err := jwtService.Generate(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return user, token
}

func latestOTPCode(t *testing.T, db *gorm.DB, phone string) models.OTPCode {
	t.Helper()

	var otp models.OTPCode
	if err := db.Where("phone_number = ?", phone).Order("id DESC").First(&otp).Error; err != nil {
		t.Fatalf("failed to fetch latest otp: %v", err)
	}

	return otp
}

func createVehicle(t *testing.T, db *gorm.DB, userID uint, seatsTotal int) models.Vehicle {
	t.Helper()

	vehicle := models.Vehicle{
		UserID:     userID,
		SeatsTotal: seatsTotal,
	}

	if err := db.Create(&vehicle).Error; err != nil {
		t.Fatalf("failed to create vehicle: %v", err)
	}

	return vehicle
}

func createTrip(t *testing.T, db *gorm.DB, seed tripSeed) models.TripOffer {
	t.Helper()

	if seed.Origin == "" {
		seed.Origin = "Dushanbe"
	}
	if seed.Destination == "" {
		seed.Destination = "Khujand"
	}
	if seed.TripDate.IsZero() {
		seed.TripDate = time.Now().UTC().Add(24 * time.Hour)
	}
	if seed.AvailableSeats == 0 && seed.TotalSeats == 0 {
		seed.AvailableSeats = 1
	}
	if seed.TotalSeats == 0 {
		seed.TotalSeats = seed.AvailableSeats
	}
	if seed.Status == "" {
		seed.Status = models.TripStatusActive
	}

	trip := models.TripOffer{
		DriverID:       seed.DriverID,
		VehicleID:      seed.VehicleID,
		Origin:         seed.Origin,
		Destination:    seed.Destination,
		TripDate:       seed.TripDate,
		AvailableSeats: seed.AvailableSeats,
		TotalSeats:     seed.TotalSeats,
		Status:         seed.Status,
		CreatedAt:      seed.CreatedAt,
		UpdatedAt:      seed.UpdatedAt,
	}

	if !seed.CreatedAt.IsZero() && seed.UpdatedAt.IsZero() {
		trip.UpdatedAt = seed.CreatedAt
	}

	if err := db.Create(&trip).Error; err != nil {
		t.Fatalf("failed to create trip: %v", err)
	}

	return trip
}

func createBooking(t *testing.T, db *gorm.DB, seed bookingSeed) models.Booking {
	t.Helper()

	if seed.RequestedSeats == 0 {
		seed.RequestedSeats = 1
	}
	if seed.Status == "" {
		seed.Status = models.BookingStatusConfirmed
	}

	booking := models.Booking{
		TripOfferID:         seed.TripOfferID,
		PassengerID:         seed.PassengerID,
		RequestedSeats:      seed.RequestedSeats,
		Status:              seed.Status,
		WarningAcknowledged: seed.WarningAcknowledged,
		CreatedAt:           seed.CreatedAt,
		UpdatedAt:           seed.UpdatedAt,
	}

	if !seed.CreatedAt.IsZero() && seed.UpdatedAt.IsZero() {
		booking.UpdatedAt = seed.CreatedAt
	}

	if err := db.Create(&booking).Error; err != nil {
		t.Fatalf("failed to create booking: %v", err)
	}

	return booking
}
