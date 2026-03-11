package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/korgx9/safar-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&models.OTPCode{}, &models.User{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

func setupRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.Default()

	authHandler := NewAuthHandler(db, "test-secret")

	auth := router.Group("/auth")
	{
		auth.POST("/send-otp", authHandler.SendOTP)
		auth.POST("/verify-otp", authHandler.VerifyOTP)
	}

	return router
}

func TestSendOTP_Success(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(t, db)

	body := []byte(`{"phone_number":"+992900000001"}`)

	req, err := http.NewRequest(http.MethodPost, "/auth/send-otp", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestSendOTP_BadRequest_WhenPhoneNumberIsMissing(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(t, db)

	body := []byte(`{}`)

	req, err := http.NewRequest(http.MethodPost, "/auth/send-otp", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestVerifyOTP_Success(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(t, db)

	sendBody := []byte(`{"phone_number":"+992900000001"}`)
	sendReq, err := http.NewRequest(http.MethodPost, "/auth/send-otp", bytes.NewBuffer(sendBody))
	if err != nil {
		t.Fatalf("failed to create send request: %v", err)
	}
	sendReq.Header.Set("Content-Type", "application/json")

	sendW := httptest.NewRecorder()
	router.ServeHTTP(sendW, sendReq)

	if sendW.Code != http.StatusOK {
		t.Fatalf("expected send-otp status 200, got %d, body=%s", sendW.Code, sendW.Body.String())
	}

	var otp models.OTPCode
	if err := db.Where("phone_number = ?", "+992900000001").
		Order("id desc").
		First(&otp).Error; err != nil {
		t.Fatalf("failed to fetch otp from db: %v", err)
	}

	user := models.User{
		PhoneNumber: "+992900000001",
		Role:        models.UserRolePassenger,
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	verifyBody := []byte(`{"phone_number":"+992900000001","code":"` + otp.Code + `"}`)

	verifyReq, err := http.NewRequest(http.MethodPost, "/auth/verify-otp", bytes.NewBuffer(verifyBody))
	if err != nil {
		t.Fatalf("failed to create verify request: %v", err)
	}
	verifyReq.Header.Set("Content-Type", "application/json")

	verifyW := httptest.NewRecorder()
	router.ServeHTTP(verifyW, verifyReq)

	if verifyW.Code != http.StatusOK {
		t.Fatalf("expected verify-otp status 200, got %d, body=%s", verifyW.Code, verifyW.Body.String())
	}
}

func TestVerifyOTP_InvalidCode(t *testing.T) {
	db := setupTestDB(t)
	router := setupRouter(t, db)

	sendBody := []byte(`{"phone_number":"+992900000001"}`)

	sendReq, err := http.NewRequest(http.MethodPost, "/auth/send-otp", bytes.NewBuffer(sendBody))
	if err != nil {
		t.Fatalf("failed to create send request: %v", err)
	}
	sendReq.Header.Set("Content-Type", "application/json")

	sendW := httptest.NewRecorder()
	router.ServeHTTP(sendW, sendReq)

	if sendW.Code != http.StatusOK {
		t.Fatalf("expected send-otp status 200, got %d, body=%s", sendW.Code, sendW.Body.String())
	}

	verifyBody := []byte(`{"phone_number":"+992900000001","code":"000000"}`)

	verifyReq, err := http.NewRequest(http.MethodPost, "/auth/verify-otp", bytes.NewBuffer(verifyBody))
	if err != nil {
		t.Fatalf("failed to create verify request: %v", err)
	}
	verifyReq.Header.Set("Content-Type", "application/json")

	verifyW := httptest.NewRecorder()
	router.ServeHTTP(verifyW, verifyReq)

	if verifyW.Code == http.StatusOK {
		t.Fatalf("expected verification to fail with invalid code, got status %d body=%s",
			verifyW.Code, verifyW.Body.String())
	}
}
