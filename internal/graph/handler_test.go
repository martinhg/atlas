package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// mockOrgResolver is a test double for OrgResolver.
type mockOrgResolver struct {
	orgID uuid.UUID
	found bool
	err   error
}

func (m *mockOrgResolver) GetOrgIDBySlug(_ context.Context, _ string) (uuid.UUID, bool, error) {
	return m.orgID, m.found, m.err
}

// mockStore is a test double for GraphStore.
type mockStore struct {
	aggregates []depAggregate
	err        error
}

func (m *mockStore) GetGraph(_ context.Context, _ uuid.UUID, _ GraphFilters) ([]depAggregate, error) {
	return m.aggregates, m.err
}

// setupRouter builds a chi router that wires the graph handler at the canonical
// path, mirroring how main.go registers routes.
func setupRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/orgs/{slug}/graph", h.HandleGetGraph)
	return r
}

// makeRequest creates a test HTTP request routed through the chi router.
func makeRequest(t *testing.T, router http.Handler, url string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestHandleGetGraph_200_emptyGraph verifies a 200 with empty nodes/edges when
// the org has no dependencies.
func TestHandleGetGraph_200_emptyGraph(t *testing.T) {
	orgID := uuid.New()
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockStore{aggregates: []depAggregate{}}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	w := makeRequest(t, router, "/orgs/acme/graph")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Truncated {
		t.Error("expected truncated=false for empty graph")
	}
	if resp.Nodes == nil {
		t.Error("expected non-nil nodes slice")
	}
	if resp.Edges == nil {
		t.Error("expected non-nil edges slice")
	}
	if len(resp.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(resp.Nodes))
	}
	if len(resp.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(resp.Edges))
	}
}

// TestHandleGetGraph_200_payloadShape verifies that a graph with one dep and
// one repo produces the correct node/edge structure.
func TestHandleGetGraph_200_payloadShape(t *testing.T) {
	orgID := uuid.New()
	depID := uuid.New()
	repoID := uuid.New()
	lang := "Go"

	agg := depAggregate{
		DepID:     depID,
		Ecosystem: "npm",
		Name:      "lodash",
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

	resolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockStore{aggregates: []depAggregate{agg}}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	w := makeRequest(t, router, "/orgs/acme/graph")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body: %s", w.Code, w.Body.String())
	}

	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Should have: 1 repo node + 1 dep node + 1 team node = 3 nodes
	if len(resp.Nodes) != 3 {
		t.Errorf("expected 3 nodes (repo+dep+team), got %d: %v", len(resp.Nodes), resp.Nodes)
	}
	// Should have: 1 repo→dep edge + 1 repo→team edge = 2 edges
	if len(resp.Edges) != 2 {
		t.Errorf("expected 2 edges (repo->dep, repo->team), got %d", len(resp.Edges))
	}
	if resp.Truncated {
		t.Error("expected truncated=false for small graph")
	}

	// Verify node ID formats
	nodesByType := make(map[string][]Node)
	for _, n := range resp.Nodes {
		nodesByType[string(n.Type)] = append(nodesByType[string(n.Type)], n)
	}
	if len(nodesByType["repo"]) != 1 {
		t.Errorf("expected 1 repo node, got %d", len(nodesByType["repo"]))
	}
	if len(nodesByType["dep"]) != 1 {
		t.Errorf("expected 1 dep node, got %d", len(nodesByType["dep"]))
	}
	if len(nodesByType["team"]) != 1 {
		t.Errorf("expected 1 team node, got %d", len(nodesByType["team"]))
	}
}

// TestHandleGetGraph_unknownSlug_404 verifies that an unknown org slug returns 404.
func TestHandleGetGraph_unknownSlug_404(t *testing.T) {
	resolver := &mockOrgResolver{found: false}
	store := &mockStore{}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	w := makeRequest(t, router, "/orgs/unknown-org/graph")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestHandleGetGraph_resolverError_500 verifies that a resolver error returns 500.
func TestHandleGetGraph_resolverError_500(t *testing.T) {
	resolver := &mockOrgResolver{err: errors.New("db failure")}
	store := &mockStore{}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	w := makeRequest(t, router, "/orgs/acme/graph")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// TestHandleGetGraph_storeError_500 verifies that a store error returns 500.
func TestHandleGetGraph_storeError_500(t *testing.T) {
	orgID := uuid.New()
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockStore{err: errors.New("query failed")}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	w := makeRequest(t, router, "/orgs/acme/graph")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// TestHandleGetGraph_invalidRisk_400 verifies that an invalid ?risk= value returns 400.
func TestHandleGetGraph_invalidRisk_400(t *testing.T) {
	orgID := uuid.New()
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockStore{aggregates: []depAggregate{}}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	w := makeRequest(t, router, "/orgs/acme/graph?risk=critical")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid risk value, got %d", w.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected error message in response body")
	}
}

// TestHandleGetGraph_validRiskValues verifies all valid risk values are accepted.
func TestHandleGetGraph_validRiskValues(t *testing.T) {
	orgID := uuid.New()
	valid := []string{"low", "medium", "high"}
	for _, v := range valid {
		t.Run(v, func(t *testing.T) {
			resolver := &mockOrgResolver{orgID: orgID, found: true}
			store := &mockStore{aggregates: []depAggregate{}}
			h := NewHandler(store, resolver)
			router := setupRouter(h)
			w := makeRequest(t, router, "/orgs/acme/graph?risk="+v)
			if w.Code != http.StatusOK {
				t.Errorf("risk=%q: expected 200, got %d", v, w.Code)
			}
		})
	}
}

// TestHandleGetGraph_truncation verifies that when the aggregate expands to
// more than edgeLimit edges, truncated=true is set and at most edgeLimit edges
// are returned.
func TestHandleGetGraph_truncation(t *testing.T) {
	orgID := uuid.New()
	resolver := &mockOrgResolver{orgID: orgID, found: true}

	// Build enough aggregates to exceed 5000 edges.
	// Each agg has 1 repo with 1 team → 2 edges per agg (repo→dep + repo→team).
	// With 2501 aggs we'd have 5002 edges.
	aggs := make([]depAggregate, 2501)
	lang := "Go"
	for i := range aggs {
		depID := uuid.New()
		repoID := uuid.New()
		aggs[i] = depAggregate{
			DepID:     depID,
			Ecosystem: "npm",
			Name:      fmt.Sprintf("dep-%d", i),
			AffectedRepos: []repoRef{
				{
					RepoID:   repoID,
					RepoName: fmt.Sprintf("repo-%d", i),
					Language: &lang,
					DepType:  "direct",
					Teams:    []string{"@acme/team"},
				},
			},
		}
	}

	store := &mockStore{aggregates: aggs}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	w := makeRequest(t, router, "/orgs/acme/graph")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !resp.Truncated {
		t.Error("expected truncated=true for large graph")
	}
	if len(resp.Edges) > edgeLimit {
		t.Errorf("edges len %d exceeds edgeLimit %d", len(resp.Edges), edgeLimit)
	}
}

// TestHandleGetGraph_noTeams verifies that an org with no team owners produces
// only repo and dep nodes with repo→dep edges (no team nodes or repo→team edges).
func TestHandleGetGraph_noTeams(t *testing.T) {
	orgID := uuid.New()
	depID := uuid.New()
	repoID := uuid.New()

	agg := depAggregate{
		DepID:     depID,
		Ecosystem: "npm",
		Name:      "react",
		AffectedRepos: []repoRef{
			{
				RepoID:   repoID,
				RepoName: "svc-a",
				Language: nil,
				DepType:  "direct",
				Teams:    []string{}, // no teams
			},
		},
	}

	resolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockStore{aggregates: []depAggregate{agg}}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	w := makeRequest(t, router, "/orgs/acme/graph")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	for _, n := range resp.Nodes {
		if n.Type == NodeTypeTeam {
			t.Error("expected no team nodes when org has no team owners")
		}
	}
	for _, e := range resp.Edges {
		if e.Label == "owns" {
			t.Error("expected no owns edges when org has no team owners")
		}
	}
}
