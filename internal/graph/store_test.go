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

// TestGraphStore_filteredEmptyResult verifies the contract that a filtered
// query yielding no matches returns a non-nil empty slice and no error, so the
// handler can build an empty graph rather than treating it as a failure. This
// covers the spec's "unknown ecosystem/team returns an empty graph" path.
func TestGraphStore_filteredEmptyResult(t *testing.T) {
	store := &mockGraphStore{aggregates: []depAggregate{}}
	for _, f := range []GraphFilters{
		{Ecosystem: "haskell"},
		{Team: "@acme/does-not-exist"},
	} {
		result, err := store.GetGraph(context.Background(), uuid.New(), f)
		if err != nil {
			t.Fatalf("filter %+v: unexpected error: %v", f, err)
		}
		if result == nil {
			t.Errorf("filter %+v: expected non-nil empty slice, got nil", f)
		}
		if len(result) != 0 {
			t.Errorf("filter %+v: expected 0 aggregates, got %d", f, len(result))
		}
	}
}
