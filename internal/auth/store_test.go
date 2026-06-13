package auth

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nesbite/atlas/migrations"
	"github.com/nesbite/atlas/internal/platform/database"
)

// getTestPool opens a pgxpool using DATABASE_URL and runs migrations.
// The test is skipped when DATABASE_URL is not set (e.g. local dev without Postgres).
func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := database.RunMigrations(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return pool
}

// makeGitHubUser builds a deterministic *User for testing.
func makeGitHubUser(githubID int64, login string) *User {
	return &User{
		GitHubID: githubID,
		Login:    login,
	}
}

// cleanUser removes the test row so tests are independent.
func cleanUser(t *testing.T, pool *pgxpool.Pool, githubID int64) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"DELETE FROM users WHERE github_id = $1", githubID)
	if err != nil {
		t.Logf("cleanup warning: %v", err)
	}
}

func TestUpsertUser_createsNew(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ghID := int64(9_000_001)
	u := makeGitHubUser(ghID, "new-user")
	t.Cleanup(func() { cleanUser(t, pool, ghID) })

	got, err := store.UpsertUser(ctx, u, "token-abc")
	if err != nil {
		t.Fatalf("UpsertUser error: %v", err)
	}
	if got.ID == uuid.Nil {
		t.Error("returned ID is nil UUID")
	}
	if got.Login != "new-user" {
		t.Errorf("Login = %q, want %q", got.Login, "new-user")
	}
	if got.GitHubID != ghID {
		t.Errorf("GitHubID = %d, want %d", got.GitHubID, ghID)
	}
}

func TestUpsertUser_updatesExisting(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ghID := int64(9_000_002)
	t.Cleanup(func() { cleanUser(t, pool, ghID) })

	_, err := store.UpsertUser(ctx, makeGitHubUser(ghID, "original-login"), "token-1")
	if err != nil {
		t.Fatalf("first UpsertUser error: %v", err)
	}

	updated, err := store.UpsertUser(ctx, makeGitHubUser(ghID, "updated-login"), "token-2")
	if err != nil {
		t.Fatalf("second UpsertUser error: %v", err)
	}
	if updated.Login != "updated-login" {
		t.Errorf("Login after update = %q, want %q", updated.Login, "updated-login")
	}
}

func TestGetUserByID_exists(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ghID := int64(9_000_003)
	t.Cleanup(func() { cleanUser(t, pool, ghID) })

	created, err := store.UpsertUser(ctx, makeGitHubUser(ghID, "lookup-user"), "token-x")
	if err != nil {
		t.Fatalf("UpsertUser error: %v", err)
	}

	got, err := store.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %v, want %v", got.ID, created.ID)
	}
	if got.Login != "lookup-user" {
		t.Errorf("Login = %q, want %q", got.Login, "lookup-user")
	}
}

func TestGetUserByID_notFound(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	randomID := uuid.New()
	_, err := store.GetUserByID(ctx, randomID)
	if err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Logf("error type: %T — %v (expected pgx.ErrNoRows or wrapping it)", err, err)
	}
}
