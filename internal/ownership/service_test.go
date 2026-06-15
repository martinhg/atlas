package ownership

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gogithub "github.com/google/go-github/v69/github"
	"github.com/google/uuid"
	ownerparser "github.com/nesbite/atlas/internal/ownership/parser"
)

// mockOwnershipStoreForService is a test double for OwnershipStore used in service tests.
type mockOwnershipStoreForService struct {
	syncCalls  []syncRepoOwnersCall
	syncErr    error
}

type syncRepoOwnersCall struct {
	repoID uuid.UUID
	owners []ownerparser.ParsedOwner
}

func (m *mockOwnershipStoreForService) SyncRepoOwners(_ context.Context, repoID uuid.UUID, owners []ownerparser.ParsedOwner) error {
	m.syncCalls = append(m.syncCalls, syncRepoOwnersCall{repoID: repoID, owners: owners})
	return m.syncErr
}

func (m *mockOwnershipStoreForService) ListByOrg(_ context.Context, _ uuid.UUID, _, _ int) ([]RepoOwnerSummary, int, error) {
	return nil, 0, nil
}

func (m *mockOwnershipStoreForService) ListByRepo(_ context.Context, _ uuid.UUID, _ string) ([]OwnerRule, error) {
	return nil, nil
}

// newGitHubTestClientWithServer creates a go-github client pointing at the given test server URL.
func newGitHubTestClientWithServer(serverURL string) *gogithub.Client {
	c := gogithub.NewClient(nil)
	c.BaseURL, _ = c.BaseURL.Parse(serverURL + "/")
	return c
}

// codeownersContent base64-encodes the given string to simulate GitHub's GetContents response.
func codeownersContent(raw string) string {
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// TestSyncRepoOwnership_rootPath verifies that if CODEOWNERS is found at the
// root path, the content is parsed and stored; the other two paths are NOT called.
func TestSyncRepoOwnership_rootPath(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if strings.Contains(r.URL.Path, "/repos/") && strings.HasSuffix(r.URL.Path, "/CODEOWNERS") &&
			!strings.Contains(r.URL.Path, ".github") && !strings.Contains(r.URL.Path, "docs") {
			// Root CODEOWNERS — return success with valid content
			content := fmt.Sprintf(`{"type":"file","encoding":"base64","content":"%s"}`,
				codeownersContent("*.go @go-team\n"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, content)
			return
		}
		// Should not be called for other paths
		t.Errorf("unexpected request to %s", r.URL.Path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer srv.Close()

	store := &mockOwnershipStoreForService{}
	svc := NewService(store)
	client := newGitHubTestClientWithServer(srv.URL)
	repoID := uuid.New()

	err := svc.SyncRepoOwnership(context.Background(), client, repoID, "myorg", "myrepo")

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoOwners call, got %d", len(store.syncCalls))
	}
	if len(store.syncCalls[0].owners) == 0 {
		t.Errorf("expected parsed owners to be stored, got empty slice")
	}
	// Only 1 GetContents call should have been made (for root CODEOWNERS)
	if callCount != 1 {
		t.Errorf("expected 1 GitHub API call, got %d (other paths should not be tried)", callCount)
	}
}

// TestSyncRepoOwnership_githubPath verifies that if the root path returns 404 but
// .github/CODEOWNERS returns 200, that content is used and docs/ is not fetched.
func TestSyncRepoOwnership_githubPath(t *testing.T) {
	callLog := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		callLog = append(callLog, path)
		if strings.Contains(path, "/contents/CODEOWNERS") &&
			!strings.Contains(path, ".github") && !strings.Contains(path, "docs") {
			// Root path → 404
			http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
			return
		}
		if strings.Contains(path, ".github%2FCODEOWNERS") || strings.Contains(path, ".github/CODEOWNERS") {
			// .github/CODEOWNERS → success
			content := fmt.Sprintf(`{"type":"file","encoding":"base64","content":"%s"}`,
				codeownersContent("src/ @org/team\n"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, content)
			return
		}
		// docs/CODEOWNERS should NOT be reached
		t.Errorf("unexpected request to %s", path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer srv.Close()

	store := &mockOwnershipStoreForService{}
	svc := NewService(store)
	client := newGitHubTestClientWithServer(srv.URL)
	repoID := uuid.New()

	err := svc.SyncRepoOwnership(context.Background(), client, repoID, "myorg", "myrepo")

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoOwners call, got %d", len(store.syncCalls))
	}
	if len(store.syncCalls[0].owners) == 0 {
		t.Errorf("expected owners from .github/CODEOWNERS to be stored")
	}
}

// TestSyncRepoOwnership_docsPath verifies fallback to docs/CODEOWNERS.
func TestSyncRepoOwnership_docsPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, ".github%2FCODEOWNERS") || strings.Contains(path, ".github/CODEOWNERS") {
			http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
			return
		}
		if strings.Contains(path, "docs%2FCODEOWNERS") || strings.Contains(path, "docs/CODEOWNERS") {
			content := fmt.Sprintf(`{"type":"file","encoding":"base64","content":"%s"}`,
				codeownersContent("docs/ @doc-team\n"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, content)
			return
		}
		// Root CODEOWNERS → 404
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	store := &mockOwnershipStoreForService{}
	svc := NewService(store)
	client := newGitHubTestClientWithServer(srv.URL)
	repoID := uuid.New()

	err := svc.SyncRepoOwnership(context.Background(), client, repoID, "myorg", "myrepo")

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoOwners call, got %d", len(store.syncCalls))
	}
	if len(store.syncCalls[0].owners) == 0 {
		t.Errorf("expected owners from docs/CODEOWNERS to be stored")
	}
}

// TestSyncRepoOwnership_allNotFound verifies that when all 3 paths return 404,
// SyncRepoOwners is called with an empty slice (to clear stale data) and nil is returned.
func TestSyncRepoOwnership_allNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	store := &mockOwnershipStoreForService{}
	svc := NewService(store)
	client := newGitHubTestClientWithServer(srv.URL)
	repoID := uuid.New()

	err := svc.SyncRepoOwnership(context.Background(), client, repoID, "myorg", "myrepo")

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	// SyncRepoOwners must be called with empty slice to clear stale data
	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoOwners call (to clear stale data), got %d", len(store.syncCalls))
	}
	if len(store.syncCalls[0].owners) != 0 {
		t.Errorf("expected empty owners slice to clear stale data, got %d owners", len(store.syncCalls[0].owners))
	}
}

// TestSyncRepoOwnership_githubNon404Error verifies that a non-404 GitHub error
// (e.g. 403) causes the function to log and return nil (error isolation).
func TestSyncRepoOwnership_githubNon404Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 403 for all requests
		http.Error(w, `{"message":"Forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	store := &mockOwnershipStoreForService{}
	svc := NewService(store)
	client := newGitHubTestClientWithServer(srv.URL)
	repoID := uuid.New()

	err := svc.SyncRepoOwnership(context.Background(), client, repoID, "myorg", "myrepo")

	if err != nil {
		t.Errorf("expected nil error (error isolation), got %v", err)
	}
	// Store must NOT be called on non-404 GitHub error
	if len(store.syncCalls) != 0 {
		t.Errorf("expected 0 SyncRepoOwners calls on GitHub error, got %d", len(store.syncCalls))
	}
}

// TestSyncRepoOwnership_storeError verifies that a store error causes the function
// to log and return nil (error isolation — store failure must not break sync loop).
func TestSyncRepoOwnership_storeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return valid CODEOWNERS content for root path
		content := fmt.Sprintf(`{"type":"file","encoding":"base64","content":"%s"}`,
			codeownersContent("*.go @go-team\n"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, content)
	}))
	defer srv.Close()

	store := &mockOwnershipStoreForService{syncErr: fmt.Errorf("db connection lost")}
	svc := NewService(store)
	client := newGitHubTestClientWithServer(srv.URL)
	repoID := uuid.New()

	err := svc.SyncRepoOwnership(context.Background(), client, repoID, "myorg", "myrepo")

	if err != nil {
		t.Errorf("expected nil error (error isolation on store failure), got %v", err)
	}
}

// TestSyncRepoOwnership_commentOnlyFile verifies that a CODEOWNERS file containing
// only comments results in an empty parsed slice, and SyncRepoOwners is still
// called with that empty slice to clear stale data.
func TestSyncRepoOwnership_commentOnlyFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only root CODEOWNERS path
		if strings.Contains(r.URL.Path, "/contents/CODEOWNERS") &&
			!strings.Contains(r.URL.Path, ".github") && !strings.Contains(r.URL.Path, "docs") {
			content := fmt.Sprintf(`{"type":"file","encoding":"base64","content":"%s"}`,
				codeownersContent("# This file intentionally left with only comments\n# No actual rules\n"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, content)
			return
		}
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	store := &mockOwnershipStoreForService{}
	svc := NewService(store)
	client := newGitHubTestClientWithServer(srv.URL)
	repoID := uuid.New()

	err := svc.SyncRepoOwnership(context.Background(), client, repoID, "myorg", "myrepo")

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	// SyncRepoOwners must still be called with an empty slice to clear stale data
	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoOwners call (empty slice clears stale), got %d", len(store.syncCalls))
	}
	if len(store.syncCalls[0].owners) != 0 {
		t.Errorf("expected empty owners slice for comment-only file, got %d owners", len(store.syncCalls[0].owners))
	}
}
