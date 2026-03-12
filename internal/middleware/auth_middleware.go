package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondWithError(c, http.StatusUnauthorized, "authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondWithError(c, http.StatusUnauthorized, "invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			respondWithError(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			respondWithError(c, http.StatusUnauthorized, "invalid token claims")
			c.Abort()
			return
		}

		userID, ok := claims["user_id"]
		if !ok {
			respondWithError(c, http.StatusUnauthorized, "user_id not found in token")
			c.Abort()
			return
		}

		phoneNumber, _ := claims["phone_number"].(string)
		role, _ := claims["role"].(string)

		c.Set("user_id", uint(userID.(float64)))
		c.Set("phone_number", phoneNumber)
		c.Set("role", role)

		c.Next()
	}
}
