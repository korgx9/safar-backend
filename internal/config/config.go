package config

import (
	"os"
)

type Config struct {
	Port      string
	DBUrl     string
	JWTSecret string
}

func LoadConfig() Config {
	port := os.Getenv("HTTP_PORT")

	db := "host=" + os.Getenv("DB_HOST") +
		" user=" + os.Getenv("DB_USER") +
		" password=" + os.Getenv("DB_PASSWORD") +
		" dbname=" + os.Getenv("DB_NAME") +
		" port=" + os.Getenv("DB_PORT") +
		" sslmode=" + os.Getenv("DB_SSLMODE")

	return Config{
		Port:      port,
		DBUrl:     db,
		JWTSecret: os.Getenv("JWT_SECRET"),
	}
}
