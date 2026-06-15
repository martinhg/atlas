package dependency

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nesbite/atlas/internal/dependency/parser"
	"github.com/nesbite/atlas/internal/platform/database"
	"github.com/nesbite/atlas/migrations"
)

// mockDepStore is a test double for the DepStore interface.
// Updated to match new ListByOrg signature with q string param.
type mockDepStore struct {
	syncCalls    []syncCall
	syncErr      error
	listResult   []DependencyWithCount
	listTotal    int
	listErr      error
	detailResult []DepDetail
	detailErr    error
}

type syncCall struct {
	repoID uuid.UUID
	deps   []parser.ParsedDep
}

func (m *mockDepStore) SyncRepoDependencies(ctx context.Context, repoID uuid.UUID, deps []parser.ParsedDep) error {
	m.syncCalls = append(m.syncCalls, syncCall{repoID: repoID, deps: deps})
	return m.syncErr
}

// ListByOrg now accepts q string param (PR1.7 RED — fails until PR1.8 updates the interface).
func (m *mockDepStore) ListByOrg(ctx context.Context, orgID uuid.UUID, q string, page, perPage int) ([]DependencyWithCount, int, error) {
	return m.listResult, m.listTotal, m.listErr
}

func (m *mockDepStore) GetDetail(ctx context.Context, orgID uuid.UUID, ecosystem, name string) ([]DepDetail, error) {
	return m.detailResult, m.detailErr
}

// TestDepStore_interface_coverage ensures the mockDepStore implements DepStore,
// providing compile-time verification that the interface shape is correct.
func TestDepStore_interface_coverage(t *testing.T) {
	var _ DepStore = &mockDepStore{}
}

// TestSyncRepoDependencies_stores_deps verifies that the store receives the
// correct repoID and deps slice.
func TestSyncRepoDependencies_stores_deps(t *testing.T) {
	store := &mockDepStore{}
	ctx := context.Background()
	repoID := uuid.New()
	deps := []parser.ParsedDep{
		{Ecosystem: "npm", Name: "react", Version: "^18.0.0", DepType: "dep", SourceFile: "package.json"},
		{Ecosystem: "npm", Name: "typescript", Version: "^5.0.0", DepType: "devDep", SourceFile: "package.json"},
	}

	err := store.SyncRepoDependencies(ctx, repoID, deps)
	if err != nil {
		t.Fatalf("SyncRepoDependencies: unexpected error: %v", err)
	}
	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 sync call, got %d", len(store.syncCalls))
	}
	if store.syncCalls[0].repoID != repoID {
		t.Errorf("repoID = %v, want %v", store.syncCalls[0].repoID, repoID)
	}
	if len(store.syncCalls[0].deps) != 2 {
		t.Errorf("deps len = %d, want 2", len(store.syncCalls[0].deps))
	}
}

// TestSyncRepoDependencies_propagates_error verifies that store errors are returned.
func TestSyncRepoDependencies_propagates_error(t *testing.T) {
	wantErr := errors.New("transaction failed")
	store := &mockDepStore{syncErr: wantErr}
	ctx := context.Background()

	err := store.SyncRepoDependencies(ctx, uuid.New(), nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

// TestListByOrg_returns_paginated_results verifies the list method returns
// the expected slice and total count (updated to new q param signature).
func TestListByOrg_returns_paginated_results(t *testing.T) {
	expected := []DependencyWithCount{
		{Ecosystem: "npm", Name: "react", RepoCount: 3},
		{Ecosystem: "npm", Name: "typescript", RepoCount: 5},
	}
	store := &mockDepStore{listResult: expected, listTotal: 42}
	ctx := context.Background()

	got, total, err := store.ListByOrg(ctx, uuid.New(), "", 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg: unexpected error: %v", err)
	}
	if total != 42 {
		t.Errorf("total = %d, want 42", total)
	}
	if len(got) != 2 {
		t.Errorf("len(got) = %d, want 2", len(got))
	}
	if got[0].Name != "react" {
		t.Errorf("first dep name = %q, want %q", got[0].Name, "react")
	}
}

// TestListByOrg_returns_empty_for_org_with_no_deps verifies the list method
// returns an empty (non-nil) slice and zero total when the org has no deps.
func TestListByOrg_returns_empty_for_org_with_no_deps(t *testing.T) {
	store := &mockDepStore{listResult: []DependencyWithCount{}, listTotal: 0}
	ctx := context.Background()

	got, total, err := store.ListByOrg(ctx, uuid.New(), "", 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg: unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if got == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 items, got %d", len(got))
	}
}

// TestListByOrg_propagates_error verifies that store errors are returned.
func TestListByOrg_propagates_error(t *testing.T) {
	wantErr := errors.New("db error")
	store := &mockDepStore{listErr: wantErr}
	ctx := context.Background()

	_, _, err := store.ListByOrg(ctx, uuid.New(), "", 1, 50)
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

// TestGetDetail_returns_results verifies the detail method returns the expected slice.
func TestGetDetail_returns_results(t *testing.T) {
	expected := []DepDetail{
		{RepoName: "web-app", Version: "^18.2.0", DepType: "dep", SourceFile: "package.json"},
		{RepoName: "api", Version: "^17.0.0", DepType: "dep", SourceFile: "package.json"},
	}
	store := &mockDepStore{detailResult: expected}
	ctx := context.Background()

	got, err := store.GetDetail(ctx, uuid.New(), "npm", "react")
	if err != nil {
		t.Fatalf("GetDetail: unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len(got) = %d, want 2", len(got))
	}
	if got[0].RepoName != "web-app" {
		t.Errorf("first repo name = %q, want %q", got[0].RepoName, "web-app")
	}
}

// TestGetDetail_returns_empty_on_not_found verifies the detail method returns
// an empty (non-nil) slice when no repos use the dependency.
func TestGetDetail_returns_empty_on_not_found(t *testing.T) {
	store := &mockDepStore{detailResult: []DepDetail{}}
	ctx := context.Background()

	got, err := store.GetDetail(ctx, uuid.New(), "npm", "unknown-lib")
	if err != nil {
		t.Fatalf("GetDetail: unexpected error: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 items, got %d", len(got))
	}
}

// TestGetDetail_propagates_error verifies that store errors are returned.
func TestGetDetail_propagates_error(t *testing.T) {
	wantErr := errors.New("db error")
	store := &mockDepStore{detailErr: wantErr}
	ctx := context.Background()

	_, err := store.GetDetail(ctx, uuid.New(), "npm", "react")
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

// ---------------------------------------------------------------------------
// PR1.7: New tests for ListByOrg with q param.
// These fail to compile until PR1.8 updates the DepStore interface.
// ---------------------------------------------------------------------------

// TestListByOrg_q_empty_returns_all verifies that q="" returns all deps.
func TestListByOrg_q_empty_returns_all(t *testing.T) {
	expected := []DependencyWithCount{
		{Ecosystem: "npm", Name: "lodash", RepoCount: 2},
		{Ecosystem: "npm", Name: "axios", RepoCount: 1},
	}
	store := &mockDepStore{listResult: expected, listTotal: 2}
	ctx := context.Background()

	got, total, err := store.ListByOrg(ctx, uuid.New(), "", 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg q='': %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(got) != 2 {
		t.Errorf("len(got) = %d, want 2", len(got))
	}
}

// TestListByOrg_q_filters_by_name verifies that q="lodash" would filter by name.
// The mock always returns its preset data; this test verifies the q param is accepted.
func TestListByOrg_q_filters_by_name(t *testing.T) {
	expected := []DependencyWithCount{
		{Ecosystem: "npm", Name: "lodash", RepoCount: 2},
		{Ecosystem: "npm", Name: "lodash-fp", RepoCount: 1},
	}
	store := &mockDepStore{listResult: expected, listTotal: 2}
	ctx := context.Background()

	got, total, err := store.ListByOrg(ctx, uuid.New(), "lodash", 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg q='lodash': %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(got) != 2 {
		t.Errorf("len(got) = %d, want 2", len(got))
	}
}

// TestListByOrg_q_no_match_returns_empty verifies that a non-matching q returns empty.
func TestListByOrg_q_no_match_returns_empty(t *testing.T) {
	store := &mockDepStore{listResult: []DependencyWithCount{}, listTotal: 0}
	ctx := context.Background()

	got, total, err := store.ListByOrg(ctx, uuid.New(), "no-match-xyz", 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg q='no-match-xyz': %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
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
	`, orgID, ghID, name, "org/"+name, "main").Scan(&repoID)
	if err != nil {
		t.Fatalf("makeTestRepo(%s): %v", name, err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM repositories WHERE id = $1", repoID) //nolint:errcheck
	})
	return repoID
}

func TestIntegration_SyncRepoDependencies(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)
	repoID := makeTestRepo(t, pool, orgID, "sync-test")

	deps := []parser.ParsedDep{
		{Ecosystem: "npm", Name: "react", Version: "^18.0.0", DepType: "dep", SourceFile: "package.json"},
		{Ecosystem: "npm", Name: "typescript", Version: "^5.0.0", DepType: "devDep", SourceFile: "package.json"},
	}

	// First sync — creates deps.
	if err := store.SyncRepoDependencies(ctx, repoID, deps); err != nil {
		t.Fatalf("first SyncRepoDependencies: %v", err)
	}

	// Verify deps exist via ListByOrg (q="").
	list, total, err := store.ListByOrg(ctx, orgID, "", 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg after first sync: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}

	// Re-sync with a different set — should replace.
	newDeps := []parser.ParsedDep{
		{Ecosystem: "npm", Name: "vue", Version: "^3.0.0", DepType: "dep", SourceFile: "package.json"},
	}
	if err := store.SyncRepoDependencies(ctx, repoID, newDeps); err != nil {
		t.Fatalf("second SyncRepoDependencies: %v", err)
	}

	// Only the new dep should be linked to this repo.
	detail, err := store.GetDetail(ctx, orgID, "npm", "vue")
	if err != nil {
		t.Fatalf("GetDetail(vue): %v", err)
	}
	if len(detail) != 1 {
		t.Errorf("expected 1 detail for vue, got %d", len(detail))
	}

	// Old deps should no longer be linked to this repo.
	oldDetail, err := store.GetDetail(ctx, orgID, "npm", "react")
	if err != nil {
		t.Fatalf("GetDetail(react): %v", err)
	}
	if len(oldDetail) != 0 {
		t.Errorf("expected 0 details for react after re-sync, got %d", len(oldDetail))
	}
}

func TestIntegration_ListByOrg_pagination(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)
	repoID := makeTestRepo(t, pool, orgID, "list-test")

	// Create 5 deps.
	deps := make([]parser.ParsedDep, 5)
	for i := range deps {
		deps[i] = parser.ParsedDep{
			Ecosystem:  "npm",
			Name:       "dep-" + string(rune('a'+i)),
			Version:    "^1.0.0",
			DepType:    "dep",
			SourceFile: "package.json",
		}
	}
	if err := store.SyncRepoDependencies(ctx, repoID, deps); err != nil {
		t.Fatalf("SyncRepoDependencies: %v", err)
	}

	// Page 1 with perPage=2 (q="").
	page1, total, err := store.ListByOrg(ctx, orgID, "", 1, 2)
	if err != nil {
		t.Fatalf("ListByOrg page 1: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(page1) != 2 {
		t.Errorf("page 1 len = %d, want 2", len(page1))
	}

	// Page 3 with perPage=2 should have 1 item.
	page3, total3, err := store.ListByOrg(ctx, orgID, "", 3, 2)
	if err != nil {
		t.Fatalf("ListByOrg page 3: %v", err)
	}
	if total3 != 5 {
		t.Errorf("total = %d, want 5", total3)
	}
	if len(page3) != 1 {
		t.Errorf("page 3 len = %d, want 1", len(page3))
	}
}

func TestIntegration_GetDetail(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)
	repo1 := makeTestRepo(t, pool, orgID, "detail-a")
	repo2 := makeTestRepo(t, pool, orgID, "detail-b")

	deps := []parser.ParsedDep{
		{Ecosystem: "npm", Name: "shared-lib", Version: "^1.0.0", DepType: "dep", SourceFile: "package.json"},
	}
	if err := store.SyncRepoDependencies(ctx, repo1, deps); err != nil {
		t.Fatalf("SyncRepoDependencies repo1: %v", err)
	}
	deps[0].Version = "^2.0.0"
	if err := store.SyncRepoDependencies(ctx, repo2, deps); err != nil {
		t.Fatalf("SyncRepoDependencies repo2: %v", err)
	}

	detail, err := store.GetDetail(ctx, orgID, "npm", "shared-lib")
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if len(detail) != 2 {
		t.Fatalf("expected 2 detail rows, got %d", len(detail))
	}

	// Verify repo names are present (order is by r.name).
	if detail[0].RepoName != "detail-a" {
		t.Errorf("first repo = %q, want %q", detail[0].RepoName, "detail-a")
	}
	if detail[1].RepoName != "detail-b" {
		t.Errorf("second repo = %q, want %q", detail[1].RepoName, "detail-b")
	}

	// Unknown dep returns empty slice.
	empty, err := store.GetDetail(ctx, orgID, "npm", "nonexistent")
	if err != nil {
		t.Fatalf("GetDetail(nonexistent): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 detail rows for unknown dep, got %d", len(empty))
	}
}

// ---------------------------------------------------------------------------
// PR1.7 Integration tests for ListByOrg with q param.
// Require DATABASE_URL. FAIL until PR1.8 updates the real store.
// ---------------------------------------------------------------------------

// TestIntegration_ListByOrg_q_filters_by_name verifies ILIKE filtering.
func TestIntegration_ListByOrg_q_filters_by_name(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)
	repoID := makeTestRepo(t, pool, orgID, "filter-test")

	deps := []parser.ParsedDep{
		{Ecosystem: "npm", Name: "lodash", Version: "^4.0.0", DepType: "dep", SourceFile: "package.json"},
		{Ecosystem: "npm", Name: "lodash-fp", Version: "^4.0.0", DepType: "dep", SourceFile: "package.json"},
		{Ecosystem: "npm", Name: "axios", Version: "^1.0.0", DepType: "dep", SourceFile: "package.json"},
	}
	if err := store.SyncRepoDependencies(ctx, repoID, deps); err != nil {
		t.Fatalf("SyncRepoDependencies: %v", err)
	}

	got, total, err := store.ListByOrg(ctx, orgID, "lodash", 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg q='lodash': %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(got) != 2 {
		t.Errorf("len(got) = %d, want 2", len(got))
	}
	for _, d := range got {
		if d.Name != "lodash" && d.Name != "lodash-fp" {
			t.Errorf("unexpected dep %q in results", d.Name)
		}
	}
}

// TestIntegration_ListByOrg_q_case_insensitive verifies ILIKE is case-insensitive.
func TestIntegration_ListByOrg_q_case_insensitive(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)
	repoID := makeTestRepo(t, pool, orgID, "ci-test")

	deps := []parser.ParsedDep{
		{Ecosystem: "npm", Name: "Lodash", Version: "^4.0.0", DepType: "dep", SourceFile: "package.json"},
	}
	if err := store.SyncRepoDependencies(ctx, repoID, deps); err != nil {
		t.Fatalf("SyncRepoDependencies: %v", err)
	}

	got, total, err := store.ListByOrg(ctx, orgID, "LODASH", 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg q='LODASH': %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(got) != 1 {
		t.Errorf("len(got) = %d, want 1", len(got))
	}
}

// TestIntegration_ListByOrg_q_no_match_returns_empty verifies no match.
func TestIntegration_ListByOrg_q_no_match_returns_empty(t *testing.T) {
	pool := getTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeTestOrg(t, pool)
	repoID := makeTestRepo(t, pool, orgID, "nomatch-test")

	deps := []parser.ParsedDep{
		{Ecosystem: "npm", Name: "axios", Version: "^1.0.0", DepType: "dep", SourceFile: "package.json"},
	}
	if err := store.SyncRepoDependencies(ctx, repoID, deps); err != nil {
		t.Fatalf("SyncRepoDependencies: %v", err)
	}

	got, total, err := store.ListByOrg(ctx, orgID, "no-match-xyz", 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg q='no-match-xyz': %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}
