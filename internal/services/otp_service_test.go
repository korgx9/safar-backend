package services

import (
	"testing"
	"time"

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

	if err := db.AutoMigrate(&models.OTPCode{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

func TestOTPService_Generate_Success(t *testing.T) {
	db := setupTestDB(t)
	service := NewOTPService(db)

	otp, err := service.Generate("+992900000001")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if otp == nil {
		t.Fatal("expected otp, got nil")
	}

	if otp.PhoneNumber != "+992900000001" {
		t.Fatalf("expected phone number +992900000001, got %s", otp.PhoneNumber)
	}

	if len(otp.Code) != 6 {
		t.Fatalf("expected code length 6, got %d, code=%s", len(otp.Code), otp.Code)
	}

	for _, ch := range otp.Code {
		if ch < '0' || ch > '9' {
			t.Fatalf("expected numeric code, got %s", otp.Code)
		}
	}

	if otp.Verified {
		t.Fatal("expected verified to be false for newly generated otp")
	}

	if otp.ExpiresAt.Before(time.Now()) {
		t.Fatalf("expected expires_at to be in the future, got %v", otp.ExpiresAt)
	}
}

func TestOTPService_Verify_Success(t *testing.T) {
	db := setupTestDB(t)
	service := NewOTPService(db)

	created, err := service.Generate("+992900000001")
	if err != nil {
		t.Fatalf("failed to generate otp: %v", err)
	}

	verified, err := service.Verify("+992900000001", created.Code)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if verified == nil {
		t.Fatal("expected verified otp, got nil")
	}

	if verified.PhoneNumber != "+992900000001" {
		t.Fatalf("expected phone number +992900000001, got %s", verified.PhoneNumber)
	}

	if verified.Code != created.Code {
		t.Fatalf("expected code %s, got %s", created.Code, verified.Code)
	}

	if !verified.Verified {
		t.Fatal("expected verified to be true after successful verification")
	}
}

func TestOTPService_Verify_InvalidCode(t *testing.T) {
	db := setupTestDB(t)
	service := NewOTPService(db)

	_, err := service.Generate("+992900000001")
	if err != nil {
		t.Fatalf("failed to generate otp: %v", err)
	}

	result, err := service.Verify("+992900000001", "000000")
	if err == nil {
		t.Fatalf("expected error for invalid code, got nil, result=%v", result)
	}
}
