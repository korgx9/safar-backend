package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	Port      string
	DBUrl     string
	JWTSecret string
}

func LoadConfig() Config {
	port := getEnvOrDefault("HTTP_PORT", "8080")
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbSSLMode := getEnvOrDefault("DB_SSLMODE", "disable")

	db := "host=" + dbHost +
		" user=" + mustGetEnv("DB_USER") +
		" password=" + mustGetEnv("DB_PASSWORD") +
		" dbname=" + mustGetEnv("DB_NAME") +
		" port=" + dbPort +
		" sslmode=" + dbSSLMode

	return Config{
		Port:      port,
		DBUrl:     db,
		JWTSecret: mustGetEnv("JWT_SECRET"),
	}
}

func getEnvOrDefault(key string, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return defaultValue
}

func mustGetEnv(key string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	log.Fatalf("missing required environment variable: %s", key)
	return ""
}
