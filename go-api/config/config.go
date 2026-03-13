package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	RedisURL       string
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioPublicURL string
}

func Load() *Config {
	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			getEnv("POSTGRES_USER", "yourusername"),
			getEnv("POSTGRES_PASSWORD", "yourpassword"),
			getEnv("POSTGRES_HOST", "db"),
			getEnv("POSTGRES_PORT", "5432"),
			getEnv("POSTGRES_DB", "yourdatabase"),
			getEnv("POSTGRES_SSLMODE", "disable"),
		)
	}

	return &Config{
		Port:           getEnv("PORT", "8000"),
		DatabaseURL:    dbURL,
		JWTSecret:      getEnv("JWT_SECRET", "allshop-secret-key-change-in-production"),
		RedisURL:       getEnv("REDIS_URL", "redis://redis:6379/0"),
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    getEnv("MINIO_BUCKET", "allshop-images"),
		MinioPublicURL: getEnv("MINIO_PUBLIC_URL", "http://localhost:9000"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
