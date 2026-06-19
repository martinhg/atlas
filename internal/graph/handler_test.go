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

// TestHandleGetGraph_duplicateRepoDepEdge verifies that when the same repo
// appears multiple times for a single dep (e.g. two source files or two
// dep_types via the UNIQUE (repo_id, dep_id, source_file) constraint), the
// handler emits exactly ONE repo→dep edge with a unique ID. Duplicate edge
// IDs would break Sigma/graphology (duplicate edge key).
func TestHandleGetGraph_duplicateRepoDepEdge(t *testing.T) {
	orgID := uuid.New()
	depID := uuid.New()
	repoID := uuid.New()
	lang := "Go"

	// Same repo appears twice for the same dep: once as "dev" (package.json),
	// once as "direct" (a second manifest). This is legal under the
	// UNIQUE (repo_id, dep_id, source_file) constraint.
	agg := depAggregate{
		DepID:     depID,
		Ecosystem: "npm",
		Name:      "lodash",
		AffectedRepos: []repoRef{
			{RepoID: repoID, RepoName: "svc-a", Language: &lang, DepType: "dev", Teams: nil},
			{RepoID: repoID, RepoName: "svc-a", Language: &lang, DepType: "direct", Teams: nil},
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

	// Exactly one repo→dep edge for this (repo, dep) pair.
	wantEdgeID := fmt.Sprintf("e:repo:%s:dep:%s", repoID, depID)
	count := 0
	seen := make(map[string]struct{})
	for _, e := range resp.Edges {
		if _, dup := seen[e.ID]; dup {
			t.Errorf("duplicate edge ID emitted: %q", e.ID)
		}
		seen[e.ID] = struct{}{}
		if e.ID == wantEdgeID {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 repo->dep edge with ID %q, got %d", wantEdgeID, count)
	}

	// The representative dep_type must be the higher-precedence "direct"
	// (weight 1.0), not "dev" (weight 0.3).
	for _, e := range resp.Edges {
		if e.ID == wantEdgeID && e.DepType != "direct" {
			t.Errorf("expected representative dep_type 'direct', got %q", e.DepType)
		}
	}

	// Only one repo node should exist.
	repoCount := 0
	for _, n := range resp.Nodes {
		if n.Type == NodeTypeRepo {
			repoCount++
		}
	}
	if repoCount != 1 {
		t.Errorf("expected 1 repo node, got %d", repoCount)
	}
}

// TestHandleGetGraph_maxRiskLevel verifies that a repo connected to multiple
// deps of differing risk takes the MAX risk level. A repo linked to a low-risk
// dep and a high-risk dep must report risk_level "high".
func TestHandleGetGraph_maxRiskLevel(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()
	lowDepID := uuid.New()
	highDepID := uuid.New()

	// Low-risk dep: single repo, dev dep type → low score.
	lowAgg := depAggregate{
		DepID:     lowDepID,
		Ecosystem: "npm",
		Name:      "tiny-dev-dep",
		AffectedRepos: []repoRef{
			{RepoID: repoID, RepoName: "svc-a", DepType: "dev", Teams: nil},
		},
	}

	// High-risk dep: many repos, direct, broad team spread → high score.
	highRepos := make([]repoRef, 0, 10)
	highRepos = append(highRepos, repoRef{RepoID: repoID, RepoName: "svc-a", DepType: "direct", Teams: []string{"@acme/a"}})
	for i := 0; i < 9; i++ {
		highRepos = append(highRepos, repoRef{
			RepoID:   uuid.New(),
			RepoName: fmt.Sprintf("svc-%d", i),
			DepType:  "direct",
			Teams:    []string{fmt.Sprintf("@acme/team-%d", i)},
		})
	}
	highAgg := depAggregate{
		DepID:         highDepID,
		Ecosystem:     "npm",
		Name:          "widely-used",
		AffectedRepos: highRepos,
	}

	resolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockStore{aggregates: []depAggregate{lowAgg, highAgg}}
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

	repoNodeID := fmt.Sprintf("repo:%s", repoID)
	var found bool
	for _, n := range resp.Nodes {
		if n.ID == repoNodeID {
			found = true
			if n.RiskLevel != "high" {
				t.Errorf("expected repo risk_level 'high' (max of connected deps), got %q", n.RiskLevel)
			}
		}
	}
	if !found {
		t.Fatalf("repo node %q not found in response", repoNodeID)
	}
}

// TestHandleGetGraph_riskFilterNoOrphans verifies that filtering by risk=high
// narrows the graph to matching dep nodes plus their connected repos only,
// leaving no orphan nodes (no repo/dep/team node without a surviving edge).
func TestHandleGetGraph_riskFilterNoOrphans(t *testing.T) {
	orgID := uuid.New()
	lowRepoID := uuid.New()
	highRepoID := uuid.New()
	lowDepID := uuid.New()
	highDepID := uuid.New()

	lowAgg := depAggregate{
		DepID:     lowDepID,
		Ecosystem: "npm",
		Name:      "low-dep",
		AffectedRepos: []repoRef{
			{RepoID: lowRepoID, RepoName: "low-repo", DepType: "dev", Teams: []string{"@acme/low"}},
		},
	}

	highRepos := make([]repoRef, 0, 10)
	highRepos = append(highRepos, repoRef{RepoID: highRepoID, RepoName: "high-repo", DepType: "direct", Teams: []string{"@acme/high"}})
	for i := 0; i < 9; i++ {
		highRepos = append(highRepos, repoRef{
			RepoID:   uuid.New(),
			RepoName: fmt.Sprintf("hr-%d", i),
			DepType:  "direct",
			Teams:    []string{fmt.Sprintf("@acme/h-%d", i)},
		})
	}
	highAgg := depAggregate{
		DepID:         highDepID,
		Ecosystem:     "npm",
		Name:          "high-dep",
		AffectedRepos: highRepos,
	}

	resolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockStore{aggregates: []depAggregate{lowAgg, highAgg}}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	w := makeRequest(t, router, "/orgs/acme/graph?risk=high")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body: %s", w.Code, w.Body.String())
	}

	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// The low-risk dep node and its isolated repo must NOT appear.
	lowDepNodeID := fmt.Sprintf("dep:%s", lowDepID)
	lowRepoNodeID := fmt.Sprintf("repo:%s", lowRepoID)
	for _, n := range resp.Nodes {
		if n.ID == lowDepNodeID {
			t.Error("low-risk dep node should be filtered out")
		}
		if n.ID == lowRepoNodeID {
			t.Error("repo connected only to the filtered dep should not appear")
		}
	}

	// No orphan nodes: every node must be referenced by some surviving edge.
	referenced := make(map[string]struct{})
	for _, e := range resp.Edges {
		referenced[e.Source] = struct{}{}
		referenced[e.Target] = struct{}{}
	}
	for _, n := range resp.Nodes {
		if _, ok := referenced[n.ID]; !ok {
			t.Errorf("orphan node with no edge: %q (type %s)", n.ID, n.Type)
		}
	}
}

// TestHandleGetGraph_truncationDeterministicNoOrphans verifies that truncation
// is deterministic (stable edge ordering), sets truncated=true, and prunes
// orphan nodes so no node survives without a surviving edge.
func TestHandleGetGraph_truncationDeterministicNoOrphans(t *testing.T) {
	orgID := uuid.New()
	resolver := &mockOrgResolver{orgID: orgID, found: true}

	// Use a small injected limit so we don't need 5000+ edges.
	prev := edgeLimit
	edgeLimit = 4
	defer func() { edgeLimit = prev }()

	// Build 6 aggregates, each 1 repo + 1 team → 2 edges each = 12 edges total.
	aggs := make([]depAggregate, 6)
	for i := range aggs {
		aggs[i] = depAggregate{
			DepID:     uuid.New(),
			Ecosystem: "npm",
			Name:      fmt.Sprintf("dep-%d", i),
			AffectedRepos: []repoRef{
				{RepoID: uuid.New(), RepoName: fmt.Sprintf("repo-%d", i), DepType: "direct", Teams: []string{fmt.Sprintf("@acme/t-%d", i)}},
			},
		}
	}

	store := &mockStore{aggregates: aggs}
	h := NewHandler(store, resolver)
	router := setupRouter(h)

	run := func() GraphResponse {
		w := makeRequest(t, router, "/orgs/acme/graph")
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp GraphResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		return resp
	}

	resp := run()

	if !resp.Truncated {
		t.Error("expected truncated=true when edges exceed limit")
	}
	if len(resp.Edges) != edgeLimit {
		t.Errorf("expected exactly %d edges after truncation, got %d", edgeLimit, len(resp.Edges))
	}

	// No orphan nodes: every node referenced by a surviving edge.
	referenced := make(map[string]struct{})
	for _, e := range resp.Edges {
		referenced[e.Source] = struct{}{}
		referenced[e.Target] = struct{}{}
	}
	for _, n := range resp.Nodes {
		if _, ok := referenced[n.ID]; !ok {
			t.Errorf("orphan node after truncation: %q (type %s)", n.ID, n.Type)
		}
	}

	// Determinism: repeated runs return the identical edge set in the same order.
	resp2 := run()
	if len(resp.Edges) != len(resp2.Edges) {
		t.Fatalf("non-deterministic edge count: %d vs %d", len(resp.Edges), len(resp2.Edges))
	}
	for i := range resp.Edges {
		if resp.Edges[i].ID != resp2.Edges[i].ID {
			t.Errorf("non-deterministic edge order at %d: %q vs %q", i, resp.Edges[i].ID, resp2.Edges[i].ID)
		}
	}
}

// TestHandleGetGraph_teamNodeIDSpecialChar verifies that an owner containing a
// character used as an ID separator (':') or whitespace is encoded so it cannot
// collide with the repo:/dep:/team: ID scheme.
func TestHandleGetGraph_teamNodeIDSpecialChar(t *testing.T) {
	orgID := uuid.New()
	depID := uuid.New()
	repoID := uuid.New()

	// Owner with a colon and whitespace.
	owner := "weird:owner name"
	agg := depAggregate{
		DepID:     depID,
		Ecosystem: "npm",
		Name:      "lodash",
		AffectedRepos: []repoRef{
			{RepoID: repoID, RepoName: "svc-a", DepType: "direct", Teams: []string{owner}},
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

	var teamNode *Node
	for i := range resp.Nodes {
		if resp.Nodes[i].Type == NodeTypeTeam {
			teamNode = &resp.Nodes[i]
		}
	}
	if teamNode == nil {
		t.Fatal("expected a team node")
	}

	// The label must preserve the original owner for display.
	if teamNode.Label != owner {
		t.Errorf("team label should preserve raw owner, got %q", teamNode.Label)
	}

	// The encoded ID must have exactly one ':' separator (the "team:" prefix),
	// so a raw ':' in the owner cannot create a second separator that collides
	// with the repo:/dep:/team: scheme.
	id := teamNode.ID
	prefix := "team:"
	if len(id) < len(prefix) || id[:len(prefix)] != prefix {
		t.Fatalf("team node ID missing 'team:' prefix: %q", id)
	}
	rest := id[len(prefix):]
	for _, c := range rest {
		if c == ':' {
			t.Errorf("encoded owner segment must not contain a raw ':', got %q", id)
		}
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
