package handlers

import (
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

func setupAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.TripOffer{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

func setupAdminRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	adminHandler := NewAdminHandler(db)
	router := gin.Default()

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware("test-secret"))
	{
		protected.POST("/admin/cleanup/expired-trips", adminHandler.CleanupExpiredTrips)
	}

	return router
}

func createAdminTestUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()

	user := models.User{
		PhoneNumber: "+992900000071",
		Role:        models.UserRolePassenger,
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return user
}

func createAdminToken(t *testing.T, user models.User) string {
	t.Helper()

	jwtService := services.NewJWTService("test-secret")
	token, err := jwtService.Generate(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return token
}

func createAdminTrip(t *testing.T, db *gorm.DB, status string, tripDate time.Time) models.TripOffer {
	t.Helper()

	trip := models.TripOffer{
		DriverID:       1,
		VehicleID:      1,
		Origin:         "Dushanbe",
		Destination:    "Khujand",
		TripDate:       tripDate,
		AvailableSeats: 2,
		TotalSeats:     4,
		Status:         status,
	}

	if err := db.Create(&trip).Error; err != nil {
		t.Fatalf("failed to create trip: %v", err)
	}

	return trip
}

func TestCleanupExpiredTrips_UnauthorizedWithoutToken(t *testing.T) {
	db := setupAdminTestDB(t)
	router := setupAdminRouter(t, db)

	req, err := http.NewRequest(http.MethodPost, "/admin/cleanup/expired-trips", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestCleanupExpiredTrips_Success(t *testing.T) {
	db := setupAdminTestDB(t)
	router := setupAdminRouter(t, db)

	user := createAdminTestUser(t, db)
	token := createAdminToken(t, user)

	oldActiveTrip := createAdminTrip(t, db, models.TripStatusActive, time.Now().AddDate(0, 0, -9))
	createAdminTrip(t, db, models.TripStatusExpired, time.Now().AddDate(0, 0, -9))
	createAdminTrip(t, db, models.TripStatusActive, time.Now().AddDate(0, 0, -1))

	req, err := http.NewRequest(http.MethodPost, "/admin/cleanup/expired-trips", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}

	if body["message"] != "cleanup completed" {
		t.Fatalf("expected message 'cleanup completed', got %v", body["message"])
	}

	expiredCount, ok := body["expired_trips_count"].(float64)
	if !ok {
		t.Fatalf("expected numeric expired_trips_count, got %T", body["expired_trips_count"])
	}

	if int64(expiredCount) != 1 {
		t.Fatalf("expected expired_trips_count=1, got %v", body["expired_trips_count"])
	}

	var updatedTrip models.TripOffer
	if err := db.First(&updatedTrip, oldActiveTrip.ID).Error; err != nil {
		t.Fatalf("failed to reload trip: %v", err)
	}

	if updatedTrip.Status != models.TripStatusExpired {
		t.Fatalf("expected trip status=%s, got %s", models.TripStatusExpired, updatedTrip.Status)
	}
}
