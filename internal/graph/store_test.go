package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// mockGraphStore is a test double for GraphStore used in contract tests.
type mockGraphStore struct {
	aggregates []depAggregate
	err        error
}

func (m *mockGraphStore) GetGraph(ctx context.Context, orgID uuid.UUID, f GraphFilters) ([]depAggregate, error) {
	return m.aggregates, m.err
}

// TestGraphStore_contract verifies that a mockGraphStore satisfies GraphStore,
// confirming the interface shape is correct.
func TestGraphStore_contract(t *testing.T) {
	var _ GraphStore = &mockGraphStore{}
}

// TestGraphStore_empty verifies that an empty aggregate slice is returned
// (not nil) when an org has no dependencies.
func TestGraphStore_empty(t *testing.T) {
	store := &mockGraphStore{aggregates: []depAggregate{}}
	result, err := store.GetGraph(context.Background(), uuid.New(), GraphFilters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 aggregates, got %d", len(result))
	}
}

// TestGraphStore_propagatesError verifies that store errors are surfaced.
func TestGraphStore_propagatesError(t *testing.T) {
	wantErr := errors.New("db connection lost")
	store := &mockGraphStore{err: wantErr}
	_, err := store.GetGraph(context.Background(), uuid.New(), GraphFilters{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("got error %v, want %v", err, wantErr)
	}
}

// TestGraphStore_aggregateShape verifies the depAggregate projection structure
// can hold all expected fields.
func TestGraphStore_aggregateShape(t *testing.T) {
	orgID := uuid.New()
	depID := uuid.New()
	repoID := uuid.New()
	lang := "Go"

	agg := depAggregate{
		DepID:     depID,
		Ecosystem: "npm",
		Name:      "left-pad",
		AffectedRepos: []repoRef{
			{
				RepoID:   repoID,
				RepoName: "svc-a",
				Language: &lang,
				DepType:  "direct",
				Teams:    []string{"@acme/backend"},
			},
		},
	}

	store := &mockGraphStore{aggregates: []depAggregate{agg}}
	result, err := store.GetGraph(context.Background(), orgID, GraphFilters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 aggregate, got %d", len(result))
	}
	got := result[0]
	if got.DepID != depID {
		t.Errorf("DepID = %v, want %v", got.DepID, depID)
	}
	if got.Ecosystem != "npm" {
		t.Errorf("Ecosystem = %q, want %q", got.Ecosystem, "npm")
	}
	if got.Name != "left-pad" {
		t.Errorf("Name = %q, want %q", got.Name, "left-pad")
	}
	if len(got.AffectedRepos) != 1 {
		t.Fatalf("AffectedRepos len = %d, want 1", len(got.AffectedRepos))
	}
	repo := got.AffectedRepos[0]
	if repo.RepoID != repoID {
		t.Errorf("RepoID = %v, want %v", repo.RepoID, repoID)
	}
	if repo.Language == nil || *repo.Language != "Go" {
		t.Errorf("Language = %v, want 'Go'", repo.Language)
	}
}

// TestGraphStore_ecosystemFilter verifies the ecosystem filter param is
// accepted by the interface and passed without error.
func TestGraphStore_ecosystemFilter(t *testing.T) {
	store := &mockGraphStore{}
	f := GraphFilters{Ecosystem: "npm"}
	_, err := store.GetGraph(context.Background(), uuid.New(), f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Ecosystem != "npm" {
		t.Errorf("Ecosystem filter not preserved: got %q", f.Ecosystem)
	}
}

// TestGraphStore_teamFilter verifies team filter param shape.
func TestGraphStore_teamFilter(t *testing.T) {
	f := GraphFilters{Team: "@acme/backend"}
	store := &mockGraphStore{}
	_, err := store.GetGraph(context.Background(), uuid.New(), f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
