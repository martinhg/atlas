package dependency

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

func (m *mockOrgResolver) GetOrgIDBySlug(_ context.Context, slug string) (uuid.UUID, bool, error) {
	return m.orgID, m.found, m.err
}

// newTestHandlerRouter creates a chi router wired to the given Handler for testing.
// Routes mirror what main.go registers.
func newTestHandlerRouter(h *Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/orgs/{slug}/dependencies", h.HandleListDependencies)
	r.Get("/orgs/{slug}/dependencies/{ecosystem}/*", h.HandleGetDependency)
	return r
}

// TestHandleListDependencies_200_with_pagination verifies the happy path:
// a valid slug returns 200 with a pagination envelope.
func TestHandleListDependencies_200_with_pagination(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{
		listResult: []DependencyWithCount{
			{Ecosystem: "npm", Name: "react", RepoCount: 3},
			{Ecosystem: "npm", Name: "typescript", RepoCount: 5},
		},
		listTotal: 2,
	}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies?page=1&per_page=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp listResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("page = %d, want 1", resp.Page)
	}
	if resp.PerPage != 10 {
		t.Errorf("per_page = %d, want 10", resp.PerPage)
	}
	if len(resp.Data) != 2 {
		t.Errorf("data len = %d, want 2", len(resp.Data))
	}
}

// TestHandleListDependencies_empty_org_200 verifies that an org with no
// dependencies returns 200 with an empty data array.
func TestHandleListDependencies_empty_org_200(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{
		listResult: []DependencyWithCount{},
		listTotal:  0,
	}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/empty-org/dependencies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp listResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
	if resp.Data == nil {
		t.Error("data must not be nil for empty org")
	}
	if len(resp.Data) != 0 {
		t.Errorf("data len = %d, want 0", len(resp.Data))
	}
}

// TestHandleListDependencies_unknown_slug_404 verifies that an unknown slug
// returns 404.
func TestHandleListDependencies_unknown_slug_404(t *testing.T) {
	orgResolver := &mockOrgResolver{found: false}
	store := &mockDepStore{}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/ghost/dependencies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleListDependencies_invalid_per_page_400 verifies that an invalid
// per_page value (0 or negative) returns 400.
func TestHandleListDependencies_invalid_per_page_400(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	tests := []struct {
		name    string
		perPage string
	}{
		{"zero", "0"},
		{"negative", "-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies?per_page="+tt.perPage, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("per_page=%s: expected 400, got %d — body: %s", tt.perPage, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleGetDependency_200_with_repos verifies the detail endpoint happy path.
func TestHandleGetDependency_200_with_repos(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{
		detailResult: []DepDetail{
			{RepoName: "web-app", Version: "^18.2.0", DepType: "dep", SourceFile: "package.json"},
			{RepoName: "api", Version: "^18.0.0", DepType: "dep", SourceFile: "package.json"},
		},
	}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies/npm/react", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp detailResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Ecosystem != "npm" {
		t.Errorf("ecosystem = %q, want %q", resp.Ecosystem, "npm")
	}
	if resp.Name != "react" {
		t.Errorf("name = %q, want %q", resp.Name, "react")
	}
	if len(resp.Repos) != 2 {
		t.Errorf("repos len = %d, want 2", len(resp.Repos))
	}
}

// TestHandleGetDependency_not_found_404 verifies that a dependency with no
// repos in the org returns 404.
func TestHandleGetDependency_not_found_404(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{detailResult: []DepDetail{}} // empty = not found
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies/npm/unknown-lib", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleListDependencies_per_page_exceeds_max_400 verifies that per_page
// values above 100 return 400 Bad Request.
func TestHandleListDependencies_per_page_exceeds_max_400(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	tests := []struct {
		name    string
		perPage string
	}{
		{"101", "101"},
		{"500", "500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies?per_page="+tt.perPage, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("per_page=%s: expected 400, got %d — body: %s", tt.perPage, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleListDependencies_invalid_page_400 verifies that page=0 and
// page=-1 return 400 Bad Request.
func TestHandleListDependencies_invalid_page_400(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	tests := []struct {
		name string
		page string
	}{
		{"zero", "0"},
		{"negative", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies?page="+tt.page, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("page=%s: expected 400, got %d — body: %s", tt.page, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleListDependencies_store_error_500 verifies that a store error in
// ListByOrg returns 500 Internal Server Error.
func TestHandleListDependencies_store_error_500(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{listErr: errors.New("database connection lost")}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleGetDependency_store_error_500 verifies that a store error in
// GetDetail returns 500 Internal Server Error.
func TestHandleGetDependency_store_error_500(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{detailErr: errors.New("database connection lost")}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies/npm/react", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleListDependencies_org_resolver_error_500 verifies that an
// OrgResolver error returns 500 Internal Server Error.
func TestHandleListDependencies_org_resolver_error_500(t *testing.T) {
	orgResolver := &mockOrgResolver{err: errors.New("resolver failure")}
	store := &mockDepStore{}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleGetDependency_org_resolver_error_500 verifies that an
// OrgResolver error in GetDetail returns 500 Internal Server Error.
func TestHandleGetDependency_org_resolver_error_500(t *testing.T) {
	orgResolver := &mockOrgResolver{err: errors.New("resolver failure")}
	store := &mockDepStore{}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies/npm/react", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHandleGetDependency_scoped_npm_package_200 verifies that scoped npm
// packages like @types/react are correctly routed via the wildcard route.
func TestHandleGetDependency_scoped_npm_package_200(t *testing.T) {
	orgID := uuid.New()
	orgResolver := &mockOrgResolver{orgID: orgID, found: true}
	store := &mockDepStore{
		detailResult: []DepDetail{
			{RepoName: "web-app", Version: "^18.2.0", DepType: "devDep", SourceFile: "package.json"},
		},
	}
	h := NewHandler(store, orgResolver)
	r := newTestHandlerRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/dependencies/npm/@types/react", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp detailResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Ecosystem != "npm" {
		t.Errorf("ecosystem = %q, want %q", resp.Ecosystem, "npm")
	}
	if resp.Name != "@types/react" {
		t.Errorf("name = %q, want %q", resp.Name, "@types/react")
	}
	if len(resp.Repos) != 1 {
		t.Errorf("repos len = %d, want 1", len(resp.Repos))
	}
}
