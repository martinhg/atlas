package ownership

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	ownerparser "github.com/nesbite/atlas/internal/ownership/parser"
	"github.com/nesbite/atlas/internal/platform/database"
	"github.com/nesbite/atlas/migrations"
)

// ---------------------------------------------------------------------------
// Mock store for unit tests
// ---------------------------------------------------------------------------

// mockOwnershipStore is a test double satisfying OwnershipStore.
type mockOwnershipStore struct {
	syncCalls  []storeSyncCall
	syncErr    error
	listResult []RepoOwnerSummary
	listTotal  int
	listErr    error
	rulesResult []OwnerRule
	rulesErr    error
}

type storeSyncCall struct {
	repoID uuid.UUID
	owners []ownerparser.ParsedOwner
}

func (m *mockOwnershipStore) SyncRepoOwners(ctx context.Context, repoID uuid.UUID, owners []ownerparser.ParsedOwner) error {
	m.syncCalls = append(m.syncCalls, storeSyncCall{repoID: repoID, owners: owners})
	return m.syncErr
}

func (m *mockOwnershipStore) ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]RepoOwnerSummary, int, error) {
	return m.listResult, m.listTotal, m.listErr
}

func (m *mockOwnershipStore) ListByRepo(ctx context.Context, orgID uuid.UUID, repoName string) ([]OwnerRule, error) {
	return m.rulesResult, m.rulesErr
}

// TestOwnershipStore_interface_coverage ensures mockOwnershipStore implements OwnershipStore.
func TestOwnershipStore_interface_coverage(t *testing.T) {
	var _ OwnershipStore = &mockOwnershipStore{}
}

// ---------------------------------------------------------------------------
// Unit tests using mock store
// ---------------------------------------------------------------------------

func TestSyncRepoOwners_stores_owners(t *testing.T) {
	store := &mockOwnershipStore{}
	ctx := context.Background()
	repoID := uuid.New()
	owners := []ownerparser.ParsedOwner{
		{Pattern: "*.go", Owner: "@go-team", OwnerType: "user", LineNumber: 1},
		{Pattern: "src/", Owner: "@org/backend", OwnerType: "team", LineNumber: 2},
	}

	err := store.SyncRepoOwners(ctx, repoID, owners)
	if err != nil {
		t.Fatalf("SyncRepoOwners: unexpected error: %v", err)
	}
	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 sync call, got %d", len(store.syncCalls))
	}
	if store.syncCalls[0].repoID != repoID {
		t.Errorf("repoID = %v, want %v", store.syncCalls[0].repoID, repoID)
	}
	if len(store.syncCalls[0].owners) != 2 {
		t.Errorf("owners len = %d, want 2", len(store.syncCalls[0].owners))
	}
}

func TestSyncRepoOwners_empty_owners_clears(t *testing.T) {
	store := &mockOwnershipStore{}
	ctx := context.Background()
	repoID := uuid.New()

	err := store.SyncRepoOwners(ctx, repoID, []ownerparser.ParsedOwner{})
	if err != nil {
		t.Fatalf("SyncRepoOwners with empty: unexpected error: %v", err)
	}
	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 sync call for empty slice, got %d", len(store.syncCalls))
	}
	if len(store.syncCalls[0].owners) != 0 {
		t.Errorf("expected empty owners, got %d", len(store.syncCalls[0].owners))
	}
}

func TestSyncRepoOwners_propagates_error(t *testing.T) {
	wantErr := errors.New("transaction failed")
	store := &mockOwnershipStore{syncErr: wantErr}
	ctx := context.Background()

	err := store.SyncRepoOwners(ctx, uuid.New(), nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

func TestListByOrg_returns_paginated_results(t *testing.T) {
	expected := []RepoOwnerSummary{
		{RepoName: "api", OwnerCount: 5, TeamCount: 2, Teams: []string{"@org/backend", "@org/platform"}},
		{RepoName: "web", OwnerCount: 3, TeamCount: 1, Teams: []string{"@org/frontend"}},
	}
	store := &mockOwnershipStore{listResult: expected, listTotal: 10}
	ctx := context.Background()

	got, total, err := store.ListByOrg(ctx, uuid.New(), 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg: unexpected error: %v", err)
	}
	if total != 10 {
		t.Errorf("total = %d, want 10", total)
	}
	if len(got) != 2 {
		t.Errorf("len(got) = %d, want 2", len(got))
	}
	if got[0].RepoName != "api" {
		t.Errorf("first repo = %q, want %q", got[0].RepoName, "api")
	}
}

func TestListByOrg_returns_empty_non_nil_for_empty_org(t *testing.T) {
	store := &mockOwnershipStore{listResult: []RepoOwnerSummary{}, listTotal: 0}
	ctx := context.Background()

	got, total, err := store.ListByOrg(ctx, uuid.New(), 1, 50)
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

func TestListByOrg_propagates_error(t *testing.T) {
	wantErr := errors.New("db error")
	store := &mockOwnershipStore{listErr: wantErr}
	ctx := context.Background()

	_, _, err := store.ListByOrg(ctx, uuid.New(), 1, 50)
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

func TestListByRepo_returns_rules_in_order(t *testing.T) {
	ln1, ln2 := 1, 3
	expected := []OwnerRule{
		{Pattern: "*.go", Owner: "@go-owner", OwnerType: "user", LineNumber: &ln1},
		{Pattern: "docs/", Owner: "@doc-team", OwnerType: "user", LineNumber: &ln2},
	}
	store := &mockOwnershipStore{rulesResult: expected}
	ctx := context.Background()

	got, err := store.ListByRepo(ctx, uuid.New(), "my-repo")
	if err != nil {
		t.Fatalf("ListByRepo: unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Pattern != "*.go" {
		t.Errorf("first rule pattern = %q, want %q", got[0].Pattern, "*.go")
	}
	if got[1].Pattern != "docs/" {
		t.Errorf("second rule pattern = %q, want %q", got[1].Pattern, "docs/")
	}
}

func TestListByRepo_returns_empty_non_nil_for_unknown_repo(t *testing.T) {
	store := &mockOwnershipStore{rulesResult: []OwnerRule{}}
	ctx := context.Background()

	got, err := store.ListByRepo(ctx, uuid.New(), "nonexistent-repo")
	if err != nil {
		t.Fatalf("ListByRepo: unexpected error: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 rules, got %d", len(got))
	}
}

func TestListByRepo_propagates_error(t *testing.T) {
	wantErr := errors.New("db error")
	store := &mockOwnershipStore{rulesErr: wantErr}
	ctx := context.Background()

	_, err := store.ListByRepo(ctx, uuid.New(), "any-repo")
	if !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}

// ---------------------------------------------------------------------------
// Integration test helpers
// ---------------------------------------------------------------------------

func getStoreTestPool(t *testing.T) *pgxpool.Pool {
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

func makeOwnerTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (github_id, login, name, email, avatar_url, access_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, 5_000_000+int64(uuid.New().ID()%1_000_000), "owner-test-user-"+uuid.NewString()[:8], nil, nil, nil, "tok").Scan(&id)
	if err != nil {
		t.Fatalf("makeOwnerTestUser: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id) //nolint:errcheck
	})
	return id
}

func makeOwnerTestOrg(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userID := makeOwnerTestUser(t, pool)
	var orgID uuid.UUID
	ghID := int64(6_000_000 + int64(uuid.New().ID()%1_000_000))
	err := pool.QueryRow(context.Background(), `
		INSERT INTO organizations (github_id, name, slug, owner_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, ghID, "owner-test-org-"+uuid.NewString()[:8], "owner-slug-"+uuid.NewString()[:8], userID).Scan(&orgID)
	if err != nil {
		t.Fatalf("makeOwnerTestOrg: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID) //nolint:errcheck
	})
	return orgID
}

func makeOwnerTestRepo(t *testing.T, pool *pgxpool.Pool, orgID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	var repoID uuid.UUID
	ghID := int64(4_000_000 + int64(uuid.New().ID()%1_000_000))
	err := pool.QueryRow(context.Background(), `
		INSERT INTO repositories (org_id, github_id, name, full_name, default_branch)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, orgID, ghID, name, "org/"+name, "main").Scan(&repoID)
	if err != nil {
		t.Fatalf("makeOwnerTestRepo(%s): %v", name, err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM repositories WHERE id = $1", repoID) //nolint:errcheck
	})
	return repoID
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

func TestIntegration_SyncRepoOwners_insertsNewRows(t *testing.T) {
	pool := getStoreTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeOwnerTestOrg(t, pool)
	repoID := makeOwnerTestRepo(t, pool, orgID, "sync-insert-test")

	ln1, ln2 := 1, 2
	owners := []ownerparser.ParsedOwner{
		{Pattern: "*.go", Owner: "@go-team", OwnerType: "user", LineNumber: ln1},
		{Pattern: "src/", Owner: "@org/backend", OwnerType: "team", LineNumber: ln2},
	}

	if err := store.SyncRepoOwners(ctx, repoID, owners); err != nil {
		t.Fatalf("SyncRepoOwners: %v", err)
	}

	rules, err := store.ListByRepo(ctx, orgID, "sync-insert-test")
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if len(rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules))
	}
}

func TestIntegration_SyncRepoOwners_replacesExisting(t *testing.T) {
	pool := getStoreTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeOwnerTestOrg(t, pool)
	repoID := makeOwnerTestRepo(t, pool, orgID, "sync-replace-test")

	ln1 := 1
	first := []ownerparser.ParsedOwner{
		{Pattern: "*.go", Owner: "@old-owner", OwnerType: "user", LineNumber: ln1},
	}
	if err := store.SyncRepoOwners(ctx, repoID, first); err != nil {
		t.Fatalf("first SyncRepoOwners: %v", err)
	}

	ln2 := 1
	second := []ownerparser.ParsedOwner{
		{Pattern: "src/", Owner: "@new-owner", OwnerType: "user", LineNumber: ln2},
	}
	if err := store.SyncRepoOwners(ctx, repoID, second); err != nil {
		t.Fatalf("second SyncRepoOwners: %v", err)
	}

	rules, err := store.ListByRepo(ctx, orgID, "sync-replace-test")
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after re-sync, got %d", len(rules))
	}
	if rules[0].Owner != "@new-owner" {
		t.Errorf("owner = %q, want %q", rules[0].Owner, "@new-owner")
	}
}

func TestIntegration_SyncRepoOwners_emptySliceClears(t *testing.T) {
	pool := getStoreTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeOwnerTestOrg(t, pool)
	repoID := makeOwnerTestRepo(t, pool, orgID, "sync-clear-test")

	ln1 := 1
	first := []ownerparser.ParsedOwner{
		{Pattern: "*.go", Owner: "@owner", OwnerType: "user", LineNumber: ln1},
	}
	if err := store.SyncRepoOwners(ctx, repoID, first); err != nil {
		t.Fatalf("first SyncRepoOwners: %v", err)
	}

	// Sync with empty slice — should clear all rows.
	if err := store.SyncRepoOwners(ctx, repoID, []ownerparser.ParsedOwner{}); err != nil {
		t.Fatalf("empty SyncRepoOwners: %v", err)
	}

	rules, err := store.ListByRepo(ctx, orgID, "sync-clear-test")
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules after empty sync, got %d", len(rules))
	}
}

func TestIntegration_ListByOrg_pagination(t *testing.T) {
	pool := getStoreTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeOwnerTestOrg(t, pool)

	// Create 3 repos each with 1 owner.
	for i, name := range []string{"repo-a", "repo-b", "repo-c"} {
		repoID := makeOwnerTestRepo(t, pool, orgID, name)
		ln := i + 1
		owners := []ownerparser.ParsedOwner{
			{Pattern: "*.go", Owner: "@owner-" + name, OwnerType: "user", LineNumber: ln},
		}
		if err := store.SyncRepoOwners(ctx, repoID, owners); err != nil {
			t.Fatalf("SyncRepoOwners(%s): %v", name, err)
		}
	}

	// Page 1, perPage=2.
	page1, total, err := store.ListByOrg(ctx, orgID, 1, 2)
	if err != nil {
		t.Fatalf("ListByOrg page 1: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(page1) != 2 {
		t.Errorf("page1 len = %d, want 2", len(page1))
	}

	// Page 2, perPage=2 — should have 1 item.
	page2, total2, err := store.ListByOrg(ctx, orgID, 2, 2)
	if err != nil {
		t.Fatalf("ListByOrg page 2: %v", err)
	}
	if total2 != 3 {
		t.Errorf("total page2 = %d, want 3", total2)
	}
	if len(page2) != 1 {
		t.Errorf("page2 len = %d, want 1", len(page2))
	}
}

func TestIntegration_ListByOrg_emptyOrg(t *testing.T) {
	pool := getStoreTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeOwnerTestOrg(t, pool)

	got, total, err := store.ListByOrg(ctx, orgID, 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if got == nil {
		t.Error("expected non-nil slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 items, got %d", len(got))
	}
}

func TestIntegration_ListByRepo_orderedByLineNumber(t *testing.T) {
	pool := getStoreTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeOwnerTestOrg(t, pool)
	repoID := makeOwnerTestRepo(t, pool, orgID, "ordered-test")

	// Insert owners in reverse line number order.
	ln3, ln1, ln5 := 3, 1, 5
	owners := []ownerparser.ParsedOwner{
		{Pattern: "docs/", Owner: "@doc-owner", OwnerType: "user", LineNumber: ln3},
		{Pattern: "*.go", Owner: "@go-owner", OwnerType: "user", LineNumber: ln1},
		{Pattern: "src/", Owner: "@src-owner", OwnerType: "user", LineNumber: ln5},
	}
	if err := store.SyncRepoOwners(ctx, repoID, owners); err != nil {
		t.Fatalf("SyncRepoOwners: %v", err)
	}

	rules, err := store.ListByRepo(ctx, orgID, "ordered-test")
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}

	// Should be ordered by line_number ASC.
	if rules[0].Owner != "@go-owner" {
		t.Errorf("first rule owner = %q, want @go-owner (line 1)", rules[0].Owner)
	}
	if rules[1].Owner != "@doc-owner" {
		t.Errorf("second rule owner = %q, want @doc-owner (line 3)", rules[1].Owner)
	}
	if rules[2].Owner != "@src-owner" {
		t.Errorf("third rule owner = %q, want @src-owner (line 5)", rules[2].Owner)
	}
}

func TestIntegration_ListByOrg_aggregation(t *testing.T) {
	pool := getStoreTestPool(t)
	store := NewStore(pool)
	ctx := context.Background()
	orgID := makeOwnerTestOrg(t, pool)
	repoID := makeOwnerTestRepo(t, pool, orgID, "agg-test")

	ln1, ln2, ln3 := 1, 2, 3
	owners := []ownerparser.ParsedOwner{
		{Pattern: "*.go", Owner: "@user1", OwnerType: "user", LineNumber: ln1},
		{Pattern: "src/", Owner: "@org/team-a", OwnerType: "team", LineNumber: ln2},
		{Pattern: "docs/", Owner: "@org/team-b", OwnerType: "team", LineNumber: ln3},
	}
	if err := store.SyncRepoOwners(ctx, repoID, owners); err != nil {
		t.Fatalf("SyncRepoOwners: %v", err)
	}

	summaries, total, err := store.ListByOrg(ctx, orgID, 1, 50)
	if err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	s := summaries[0]
	if s.RepoName != "agg-test" {
		t.Errorf("repo_name = %q, want %q", s.RepoName, "agg-test")
	}
	if s.OwnerCount != 3 {
		t.Errorf("owner_count = %d, want 3", s.OwnerCount)
	}
	if s.TeamCount != 2 {
		t.Errorf("team_count = %d, want 2", s.TeamCount)
	}
	if s.Teams == nil {
		t.Error("teams must not be nil")
	}
	if len(s.Teams) != 2 {
		t.Errorf("len(teams) = %d, want 2", len(s.Teams))
	}
}
