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

func setupPassengerSearchTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Vehicle{},
		&models.TripOffer{},
		&models.SearchSession{},
	); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

func setupPassengerSearchRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	handler := NewPassengerSearchHandler(db)
	router := gin.Default()

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware("test-secret"))
	{
		protected.POST("/passenger/search", handler.Search)
		protected.POST("/passenger/search/:sessionId/next", handler.Next)
	}

	return router
}

func createPassengerSearchUser(t *testing.T, db *gorm.DB, phone string, role models.UserRole) models.User {
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

func createPassengerSearchToken(t *testing.T, user models.User) string {
	t.Helper()

	jwtService := services.NewJWTService("test-secret")
	token, err := jwtService.Generate(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return token
}

func createPassengerSearchVehicle(t *testing.T, db *gorm.DB, userID uint, seats int) models.Vehicle {
	t.Helper()

	vehicle := models.Vehicle{
		UserID:     userID,
		SeatsTotal: seats,
	}

	if err := db.Create(&vehicle).Error; err != nil {
		t.Fatalf("failed to create vehicle: %v", err)
	}

	return vehicle
}

func createPassengerSearchTrip(
	t *testing.T,
	db *gorm.DB,
	driverID uint,
	vehicleID uint,
	origin string,
	destination string,
	tripDate time.Time,
	availableSeats int,
	status string,
	createdAt time.Time,
) models.TripOffer {
	t.Helper()

	trip := models.TripOffer{
		DriverID:       driverID,
		VehicleID:      vehicleID,
		Origin:         origin,
		Destination:    destination,
		TripDate:       tripDate,
		AvailableSeats: availableSeats,
		TotalSeats:     4,
		Status:         status,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}

	if err := db.Create(&trip).Error; err != nil {
		t.Fatalf("failed to create trip: %v", err)
	}

	return trip
}

func makePassengerSearchBody(t *testing.T, payload map[string]interface{}) []byte {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	return body
}

func TestPassengerSearch_Success_ReturnsFirstTripInQueue(t *testing.T) {
	db := setupPassengerSearchTestDB(t)
	router := setupPassengerSearchRouter(t, db)

	passenger := createPassengerSearchUser(t, db, "+992900000031", models.UserRolePassenger)
	token := createPassengerSearchToken(t, passenger)

	driver1 := createPassengerSearchUser(t, db, "+992900000032", models.UserRoleDriver)
	driver2 := createPassengerSearchUser(t, db, "+992900000033", models.UserRoleDriver)

	driver1Vehicle := createPassengerSearchVehicle(t, db, driver1.ID, 4)
	driver2Vehicle := createPassengerSearchVehicle(t, db, driver2.ID, 6)

	searchDay := time.Now().UTC().Add(48 * time.Hour)
	searchDayStart, _ := dayRangeUTC(searchDay)
	baseCreatedAt := time.Now().UTC().Add(-2 * time.Hour)

	firstMatchingTrip := createPassengerSearchTrip(
		t, db, driver1.ID, driver1Vehicle.ID, "Dushanbe", "Khujand",
		searchDayStart.Add(10*time.Hour), 3, models.TripStatusActive, baseCreatedAt,
	)
	createPassengerSearchTrip(
		t, db, driver2.ID, driver2Vehicle.ID, "Dushanbe", "Khujand",
		searchDayStart.Add(12*time.Hour), 4, models.TripStatusActive, baseCreatedAt.Add(1*time.Minute),
	)

	createPassengerSearchTrip(
		t, db, driver2.ID, driver2Vehicle.ID, "Dushanbe", "Khujand",
		searchDayStart.Add(14*time.Hour), 4, models.TripStatusCancelled, baseCreatedAt.Add(-1*time.Minute),
	)
	createPassengerSearchTrip(
		t, db, driver2.ID, driver2Vehicle.ID, "Dushanbe", "Khujand",
		searchDayStart.Add(15*time.Hour), 1, models.TripStatusActive, baseCreatedAt.Add(-2*time.Minute),
	)

	body := makePassengerSearchBody(t, map[string]interface{}{
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       searchDayStart.Format(time.RFC3339),
		"requested_seats": 2,
	})

	req, err := http.NewRequest(http.MethodPost, "/passenger/search", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp models.SearchTripResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}

	if resp.SearchSessionID == 0 {
		t.Fatalf("expected non-zero search_session_id, got %d", resp.SearchSessionID)
	}

	if !resp.IsFirstInQueue {
		t.Fatal("expected is_first_in_queue=true")
	}

	if resp.Warning != nil {
		t.Fatalf("expected warning to be null, got %v", *resp.Warning)
	}

	if resp.Trip.ID != firstMatchingTrip.ID {
		t.Fatalf("expected trip id=%d, got %d", firstMatchingTrip.ID, resp.Trip.ID)
	}

	var session models.SearchSession
	if err := db.First(&session, resp.SearchSessionID).Error; err != nil {
		t.Fatalf("failed to load search session: %v", err)
	}

	if session.CurrentOffset != 0 {
		t.Fatalf("expected current_offset=0, got %d", session.CurrentOffset)
	}
}

func TestPassengerSearch_NotFound_StillCreatesSession(t *testing.T) {
	db := setupPassengerSearchTestDB(t)
	router := setupPassengerSearchRouter(t, db)

	passenger := createPassengerSearchUser(t, db, "+992900000034", models.UserRolePassenger)
	token := createPassengerSearchToken(t, passenger)

	searchDay := time.Now().UTC().Add(72 * time.Hour)
	searchDayStart, _ := dayRangeUTC(searchDay)

	body := makePassengerSearchBody(t, map[string]interface{}{
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       searchDayStart.Format(time.RFC3339),
		"requested_seats": 2,
	})

	req, err := http.NewRequest(http.MethodPost, "/passenger/search", bytes.NewBuffer(body))
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

	var sessions []models.SearchSession
	if err := db.Where("passenger_id = ?", passenger.ID).Find(&sessions).Error; err != nil {
		t.Fatalf("failed to query search sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 search session, got %d", len(sessions))
	}

	if sessions[0].CurrentOffset != 0 {
		t.Fatalf("expected current_offset=0, got %d", sessions[0].CurrentOffset)
	}
}

func TestPassengerSearchNext_Success_ReturnsNextTripAndUpdatesOffset(t *testing.T) {
	db := setupPassengerSearchTestDB(t)
	router := setupPassengerSearchRouter(t, db)

	passenger := createPassengerSearchUser(t, db, "+992900000035", models.UserRolePassenger)
	token := createPassengerSearchToken(t, passenger)

	driver := createPassengerSearchUser(t, db, "+992900000036", models.UserRoleDriver)
	vehicle := createPassengerSearchVehicle(t, db, driver.ID, 4)

	searchDay := time.Now().UTC().Add(48 * time.Hour)
	searchDayStart, _ := dayRangeUTC(searchDay)
	baseCreatedAt := time.Now().UTC().Add(-1 * time.Hour)

	createPassengerSearchTrip(
		t, db, driver.ID, vehicle.ID, "Dushanbe", "Khujand",
		searchDayStart.Add(9*time.Hour), 3, models.TripStatusActive, baseCreatedAt,
	)
	secondTrip := createPassengerSearchTrip(
		t, db, driver.ID, vehicle.ID, "Dushanbe", "Khujand",
		searchDayStart.Add(11*time.Hour), 3, models.TripStatusActive, baseCreatedAt.Add(1*time.Minute),
	)

	searchBody := makePassengerSearchBody(t, map[string]interface{}{
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       searchDayStart.Format(time.RFC3339),
		"requested_seats": 2,
	})

	searchReq, err := http.NewRequest(http.MethodPost, "/passenger/search", bytes.NewBuffer(searchBody))
	if err != nil {
		t.Fatalf("failed to create search request: %v", err)
	}
	searchReq.Header.Set("Content-Type", "application/json")
	searchReq.Header.Set("Authorization", "Bearer "+token)

	searchRes := httptest.NewRecorder()
	router.ServeHTTP(searchRes, searchReq)
	if searchRes.Code != http.StatusOK {
		t.Fatalf("expected search status 200, got %d, body=%s", searchRes.Code, searchRes.Body.String())
	}

	var searchResp models.SearchTripResponse
	if err := json.Unmarshal(searchRes.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("failed to parse search response: %v, body=%s", err, searchRes.Body.String())
	}

	nextReq, err := http.NewRequest(
		http.MethodPost,
		"/passenger/search/"+strconv.FormatUint(uint64(searchResp.SearchSessionID), 10)+"/next",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create next request: %v", err)
	}
	nextReq.Header.Set("Authorization", "Bearer "+token)

	nextRes := httptest.NewRecorder()
	router.ServeHTTP(nextRes, nextReq)

	if nextRes.Code != http.StatusOK {
		t.Fatalf("expected next status 200, got %d, body=%s", nextRes.Code, nextRes.Body.String())
	}

	var nextResp models.SearchTripResponse
	if err := json.Unmarshal(nextRes.Body.Bytes(), &nextResp); err != nil {
		t.Fatalf("failed to parse next response: %v, body=%s", err, nextRes.Body.String())
	}

	if nextResp.Trip.ID != secondTrip.ID {
		t.Fatalf("expected next trip id=%d, got %d", secondTrip.ID, nextResp.Trip.ID)
	}

	if nextResp.IsFirstInQueue {
		t.Fatal("expected is_first_in_queue=false for next response")
	}

	if nextResp.Warning == nil || *nextResp.Warning != nextTripWarningMessage {
		t.Fatalf("expected warning=%q, got %v", nextTripWarningMessage, nextResp.Warning)
	}

	var session models.SearchSession
	if err := db.First(&session, searchResp.SearchSessionID).Error; err != nil {
		t.Fatalf("failed to load updated search session: %v", err)
	}

	if session.CurrentOffset != 1 {
		t.Fatalf("expected current_offset=1, got %d", session.CurrentOffset)
	}
}

func TestPassengerSearchNext_Forbidden_WhenSessionBelongsToAnotherUser(t *testing.T) {
	db := setupPassengerSearchTestDB(t)
	router := setupPassengerSearchRouter(t, db)

	passenger1 := createPassengerSearchUser(t, db, "+992900000037", models.UserRolePassenger)
	passenger2 := createPassengerSearchUser(t, db, "+992900000038", models.UserRolePassenger)
	token1 := createPassengerSearchToken(t, passenger1)
	token2 := createPassengerSearchToken(t, passenger2)

	driver := createPassengerSearchUser(t, db, "+992900000039", models.UserRoleDriver)
	vehicle := createPassengerSearchVehicle(t, db, driver.ID, 4)

	searchDay := time.Now().UTC().Add(48 * time.Hour)
	searchDayStart, _ := dayRangeUTC(searchDay)
	createPassengerSearchTrip(
		t, db, driver.ID, vehicle.ID, "Dushanbe", "Khujand",
		searchDayStart.Add(10*time.Hour), 3, models.TripStatusActive, time.Now().UTC(),
	)

	searchBody := makePassengerSearchBody(t, map[string]interface{}{
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       searchDayStart.Format(time.RFC3339),
		"requested_seats": 2,
	})

	searchReq, err := http.NewRequest(http.MethodPost, "/passenger/search", bytes.NewBuffer(searchBody))
	if err != nil {
		t.Fatalf("failed to create search request: %v", err)
	}
	searchReq.Header.Set("Content-Type", "application/json")
	searchReq.Header.Set("Authorization", "Bearer "+token1)

	searchRes := httptest.NewRecorder()
	router.ServeHTTP(searchRes, searchReq)
	if searchRes.Code != http.StatusOK {
		t.Fatalf("expected search status 200, got %d, body=%s", searchRes.Code, searchRes.Body.String())
	}

	var searchResp models.SearchTripResponse
	if err := json.Unmarshal(searchRes.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("failed to parse search response: %v, body=%s", err, searchRes.Body.String())
	}

	nextReq, err := http.NewRequest(
		http.MethodPost,
		"/passenger/search/"+strconv.FormatUint(uint64(searchResp.SearchSessionID), 10)+"/next",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create next request: %v", err)
	}
	nextReq.Header.Set("Authorization", "Bearer "+token2)

	nextRes := httptest.NewRecorder()
	router.ServeHTTP(nextRes, nextReq)

	if nextRes.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d, body=%s", nextRes.Code, nextRes.Body.String())
	}
}

func TestPassengerSearchNext_NotFound_WhenNoMoreTripsInQueue(t *testing.T) {
	db := setupPassengerSearchTestDB(t)
	router := setupPassengerSearchRouter(t, db)

	passenger := createPassengerSearchUser(t, db, "+992900000040", models.UserRolePassenger)
	token := createPassengerSearchToken(t, passenger)

	driver := createPassengerSearchUser(t, db, "+992900000041", models.UserRoleDriver)
	vehicle := createPassengerSearchVehicle(t, db, driver.ID, 4)

	searchDay := time.Now().UTC().Add(48 * time.Hour)
	searchDayStart, _ := dayRangeUTC(searchDay)
	createPassengerSearchTrip(
		t, db, driver.ID, vehicle.ID, "Dushanbe", "Khujand",
		searchDayStart.Add(10*time.Hour), 3, models.TripStatusActive, time.Now().UTC(),
	)

	searchBody := makePassengerSearchBody(t, map[string]interface{}{
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       searchDayStart.Format(time.RFC3339),
		"requested_seats": 2,
	})

	searchReq, err := http.NewRequest(http.MethodPost, "/passenger/search", bytes.NewBuffer(searchBody))
	if err != nil {
		t.Fatalf("failed to create search request: %v", err)
	}
	searchReq.Header.Set("Content-Type", "application/json")
	searchReq.Header.Set("Authorization", "Bearer "+token)

	searchRes := httptest.NewRecorder()
	router.ServeHTTP(searchRes, searchReq)
	if searchRes.Code != http.StatusOK {
		t.Fatalf("expected search status 200, got %d, body=%s", searchRes.Code, searchRes.Body.String())
	}

	var searchResp models.SearchTripResponse
	if err := json.Unmarshal(searchRes.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("failed to parse search response: %v, body=%s", err, searchRes.Body.String())
	}

	nextReq, err := http.NewRequest(
		http.MethodPost,
		"/passenger/search/"+strconv.FormatUint(uint64(searchResp.SearchSessionID), 10)+"/next",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create next request: %v", err)
	}
	nextReq.Header.Set("Authorization", "Bearer "+token)

	nextRes := httptest.NewRecorder()
	router.ServeHTTP(nextRes, nextReq)

	if nextRes.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", nextRes.Code, nextRes.Body.String())
	}
}

func TestPassengerSearch_BadRequest_WhenRequestedSeatsInvalid(t *testing.T) {
	db := setupPassengerSearchTestDB(t)
	router := setupPassengerSearchRouter(t, db)

	passenger := createPassengerSearchUser(t, db, "+992900000042", models.UserRolePassenger)
	token := createPassengerSearchToken(t, passenger)

	searchDay := time.Now().UTC().Add(48 * time.Hour)
	searchDayStart, _ := dayRangeUTC(searchDay)
	body := makePassengerSearchBody(t, map[string]interface{}{
		"origin":          "Dushanbe",
		"destination":     "Khujand",
		"trip_date":       searchDayStart.Format(time.RFC3339),
		"requested_seats": 0,
	})

	req, err := http.NewRequest(http.MethodPost, "/passenger/search", bytes.NewBuffer(body))
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

func TestPassengerSearchNext_NotFound_WhenSessionDoesNotExist(t *testing.T) {
	db := setupPassengerSearchTestDB(t)
	router := setupPassengerSearchRouter(t, db)

	passenger := createPassengerSearchUser(t, db, "+992900000043", models.UserRolePassenger)
	token := createPassengerSearchToken(t, passenger)

	req, err := http.NewRequest(http.MethodPost, "/passenger/search/999999/next", nil)
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
