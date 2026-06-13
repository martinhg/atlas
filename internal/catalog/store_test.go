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
		pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID)
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
		pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
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

	repos, err := store.GetRepositoriesByOrgID(ctx, orgID)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID error: %v", err)
	}
	if len(repos) != 3 {
		t.Errorf("expected 3 repos, got %d", len(repos))
	}
}

func TestGetRepositoriesByOrgID_returns_empty(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)

	repos, err := store.GetRepositoriesByOrgID(ctx, orgID)
	if err != nil {
		t.Fatalf("GetRepositoriesByOrgID error: %v", err)
	}
	if repos == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(repos))
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
