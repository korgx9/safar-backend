package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/middleware"
	"github.com/korgx9/safar-backend/internal/models"
	"github.com/korgx9/safar-backend/internal/services"
)

func setupVehicleTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Vehicle{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

func setupVehicleRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	vehicleHandler := NewVehicleHandler(db)
	router := gin.Default()

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware("test-secret"))
	{
		protected.POST("/vehicles", vehicleHandler.Create)
		protected.GET("/vehicles", vehicleHandler.List)
	}

	return router
}

func createVehicleUser(t *testing.T, db *gorm.DB, phone string) models.User {
	t.Helper()

	user := models.User{
		PhoneNumber: phone,
		Role:        models.UserRolePassenger,
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return user
}

func createVehicleToken(t *testing.T, user models.User) string {
	t.Helper()

	jwtService := services.NewJWTService("test-secret")
	token, err := jwtService.Generate(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return token
}

func TestVehicleCreate_Unauthorized_WithoutToken(t *testing.T) {
	db := setupVehicleTestDB(t)
	router := setupVehicleRouter(t, db)

	body := []byte(`{"seats_total":4}`)
	req, err := http.NewRequest(http.MethodPost, "/vehicles", bytes.NewBuffer(body))
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

func TestVehicleCreate_BadRequest_InvalidSeatsTotal(t *testing.T) {
	db := setupVehicleTestDB(t)
	router := setupVehicleRouter(t, db)

	user := createVehicleUser(t, db, "+992900000010")
	token := createVehicleToken(t, user)

	body := []byte(`{"seats_total":0}`)
	req, err := http.NewRequest(http.MethodPost, "/vehicles", bytes.NewBuffer(body))
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

func TestVehicleCreate_Success(t *testing.T) {
	db := setupVehicleTestDB(t)
	router := setupVehicleRouter(t, db)

	user := createVehicleUser(t, db, "+992900000011")
	token := createVehicleToken(t, user)

	body := []byte(`{"seats_total":4}`)
	req, err := http.NewRequest(http.MethodPost, "/vehicles", bytes.NewBuffer(body))
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

	var vehicle models.Vehicle
	if err := json.Unmarshal(w.Body.Bytes(), &vehicle); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}

	if vehicle.UserID != user.ID {
		t.Fatalf("expected user_id=%d, got %d", user.ID, vehicle.UserID)
	}

	if vehicle.SeatsTotal != 4 {
		t.Fatalf("expected seats_total=4, got %d", vehicle.SeatsTotal)
	}
}

func TestVehicleList_Success_ReturnsOnlyOwnVehicles(t *testing.T) {
	db := setupVehicleTestDB(t)
	router := setupVehicleRouter(t, db)

	user1 := createVehicleUser(t, db, "+992900000012")
	user2 := createVehicleUser(t, db, "+992900000013")
	token := createVehicleToken(t, user1)

	vehicles := []models.Vehicle{
		{UserID: user1.ID, SeatsTotal: 4},
		{UserID: user1.ID, SeatsTotal: 6},
		{UserID: user2.ID, SeatsTotal: 2},
	}

	if err := db.Create(&vehicles).Error; err != nil {
		t.Fatalf("failed to seed vehicles: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "/vehicles", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var result []models.Vehicle
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 vehicles, got %d, body=%s", len(result), w.Body.String())
	}

	for _, vehicle := range result {
		if vehicle.UserID != user1.ID {
			t.Fatalf("expected only vehicles for user_id=%d, got vehicle with user_id=%d", user1.ID, vehicle.UserID)
		}
	}
}
