package catalog

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nesbite/atlas/internal/platform/database"
	"github.com/nesbite/atlas/migrations"
)

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

func TestUpsertRepository_creates_new(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	repo := &Repository{
		OrgID:         orgID,
		GitHubID:      50001,
		Name:          "my-repo",
		FullName:      "org/my-repo",
		DefaultBranch: "main",
	}

	got, err := store.UpsertRepository(ctx, repo)
	if err != nil {
		t.Fatalf("UpsertRepository error: %v", err)
	}
	if got.ID == uuid.Nil {
		t.Error("returned ID is nil UUID")
	}
	if got.Name != "my-repo" {
		t.Errorf("Name = %q, want %q", got.Name, "my-repo")
	}
	if got.FullName != "org/my-repo" {
		t.Errorf("FullName = %q, want %q", got.FullName, "org/my-repo")
	}
}

func TestUpsertRepository_updates_existing(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	_, err := store.UpsertRepository(ctx, &Repository{
		OrgID:         orgID,
		GitHubID:      50002,
		Name:          "old-name",
		FullName:      "org/old-name",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("first UpsertRepository error: %v", err)
	}

	updated, err := store.UpsertRepository(ctx, &Repository{
		OrgID:         orgID,
		GitHubID:      50002,
		Name:          "new-name",
		FullName:      "org/new-name",
		DefaultBranch: "develop",
		Stars:         42,
	})
	if err != nil {
		t.Fatalf("second UpsertRepository error: %v", err)
	}
	if updated.Name != "new-name" {
		t.Errorf("Name = %q, want %q", updated.Name, "new-name")
	}
	if updated.DefaultBranch != "develop" {
		t.Errorf("DefaultBranch = %q, want %q", updated.DefaultBranch, "develop")
	}
	if updated.Stars != 42 {
		t.Errorf("Stars = %d, want 42", updated.Stars)
	}
}

// TestGetRepositoriesByOrgID_returns_repos uses the updated signature:
// q="", page=1, perPage=100 — matches prior behaviour.
func TestGetRepositoriesByOrgID_returns_repos(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	for i, name := range []string{"repo-a", "repo-b", "repo-c"} {
		_, err := store.UpsertRepository(ctx, &Repository{
			OrgID:         orgID,
			GitHubID:      int64(60000 + i),
			Name:          name,
			FullName:      "org/" + name,
			DefaultBranch: "main",
		})
		if err != nil {
			t.Fatalf("UpsertRepository(%s) error: %v", name, err)
		}
	}

	repos, total, err := store.GetRepositoriesByOrgID(ctx, orgID, "", 1, 100)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID error: %v", err)
	}
	if len(repos) != 3 {
		t.Errorf("expected 3 repos, got %d", len(repos))
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
}

// TestMigration_000006_indexes_exist verifies that migration 000006 creates
// the expected text_pattern_ops indexes on the repositories table.
func TestMigration_000006_indexes_exist(t *testing.T) {
	pool := getTestPool(t)
	ctx := context.Background()

	wantIndexes := []string{
		"idx_repositories_name_pattern",
		"idx_repositories_full_name_pattern",
	}

	for _, idx := range wantIndexes {
		var count int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM pg_indexes
			WHERE tablename = 'repositories'
			  AND indexname = $1
		`, idx).Scan(&count)
		if err != nil {
			t.Fatalf("query pg_indexes for %q: %v", idx, err)
		}
		if count == 0 {
			t.Errorf("index %q not found in pg_indexes", idx)
		}
	}
}

// TestGetRepositoriesByOrgID_returns_empty uses the updated signature.
func TestGetRepositoriesByOrgID_returns_empty(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	repos, total, err := store.GetRepositoriesByOrgID(ctx, orgID, "", 1, 100)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID error: %v", err)
	}
	if repos == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(repos))
	}
	if total != 0 {
		t.Errorf("expected total=0, got %d", total)
	}
}

func TestUpsertRepository_with_optional_fields(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	desc := "A cool project"
	lang := "Go"
	repo := &Repository{
		OrgID:         orgID,
		GitHubID:      50003,
		Name:          "described-repo",
		FullName:      "org/described-repo",
		DefaultBranch: "main",
		Description:   &desc,
		Language:      &lang,
		Private:       true,
		Fork:          true,
		Stars:         100,
	}

	got, err := store.UpsertRepository(ctx, repo)
	if err != nil {
		t.Fatalf("UpsertRepository error: %v", err)
	}
	if got.Description == nil || *got.Description != desc {
		t.Errorf("Description = %v, want %q", got.Description, desc)
	}
	if got.Language == nil || *got.Language != lang {
		t.Errorf("Language = %v, want %q", got.Language, lang)
	}
	if !got.Private {
		t.Error("expected Private=true")
	}
	if !got.Fork {
		t.Error("expected Fork=true")
	}
	if got.Stars != 100 {
		t.Errorf("Stars = %d, want 100", got.Stars)
	}
}

// ---------------------------------------------------------------------------
// PR1.3: New tests for GetRepositoriesByOrgID with q/page/perPage params.
// These tests fail to compile until PR1.4 updates the store signature.
// ---------------------------------------------------------------------------

// TestGetRepositoriesByOrgID_filter_by_name verifies that q filters by name.
func TestGetRepositoriesByOrgID_filter_by_name(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	names := []struct {
		name     string
		fullName string
		ghID     int64
	}{
		{"react-dashboard", "org/react-dashboard", 70001},
		{"vue-app", "org/vue-app", 70002},
		{"react-native-lib", "org/react-native-lib", 70003},
	}
	for _, n := range names {
		_, err := store.UpsertRepository(ctx, &Repository{
			OrgID: orgID, GitHubID: n.ghID, Name: n.name,
			FullName: n.fullName, DefaultBranch: "main",
		})
		if err != nil {
			t.Fatalf("UpsertRepository(%s): %v", n.name, err)
		}
	}

	repos, total, err := store.GetRepositoriesByOrgID(ctx, orgID, "react", 1, 100)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(repos) != 2 {
		t.Errorf("len(repos) = %d, want 2", len(repos))
	}
	for _, r := range repos {
		if r.Name != "react-dashboard" && r.Name != "react-native-lib" {
			t.Errorf("unexpected repo %q in results", r.Name)
		}
	}
}

// TestGetRepositoriesByOrgID_filter_case_insensitive verifies ILIKE is case-insensitive.
func TestGetRepositoriesByOrgID_filter_case_insensitive(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	_, err := store.UpsertRepository(ctx, &Repository{
		OrgID: orgID, GitHubID: 71001, Name: "ReactApp",
		FullName: "org/ReactApp", DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("UpsertRepository: %v", err)
	}

	repos, total, err := store.GetRepositoriesByOrgID(ctx, orgID, "REACT", 1, 100)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(repos) != 1 {
		t.Errorf("len(repos) = %d, want 1", len(repos))
	}
}

// TestGetRepositoriesByOrgID_filter_by_description verifies COALESCE(description,'') ILIKE.
func TestGetRepositoriesByOrgID_filter_by_description(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	desc := "A toolkit for building UIs"
	_, err := store.UpsertRepository(ctx, &Repository{
		OrgID: orgID, GitHubID: 72001, Name: "ui-kit",
		FullName: "org/ui-kit", DefaultBranch: "main", Description: &desc,
	})
	if err != nil {
		t.Fatalf("UpsertRepository: %v", err)
	}

	repos, total, err := store.GetRepositoriesByOrgID(ctx, orgID, "toolkit", 1, 100)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(repos) != 1 || repos[0].Name != "ui-kit" {
		t.Errorf("expected ui-kit, got %v", repos)
	}
}

// TestGetRepositoriesByOrgID_no_match returns empty slice with total=0.
func TestGetRepositoriesByOrgID_no_match(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	_, err := store.UpsertRepository(ctx, &Repository{
		OrgID: orgID, GitHubID: 73001, Name: "some-repo",
		FullName: "org/some-repo", DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("UpsertRepository: %v", err)
	}

	repos, total, err := store.GetRepositoriesByOrgID(ctx, orgID, "xyz-no-match", 1, 100)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(repos) != 0 {
		t.Errorf("len(repos) = %d, want 0", len(repos))
	}
}

// TestGetRepositoriesByOrgID_null_description_no_crash verifies COALESCE handles NULL description.
func TestGetRepositoriesByOrgID_null_description_no_crash(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	// Insert repo with NULL description.
	_, err := store.UpsertRepository(ctx, &Repository{
		OrgID: orgID, GitHubID: 74001, Name: "null-desc-repo",
		FullName: "org/null-desc-repo", DefaultBranch: "main",
		// Description intentionally nil
	})
	if err != nil {
		t.Fatalf("UpsertRepository: %v", err)
	}

	// Should not panic or error.
	repos, _, err := store.GetRepositoriesByOrgID(ctx, orgID, "anything", 1, 100)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID with null description: %v", err)
	}
	// The repo name doesn't match "anything" and description is null, so should be 0.
	_ = repos
}

// TestGetRepositoriesByOrgID_pagination_page1 verifies perPage limits results.
func TestGetRepositoriesByOrgID_pagination_page1(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	for i := 0; i < 5; i++ {
		_, err := store.UpsertRepository(ctx, &Repository{
			OrgID: orgID, GitHubID: int64(75000 + i),
			Name: "pg-repo-" + string(rune('a'+i)), FullName: "org/pg-repo-" + string(rune('a'+i)),
			DefaultBranch: "main",
		})
		if err != nil {
			t.Fatalf("UpsertRepository: %v", err)
		}
	}

	repos, total, err := store.GetRepositoriesByOrgID(ctx, orgID, "", 1, 2)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID page 1: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(repos) != 2 {
		t.Errorf("len(repos) = %d, want 2", len(repos))
	}
}

// TestGetRepositoriesByOrgID_pagination_page2 verifies second page results.
func TestGetRepositoriesByOrgID_pagination_page2(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	for i := 0; i < 5; i++ {
		_, err := store.UpsertRepository(ctx, &Repository{
			OrgID: orgID, GitHubID: int64(76000 + i),
			Name: "p2-repo-" + string(rune('a'+i)), FullName: "org/p2-repo-" + string(rune('a'+i)),
			DefaultBranch: "main",
		})
		if err != nil {
			t.Fatalf("UpsertRepository: %v", err)
		}
	}

	repos, total, err := store.GetRepositoriesByOrgID(ctx, orgID, "", 2, 2)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID page 2: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(repos) != 2 {
		t.Errorf("len(repos) = %d, want 2", len(repos))
	}
}
