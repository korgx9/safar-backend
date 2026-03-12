package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/models"
	"github.com/korgx9/safar-backend/internal/services"
)

type AuthHandler struct {
	db         *gorm.DB
	otpService *services.OTPService
	jwtService *services.JWTService
}

func NewAuthHandler(db *gorm.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		db:         db,
		otpService: services.NewOTPService(db),
		jwtService: services.NewJWTService(jwtSecret),
	}
}

// SendOTP godoc
// @Summary Send OTP code
// @Description Generates and stores OTP code for a phone number.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.SendOTPRequest true "Send OTP request"
// @Success 200 {object} models.SendOTPResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/send-otp [post]
func (h *AuthHandler) SendOTP(c *gin.Context) {
	var req models.SendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	otp, err := h.otpService.Generate(req.PhoneNumber)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "failed to generate otp")
		return
	}

	c.JSON(http.StatusOK, models.SendOTPResponse{
		Message:   "OTP generated successfully",
		OTPCode:   otp.Code,
		ExpiresAt: otp.ExpiresAt,
	})
}

// VerifyOTP godoc
// @Summary Verify OTP code
// @Description Verifies OTP and returns JWT access token with user profile.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.VerifyOTPRequest true "Verify OTP request"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req models.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	_, err := h.otpService.Verify(req.PhoneNumber, req.Code)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, "invalid or expired otp")
		return
	}

	var user models.User
	err = h.db.Where("phone_number = ?", req.PhoneNumber).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = models.User{
				PhoneNumber: req.PhoneNumber,
				Role:        models.UserRolePassenger,
			}
			if err := h.db.Create(&user).Error; err != nil {
				respondWithError(c, http.StatusInternalServerError, "failed to create user")
				return
			}
		} else {
			respondWithError(c, http.StatusInternalServerError, "failed to fetch user")
			return
		}
	}

	token, err := h.jwtService.Generate(user)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken: token,
		User:        user,
	})
}
