package org

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nesbite/atlas/internal/platform/database"
	"github.com/nesbite/atlas/migrations"
)

// getTestPool opens a pgxpool using DATABASE_URL and runs migrations.
// The test is skipped when DATABASE_URL is not set.
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

// makeTestUser inserts a test user and returns its ID.
func makeTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (github_id, login, name, email, avatar_url, access_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, 8_000_000+int64(uuid.New().ID()), "test-user-"+uuid.NewString()[:8], nil, nil, nil, "tok").Scan(&id)
	if err != nil {
		t.Fatalf("makeTestUser: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
	})
	return id
}

// cleanOrg removes a test org row so tests are independent.
func cleanOrg(t *testing.T, pool *pgxpool.Pool, githubID int64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "DELETE FROM organizations WHERE github_id = $1", githubID)
	if err != nil {
		t.Logf("cleanup warning: %v", err)
	}
}

func TestUpsertOrg_creates_new(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ownerID := makeTestUser(t, pool)
	ghID := int64(7_000_001)
	t.Cleanup(func() { cleanOrg(t, pool, ghID) })

	org := &Organization{
		GitHubID: ghID,
		Name:     "Test Org",
		Slug:     "test-org-" + uuid.NewString()[:8],
		OwnerID:  ownerID,
	}

	got, err := store.UpsertOrg(ctx, org)
	if err != nil {
		t.Fatalf("UpsertOrg error: %v", err)
	}
	if got.ID == uuid.Nil {
		t.Error("returned ID is nil UUID")
	}
	if got.GitHubID != ghID {
		t.Errorf("GitHubID = %d, want %d", got.GitHubID, ghID)
	}
	if got.Slug != org.Slug {
		t.Errorf("Slug = %q, want %q", got.Slug, org.Slug)
	}
	if got.OwnerID != ownerID {
		t.Errorf("OwnerID = %v, want %v", got.OwnerID, ownerID)
	}
}

func TestUpsertOrg_updates_existing(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ownerID := makeTestUser(t, pool)
	ghID := int64(7_000_002)
	slug := "test-org-update-" + uuid.NewString()[:8]
	t.Cleanup(func() { cleanOrg(t, pool, ghID) })

	_, err := store.UpsertOrg(ctx, &Organization{
		GitHubID: ghID,
		Name:     "Original Name",
		Slug:     slug,
		OwnerID:  ownerID,
	})
	if err != nil {
		t.Fatalf("first UpsertOrg error: %v", err)
	}

	updated, err := store.UpsertOrg(ctx, &Organization{
		GitHubID: ghID,
		Name:     "Updated Name",
		Slug:     slug,
		OwnerID:  ownerID,
	})
	if err != nil {
		t.Fatalf("second UpsertOrg error: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Name after update = %q, want %q", updated.Name, "Updated Name")
	}
}

func TestGetOrgBySlug_found(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ownerID := makeTestUser(t, pool)
	ghID := int64(7_000_003)
	slug := "test-slug-found-" + uuid.NewString()[:8]
	t.Cleanup(func() { cleanOrg(t, pool, ghID) })

	_, err := store.UpsertOrg(ctx, &Organization{
		GitHubID: ghID,
		Name:     "Slug Test Org",
		Slug:     slug,
		OwnerID:  ownerID,
	})
	if err != nil {
		t.Fatalf("UpsertOrg error: %v", err)
	}

	got, err := store.GetOrgBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("GetOrgBySlug error: %v", err)
	}
	if got == nil {
		t.Fatal("expected org, got nil")
	}
	if got.Slug != slug {
		t.Errorf("Slug = %q, want %q", got.Slug, slug)
	}
}

func TestGetOrgBySlug_not_found_returns_nil(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	got, err := store.GetOrgBySlug(ctx, "nonexistent-slug-xyz-"+uuid.NewString()[:8])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestGetOrgsByOwnerID_returns_empty(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ownerID := makeTestUser(t, pool)

	orgs, err := store.GetOrgsByOwnerID(ctx, ownerID)
	if err != nil {
		t.Fatalf("GetOrgsByOwnerID error: %v", err)
	}
	if orgs == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(orgs) != 0 {
		t.Errorf("expected 0 orgs, got %d", len(orgs))
	}
}

func TestGetOrgsByOwnerID_returns_orgs(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ownerID := makeTestUser(t, pool)
	ghIDs := []int64{7_000_004, 7_000_005}
	for _, id := range ghIDs {
		id := id
		t.Cleanup(func() { cleanOrg(t, pool, id) })
		_, err := store.UpsertOrg(ctx, &Organization{
			GitHubID: id,
			Name:     "Org",
			Slug:     "owner-test-" + uuid.NewString()[:8],
			OwnerID:  ownerID,
		})
		if err != nil {
			t.Fatalf("UpsertOrg error: %v", err)
		}
	}

	orgs, err := store.GetOrgsByOwnerID(ctx, ownerID)
	if err != nil {
		t.Fatalf("GetOrgsByOwnerID error: %v", err)
	}
	if len(orgs) != 2 {
		t.Errorf("expected 2 orgs, got %d", len(orgs))
	}
}

func TestGetOrgByInstallationID_found(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ownerID := makeTestUser(t, pool)
	ghID := int64(7_000_006)
	installationID := int64(98_000_001)
	slug := "test-install-" + uuid.NewString()[:8]
	t.Cleanup(func() { cleanOrg(t, pool, ghID) })

	_, err := store.UpsertOrg(ctx, &Organization{
		GitHubID:             ghID,
		Name:                 "Install Org",
		Slug:                 slug,
		GitHubInstallationID: &installationID,
		OwnerID:              ownerID,
	})
	if err != nil {
		t.Fatalf("UpsertOrg error: %v", err)
	}

	got, err := store.GetOrgByInstallationID(ctx, installationID)
	if err != nil {
		t.Fatalf("GetOrgByInstallationID error: %v", err)
	}
	if got == nil {
		t.Fatal("expected org, got nil")
	}
	if *got.GitHubInstallationID != installationID {
		t.Errorf("InstallationID = %d, want %d", *got.GitHubInstallationID, installationID)
	}
}

func TestSetInstallationID(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ownerID := makeTestUser(t, pool)
	ghID := int64(7_000_007)
	slug := "test-set-install-" + uuid.NewString()[:8]
	t.Cleanup(func() { cleanOrg(t, pool, ghID) })

	created, err := store.UpsertOrg(ctx, &Organization{
		GitHubID: ghID,
		Name:     "Set Install Org",
		Slug:     slug,
		OwnerID:  ownerID,
	})
	if err != nil {
		t.Fatalf("UpsertOrg error: %v", err)
	}

	installationID := int64(98_000_002)
	if err := store.SetInstallationID(ctx, created.ID, installationID); err != nil {
		t.Fatalf("SetInstallationID error: %v", err)
	}

	got, err := store.GetOrgBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("GetOrgBySlug error: %v", err)
	}
	if got.GitHubInstallationID == nil || *got.GitHubInstallationID != installationID {
		t.Errorf("InstallationID not set correctly, got %v", got.GitHubInstallationID)
	}
}

func TestSetLastSyncedAt(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	ownerID := makeTestUser(t, pool)
	ghID := int64(7_000_008)
	slug := "test-last-synced-" + uuid.NewString()[:8]
	t.Cleanup(func() { cleanOrg(t, pool, ghID) })

	created, err := store.UpsertOrg(ctx, &Organization{
		GitHubID: ghID,
		Name:     "Sync Test Org",
		Slug:     slug,
		OwnerID:  ownerID,
	})
	if err != nil {
		t.Fatalf("UpsertOrg error: %v", err)
	}
	if created.LastSyncedAt != nil {
		t.Error("expected LastSyncedAt to be nil initially")
	}

	syncTime := time.Now().UTC().Truncate(time.Microsecond)
	if err := store.SetLastSyncedAt(ctx, created.ID, syncTime); err != nil {
		t.Fatalf("SetLastSyncedAt error: %v", err)
	}

	got, err := store.GetOrgBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("GetOrgBySlug error: %v", err)
	}
	if got.LastSyncedAt == nil {
		t.Fatal("expected LastSyncedAt to be set")
	}
	diff := got.LastSyncedAt.Sub(syncTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Millisecond {
		t.Errorf("LastSyncedAt = %v, want ~%v (diff %v)", got.LastSyncedAt, syncTime, diff)
	}
}
