package config

import (
	"testing"
)

// setRequiredVars sets the three mandatory env vars that have no default.
func setRequiredVars(t *testing.T) {
	t.Helper()
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
}

func TestLoad_allVarsPresent(t *testing.T) {
	setRequiredVars(t)
	t.Setenv("DATABASE_URL", "postgres://user:pass@host:5432/db?sslmode=disable")
	t.Setenv("PORT", "9090")
	t.Setenv("WEB_URL", "http://example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.GitHubClientID != "test-client-id" {
		t.Errorf("GitHubClientID = %q, want %q", cfg.GitHubClientID, "test-client-id")
	}
	if cfg.GitHubClientSecret != "test-client-secret" {
		t.Errorf("GitHubClientSecret = %q, want %q", cfg.GitHubClientSecret, "test-client-secret")
	}
	if cfg.JWTSecret != "test-jwt-secret" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "test-jwt-secret")
	}
	if cfg.DatabaseURL != "postgres://user:pass@host:5432/db?sslmode=disable" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "9090")
	}
	if cfg.WebURL != "http://example.com" {
		t.Errorf("WebURL = %q, want %q", cfg.WebURL, "http://example.com")
	}
}

func TestLoad_defaults(t *testing.T) {
	setRequiredVars(t)
	// DATABASE_URL and PORT are intentionally not set — they should use defaults.

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	const defaultDB = "postgres://atlas:atlas@localhost:5432/atlas?sslmode=disable"
	if cfg.DatabaseURL != defaultDB {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDB)
	}
	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "8080")
	}
}

func TestLoad_missingGitHubClientID(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	t.Setenv("JWT_SECRET", "test-jwt-secret")

	_, err := Load()
	if err == nil {
		t.Error("expected error when GITHUB_CLIENT_ID is missing, got nil")
	}
}

func TestLoad_missingGitHubClientSecret(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	t.Setenv("JWT_SECRET", "test-jwt-secret")

	_, err := Load()
	if err == nil {
		t.Error("expected error when GITHUB_CLIENT_SECRET is missing, got nil")
	}
}

func TestLoad_missingJWTSecret(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Error("expected error when JWT_SECRET is missing, got nil")
	}
}
