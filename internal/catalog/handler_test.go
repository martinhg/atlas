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

func newRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/orgs/{orgID}/repos", h.HandleListRepos)
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
	h := NewHandler(store)
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/"+orgID.String()+"/repos", nil)
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
	h := NewHandler(&mockRepoStore{})
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/"+uuid.New().String()+"/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleListRepos_400_invalid_org_id(t *testing.T) {
	h := NewHandler(&mockRepoStore{})
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/not-a-uuid/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleListRepos_500_store_error(t *testing.T) {
	h := NewHandler(&mockRepoStore{listErr: fmt.Errorf("db down")})
	r := newRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/orgs/"+uuid.New().String()+"/repos", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
