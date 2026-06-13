package org

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	gogithub "github.com/google/go-github/v69/github"
	"github.com/nesbite/atlas/internal/catalog"
)

type mockOrgStore struct {
	lastSyncedCalls []uuid.UUID
	lastSyncedErr   error
}

func (m *mockOrgStore) UpsertOrg(_ context.Context, o *Organization) (*Organization, error) {
	return o, nil
}
func (m *mockOrgStore) GetOrgBySlug(_ context.Context, _ string) (*Organization, error) {
	return nil, nil
}
func (m *mockOrgStore) GetOrgsByOwnerID(_ context.Context, _ uuid.UUID) ([]Organization, error) {
	return nil, nil
}
func (m *mockOrgStore) GetOrgByInstallationID(_ context.Context, _ int64) (*Organization, error) {
	return nil, nil
}
func (m *mockOrgStore) SetInstallationID(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}
func (m *mockOrgStore) SetLastSyncedAt(_ context.Context, orgID uuid.UUID, _ time.Time) error {
	m.lastSyncedCalls = append(m.lastSyncedCalls, orgID)
	return m.lastSyncedErr
}

type mockCatalogStore struct {
	upsertCalls []*catalog.Repository
	upsertErr   error
}

func (m *mockCatalogStore) UpsertRepository(_ context.Context, r *catalog.Repository) (*catalog.Repository, error) {
	m.upsertCalls = append(m.upsertCalls, r)
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	return r, nil
}

func (m *mockCatalogStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID) ([]catalog.Repository, error) {
	return nil, nil
}

func newMockGitHubServer(repos []*gogithub.Repository, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if statusCode != 0 && statusCode != http.StatusOK {
			http.Error(w, "error", statusCode)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(repos)
	}))
}

func TestSyncRepos_success(t *testing.T) {
	name1, name2, name3 := "repo-1", "repo-2", "repo-3"
	full1, full2, full3 := "org/repo-1", "org/repo-2", "org/repo-3"
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1},
		{ID: ptr(int64(2)), Name: &name2, FullName: &full2},
		{ID: ptr(int64(3)), Name: &name3, FullName: &full3},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgID := uuid.New()
	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{}

	syncRepos(client, orgS, catS, orgID, "org")

	if len(catS.upsertCalls) != 3 {
		t.Errorf("expected 3 UpsertRepository calls, got %d", len(catS.upsertCalls))
	}
	if len(orgS.lastSyncedCalls) != 1 {
		t.Errorf("expected 1 SetLastSyncedAt call, got %d", len(orgS.lastSyncedCalls))
	}
}

func TestSyncRepos_github_error(t *testing.T) {
	srv := newMockGitHubServer(nil, http.StatusInternalServerError)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{}

	syncRepos(client, orgS, catS, uuid.New(), "org")

	if len(catS.upsertCalls) != 0 {
		t.Errorf("expected 0 UpsertRepository calls, got %d", len(catS.upsertCalls))
	}
	if len(orgS.lastSyncedCalls) != 0 {
		t.Errorf("expected 0 SetLastSyncedAt calls, got %d", len(orgS.lastSyncedCalls))
	}
}

func TestSyncRepos_upsert_error(t *testing.T) {
	name1, name2 := "repo-1", "repo-2"
	full1, full2 := "org/repo-1", "org/repo-2"
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1},
		{ID: ptr(int64(2)), Name: &name2, FullName: &full2},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{upsertErr: fmt.Errorf("db connection lost")}

	syncRepos(client, orgS, catS, uuid.New(), "org")

	if len(catS.upsertCalls) != 2 {
		t.Errorf("expected 2 UpsertRepository calls (continues on error), got %d", len(catS.upsertCalls))
	}
	if len(orgS.lastSyncedCalls) != 0 {
		t.Errorf("expected 0 SetLastSyncedAt calls (partial sync), got %d", len(orgS.lastSyncedCalls))
	}
}

func ptr[T any](v T) *T { return &v }
