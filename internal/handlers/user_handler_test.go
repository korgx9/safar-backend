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
)

func setupMeTestDB(t *testing.T) *gorm.DB {
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

func setupMeRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	userHandler := NewUserHandler(db)

	router := gin.Default()
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware("test-secret"))
	{
		protected.GET("/me", userHandler.Me)
	}

	return router
}

func TestMe_Unauthorized_WithoutToken(t *testing.T) {
	db := setupMeTestDB(t)
	router := setupMeRouter(t, db)

	req, err := http.NewRequest(http.MethodGet, "/me", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestMe_Unauthorized_InvalidToken(t *testing.T) {
	db := setupMeTestDB(t)
	router := setupMeRouter(t, db)

	req, err := http.NewRequest(http.MethodGet, "/me", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer invalid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestMe_Success(t *testing.T) {
	db := setupMeTestDB(t)
	authRouter := setupRouter(t, db)
	meRouter := setupMeRouter(t, db)

	phoneNumber := "+992900000001"

	sendOTPRequest(t, authRouter, phoneNumber)
	otp := fetchLatestOTP(t, db, phoneNumber)
	createTestUser(t, db, phoneNumber)
	token := verifyOTPAndGetToken(t, authRouter, phoneNumber, otp.Code)

	meReq, err := http.NewRequest(http.MethodGet, "/me", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	meReq.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	meRouter.ServeHTTP(w, meReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func sendOTPRequest(t *testing.T, router *gin.Engine, phoneNumber string) {
	t.Helper()

	body := []byte(`{"phone_number":"` + phoneNumber + `"}`)
	req, err := http.NewRequest(http.MethodPost, "/auth/send-otp", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create send request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected send-otp status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func fetchLatestOTP(t *testing.T, db *gorm.DB, phoneNumber string) models.OTPCode {
	t.Helper()

	var otp models.OTPCode
	if err := db.Where("phone_number = ?", phoneNumber).
		Order("id desc").
		First(&otp).Error; err != nil {
		t.Fatalf("failed to fetch otp from db: %v", err)
	}

	return otp
}

func createTestUser(t *testing.T, db *gorm.DB, phoneNumber string) models.User {
	t.Helper()

	user := models.User{
		PhoneNumber: phoneNumber,
		Role:        models.UserRolePassenger,
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func verifyOTPAndGetToken(t *testing.T, router *gin.Engine, phoneNumber, code string) string {
	t.Helper()

	verifyBody := []byte(`{"phone_number":"` + phoneNumber + `","code":"` + code + `"}`)
	verifyReq, err := http.NewRequest(http.MethodPost, "/auth/verify-otp", bytes.NewBuffer(verifyBody))
	if err != nil {
		t.Fatalf("failed to create verify request: %v", err)
	}
	verifyReq.Header.Set("Content-Type", "application/json")

	verifyRes := httptest.NewRecorder()
	router.ServeHTTP(verifyRes, verifyReq)

	if verifyRes.Code != http.StatusOK {
		t.Fatalf("expected verify-otp status 200, got %d, body=%s", verifyRes.Code, verifyRes.Body.String())
	}

	var verifyResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(verifyRes.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("failed to parse verify response: %v, body=%s", err, verifyRes.Body.String())
	}

	if verifyResp.AccessToken == "" {
		t.Fatalf("expected access_token, got empty string, body=%s", verifyRes.Body.String())
	}

	return verifyResp.AccessToken
}
