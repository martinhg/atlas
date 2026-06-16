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

func (m *mockCatalogStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]catalog.Repository, int, error) {
	return nil, 0, nil
}

func (m *mockCatalogStore) GetRepoByName(_ context.Context, _ uuid.UUID, _ string) (*catalog.Repository, error) {
	return nil, nil
}

func newMockGitHubServer(repos []*gogithub.Repository, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if statusCode != 0 && statusCode != http.StatusOK {
			http.Error(w, "error", statusCode)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		count := len(repos)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count":  count,
			"repositories": repos,
		})
	}))
}

func TestSyncRepos_success(t *testing.T) {
	name1, name2, name3 := "repo-1", "repo-2", "repo-3"
	full1, full2, full3 := "org/repo-1", "org/repo-2", "org/repo-3"
	ownerLogin := "org"
	owner := &gogithub.User{Login: &ownerLogin}
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1, Owner: owner},
		{ID: ptr(int64(2)), Name: &name2, FullName: &full2, Owner: owner},
		{ID: ptr(int64(3)), Name: &name3, FullName: &full3, Owner: owner},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgID := uuid.New()
	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{}

	syncRepos(client, orgS, catS, nil, nil, orgID)

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

	syncRepos(client, orgS, catS, nil, nil, uuid.New())

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
	ownerLogin := "org"
	owner := &gogithub.User{Login: &ownerLogin}
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1, Owner: owner},
		{ID: ptr(int64(2)), Name: &name2, FullName: &full2, Owner: owner},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{upsertErr: fmt.Errorf("db connection lost")}

	syncRepos(client, orgS, catS, nil, nil, uuid.New())

	if len(catS.upsertCalls) != 2 {
		t.Errorf("expected 2 UpsertRepository calls (continues on error), got %d", len(catS.upsertCalls))
	}
	if len(orgS.lastSyncedCalls) != 0 {
		t.Errorf("expected 0 SetLastSyncedAt calls (partial sync), got %d", len(orgS.lastSyncedCalls))
	}
}

func ptr[T any](v T) *T { return &v }

// mockDepSyncer is a test double for the DepSyncer interface in the org package.
type mockDepSyncer struct {
	syncCalls  []depSyncCall
	errForRepo map[string]error // keyed by repo name
}

type depSyncCall struct {
	owner  string
	repo   string
	branch string
}

func (m *mockDepSyncer) SyncRepoDeps(_ context.Context, _ *gogithub.Client, _ uuid.UUID, owner, repo, branch string) error {
	m.syncCalls = append(m.syncCalls, depSyncCall{owner: owner, repo: repo, branch: branch})
	if m.errForRepo != nil {
		if err, ok := m.errForRepo[repo]; ok {
			return err
		}
	}
	return nil
}

// TestSyncRepos_depSyncer_called_after_upsert verifies that depSyncer.SyncRepoDeps
// is called once per repo after the catalog upsert succeeds.
func TestSyncRepos_depSyncer_called_after_upsert(t *testing.T) {
	name1, name2 := "repo-1", "repo-2"
	full1, full2 := "org/repo-1", "org/repo-2"
	branch := "main"
	ownerLogin := "org"
	owner := &gogithub.User{Login: &ownerLogin}
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1, DefaultBranch: &branch, Owner: owner},
		{ID: ptr(int64(2)), Name: &name2, FullName: &full2, DefaultBranch: &branch, Owner: owner},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgID := uuid.New()
	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{}
	depS := &mockDepSyncer{}

	syncRepos(client, orgS, catS, depS, nil, orgID)

	if len(depS.syncCalls) != 2 {
		t.Errorf("expected 2 SyncRepoDeps calls, got %d", len(depS.syncCalls))
	}
}

// TestSyncRepos_depSyncer_nil_guard verifies that passing nil as depSyncer
// causes syncRepos to skip dep sync without panicking.
func TestSyncRepos_depSyncer_nil_guard(t *testing.T) {
	name1 := "repo-1"
	full1 := "org/repo-1"
	branch := "main"
	ownerLogin := "org"
	owner := &gogithub.User{Login: &ownerLogin}
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1, DefaultBranch: &branch, Owner: owner},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{}

	// Must not panic.
	syncRepos(client, orgS, catS, nil, nil, uuid.New())

	if len(catS.upsertCalls) != 1 {
		t.Errorf("expected 1 UpsertRepository call, got %d", len(catS.upsertCalls))
	}
}

// TestSyncRepos_depSyncer_error_isolation verifies that if dep sync fails for
// one repo, the remaining repos continue to be processed.
func TestSyncRepos_depSyncer_error_isolation(t *testing.T) {
	name1, name2, name3 := "repo-1", "repo-2", "repo-3"
	full1, full2, full3 := "org/repo-1", "org/repo-2", "org/repo-3"
	branch := "main"
	ownerLogin := "org"
	owner := &gogithub.User{Login: &ownerLogin}
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1, DefaultBranch: &branch, Owner: owner},
		{ID: ptr(int64(2)), Name: &name2, FullName: &full2, DefaultBranch: &branch, Owner: owner},
		{ID: ptr(int64(3)), Name: &name3, FullName: &full3, DefaultBranch: &branch, Owner: owner},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{}
	depS := &mockDepSyncer{
		errForRepo: map[string]error{
			"repo-2": fmt.Errorf("github 403 for repo-2"),
		},
	}

	syncRepos(client, orgS, catS, depS, nil, uuid.New())

	// All three catalog upserts must succeed regardless of dep sync error.
	if len(catS.upsertCalls) != 3 {
		t.Errorf("expected 3 UpsertRepository calls, got %d", len(catS.upsertCalls))
	}

	// All three dep sync calls must have been attempted.
	if len(depS.syncCalls) != 3 {
		t.Errorf("expected 3 SyncRepoDeps calls (error isolation), got %d", len(depS.syncCalls))
	}
}

// mockOwnershipSyncer is a test double for the OwnershipSyncer interface in the org package.
type mockOwnershipSyncer struct {
	syncCalls  []ownershipSyncCall
	errForRepo map[string]error // keyed by repo name
}

type ownershipSyncCall struct {
	owner string
	repo  string
}

func (m *mockOwnershipSyncer) SyncRepoOwnership(_ context.Context, _ *gogithub.Client, _ uuid.UUID, owner, repo string) error {
	m.syncCalls = append(m.syncCalls, ownershipSyncCall{owner: owner, repo: repo})
	if m.errForRepo != nil {
		if err, ok := m.errForRepo[repo]; ok {
			return err
		}
	}
	return nil
}

// TestSyncRepos_ownershipSyncer_calledPerRepo verifies that ownershipSyncer.SyncRepoOwnership
// is called once per repo after a successful catalog upsert.
func TestSyncRepos_ownershipSyncer_calledPerRepo(t *testing.T) {
	name1, name2 := "repo-1", "repo-2"
	full1, full2 := "org/repo-1", "org/repo-2"
	branch := "main"
	ownerLogin := "org"
	owner := &gogithub.User{Login: &ownerLogin}
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1, DefaultBranch: &branch, Owner: owner},
		{ID: ptr(int64(2)), Name: &name2, FullName: &full2, DefaultBranch: &branch, Owner: owner},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgID := uuid.New()
	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{}
	ownerS := &mockOwnershipSyncer{}

	syncRepos(client, orgS, catS, nil, ownerS, orgID)

	if len(ownerS.syncCalls) != 2 {
		t.Errorf("expected 2 SyncRepoOwnership calls, got %d", len(ownerS.syncCalls))
	}
}

// TestSyncRepos_ownershipSyncer_nilGuard verifies that passing nil as ownershipSyncer
// does not panic and sync completes normally.
func TestSyncRepos_ownershipSyncer_nilGuard(t *testing.T) {
	name1 := "repo-1"
	full1 := "org/repo-1"
	branch := "main"
	ownerLogin := "org"
	owner := &gogithub.User{Login: &ownerLogin}
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1, DefaultBranch: &branch, Owner: owner},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{}

	// Must not panic with nil ownershipSyncer.
	syncRepos(client, orgS, catS, nil, nil, uuid.New())

	if len(catS.upsertCalls) != 1 {
		t.Errorf("expected 1 UpsertRepository call, got %d", len(catS.upsertCalls))
	}
}

// TestSyncRepos_ownershipSyncer_errorIsolation verifies that if ownership sync
// fails for one repo, the remaining repos are still processed.
func TestSyncRepos_ownershipSyncer_errorIsolation(t *testing.T) {
	name1, name2, name3 := "repo-1", "repo-2", "repo-3"
	full1, full2, full3 := "org/repo-1", "org/repo-2", "org/repo-3"
	branch := "main"
	ownerLogin := "org"
	owner := &gogithub.User{Login: &ownerLogin}
	repos := []*gogithub.Repository{
		{ID: ptr(int64(1)), Name: &name1, FullName: &full1, DefaultBranch: &branch, Owner: owner},
		{ID: ptr(int64(2)), Name: &name2, FullName: &full2, DefaultBranch: &branch, Owner: owner},
		{ID: ptr(int64(3)), Name: &name3, FullName: &full3, DefaultBranch: &branch, Owner: owner},
	}

	srv := newMockGitHubServer(repos, 0)
	defer srv.Close()

	client := gogithub.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(srv.URL + "/")

	orgS := &mockOrgStore{}
	catS := &mockCatalogStore{}
	ownerS := &mockOwnershipSyncer{
		errForRepo: map[string]error{
			"repo-2": fmt.Errorf("ownership sync failed for repo-2"),
		},
	}

	syncRepos(client, orgS, catS, nil, ownerS, uuid.New())

	// All three repos should have had ownership sync attempted.
	if len(ownerS.syncCalls) != 3 {
		t.Errorf("expected 3 SyncRepoOwnership calls (error isolation), got %d", len(ownerS.syncCalls))
	}
	// All three catalog upserts must succeed.
	if len(catS.upsertCalls) != 3 {
		t.Errorf("expected 3 UpsertRepository calls, got %d", len(catS.upsertCalls))
	}
}
