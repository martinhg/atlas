package dependency

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gogithub "github.com/google/go-github/v69/github"
	"github.com/google/uuid"
)

// newGitHubClientForTest creates a gogithub.Client pointed at the given test server URL.
func newGitHubClientForTest(t *testing.T, serverURL string) *gogithub.Client {
	t.Helper()
	client := gogithub.NewClient(nil)
	var err error
	client.BaseURL, err = client.BaseURL.Parse(serverURL + "/")
	if err != nil {
		t.Fatalf("parse base URL: %v", err)
	}
	client.UploadURL, err = client.UploadURL.Parse(serverURL + "/")
	if err != nil {
		t.Fatalf("parse upload URL: %v", err)
	}
	return client
}

// treeResponse is the minimal shape the GitHub Git.GetTree API returns.
type treeResponse struct {
	SHA       string       `json:"sha"`
	Truncated bool         `json:"truncated"`
	Tree      []treeEntry  `json:"tree"`
}

type treeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

// TestSyncRepoDeps_filters_node_modules verifies that paths containing
// node_modules/ are excluded from the package.json discovery.
func TestSyncRepoDeps_filters_node_modules(t *testing.T) {
	var fetchedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Tree endpoint: returns root package.json + node_modules one.
		if strings.Contains(r.URL.Path, "/git/trees/") {
			resp := treeResponse{
				SHA:       "abc123",
				Truncated: false,
				Tree: []treeEntry{
					{Path: "package.json", Type: "blob"},
					{Path: "node_modules/lodash/package.json", Type: "blob"},
					{Path: "src/index.ts", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Contents endpoint: track which paths were fetched.
		if strings.Contains(r.URL.Path, "/contents/") {
			// Extract path from URL.
			parts := strings.SplitN(r.URL.Path, "/contents/", 2)
			if len(parts) == 2 {
				fetchedPaths = append(fetchedPaths, parts[1])
			}

			// Return a minimal package.json with base64-encoded content.
			pkgJSON := `{"dependencies":{"react":"^18.0.0"}}`
			encoded := base64.StdEncoding.EncodeToString([]byte(pkgJSON))
			resp := map[string]string{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
				"name":     "package.json",
				"path":     "package.json",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := newGitHubClientForTest(t, srv.URL)
	store := &mockDepStore{}
	svc := NewService(store)

	err := svc.SyncRepoDeps(context.Background(), client, uuid.New(), "acme", "web-app", "main")
	if err != nil {
		t.Fatalf("SyncRepoDeps: unexpected error: %v", err)
	}

	// Only package.json at root should have been fetched, not the node_modules one.
	for _, p := range fetchedPaths {
		if strings.Contains(p, "node_modules") {
			t.Errorf("fetched node_modules path: %q — should have been excluded", p)
		}
	}
	if len(fetchedPaths) != 1 {
		t.Errorf("expected 1 content fetch, got %d: %v", len(fetchedPaths), fetchedPaths)
	}
}

// TestSyncRepoDeps_truncated_tree_logs_warning verifies that a truncated tree
// response is processed (not aborted) and the test completes without error.
// (Log output is not asserted — we rely on the service not returning an error.)
func TestSyncRepoDeps_truncated_tree_processes_partial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/git/trees/") {
			resp := treeResponse{
				SHA:       "def456",
				Truncated: true, // truncated!
				Tree: []treeEntry{
					{Path: "package.json", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/contents/") {
			pkgJSON := `{"dependencies":{"lodash":"^4.17.21"}}`
			encoded := base64.StdEncoding.EncodeToString([]byte(pkgJSON))
			resp := map[string]string{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
				"name":     "package.json",
				"path":     "package.json",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := newGitHubClientForTest(t, srv.URL)
	store := &mockDepStore{}
	svc := NewService(store)

	// Truncated tree must not abort — err should be nil.
	err := svc.SyncRepoDeps(context.Background(), client, uuid.New(), "acme", "web-app", "main")
	if err != nil {
		t.Fatalf("SyncRepoDeps with truncated tree: unexpected error: %v", err)
	}

	// At least one sync call should have been made for the partial tree.
	if len(store.syncCalls) != 1 {
		t.Errorf("expected 1 SyncRepoDependencies call, got %d", len(store.syncCalls))
	}
}

// TestSyncRepoDeps_no_package_json_is_noop verifies that a repo without any
// package.json completes without error and makes no store calls.
func TestSyncRepoDeps_no_package_json_is_noop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/git/trees/") {
			// No package.json files.
			resp := treeResponse{
				SHA:       "ghi789",
				Truncated: false,
				Tree: []treeEntry{
					{Path: "main.go", Type: "blob"},
					{Path: "README.md", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := newGitHubClientForTest(t, srv.URL)
	store := &mockDepStore{}
	svc := NewService(store)

	err := svc.SyncRepoDeps(context.Background(), client, uuid.New(), "acme", "go-service", "main")
	if err != nil {
		t.Fatalf("SyncRepoDeps with no package.json: unexpected error: %v", err)
	}

	// No store calls — nothing to sync.
	if len(store.syncCalls) != 0 {
		t.Errorf("expected 0 SyncRepoDependencies calls, got %d", len(store.syncCalls))
	}
}

// TestSyncRepoDeps_github_403_returns_nil verifies that when the GitHub API
// returns a 403, SyncRepoDeps logs the error and returns nil (error isolation).
func TestSyncRepoDeps_github_403_returns_nil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a 403 on every request (including GetTree).
		http.Error(w, `{"message":"API rate limit exceeded"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	client := newGitHubClientForTest(t, srv.URL)
	store := &mockDepStore{}
	svc := NewService(store)

	// The service must log the error and return nil — error isolation contract.
	err := svc.SyncRepoDeps(context.Background(), client, uuid.New(), "acme", "web-app", "main")
	if err != nil {
		t.Fatalf("SyncRepoDeps on 403: expected nil error (error isolation), got: %v", err)
	}

	// No store calls — sync aborted after the 403.
	if len(store.syncCalls) != 0 {
		t.Errorf("expected 0 SyncRepoDependencies calls on 403, got %d", len(store.syncCalls))
	}
}

// TestService_implements_DepSyncer is a compile-time check that *Service
// satisfies the DepSyncer interface.
func TestService_implements_DepSyncer(t *testing.T) {
	var _ DepSyncer = (*Service)(nil)
}

// Compile-time check: mockDepStore satisfies DepStore (defined in store_test.go).
// We also need a local check here so this test file compiles independently.
func TestMockDepStoreForService_satisfies_DepStore(t *testing.T) {
	var _ DepStore = &mockDepStore{}
}

// TestIsPackageJSON verifies that isPackageJSON correctly identifies valid
// package.json paths and rejects false positives like "my-package.json".
func TestIsPackageJSON(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"package.json", true},
		{"apps/web/package.json", true},
		{"packages/ui/package.json", true},
		{"node_modules/lodash/package.json", false},
		{"deep/node_modules/react/package.json", false},
		{"my-package.json", false},
		{"not-a-package.json", false},
		{"src/fake-package.json", false},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isPackageJSON(tt.path)
			if got != tt.want {
				t.Errorf("isPackageJSON(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

