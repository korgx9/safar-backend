package models

import "time"

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type SendOTPResponse struct {
	Message   string    `json:"message"`
	OTPCode   string    `json:"otp_code"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CleanupExpiredTripsResponse struct {
	Message           string `json:"message"`
	ExpiredTripsCount int64  `json:"expired_trips_count"`
}
