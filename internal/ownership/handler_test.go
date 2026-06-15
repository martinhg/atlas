package ownership

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	ownerparser "github.com/nesbite/atlas/internal/ownership/parser"
)

// --- Mock store for handler tests ---
// Named mockHandlerStore to avoid conflict with mockHandlerStore in store_test.go.

type mockHandlerStore struct {
	listByOrgResult  []RepoOwnerSummary
	listByOrgTotal   int
	listByOrgErr     error
	listByRepoResult []OwnerRule
	listByRepoErr    error
	// capture last pagination params passed
	lastPage    int
	lastPerPage int
}

func (m *mockHandlerStore) SyncRepoOwners(_ context.Context, _ uuid.UUID, _ []ownerparser.ParsedOwner) error {
	return nil
}

func (m *mockHandlerStore) ListByOrg(_ context.Context, _ uuid.UUID, page, perPage int) ([]RepoOwnerSummary, int, error) {
	m.lastPage = page
	m.lastPerPage = perPage
	return m.listByOrgResult, m.listByOrgTotal, m.listByOrgErr
}

func (m *mockHandlerStore) ListByRepo(_ context.Context, _ uuid.UUID, _ string) ([]OwnerRule, error) {
	return m.listByRepoResult, m.listByRepoErr
}

// --- Mock org resolver ---

type mockOrgResolver struct {
	orgID uuid.UUID
	found bool
	err   error
}

func (m *mockOrgResolver) GetOrgIDBySlug(_ context.Context, _ string) (uuid.UUID, bool, error) {
	return m.orgID, m.found, m.err
}

// --- Helpers ---

// newHandlerWithRouter creates a Handler and a chi router with both routes registered.
func newHandlerWithRouter(store *mockHandlerStore, resolver *mockOrgResolver) (*Handler, *chi.Mux) {
	h := NewHandler(store, resolver)
	r := chi.NewRouter()
	r.Get("/orgs/{slug}/ownership", h.HandleListOwnership)
	r.Get("/orgs/{slug}/ownership/{repo}", h.HandleGetRepoOwnership)
	return h, r
}

// --- HandleListOwnership tests ---

func TestHandleListOwnership_returnsData(t *testing.T) {
	orgID := uuid.New()
	store := &mockHandlerStore{
		listByOrgResult: []RepoOwnerSummary{
			{RepoName: "api", OwnerCount: 3, TeamCount: 1, Teams: []string{"@org/backend"}},
			{RepoName: "web", OwnerCount: 2, TeamCount: 0, Teams: []string{}},
		},
		listByOrgTotal: 2,
	}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data    []RepoOwnerSummary `json:"data"`
		Total   int                `json:"total"`
		Page    int                `json:"page"`
		PerPage int                `json:"per_page"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Data))
	}
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
	if resp.PerPage != 50 {
		t.Errorf("expected per_page 50, got %d", resp.PerPage)
	}
}

func TestHandleListOwnership_emptyOrg(t *testing.T) {
	orgID := uuid.New()
	store := &mockHandlerStore{
		listByOrgResult: []RepoOwnerSummary{},
		listByOrgTotal:  0,
	}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data  []RepoOwnerSummary `json:"data"`
		Total int                `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data == nil {
		t.Errorf("expected non-nil data array, got nil")
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected empty data array, got %d items", len(resp.Data))
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}

func TestHandleListOwnership_orgNotFound(t *testing.T) {
	store := &mockHandlerStore{}
	resolver := &mockOrgResolver{found: false}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/ghost/ownership", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "org not found" {
		t.Errorf("expected error 'org not found', got %q", resp["error"])
	}
}

func TestHandleListOwnership_storeError(t *testing.T) {
	orgID := uuid.New()
	store := &mockHandlerStore{listByOrgErr: fmt.Errorf("db error")}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "internal error" {
		t.Errorf("expected error 'internal error', got %q", resp["error"])
	}
}

func TestHandleListOwnership_invalidPerPage_zero(t *testing.T) {
	orgID := uuid.New()
	store := &mockHandlerStore{}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership?per_page=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleListOwnership_invalidPerPage_overMax(t *testing.T) {
	orgID := uuid.New()
	store := &mockHandlerStore{}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership?per_page=200", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleListOwnership_invalidPage_zero(t *testing.T) {
	orgID := uuid.New()
	store := &mockHandlerStore{}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership?page=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleListOwnership_defaultPagination(t *testing.T) {
	orgID := uuid.New()
	store := &mockHandlerStore{listByOrgResult: []RepoOwnerSummary{}}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	// No query params — should default to page=1, per_page=50
	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if store.lastPage != 1 {
		t.Errorf("expected page=1 passed to store, got %d", store.lastPage)
	}
	if store.lastPerPage != 50 {
		t.Errorf("expected per_page=50 passed to store, got %d", store.lastPerPage)
	}
}

// --- HandleGetRepoOwnership tests ---

func TestHandleGetRepoOwnership_returnsRules(t *testing.T) {
	orgID := uuid.New()
	lineNum := 5
	store := &mockHandlerStore{
		listByRepoResult: []OwnerRule{
			{Pattern: "src/api/**", Owner: "@org/backend-team", OwnerType: "team", LineNumber: &lineNum},
			{Pattern: "*.go", Owner: "@username", OwnerType: "user"},
			{Pattern: "docs/", Owner: "user@example.com", OwnerType: "email"},
		},
	}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership/myrepo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Repo  string      `json:"repo"`
		Rules []OwnerRule `json:"rules"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Repo != "myrepo" {
		t.Errorf("expected repo 'myrepo', got %q", resp.Repo)
	}
	if len(resp.Rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(resp.Rules))
	}
}

func TestHandleGetRepoOwnership_emptyRules(t *testing.T) {
	orgID := uuid.New()
	store := &mockHandlerStore{listByRepoResult: []OwnerRule{}}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership/myrepo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Empty rules → 200, NOT 404
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty rules, got %d", w.Code)
	}

	var resp struct {
		Repo  string      `json:"repo"`
		Rules []OwnerRule `json:"rules"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Rules == nil {
		t.Errorf("expected non-nil rules array, got nil")
	}
	if len(resp.Rules) != 0 {
		t.Errorf("expected empty rules array, got %d items", len(resp.Rules))
	}
}

func TestHandleGetRepoOwnership_orgNotFound(t *testing.T) {
	store := &mockHandlerStore{}
	resolver := &mockOrgResolver{found: false}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/ghost/ownership/myrepo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "org not found" {
		t.Errorf("expected error 'org not found', got %q", resp["error"])
	}
}

func TestHandleGetRepoOwnership_storeError(t *testing.T) {
	orgID := uuid.New()
	store := &mockHandlerStore{listByRepoErr: fmt.Errorf("db error")}
	resolver := &mockOrgResolver{orgID: orgID, found: true}
	_, r := newHandlerWithRouter(store, resolver)

	req := httptest.NewRequest(http.MethodGet, "/orgs/myorg/ownership/myrepo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "internal error" {
		t.Errorf("expected error 'internal error', got %q", resp["error"])
	}
}
