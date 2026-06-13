package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL          string
	ServerPort           string
	GitHubClientID       string
	GitHubClientSecret   string
	JWTSecret            string
	WebURL               string
	GitHubAppID          int64
	GitHubAppPrivateKey  []byte
	GitHubWebhookSecret  string
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

	// GitHub App configuration
	appIDStr := os.Getenv("GITHUB_APP_ID")
	if appIDStr == "" {
		return nil, fmt.Errorf("GITHUB_APP_ID is required")
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("GITHUB_APP_ID must be a valid integer: %w", err)
	}
	cfg.GitHubAppID = appID

	privateKeyRaw := os.Getenv("GITHUB_APP_PRIVATE_KEY")
	if privateKeyRaw == "" {
		return nil, fmt.Errorf("GITHUB_APP_PRIVATE_KEY is required")
	}
	// Attempt base64 decode first; fall back to raw string
	decoded, decodeErr := base64.StdEncoding.DecodeString(privateKeyRaw)
	if decodeErr != nil {
		cfg.GitHubAppPrivateKey = []byte(privateKeyRaw)
	} else {
		cfg.GitHubAppPrivateKey = decoded
	}

	cfg.GitHubWebhookSecret = os.Getenv("GITHUB_WEBHOOK_SECRET")
	if cfg.GitHubWebhookSecret == "" {
		return nil, fmt.Errorf("GITHUB_WEBHOOK_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
