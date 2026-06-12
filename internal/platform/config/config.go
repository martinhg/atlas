package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL        string
	ServerPort         string
	GitHubClientID     string
	GitHubClientSecret string
	JWTSecret          string
	WebURL             string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://atlas:atlas@localhost:5432/atlas?sslmode=disable"),
		ServerPort:         getEnv("PORT", "8080"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		WebURL:             getEnv("WEB_URL", "http://localhost:5173"),
	}

	if cfg.GitHubClientID == "" {
		return nil, fmt.Errorf("GITHUB_CLIENT_ID is required")
	}
	if cfg.GitHubClientSecret == "" {
		return nil, fmt.Errorf("GITHUB_CLIENT_SECRET is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
