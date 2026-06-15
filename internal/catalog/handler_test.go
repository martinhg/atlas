package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// mockRepoStore implements the updated RepoStore interface (PR1.3/PR1.4 signature).
type mockRepoStore struct {
	repos    []Repository
	total    int
	listErr  error
	// capture what was passed in the last call
	capturedQ       string
	capturedPage    int
	capturedPerPage int
}

func (m *mockRepoStore) UpsertRepository(_ context.Context, r *Repository) (*Repository, error) {
	return r, nil
}

func (m *mockRepoStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID, q string, page, perPage int) ([]Repository, int, error) {
	m.capturedQ = q
	m.capturedPage = page
	m.capturedPerPage = perPage
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	if m.repos == nil {
		return []Repository{}, m.total, nil
	}
	return m.repos, m.total, nil
}

type mockOrgResolver struct {
	orgID uuid.UUID
	found bool
	err   error
}

func (m *mockOrgResolver) GetOrgIDBySlug(_ context.Context, _ string) (uuid.UUID, bool, error) {
	return m.orgID, m.found, m.err
}

func newRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/orgs/{slug}/repos", h.HandleListRepos)
	return r
}

// ---------------------------------------------------------------------------
// Original tests — updated to expect the new envelope response shape
// ---------------------------------------------------------------------------

func TestHandleListRepos_200(t *testing.T) {
	orgID := uuid.New()
	store := &mockRepoStore{
		repos: []Repository{
			{ID: uuid.New(), OrgID: orgID, Name: "repo-1", FullName: "org/repo-1"},
			{ID: uuid.New(), OrgID: orgID, Name: "repo-2", FullName: "org/repo-2"},
		},
		total: 2,
	}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	h := NewHandler(store, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp repoListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(resp.Data))
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

func TestHandleListRepos_200_empty(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	h := NewHandler(&mockRepoStore{}, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp repoListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data == nil {
		t.Error("data must not be nil for empty result")
	}
}

func TestHandleListRepos_404_org_not_found(t *testing.T) {
	resolver := &mockOrgResolver{found: false}
	h := NewHandler(&mockRepoStore{}, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/unknown-org/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleListRepos_500_resolver_error(t *testing.T) {
	resolver := &mockOrgResolver{err: fmt.Errorf("db down")}
	h := NewHandler(&mockRepoStore{}, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleListRepos_500_store_error(t *testing.T) {
	resolver := &mockOrgResolver{orgID: uuid.New(), found: true}
	h := NewHandler(&mockRepoStore{listErr: fmt.Errorf("db down")}, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ---------------------------------------------------------------------------
// PR1.5: New handler tests for q/page/per_page parsing and envelope shape.
// These FAIL until PR1.6 updates the handler.
// ---------------------------------------------------------------------------

// TestHandleListRepos_q_param_passed_to_store verifies ?q= is parsed and forwarded.
func TestHandleListRepos_q_param_passed_to_store(t *testing.T) {
	orgID := uuid.New()
	store := &mockRepoStore{repos: []Repository{}, total: 0}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	h := NewHandler(store, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos?q=react", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if store.capturedQ != "react" {
		t.Errorf("capturedQ = %q, want %q", store.capturedQ, "react")
	}
}

// TestHandleListRepos_empty_q_param passes "" to store.
func TestHandleListRepos_empty_q_param(t *testing.T) {
	orgID := uuid.New()
	store := &mockRepoStore{repos: []Repository{}, total: 0}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	h := NewHandler(store, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos?q=", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if store.capturedQ != "" {
		t.Errorf("capturedQ = %q, want empty string", store.capturedQ)
	}
}

// TestHandleListRepos_no_q_param defaults q to "".
func TestHandleListRepos_no_q_param(t *testing.T) {
	orgID := uuid.New()
	store := &mockRepoStore{repos: []Repository{}, total: 0}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	h := NewHandler(store, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if store.capturedQ != "" {
		t.Errorf("capturedQ = %q, want empty string", store.capturedQ)
	}
}

// TestHandleListRepos_page_and_per_page_parsed verifies pagination params are forwarded.
func TestHandleListRepos_page_and_per_page_parsed(t *testing.T) {
	orgID := uuid.New()
	store := &mockRepoStore{repos: []Repository{}, total: 0}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	h := NewHandler(store, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos?page=2&per_page=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if store.capturedPage != 2 {
		t.Errorf("capturedPage = %d, want 2", store.capturedPage)
	}
	if store.capturedPerPage != 10 {
		t.Errorf("capturedPerPage = %d, want 10", store.capturedPerPage)
	}
}

// TestHandleListRepos_page_zero_400 verifies page=0 returns 400.
func TestHandleListRepos_page_zero_400(t *testing.T) {
	orgID := uuid.New()
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	h := NewHandler(&mockRepoStore{}, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos?page=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestHandleListRepos_per_page_exceeds_max_400 verifies per_page>100 returns 400.
func TestHandleListRepos_per_page_exceeds_max_400(t *testing.T) {
	orgID := uuid.New()
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	h := NewHandler(&mockRepoStore{}, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos?per_page=200", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestHandleListRepos_per_page_negative_400 verifies per_page=-1 returns 400.
func TestHandleListRepos_per_page_negative_400(t *testing.T) {
	orgID := uuid.New()
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	h := NewHandler(&mockRepoStore{}, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos?per_page=-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestHandleListRepos_envelope_shape verifies the response is {data, total, page, per_page}.
func TestHandleListRepos_envelope_shape(t *testing.T) {
	orgID := uuid.New()
	store := &mockRepoStore{
		repos: []Repository{
			{ID: uuid.New(), OrgID: orgID, Name: "r1", FullName: "org/r1"},
		},
		total: 10,
	}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	h := NewHandler(store, resolver)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos?page=2&per_page=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — body: %s", w.Code, w.Body.String())
	}

	var resp repoListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 10 {
		t.Errorf("total = %d, want 10", resp.Total)
	}
	if resp.Page != 2 {
		t.Errorf("page = %d, want 2", resp.Page)
	}
	if resp.PerPage != 5 {
		t.Errorf("per_page = %d, want 5", resp.PerPage)
	}
	if len(resp.Data) != 1 {
		t.Errorf("data len = %d, want 1", len(resp.Data))
	}
}
