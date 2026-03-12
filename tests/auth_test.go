package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/korgx9/safar-backend/internal/models"
	"github.com/korgx9/safar-backend/internal/services"
)

func TestAuthSendOTPReturns200(t *testing.T) {
	_, router := setupTestEnv(t)

	w := performRequest(
		t,
		router,
		http.MethodPost,
		"/auth/send-otp",
		map[string]interface{}{"phone_number": "+992900001001"},
		"",
	)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAuthVerifyOTPReturnsToken(t *testing.T) {
	db, router := setupTestEnv(t)
	phone := "+992900001002"

	sendRes := performRequest(
		t,
		router,
		http.MethodPost,
		"/auth/send-otp",
		map[string]interface{}{"phone_number": phone},
		"",
	)
	if sendRes.Code != http.StatusOK {
		t.Fatalf("expected send-otp status 200, got %d, body=%s", sendRes.Code, sendRes.Body.String())
	}

	otp := latestOTPCode(t, db, phone)

	verifyRes := performRequest(
		t,
		router,
		http.MethodPost,
		"/auth/verify-otp",
		map[string]interface{}{
			"phone_number": phone,
			"code":         otp.Code,
		},
		"",
	)
	if verifyRes.Code != http.StatusOK {
		t.Fatalf("expected verify-otp status 200, got %d, body=%s", verifyRes.Code, verifyRes.Body.String())
	}

	resp := decodeJSON[models.AuthResponse](t, verifyRes)
	if resp.AccessToken == "" {
		t.Fatalf("expected access token, got empty, body=%s", verifyRes.Body.String())
	}
}

func TestAuthMeWithTokenWorks(t *testing.T) {
	db, router := setupTestEnv(t)
	user, token := createTestUserAndToken(t, db, "+992900001003", models.UserRolePassenger)

	w := performRequest(t, router, http.MethodGet, "/me", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	got := decodeJSON[models.User](t, w)
	if got.ID != user.ID {
		t.Fatalf("expected user id=%d, got %d", user.ID, got.ID)
	}
}

func TestAuthMeWithoutTokenFails(t *testing.T) {
	_, router := setupTestEnv(t)

	w := performRequest(t, router, http.MethodGet, "/me", nil, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestOTPServiceGenerateAndVerify(t *testing.T) {
	db, _ := setupTestEnv(t)
	service := services.NewOTPService(db)

	otp, err := service.Generate("+992900001004")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if otp == nil {
		t.Fatal("expected otp, got nil")
	}
	if len(otp.Code) != 6 {
		t.Fatalf("expected otp code length 6, got %d", len(otp.Code))
	}
	if otp.Verified {
		t.Fatal("expected new otp to be unverified")
	}
	if otp.ExpiresAt.Before(time.Now()) {
		t.Fatalf("expected otp expiry in future, got %v", otp.ExpiresAt)
	}

	verified, err := service.Verify("+992900001004", otp.Code)
	if err != nil {
		t.Fatalf("expected verify success, got %v", err)
	}
	if verified == nil || !verified.Verified {
		t.Fatal("expected verified otp")
	}
}

func TestOTPServiceVerifyInvalidCode(t *testing.T) {
	db, _ := setupTestEnv(t)
	service := services.NewOTPService(db)

	_, err := service.Generate("+992900001005")
	if err != nil {
		t.Fatalf("failed to generate otp: %v", err)
	}

	_, err = service.Verify("+992900001005", "000000")
	if err == nil {
		t.Fatal("expected error for invalid otp code, got nil")
	}
}
