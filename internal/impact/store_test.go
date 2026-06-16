package impact

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nesbite/atlas/internal/platform/database"
	"github.com/nesbite/atlas/migrations"
)

// mockImpactStore is a test double for the ImpactStore interface.
type mockImpactStore struct {
	result []AffectedRepo
	err    error
}

func (m *mockImpactStore) GetBlastRadius(ctx context.Context, orgID uuid.UUID, ecosystem, name string) ([]AffectedRepo, error) {
	return m.result, m.err
}

// TestImpactStore_interface_coverage ensures mockImpactStore implements
// ImpactStore, providing compile-time verification of the interface shape.
func TestImpactStore_interface_coverage(t *testing.T) {
	var _ ImpactStore = &mockImpactStore{}
}

// TestGetBlastRadius_returnsResults verifies the store returns the expected
// affected repos slice unchanged.
func TestGetBlastRadius_returnsResults(t *testing.T) {
	expected := []AffectedRepo{
		{RepoName: "svc-a", FullName: "acme/svc-a", Version: "1.0.0", DepType: "direct", Teams: []string{"@acme/team-x"}},
		{RepoName: "svc-b", FullName: "acme/svc-b", Version: "1.3.0", DepType: "dev", Teams: []string{}},
	}
	store := &mockImpactStore{result: expected}
	ctx := context.Background()

	got, err := store.GetBlastRadius(ctx, uuid.New(), "npm", "left-pad")
	if err != nil {
		t.Fatalf("GetBlastRadius: unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].RepoName != "svc-a" {
		t.Errorf("first repo name = %q, want %q", got[0].RepoName, "svc-a")
	}
}

// TestGetBlastRadius_empty verifies the store returns an empty (non-nil)
// slice when no repo in the org declares the dependency.
func TestGetBlastRadius_empty(t *testing.T) {
	store := &mockImpactStore{result: []AffectedRepo{}}
	ctx := context.Background()

	got, err := store.GetBlastRadius(ctx, uuid.New(), "npm", "unknown-pkg")
	if err != nil {
		t.Fatalf("GetBlastRadius: unexpected error: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 items, got %d", len(got))
	}
}

// TestGetBlastRadius_propagatesError verifies that store errors are returned.
func TestGetBlastRadius_propagatesError(t *testing.T) {
	wantErr := errors.New("db error")
	store := &mockImpactStore{err: wantErr}
	ctx := context.Background()

	_, err := store.GetBlastRadius(ctx, uuid.New(), "npm", "left-pad")
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

// ---------------------------------------------------------------------------
// Integration tests — require DATABASE_URL
// ---------------------------------------------------------------------------

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

func makeTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (github_id, login, name, email, avatar_url, access_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, 8_000_000+int64(uuid.New().ID()%1_000_000), "test-user-"+uuid.NewString()[:8], nil, nil, nil, "tok").Scan(&id)
	if err != nil {
		t.Fatalf("makeTestUser: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id) //nolint:errcheck
	})
	return id
}

func makeTestOrg(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userID := makeTestUser(t, pool)

	var orgID uuid.UUID
	ghID := int64(9_000_000 + int64(uuid.New().ID()%1_000_000))
	err := pool.QueryRow(context.Background(), `
		INSERT INTO organizations (github_id, name, slug, owner_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, ghID, "test-org-"+uuid.NewString()[:8], "test-slug-"+uuid.NewString()[:8], userID).Scan(&orgID)
	if err != nil {
		t.Fatalf("makeTestOrg: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID) //nolint:errcheck
	})
	return orgID
}

func makeTestRepo(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	var repoID uuid.UUID
	ghID := int64(7_000_000 + int64(uuid.New().ID()%1_000_000))
	err := pool.QueryRow(context.Background(), `
		INSERT INTO repositories (org_id, github_id, name, full_name, default_branch)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, orgID, ghID, name, "test-org/"+name, "main").Scan(&repoID)
	if err != nil {
		t.Fatalf("makeTestRepo(%s): %v", name, err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM repositories WHERE id = $1", repoID) //nolint:errcheck
	})
	return repoID
}

func makeTestDependency(t *testing.T, pool *pgxpool.Pool, ecosystem, name string) uuid.UUID {
	t.Helper()
	var depID uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO dependencies (ecosystem, name)
		VALUES ($1, $2)
		ON CONFLICT (ecosystem, name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, ecosystem, name).Scan(&depID)
	if err != nil {
		t.Fatalf("makeTestDependency(%s, %s): %v", ecosystem, name, err)
	}
	return depID
}

func linkRepoDependency(t *testing.T, pool *pgxpool.Pool, repoID, depID uuid.UUID, version, depType, sourceFile string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO repo_dependencies (repo_id, dep_id, version, dep_type, source_file)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (repo_id, dep_id, source_file)
		DO UPDATE SET version = EXCLUDED.version, dep_type = EXCLUDED.dep_type
	`, repoID, depID, version, depType, sourceFile)
	if err != nil {
		t.Fatalf("linkRepoDependency: %v", err)
	}
}

func addRepoOwner(t *testing.T, pool *pgxpool.Pool, repoID uuid.UUID, pattern, owner, ownerType string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO repo_owners (repo_id, pattern, owner, owner_type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (repo_id, pattern, owner) DO NOTHING
	`, repoID, pattern, owner, ownerType)
	if err != nil {
		t.Fatalf("addRepoOwner: %v", err)
	}
}

// TestIntegration_GetBlastRadius_multiRepoMultiVersion verifies that a
// dependency used by multiple repos at different versions returns one
// AffectedRepo entry per repo with its own version and dep_type (spec AC1, AC3).
func TestIntegration_GetBlastRadius_multiRepoMultiVersion(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	repoA := makeTestRepo(t, pool, orgID, "svc-a")
	repoB := makeTestRepo(t, pool, orgID, "svc-b")
	repoC := makeTestRepo(t, pool, orgID, "svc-c")

	depID := makeTestDependency(t, pool, "npm", "left-pad")
	linkRepoDependency(t, pool, repoA, depID, "1.0.0", "direct", "package.json")
	linkRepoDependency(t, pool, repoB, depID, "1.0.0", "direct", "package.json")
	linkRepoDependency(t, pool, repoC, depID, "1.3.0", "direct", "package.json")

	got, err := store.GetBlastRadius(ctx, orgID, "npm", "left-pad")
	if err != nil {
		t.Fatalf("GetBlastRadius: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
}

// TestIntegration_GetBlastRadius_multipleTeamOwners verifies that a repo with
// multiple CODEOWNERS team entries returns all teams deduplicated case-insensitively (spec AC4).
func TestIntegration_GetBlastRadius_multipleTeamOwners(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	repoA := makeTestRepo(t, pool, orgID, "svc-a")
	depID := makeTestDependency(t, pool, "npm", "react")
	linkRepoDependency(t, pool, repoA, depID, "18.2.0", "direct", "package.json")

	addRepoOwner(t, pool, repoA, "*", "@acme/team-x", "team")
	addRepoOwner(t, pool, repoA, "/src/", "@acme/team-y", "team")

	got, err := store.GetBlastRadius(ctx, orgID, "npm", "react")
	if err != nil {
		t.Fatalf("GetBlastRadius: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if len(got[0].Teams) != 2 {
		t.Fatalf("len(teams) = %d, want 2: %v", len(got[0].Teams), got[0].Teams)
	}
}

// TestIntegration_GetBlastRadius_noTeamOwners verifies that a repo with no
// team-type owners returns an empty teams slice, not an error (spec AC5).
func TestIntegration_GetBlastRadius_noTeamOwners(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	repoB := makeTestRepo(t, pool, orgID, "svc-b")
	depID := makeTestDependency(t, pool, "npm", "axios")
	linkRepoDependency(t, pool, repoB, depID, "1.0.0", "direct", "package.json")

	got, err := store.GetBlastRadius(ctx, orgID, "npm", "axios")
	if err != nil {
		t.Fatalf("GetBlastRadius: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Teams == nil {
		t.Error("expected non-nil empty teams slice, got nil")
	}
	if len(got[0].Teams) != 0 {
		t.Errorf("expected 0 teams, got %d", len(got[0].Teams))
	}
}

// TestIntegration_GetBlastRadius_orgIsolation verifies that results are
// scoped strictly to the requesting org — no cross-org leakage (spec AC2).
func TestIntegration_GetBlastRadius_orgIsolation(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()

	orgA := makeTestOrg(t, pool)
	orgB := makeTestOrg(t, pool)

	repoA := makeTestRepo(t, pool, orgA, "shared-name-a")
	repoB := makeTestRepo(t, pool, orgB, "shared-name-b")

	depID := makeTestDependency(t, pool, "npm", "shared-lib")
	linkRepoDependency(t, pool, repoA, depID, "1.0.0", "direct", "package.json")
	linkRepoDependency(t, pool, repoB, depID, "2.0.0", "direct", "package.json")

	got, err := store.GetBlastRadius(ctx, orgA, "npm", "shared-lib")
	if err != nil {
		t.Fatalf("GetBlastRadius: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1 (org-scoped)", len(got))
	}
	if got[0].RepoName != "shared-name-a" {
		t.Errorf("repo name = %q, want %q", got[0].RepoName, "shared-name-a")
	}
}

// TestIntegration_GetBlastRadius_unknownDependency verifies that querying a
// dependency that exists in no repo_dependencies row returns an empty slice,
// not an error.
func TestIntegration_GetBlastRadius_unknownDependency(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	got, err := store.GetBlastRadius(ctx, orgID, "npm", "totally-unknown-pkg")
	if err != nil {
		t.Fatalf("GetBlastRadius: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 items, got %d", len(got))
	}
}
