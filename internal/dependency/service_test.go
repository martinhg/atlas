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
	SHA       string      `json:"sha"`
	Truncated bool        `json:"truncated"`
	Tree      []treeEntry `json:"tree"`
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

// TestMatchNpm verifies that matchNpm correctly identifies valid
// package.json paths and rejects false positives like "my-package.json".
func TestMatchNpm(t *testing.T) {
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
			got := matchNpm(tt.path)
			if got != tt.want {
				t.Errorf("matchNpm(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestMatchComposer verifies that matchComposer correctly identifies valid
// composer.json paths and excludes anything under vendor/.
func TestMatchComposer(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"composer.json", true},
		{"modules/billing/composer.json", true},
		{"vendor/monolog/monolog/composer.json", false},
		{"deep/vendor/acme/lib/composer.json", false},
		{"my-composer.json", false},
		{"not-a-composer.json", false},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := matchComposer(tt.path)
			if got != tt.want {
				t.Errorf("matchComposer(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestMatchGoMod verifies that matchGoMod correctly identifies valid go.mod
// paths and excludes anything under vendor/.
func TestMatchGoMod(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"go.mod", true},
		{"modules/billing/go.mod", true},
		{"vendor/github.com/foo/bar/go.mod", false},
		{"deep/vendor/acme/lib/go.mod", false},
		{"my-go.mod", false},
		{"not-a-go.mod", false},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := matchGoMod(tt.path)
			if got != tt.want {
				t.Errorf("matchGoMod(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestSyncRepoDeps_discovers_go_mod verifies that SyncRepoDeps discovers,
// fetches, and parses a go.mod alongside a package.json in the same tree —
// proving the ecosystem dispatch table handles the Go ecosystem too.
func TestSyncRepoDeps_discovers_go_mod(t *testing.T) {
	var fetchedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/git/trees/") {
			resp := treeResponse{
				SHA:       "abc123",
				Truncated: false,
				Tree: []treeEntry{
					{Path: "package.json", Type: "blob"},
					{Path: "go.mod", Type: "blob"},
					{Path: "vendor/github.com/foo/bar/go.mod", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/contents/") {
			parts := strings.SplitN(r.URL.Path, "/contents/", 2)
			path := ""
			if len(parts) == 2 {
				path = parts[1]
				fetchedPaths = append(fetchedPaths, path)
			}

			var content string
			switch path {
			case "go.mod":
				content = "module acme\n\ngo 1.26\n\nrequire github.com/google/uuid v1.6.0\n"
			default:
				content = `{"dependencies":{"react":"^18.0.0"}}`
			}
			encoded := base64.StdEncoding.EncodeToString([]byte(content))
			resp := map[string]string{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
				"name":     path,
				"path":     path,
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

	for _, p := range fetchedPaths {
		if strings.Contains(p, "vendor/") {
			t.Errorf("fetched vendor/ path: %q — should have been excluded", p)
		}
	}
	if len(fetchedPaths) != 2 {
		t.Fatalf("expected 2 content fetches (package.json + go.mod), got %d: %v", len(fetchedPaths), fetchedPaths)
	}

	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoDependencies call, got %d", len(store.syncCalls))
	}
	deps := store.syncCalls[0].deps
	if len(deps) != 2 {
		t.Fatalf("expected 2 synced deps, got %d: %+v", len(deps), deps)
	}
	byEcosystem := map[string]bool{}
	for _, d := range deps {
		byEcosystem[d.Ecosystem] = true
	}
	if !byEcosystem["npm"] || !byEcosystem["Go"] {
		t.Errorf("expected npm and Go ecosystems in synced deps, got %+v", deps)
	}
}

// TestMatchRequirementsTxt verifies that matchRequirementsTxt correctly
// identifies valid requirements.txt paths and excludes anything under
// Python virtual environment or cache directories.
func TestMatchRequirementsTxt(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"requirements.txt", true},
		{"services/worker/requirements.txt", true},
		{".venv/lib/requirements.txt", false},
		{"venv/lib/requirements.txt", false},
		{".tox/py311/requirements.txt", false},
		{"__pycache__/requirements.txt", false},
		{"deep/.venv/lib/requirements.txt", false},
		{"my-requirements.txt", false},
		{"not-a-requirements.txt", false},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := matchRequirementsTxt(tt.path)
			if got != tt.want {
				t.Errorf("matchRequirementsTxt(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestSyncRepoDeps_discovers_requirements_txt verifies that SyncRepoDeps
// discovers, fetches, and parses a requirements.txt alongside a
// package.json in the same tree — proving the ecosystem dispatch table
// handles the PyPI ecosystem too.
func TestSyncRepoDeps_discovers_requirements_txt(t *testing.T) {
	var fetchedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/git/trees/") {
			resp := treeResponse{
				SHA:       "abc123",
				Truncated: false,
				Tree: []treeEntry{
					{Path: "package.json", Type: "blob"},
					{Path: "requirements.txt", Type: "blob"},
					{Path: ".venv/lib/requirements.txt", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/contents/") {
			parts := strings.SplitN(r.URL.Path, "/contents/", 2)
			path := ""
			if len(parts) == 2 {
				path = parts[1]
				fetchedPaths = append(fetchedPaths, path)
			}

			var content string
			switch path {
			case "requirements.txt":
				content = "requests==2.28.1\n"
			default:
				content = `{"dependencies":{"react":"^18.0.0"}}`
			}
			encoded := base64.StdEncoding.EncodeToString([]byte(content))
			resp := map[string]string{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
				"name":     path,
				"path":     path,
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

	for _, p := range fetchedPaths {
		if strings.Contains(p, ".venv/") {
			t.Errorf("fetched .venv/ path: %q — should have been excluded", p)
		}
	}
	if len(fetchedPaths) != 2 {
		t.Fatalf("expected 2 content fetches (package.json + requirements.txt), got %d: %v", len(fetchedPaths), fetchedPaths)
	}

	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoDependencies call, got %d", len(store.syncCalls))
	}
	deps := store.syncCalls[0].deps
	if len(deps) != 2 {
		t.Fatalf("expected 2 synced deps, got %d: %+v", len(deps), deps)
	}
	byEcosystem := map[string]bool{}
	for _, d := range deps {
		byEcosystem[d.Ecosystem] = true
	}
	if !byEcosystem["npm"] || !byEcosystem["PyPI"] {
		t.Errorf("expected npm and PyPI ecosystems in synced deps, got %+v", deps)
	}
}

// TestSyncRepoDeps_discovers_composer_json verifies that SyncRepoDeps
// discovers, fetches, and parses a composer.json alongside a package.json
// in the same tree — proving the ecosystem dispatch table handles multiple
// ecosystems in a single sync.
func TestSyncRepoDeps_discovers_composer_json(t *testing.T) {
	var fetchedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/git/trees/") {
			resp := treeResponse{
				SHA:       "abc123",
				Truncated: false,
				Tree: []treeEntry{
					{Path: "package.json", Type: "blob"},
					{Path: "composer.json", Type: "blob"},
					{Path: "vendor/monolog/monolog/composer.json", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/contents/") {
			parts := strings.SplitN(r.URL.Path, "/contents/", 2)
			path := ""
			if len(parts) == 2 {
				path = parts[1]
				fetchedPaths = append(fetchedPaths, path)
			}

			var content string
			switch path {
			case "composer.json":
				content = `{"require":{"monolog/monolog":"^2.0"}}`
			default:
				content = `{"dependencies":{"react":"^18.0.0"}}`
			}
			encoded := base64.StdEncoding.EncodeToString([]byte(content))
			resp := map[string]string{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
				"name":     path,
				"path":     path,
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

	for _, p := range fetchedPaths {
		if strings.Contains(p, "vendor/") {
			t.Errorf("fetched vendor/ path: %q — should have been excluded", p)
		}
	}
	if len(fetchedPaths) != 2 {
		t.Fatalf("expected 2 content fetches (package.json + composer.json), got %d: %v", len(fetchedPaths), fetchedPaths)
	}

	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoDependencies call, got %d", len(store.syncCalls))
	}
	deps := store.syncCalls[0].deps
	if len(deps) != 2 {
		t.Fatalf("expected 2 synced deps, got %d: %+v", len(deps), deps)
	}
	byEcosystem := map[string]bool{}
	for _, d := range deps {
		byEcosystem[d.Ecosystem] = true
	}
	if !byEcosystem["npm"] || !byEcosystem["Packagist"] {
		t.Errorf("expected npm and Packagist ecosystems in synced deps, got %+v", deps)
	}
}

// TestMatchCargoToml verifies that matchCargoToml correctly identifies
// valid Cargo.toml paths and excludes anything under a target/ directory
// (Cargo's build output directory).
func TestMatchCargoToml(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"Cargo.toml", true},
		{"crates/core/Cargo.toml", true},
		{"target/debug/Cargo.toml", false},
		{"deep/target/debug/Cargo.toml", false},
		{"my-Cargo.toml", false},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := matchCargoToml(tt.path)
			if got != tt.want {
				t.Errorf("matchCargoToml(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestSyncRepoDeps_discovers_cargo_toml verifies that SyncRepoDeps
// discovers, fetches, and parses a Cargo.toml alongside a package.json in
// the same tree — proving the ecosystem dispatch table handles the
// crates.io ecosystem too.
func TestSyncRepoDeps_discovers_cargo_toml(t *testing.T) {
	var fetchedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/git/trees/") {
			resp := treeResponse{
				SHA:       "abc123",
				Truncated: false,
				Tree: []treeEntry{
					{Path: "package.json", Type: "blob"},
					{Path: "Cargo.toml", Type: "blob"},
					{Path: "target/debug/Cargo.toml", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/contents/") {
			parts := strings.SplitN(r.URL.Path, "/contents/", 2)
			path := ""
			if len(parts) == 2 {
				path = parts[1]
				fetchedPaths = append(fetchedPaths, path)
			}

			var content string
			switch path {
			case "Cargo.toml":
				content = "[dependencies]\nserde = \"1.0\"\n"
			default:
				content = `{"dependencies":{"react":"^18.0.0"}}`
			}
			encoded := base64.StdEncoding.EncodeToString([]byte(content))
			resp := map[string]string{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
				"name":     path,
				"path":     path,
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

	for _, p := range fetchedPaths {
		if strings.Contains(p, "target/") {
			t.Errorf("fetched target/ path: %q — should have been excluded", p)
		}
	}
	if len(fetchedPaths) != 2 {
		t.Fatalf("expected 2 content fetches (package.json + Cargo.toml), got %d: %v", len(fetchedPaths), fetchedPaths)
	}

	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoDependencies call, got %d", len(store.syncCalls))
	}
	deps := store.syncCalls[0].deps
	if len(deps) != 2 {
		t.Fatalf("expected 2 synced deps, got %d: %+v", len(deps), deps)
	}
	byEcosystem := map[string]bool{}
	for _, d := range deps {
		byEcosystem[d.Ecosystem] = true
	}
	if !byEcosystem["npm"] || !byEcosystem["crates.io"] {
		t.Errorf("expected npm and crates.io ecosystems in synced deps, got %+v", deps)
	}
}

// TestMatchPomXML verifies that matchPomXML correctly identifies valid
// pom.xml paths and excludes anything under target/ (Maven's build output
// directory) or .mvn/ (the Maven Wrapper directory).
func TestMatchPomXML(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"pom.xml", true},
		{"modules/billing/pom.xml", true},
		{"target/classes/pom.xml", false},
		{"deep/target/classes/pom.xml", false},
		{".mvn/wrapper/pom.xml", false},
		{"my-pom.xml", false},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := matchPomXML(tt.path)
			if got != tt.want {
				t.Errorf("matchPomXML(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestSyncRepoDeps_discovers_pom_xml verifies that SyncRepoDeps discovers,
// fetches, and parses a pom.xml alongside a package.json in the same tree —
// proving the ecosystem dispatch table handles the Maven ecosystem too.
func TestSyncRepoDeps_discovers_pom_xml(t *testing.T) {
	var fetchedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/git/trees/") {
			resp := treeResponse{
				SHA:       "abc123",
				Truncated: false,
				Tree: []treeEntry{
					{Path: "package.json", Type: "blob"},
					{Path: "pom.xml", Type: "blob"},
					{Path: "target/classes/pom.xml", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/contents/") {
			parts := strings.SplitN(r.URL.Path, "/contents/", 2)
			path := ""
			if len(parts) == 2 {
				path = parts[1]
				fetchedPaths = append(fetchedPaths, path)
			}

			var content string
			switch path {
			case "pom.xml":
				content = "<project><dependencies><dependency>" +
					"<groupId>com.google.guava</groupId><artifactId>guava</artifactId><version>31.1-jre</version>" +
					"</dependency></dependencies></project>"
			default:
				content = `{"dependencies":{"react":"^18.0.0"}}`
			}
			encoded := base64.StdEncoding.EncodeToString([]byte(content))
			resp := map[string]string{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
				"name":     path,
				"path":     path,
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

	for _, p := range fetchedPaths {
		if strings.Contains(p, "target/") {
			t.Errorf("fetched target/ path: %q — should have been excluded", p)
		}
	}
	if len(fetchedPaths) != 2 {
		t.Fatalf("expected 2 content fetches (package.json + pom.xml), got %d: %v", len(fetchedPaths), fetchedPaths)
	}

	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoDependencies call, got %d", len(store.syncCalls))
	}
	deps := store.syncCalls[0].deps
	if len(deps) != 2 {
		t.Fatalf("expected 2 synced deps, got %d: %+v", len(deps), deps)
	}
	byEcosystem := map[string]bool{}
	for _, d := range deps {
		byEcosystem[d.Ecosystem] = true
	}
	if !byEcosystem["npm"] || !byEcosystem["Maven"] {
		t.Errorf("expected npm and Maven ecosystems in synced deps, got %+v", deps)
	}
}

// TestMatchPyproject verifies that matchPyproject correctly identifies
// valid pyproject.toml paths and excludes anything under a Python virtual
// environment or cache directory (.venv/, venv/, .tox/, __pycache__/).
func TestMatchPyproject(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"pyproject.toml", true},
		{"services/api/pyproject.toml", true},
		{".venv/lib/pyproject.toml", false},
		{"venv/lib/pyproject.toml", false},
		{".tox/py311/pyproject.toml", false},
		{"__pycache__/pyproject.toml", false},
		{"my-pyproject.toml", false},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := matchPyproject(tt.path)
			if got != tt.want {
				t.Errorf("matchPyproject(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestSyncRepoDeps_discovers_pyproject_toml verifies that SyncRepoDeps
// discovers, fetches, and parses a pyproject.toml alongside a package.json
// in the same tree — proving the ecosystem dispatch table handles the
// pyproject (PyPI) ecosystem too.
func TestSyncRepoDeps_discovers_pyproject_toml(t *testing.T) {
	var fetchedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/git/trees/") {
			resp := treeResponse{
				SHA:       "abc123",
				Truncated: false,
				Tree: []treeEntry{
					{Path: "package.json", Type: "blob"},
					{Path: "pyproject.toml", Type: "blob"},
					{Path: ".venv/lib/pyproject.toml", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/contents/") {
			parts := strings.SplitN(r.URL.Path, "/contents/", 2)
			path := ""
			if len(parts) == 2 {
				path = parts[1]
				fetchedPaths = append(fetchedPaths, path)
			}

			var content string
			switch path {
			case "pyproject.toml":
				content = "[project]\nname = \"acme-app\"\ndependencies = [\"requests>=2.28\"]\n"
			default:
				content = `{"dependencies":{"react":"^18.0.0"}}`
			}
			encoded := base64.StdEncoding.EncodeToString([]byte(content))
			resp := map[string]string{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
				"name":     path,
				"path":     path,
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

	for _, p := range fetchedPaths {
		if strings.Contains(p, ".venv/") {
			t.Errorf("fetched .venv/ path: %q — should have been excluded", p)
		}
	}
	if len(fetchedPaths) != 2 {
		t.Fatalf("expected 2 content fetches (package.json + pyproject.toml), got %d: %v", len(fetchedPaths), fetchedPaths)
	}

	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoDependencies call, got %d", len(store.syncCalls))
	}
	deps := store.syncCalls[0].deps
	if len(deps) != 2 {
		t.Fatalf("expected 2 synced deps, got %d: %+v", len(deps), deps)
	}
	byEcosystem := map[string]bool{}
	for _, d := range deps {
		byEcosystem[d.Ecosystem] = true
	}
	if !byEcosystem["npm"] || !byEcosystem["PyPI"] {
		t.Errorf("expected npm and PyPI ecosystems in synced deps, got %+v", deps)
	}
}

// TestMatchGemfile verifies that matchGemfile correctly identifies valid
// Gemfile paths and excludes anything under a vendor/ directory (Bundler's
// deployment install directory).
func TestMatchGemfile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"Gemfile", true},
		{"services/api/Gemfile", true},
		{"vendor/bundle/Gemfile", false},
		{"deep/vendor/bundle/Gemfile", false},
		{"my-Gemfile", false},
		{"Gemfile.lock", false},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := matchGemfile(tt.path)
			if got != tt.want {
				t.Errorf("matchGemfile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestSyncRepoDeps_discovers_gemfile verifies that SyncRepoDeps discovers,
// fetches, and parses a Gemfile alongside a package.json in the same tree —
// proving the ecosystem dispatch table handles the RubyGems ecosystem too.
func TestSyncRepoDeps_discovers_gemfile(t *testing.T) {
	var fetchedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/git/trees/") {
			resp := treeResponse{
				SHA:       "abc123",
				Truncated: false,
				Tree: []treeEntry{
					{Path: "package.json", Type: "blob"},
					{Path: "Gemfile", Type: "blob"},
					{Path: "vendor/bundle/Gemfile", Type: "blob"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/contents/") {
			parts := strings.SplitN(r.URL.Path, "/contents/", 2)
			path := ""
			if len(parts) == 2 {
				path = parts[1]
				fetchedPaths = append(fetchedPaths, path)
			}

			var content string
			switch path {
			case "Gemfile":
				content = "gem 'rails', '~> 7.0'\n"
			default:
				content = `{"dependencies":{"react":"^18.0.0"}}`
			}
			encoded := base64.StdEncoding.EncodeToString([]byte(content))
			resp := map[string]string{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
				"name":     path,
				"path":     path,
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

	for _, p := range fetchedPaths {
		if strings.Contains(p, "vendor/") {
			t.Errorf("fetched vendor/ path: %q — should have been excluded", p)
		}
	}
	if len(fetchedPaths) != 2 {
		t.Fatalf("expected 2 content fetches (package.json + Gemfile), got %d: %v", len(fetchedPaths), fetchedPaths)
	}

	if len(store.syncCalls) != 1 {
		t.Fatalf("expected 1 SyncRepoDependencies call, got %d", len(store.syncCalls))
	}
	deps := store.syncCalls[0].deps
	if len(deps) != 2 {
		t.Fatalf("expected 2 synced deps, got %d: %+v", len(deps), deps)
	}
	byEcosystem := map[string]bool{}
	for _, d := range deps {
		byEcosystem[d.Ecosystem] = true
	}
	if !byEcosystem["npm"] || !byEcosystem["RubyGems"] {
		t.Errorf("expected npm and RubyGems ecosystems in synced deps, got %+v", deps)
	}
}
