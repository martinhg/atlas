package dependency

import (
	"context"
	"log/slog"
	"strings"

	gogithub "github.com/google/go-github/v69/github"
	"github.com/google/uuid"
	"github.com/nesbite/atlas/internal/ingest/parsers/cargo"
	"github.com/nesbite/atlas/internal/ingest/parsers/composer"
	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
	"github.com/nesbite/atlas/internal/ingest/parsers/gemfile"
	"github.com/nesbite/atlas/internal/ingest/parsers/gomod"
	"github.com/nesbite/atlas/internal/ingest/parsers/maven"
	"github.com/nesbite/atlas/internal/ingest/parsers/npm"
	"github.com/nesbite/atlas/internal/ingest/parsers/pip"
	"github.com/nesbite/atlas/internal/ingest/parsers/pyproject"
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

// ecosystemEntry pairs a manifest path matcher with its parse function for
// one dependency ecosystem. Adding a new ecosystem means appending one
// entry to the ecosystems slice below — SyncRepoDeps itself never changes.
type ecosystemEntry struct {
	Name      string
	MatchPath func(path string) bool
	Parse     func(data []byte, sourcePath string) []depmodel.ParsedDep
}

// ecosystems is the dispatch table SyncRepoDeps iterates over to discover
// and parse every supported manifest type found in a repo's tree.
var ecosystems = []ecosystemEntry{
	{Name: "npm", MatchPath: matchNpm, Parse: npm.ParsePackageJSON},
	{Name: "composer", MatchPath: matchComposer, Parse: composer.ParseComposerJSON},
	{Name: "gomod", MatchPath: matchGoMod, Parse: gomod.ParseGoMod},
	{Name: "pip", MatchPath: matchRequirementsTxt, Parse: pip.ParseRequirementsTxt},
	{Name: "cargo", MatchPath: matchCargoToml, Parse: cargo.ParseCargoToml},
	{Name: "maven", MatchPath: matchPomXML, Parse: maven.ParsePomXML},
	{Name: "pyproject", MatchPath: matchPyproject, Parse: pyproject.ParsePyprojectToml},
	{Name: "gemfile", MatchPath: matchGemfile, Parse: gemfile.ParseGemfile},
}

// SyncRepoDeps discovers every supported manifest file in the repo (see the
// ecosystems dispatch table), fetches and parses each one, and persists the
// combined results via the store. It satisfies the DepSyncer interface.
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

	// Step 2: Match every blob against the ecosystem dispatch table.
	type manifest struct {
		path  string
		parse func(data []byte, sourcePath string) []depmodel.ParsedDep
	}
	var manifests []manifest
	for _, entry := range tree.Entries {
		path := entry.GetPath()
		if entry.GetType() != "blob" {
			continue
		}
		for _, eco := range ecosystems {
			if eco.MatchPath(path) {
				manifests = append(manifests, manifest{path: path, parse: eco.Parse})
				break
			}
		}
	}

	if len(manifests) == 0 {
		return nil // no supported manifests — nothing to sync
	}

	// Step 3: Fetch and parse each manifest.
	var allDeps []depmodel.ParsedDep
	for _, m := range manifests {
		fileContent, _, _, err := ghClient.Repositories.GetContents(ctx, owner, repo, m.path, nil)
		if err != nil {
			slog.Error("dependency sync: failed to fetch manifest",
				"owner", owner, "repo", repo, "path", m.path, "error", err)
			continue // skip this file, continue with others
		}

		raw, err := fileContent.GetContent()
		if err != nil {
			slog.Error("dependency sync: failed to decode manifest content",
				"owner", owner, "repo", repo, "path", m.path, "error", err)
			continue
		}

		deps := m.parse([]byte(raw), m.path)
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

// matchNpm returns true when path points to a package.json file that is NOT
// inside a node_modules directory. The filename component must be exactly
// "package.json" — e.g. "my-package.json" does NOT match.
func matchNpm(path string) bool {
	if strings.Contains(path, "node_modules/") {
		return false
	}
	return path == "package.json" || strings.HasSuffix(path, "/package.json")
}

// matchComposer returns true when path points to a composer.json file that
// is NOT inside a vendor directory. The filename component must be exactly
// "composer.json" — e.g. "my-composer.json" does NOT match.
func matchComposer(path string) bool {
	if strings.Contains(path, "vendor/") {
		return false
	}
	return path == "composer.json" || strings.HasSuffix(path, "/composer.json")
}

// matchGoMod returns true when path points to a go.mod file that is NOT
// inside a vendor directory. The filename component must be exactly
// "go.mod" — e.g. "my-go.mod" does NOT match.
func matchGoMod(path string) bool {
	if strings.Contains(path, "vendor/") {
		return false
	}
	return path == "go.mod" || strings.HasSuffix(path, "/go.mod")
}

// matchRequirementsTxt returns true when path points to a requirements.txt
// file that is NOT inside a Python virtual environment or cache directory
// (.venv/, venv/, .tox/, __pycache__/). The filename component must be
// exactly "requirements.txt" — e.g. "my-requirements.txt" does NOT match.
func matchRequirementsTxt(path string) bool {
	for _, dir := range []string{".venv/", "venv/", ".tox/", "__pycache__/"} {
		if strings.Contains(path, dir) {
			return false
		}
	}
	return path == "requirements.txt" || strings.HasSuffix(path, "/requirements.txt")
}

// matchCargoToml returns true when path points to a Cargo.toml file that is
// NOT inside a target directory (Cargo's build output directory). The
// filename component must be exactly "Cargo.toml" — e.g. "my-Cargo.toml"
// does NOT match.
func matchCargoToml(path string) bool {
	if strings.Contains(path, "target/") {
		return false
	}
	return path == "Cargo.toml" || strings.HasSuffix(path, "/Cargo.toml")
}

// matchPomXML returns true when path points to a pom.xml file that is NOT
// inside a target directory (Maven's build output directory) or a .mvn
// directory (the Maven Wrapper directory). The filename component must be
// exactly "pom.xml" — e.g. "my-pom.xml" does NOT match.
func matchPomXML(path string) bool {
	if strings.Contains(path, "target/") || strings.Contains(path, ".mvn/") {
		return false
	}
	return path == "pom.xml" || strings.HasSuffix(path, "/pom.xml")
}

// matchPyproject returns true when path points to a pyproject.toml file
// that is NOT inside a Python virtual environment or cache directory
// (.venv/, venv/, .tox/, __pycache__/). The filename component must be
// exactly "pyproject.toml" — e.g. "my-pyproject.toml" does NOT match.
func matchPyproject(path string) bool {
	for _, dir := range []string{".venv/", "venv/", ".tox/", "__pycache__/"} {
		if strings.Contains(path, dir) {
			return false
		}
	}
	return path == "pyproject.toml" || strings.HasSuffix(path, "/pyproject.toml")
}

// matchGemfile returns true when path points to a Gemfile that is NOT
// inside a vendor directory (Bundler's `bundle install --deployment`
// install directory). The filename component must be exactly "Gemfile" —
// e.g. "my-Gemfile" and "Gemfile.lock" do NOT match.
func matchGemfile(path string) bool {
	if strings.Contains(path, "vendor/") {
		return false
	}
	return path == "Gemfile" || strings.HasSuffix(path, "/Gemfile")
}
