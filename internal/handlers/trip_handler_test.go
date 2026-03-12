package handlers

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

	"github.com/korgx9/safar-backend/internal/middleware"
	"github.com/korgx9/safar-backend/internal/models"
	"github.com/korgx9/safar-backend/internal/services"
)

func setupTripTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Vehicle{}, &models.TripOffer{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

func setupTripRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	tripHandler := NewTripHandler(db)
	router := gin.Default()

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware("test-secret"))
	{
		protected.POST("/driver/trips", tripHandler.Create)
		protected.GET("/driver/trips", tripHandler.ListMy)
	}

	return router
}

func createTripUser(t *testing.T, db *gorm.DB, phone string) models.User {
	t.Helper()

	user := models.User{
		PhoneNumber: phone,
		Role:        models.UserRoleDriver,
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return user
}

func createTripToken(t *testing.T, user models.User) string {
	t.Helper()

	jwtService := services.NewJWTService("test-secret")
	token, err := jwtService.Generate(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return token
}

func createTripVehicle(t *testing.T, db *gorm.DB, userID uint, seatsTotal int) models.Vehicle {
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

func makeTripRequestBody(t *testing.T, payload map[string]interface{}) []byte {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	return body
}

func TestTripCreate_Success(t *testing.T) {
	db := setupTripTestDB(t)
	router := setupTripRouter(t, db)

	user := createTripUser(t, db, "+992900000021")
	token := createTripToken(t, user)
	vehicle := createTripVehicle(t, db, user.ID, 4)

	body := makeTripRequestBody(t, map[string]interface{}{
		"vehicle_id":      vehicle.ID,
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
		"available_seats": 3,
	})

	req, err := http.NewRequest(http.MethodPost, "/driver/trips", bytes.NewBuffer(body))
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

	var trip models.TripOffer
	if err := json.Unmarshal(w.Body.Bytes(), &trip); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}

	if trip.DriverID != user.ID {
		t.Fatalf("expected driver_id=%d, got %d", user.ID, trip.DriverID)
	}

	if trip.VehicleID != vehicle.ID {
		t.Fatalf("expected vehicle_id=%d, got %d", vehicle.ID, trip.VehicleID)
	}

	if trip.TotalSeats != vehicle.SeatsTotal {
		t.Fatalf("expected total_seats=%d, got %d", vehicle.SeatsTotal, trip.TotalSeats)
	}

	if trip.Status != models.TripStatusActive {
		t.Fatalf("expected status=%s, got %s", models.TripStatusActive, trip.Status)
	}
}

func TestTripCreate_Forbidden_WhenVehicleBelongsToAnotherUser(t *testing.T) {
	db := setupTripTestDB(t)
	router := setupTripRouter(t, db)

	user := createTripUser(t, db, "+992900000022")
	otherUser := createTripUser(t, db, "+992900000023")
	token := createTripToken(t, user)
	otherVehicle := createTripVehicle(t, db, otherUser.ID, 4)

	body := makeTripRequestBody(t, map[string]interface{}{
		"vehicle_id":      otherVehicle.ID,
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
		"available_seats": 2,
	})

	req, err := http.NewRequest(http.MethodPost, "/driver/trips", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestTripCreate_NotFound_WhenVehicleDoesNotExist(t *testing.T) {
	db := setupTripTestDB(t)
	router := setupTripRouter(t, db)

	user := createTripUser(t, db, "+992900000024")
	token := createTripToken(t, user)

	body := makeTripRequestBody(t, map[string]interface{}{
		"vehicle_id":      999999,
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
		"available_seats": 2,
	})

	req, err := http.NewRequest(http.MethodPost, "/driver/trips", bytes.NewBuffer(body))
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

func TestTripCreate_BadRequest_WhenAvailableSeatsExceedsVehicleSeats(t *testing.T) {
	db := setupTripTestDB(t)
	router := setupTripRouter(t, db)

	user := createTripUser(t, db, "+992900000025")
	token := createTripToken(t, user)
	vehicle := createTripVehicle(t, db, user.ID, 4)

	body := makeTripRequestBody(t, map[string]interface{}{
		"vehicle_id":      vehicle.ID,
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
		"available_seats": 5,
	})

	req, err := http.NewRequest(http.MethodPost, "/driver/trips", bytes.NewBuffer(body))
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

func TestTripCreate_BadRequest_WhenTripDateIsInPast(t *testing.T) {
	db := setupTripTestDB(t)
	router := setupTripRouter(t, db)

	user := createTripUser(t, db, "+992900000026")
	token := createTripToken(t, user)
	vehicle := createTripVehicle(t, db, user.ID, 4)

	body := makeTripRequestBody(t, map[string]interface{}{
		"vehicle_id":      vehicle.ID,
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
		"available_seats": 2,
	})

	req, err := http.NewRequest(http.MethodPost, "/driver/trips", bytes.NewBuffer(body))
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

func TestTripCreate_BadRequest_WhenOriginEqualsDestination(t *testing.T) {
	db := setupTripTestDB(t)
	router := setupTripRouter(t, db)

	user := createTripUser(t, db, "+992900000027")
	token := createTripToken(t, user)
	vehicle := createTripVehicle(t, db, user.ID, 4)

	body := makeTripRequestBody(t, map[string]interface{}{
		"vehicle_id":      vehicle.ID,
		"origin":          "Dushanbe",
		"destination":     "Dushanbe",
		"trip_date":       time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
		"available_seats": 2,
	})

	req, err := http.NewRequest(http.MethodPost, "/driver/trips", bytes.NewBuffer(body))
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

func TestTripListMy_Success_ReturnsOnlyOwnTripsAndSortedByCreatedAtDesc(t *testing.T) {
	db := setupTripTestDB(t)
	router := setupTripRouter(t, db)

	user1 := createTripUser(t, db, "+992900000028")
	user2 := createTripUser(t, db, "+992900000029")
	token := createTripToken(t, user1)

	vehicle1 := createTripVehicle(t, db, user1.ID, 4)
	vehicle2 := createTripVehicle(t, db, user2.ID, 4)

	baseTime := time.Now().UTC().Add(24 * time.Hour)

	tripOld := models.TripOffer{
		DriverID:       user1.ID,
		VehicleID:      vehicle1.ID,
		Origin:         "Dushanbe",
		Destination:    "Khujand",
		TripDate:       baseTime,
		AvailableSeats: 2,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
		CreatedAt:      baseTime.Add(-2 * time.Hour),
		UpdatedAt:      baseTime.Add(-2 * time.Hour),
	}

	tripNew := models.TripOffer{
		DriverID:       user1.ID,
		VehicleID:      vehicle1.ID,
		Origin:         "Dushanbe",
		Destination:    "Kulob",
		TripDate:       baseTime,
		AvailableSeats: 3,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
		CreatedAt:      baseTime.Add(-1 * time.Hour),
		UpdatedAt:      baseTime.Add(-1 * time.Hour),
	}

	otherUserTrip := models.TripOffer{
		DriverID:       user2.ID,
		VehicleID:      vehicle2.ID,
		Origin:         "Khujand",
		Destination:    "Dushanbe",
		TripDate:       baseTime,
		AvailableSeats: 2,
		TotalSeats:     4,
		Status:         models.TripStatusActive,
		CreatedAt:      baseTime,
		UpdatedAt:      baseTime,
	}

	if err := db.Create(&tripOld).Error; err != nil {
		t.Fatalf("failed to seed old trip: %v", err)
	}
	if err := db.Create(&tripNew).Error; err != nil {
		t.Fatalf("failed to seed new trip: %v", err)
	}
	if err := db.Create(&otherUserTrip).Error; err != nil {
		t.Fatalf("failed to seed other user trip: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/driver/trips", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var trips []models.TripOffer
	if err := json.Unmarshal(w.Body.Bytes(), &trips); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}

	if len(trips) != 2 {
		t.Fatalf("expected 2 trips, got %d, body=%s", len(trips), w.Body.String())
	}

	for _, trip := range trips {
		if trip.DriverID != user1.ID {
			t.Fatalf("expected only trips for driver_id=%d, got driver_id=%d", user1.ID, trip.DriverID)
		}
	}

	if trips[0].ID != tripNew.ID {
		t.Fatalf("expected first trip id=%d, got %d", tripNew.ID, trips[0].ID)
	}
}
