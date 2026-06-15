package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockRepoStore struct {
	repos   []Repository
	listErr error
}

func (m *mockRepoStore) UpsertRepository(_ context.Context, r *Repository) (*Repository, error) {
	return r, nil
}

func (m *mockRepoStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID) ([]Repository, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.repos == nil {
		return []Repository{}, nil
	}
	return m.repos, nil
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

func TestHandleListRepos_200(t *testing.T) {
	orgID := uuid.New()
	store := &mockRepoStore{
		repos: []Repository{
			{ID: uuid.New(), OrgID: orgID, Name: "repo-1", FullName: "org/repo-1"},
			{ID: uuid.New(), OrgID: orgID, Name: "repo-2", FullName: "org/repo-2"},
		},
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
