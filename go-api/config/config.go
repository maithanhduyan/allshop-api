package config

import "os"

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
	return &Config{
		Port:           getEnv("PORT", "8000"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://yourusername:yourpassword@db:5432/yourdatabase?sslmode=disable"),
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
