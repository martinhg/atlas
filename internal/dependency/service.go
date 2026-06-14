package dependency

import (
	"context"
	"log/slog"
	"strings"

	gogithub "github.com/google/go-github/v69/github"
	"github.com/google/uuid"
	"github.com/nesbite/atlas/internal/dependency/parser"
)

// DepSyncer is the interface consumed by org.syncRepos. It decouples the org
// package from dependency internals — the org package only calls this method.
type DepSyncer interface {
	SyncRepoDeps(ctx context.Context, ghClient *gogithub.Client, repoID uuid.UUID, owner, repo, branch string) error
}

// Service implements DepSyncer. It orchestrates tree discovery, content fetch,
// parsing, and storage for a single repository.
type Service struct {
	store DepStore
}

// NewService constructs a Service backed by the given DepStore.
func NewService(store DepStore) *Service {
	return &Service{store: store}
}

// SyncRepoDeps discovers all package.json files in the repo (excluding
// node_modules/), fetches and parses each one, and persists the results via
// the store. It satisfies the DepSyncer interface.
//
// Error policy: GitHub API errors are logged and the function returns nil
// so that sync of other repos continues uninterrupted.
func (s *Service) SyncRepoDeps(ctx context.Context, ghClient *gogithub.Client, repoID uuid.UUID, owner, repo, branch string) error {
	// Step 1: Fetch the recursive tree for the default branch.
	tree, _, err := ghClient.Git.GetTree(ctx, owner, repo, branch, true)
	if err != nil {
		slog.Error("dependency sync: failed to get tree",
			"owner", owner, "repo", repo, "branch", branch, "error", err)
		return nil // error isolation — do not propagate
	}

	// Log a warning when the tree is truncated but continue with the partial result.
	if tree.GetTruncated() {
		slog.Warn("dependency sync: tree response is truncated, processing partial tree",
			"owner", owner, "repo", repo)
	}

	// Step 2: Collect package.json paths, excluding node_modules/.
	var pkgPaths []string
	for _, entry := range tree.Entries {
		path := entry.GetPath()
		if entry.GetType() != "blob" {
			continue
		}
		if !isPackageJSON(path) {
			continue
		}
		pkgPaths = append(pkgPaths, path)
	}

	if len(pkgPaths) == 0 {
		return nil // no package.json — nothing to sync
	}

	// Step 3: Fetch and parse each package.json.
	var allDeps []parser.ParsedDep
	for _, path := range pkgPaths {
		fileContent, _, _, err := ghClient.Repositories.GetContents(ctx, owner, repo, path, nil)
		if err != nil {
			slog.Error("dependency sync: failed to fetch package.json",
				"owner", owner, "repo", repo, "path", path, "error", err)
			continue // skip this file, continue with others
		}

		raw, err := fileContent.GetContent()
		if err != nil {
			slog.Error("dependency sync: failed to decode package.json content",
				"owner", owner, "repo", repo, "path", path, "error", err)
			continue
		}

		deps := parser.ParsePackageJSON([]byte(raw), path)
		allDeps = append(allDeps, deps...)
	}

	// Step 4: Persist via store (delete-then-insert in transaction).
	if err := s.store.SyncRepoDependencies(ctx, repoID, allDeps); err != nil {
		slog.Error("dependency sync: failed to sync repo dependencies",
			"repo_id", repoID, "error", err)
		return nil // error isolation
	}

	return nil
}

// isPackageJSON returns true when path points to a package.json file
// that is NOT inside a node_modules directory. The filename component
// must be exactly "package.json" — e.g. "my-package.json" does NOT match.
func isPackageJSON(path string) bool {
	if strings.Contains(path, "node_modules/") {
		return false
	}
	return path == "package.json" || strings.HasSuffix(path, "/package.json")
}
