package impact

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// mockOrgResolver is a test double for the OrgResolver interface.
type mockOrgResolver struct {
	orgID uuid.UUID
	found bool
	err   error
}

func (m *mockOrgResolver) GetOrgIDBySlug(_ context.Context, _ string) (uuid.UUID, bool, error) {
	return m.orgID, m.found, m.err
}

// newTestHandlerRouter creates a chi router wired to the given Handler for
// testing. Routes mirror what main.go registers.
func newTestHandlerRouter(h *Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/orgs/{slug}/impact", h.HandleAnalyzeImpact)
	return r
}

// postImpact issues a POST to the impact endpoint with the given raw JSON body.
func postImpact(r chi.Router, slug, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/orgs/"+slug+"/impact", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestHandleAnalyzeImpact_200 verifies the happy path: a valid request
// against a known org and dependency returns 200 with the full envelope.
func TestHandleAnalyzeImpact_200(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockImpactStore{
		result: []AffectedRepo{
			{RepoName: "svc-a", FullName: "acme/svc-a", Version: "1.0.0", DepType: "direct", Teams: []string{"@acme/team-x"}},
			{RepoName: "svc-b", FullName: "acme/svc-b", Version: "1.3.0", DepType: "direct", Teams: []string{}},
		},
	}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	w := postImpact(r, "acme", `{"ecosystem":"npm","name":"left-pad"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp BlastRadius
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Dependency.Ecosystem != "npm" {
		t.Errorf("dependency.ecosystem = %q, want %q", resp.Dependency.Ecosystem, "npm")
	}
	if resp.Dependency.Name != "left-pad" {
		t.Errorf("dependency.name = %q, want %q", resp.Dependency.Name, "left-pad")
	}
	if resp.TotalRepos != 2 {
		t.Errorf("total_repos = %d, want 2", resp.TotalRepos)
	}
	if len(resp.AffectedRepos) != 2 {
		t.Errorf("affected_repos len = %d, want 2", len(resp.AffectedRepos))
	}
	if len(resp.VersionDistribution) != 2 {
		t.Errorf("version_distribution len = %d, want 2", len(resp.VersionDistribution))
	}
}

// TestHandleAnalyzeImpact_zeroRepos_200 verifies that a dependency with no
// affected repos returns 200 with an empty result set, per spec — this is a
// valid empty state, not an error.
func TestHandleAnalyzeImpact_zeroRepos_200(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockImpactStore{result: []AffectedRepo{}}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	w := postImpact(r, "acme", `{"ecosystem":"npm","name":"unused-pkg"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp BlastRadius
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.TotalRepos != 0 {
		t.Errorf("total_repos = %d, want 0", resp.TotalRepos)
	}
	if resp.RiskLevel != RiskLow {
		t.Errorf("risk_level = %q, want %q", resp.RiskLevel, RiskLow)
	}
	if resp.AffectedRepos == nil {
		t.Error("affected_repos must not be nil")
	}
	if resp.VersionDistribution == nil {
		t.Error("version_distribution must not be nil")
	}
}

// TestHandleAnalyzeImpact_unknownSlug_404 verifies that an unknown org slug
// returns 404.
func TestHandleAnalyzeImpact_unknownSlug_404(t *testing.T) {
	orgResolver := &mockOrgResolver{found: false}
	store := &mockImpactStore{}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	w := postImpact(r, "ghost", `{"ecosystem":"npm","name":"left-pad"}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleAnalyzeImpact_missingBody_400 verifies that a request missing
// required fields (ecosystem or name) returns 400.
func TestHandleAnalyzeImpact_missingBody_400(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockImpactStore{}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	tests := []struct {
		name string
		body string
	}{
		{"missing name", `{"ecosystem":"npm"}`},
		{"missing ecosystem", `{"name":"left-pad"}`},
		{"empty body", `{}`},
		{"malformed json", `not-json`},
		{"empty ecosystem string", `{"ecosystem":"","name":"left-pad"}`},
		{"empty name string", `{"ecosystem":"npm","name":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := postImpact(r, "acme", tt.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("body=%s: expected 400, got %d — body: %s", tt.body, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleAnalyzeImpact_storeError_500 verifies that a store error returns
// 500 Internal Server Error.
func TestHandleAnalyzeImpact_storeError_500(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockImpactStore{err: errors.New("database connection lost")}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	w := postImpact(r, "acme", `{"ecosystem":"npm","name":"left-pad"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleAnalyzeImpact_resolverError_500 verifies that an OrgResolver
// error returns 500 Internal Server Error.
func TestHandleAnalyzeImpact_resolverError_500(t *testing.T) {
	orgResolver := &mockOrgResolver{err: errors.New("resolver failure")}
	store := &mockImpactStore{}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	w := postImpact(r, "acme", `{"ecosystem":"npm","name":"left-pad"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleAnalyzeImpact_riskMapping verifies that the response risk_level
// and risk_score reflect computeRiskScore over the store's affected repos.
func TestHandleAnalyzeImpact_riskMapping(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockImpactStore{
		result: []AffectedRepo{
			{RepoName: "svc-a", FullName: "acme/svc-a", Version: "1.0.0", DepType: "dev", Teams: []string{}},
		},
	}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	w := postImpact(r, "acme", `{"ecosystem":"npm","name":"left-pad"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp BlastRadius
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.RiskLevel != RiskLow {
		t.Errorf("risk_level = %q, want %q (single dev dep)", resp.RiskLevel, RiskLow)
	}
}

// TestHandleAnalyzeImpact_versionDistribution verifies that repos sharing
// the same version are grouped into a single version_distribution entry.
func TestHandleAnalyzeImpact_versionDistribution(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockImpactStore{
		result: []AffectedRepo{
			{RepoName: "svc-a", FullName: "acme/svc-a", Version: "1.0.0", DepType: "direct", Teams: []string{}},
			{RepoName: "svc-b", FullName: "acme/svc-b", Version: "1.0.0", DepType: "direct", Teams: []string{}},
			{RepoName: "svc-c", FullName: "acme/svc-c", Version: "1.3.0", DepType: "direct", Teams: []string{}},
		},
	}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	w := postImpact(r, "acme", `{"ecosystem":"npm","name":"left-pad"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp BlastRadius
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.VersionDistribution) != 2 {
		t.Fatalf("version_distribution len = %d, want 2", len(resp.VersionDistribution))
	}

	counts := make(map[string]int)
	for _, v := range resp.VersionDistribution {
		counts[v.Version] = v.RepoCount
	}
	if counts["1.0.0"] != 2 {
		t.Errorf("repo_count for 1.0.0 = %d, want 2", counts["1.0.0"])
	}
	if counts["1.3.0"] != 1 {
		t.Errorf("repo_count for 1.3.0 = %d, want 1", counts["1.3.0"])
	}
}
