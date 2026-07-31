package scan

import (
	"encoding/json"

	"github.com/nesbite/atlas/internal/ingest/parsers/npm"
)

// NpmScanner discovers and parses package.json files within a directory
// tree, recursively, skipping any node_modules/ directory (matching the
// server-side discovery behavior in internal/dependency). It is a thin
// WalkConfig wrapper around WalkDirScanner — see walk.go for the shared
// implementation.
type NpmScanner struct{}

// Name identifies this scanner in warning messages.
func (NpmScanner) Name() string { return "npm" }

// Scan walks dir recursively looking for package.json files (excluding
// node_modules/), parses each one, and appends the resulting dependencies to
// report.Dependencies. A malformed or unreadable file produces a warning on
// report and is skipped — it never aborts the whole scan.
func (s NpmScanner) Scan(dir string, report *Report) error {
	scanner := NewWalkDirScanner(WalkConfig{
		Name:     s.Name(),
		FileName: "package.json",
		SkipDirs: []string{"node_modules"},
		Parse:    npm.ParsePackageJSON,
		Validate: json.Valid,
	})
	return scanner.Scan(dir, report)
}
