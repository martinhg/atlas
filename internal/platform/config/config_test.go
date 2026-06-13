package config

import (
	"encoding/base64"
	"testing"
)

// setRequiredVars sets the three mandatory env vars that have no default.
func setRequiredVars(t *testing.T) {
	t.Helper()
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
}

// setGitHubAppVars sets the three new GitHub App env vars.
func setGitHubAppVars(t *testing.T) {
	t.Helper()
	t.Setenv("GITHUB_APP_ID", "12345")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "test-private-key-pem")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "test-webhook-secret")
}

func TestLoad_allVarsPresent(t *testing.T) {
	setRequiredVars(t)
	setGitHubAppVars(t)
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
	setGitHubAppVars(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", "")
	t.Setenv("WEB_URL", "")

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
	setGitHubAppVars(t)

	_, err := Load()
	if err == nil {
		t.Error("expected error when GITHUB_CLIENT_ID is missing, got nil")
	}
}

func TestLoad_missingGitHubClientSecret(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	setGitHubAppVars(t)

	_, err := Load()
	if err == nil {
		t.Error("expected error when GITHUB_CLIENT_SECRET is missing, got nil")
	}
}

func TestLoad_missingJWTSecret(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	t.Setenv("JWT_SECRET", "")
	setGitHubAppVars(t)

	_, err := Load()
	if err == nil {
		t.Error("expected error when JWT_SECRET is missing, got nil")
	}
}

func TestLoad_missingGitHubAppID(t *testing.T) {
	setRequiredVars(t)
	t.Setenv("GITHUB_APP_ID", "")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "test-private-key-pem")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "test-webhook-secret")

	_, err := Load()
	if err == nil {
		t.Error("expected error when GITHUB_APP_ID is missing, got nil")
	}
}

func TestLoad_invalidGitHubAppID(t *testing.T) {
	setRequiredVars(t)
	t.Setenv("GITHUB_APP_ID", "not-a-number")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "test-private-key-pem")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "test-webhook-secret")

	_, err := Load()
	if err == nil {
		t.Error("expected error when GITHUB_APP_ID is not a number, got nil")
	}
}

func TestLoad_missingGitHubAppPrivateKey(t *testing.T) {
	setRequiredVars(t)
	t.Setenv("GITHUB_APP_ID", "12345")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "test-webhook-secret")

	_, err := Load()
	if err == nil {
		t.Error("expected error when GITHUB_APP_PRIVATE_KEY is missing, got nil")
	}
}

func TestLoad_missingGitHubWebhookSecret(t *testing.T) {
	setRequiredVars(t)
	t.Setenv("GITHUB_APP_ID", "12345")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "test-private-key-pem")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Error("expected error when GITHUB_WEBHOOK_SECRET is missing, got nil")
	}
}

func TestLoad_gitHubAppID_parsed(t *testing.T) {
	setRequiredVars(t)
	t.Setenv("GITHUB_APP_ID", "98765")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "test-private-key-pem")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "test-webhook-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.GitHubAppID != 98765 {
		t.Errorf("GitHubAppID = %d, want 98765", cfg.GitHubAppID)
	}
}

func TestLoad_gitHubAppPrivateKey_rawString(t *testing.T) {
	setRequiredVars(t)
	t.Setenv("GITHUB_APP_ID", "12345")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "raw-pem-content")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "test-webhook-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if string(cfg.GitHubAppPrivateKey) != "raw-pem-content" {
		t.Errorf("GitHubAppPrivateKey = %q, want %q", string(cfg.GitHubAppPrivateKey), "raw-pem-content")
	}
}

func TestLoad_gitHubAppPrivateKey_base64Encoded(t *testing.T) {
	setRequiredVars(t)
	t.Setenv("GITHUB_APP_ID", "12345")

	original := "raw-pem-content"
	encoded := base64.StdEncoding.EncodeToString([]byte(original))
	t.Setenv("GITHUB_APP_PRIVATE_KEY", encoded)
	t.Setenv("GITHUB_WEBHOOK_SECRET", "test-webhook-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if string(cfg.GitHubAppPrivateKey) != original {
		t.Errorf("GitHubAppPrivateKey = %q, want decoded %q", string(cfg.GitHubAppPrivateKey), original)
	}
}

func TestLoad_gitHubWebhookSecret_stored(t *testing.T) {
	setRequiredVars(t)
	t.Setenv("GITHUB_APP_ID", "12345")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "test-private-key-pem")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "my-webhook-secret-abc")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.GitHubWebhookSecret != "my-webhook-secret-abc" {
		t.Errorf("GitHubWebhookSecret = %q, want %q", cfg.GitHubWebhookSecret, "my-webhook-secret-abc")
	}
}
