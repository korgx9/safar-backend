package services

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/korgx9/safar-backend/internal/models"
)

type JWTService struct {
	secret string
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: secret}
}

func (s *JWTService) Generate(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":      user.ID,
		"phone_number": user.PhoneNumber,
		"role":         user.Role,
		"exp":          time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}
