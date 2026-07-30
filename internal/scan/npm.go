package scan

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nesbite/atlas/internal/ingest/parsers/npm"
)

// NpmScanner discovers and parses package.json files within a directory
// tree, recursively, skipping any node_modules/ directory (matching the
// server-side discovery behavior in internal/dependency).
type NpmScanner struct{}

// Name identifies this scanner in warning messages.
func (NpmScanner) Name() string { return "npm" }

// Scan walks dir recursively looking for package.json files (excluding
// node_modules/), parses each one, and appends the resulting dependencies to
// report.Dependencies. A malformed or unreadable file produces a warning on
// report and is skipped — it never aborts the whole scan.
func (s NpmScanner) Scan(dir string, report *Report) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() != "package.json" {
			return nil
		}

		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			rel = path
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			addWarning(report, s.Name(), "failed to read %s: %v", rel, readErr)
			return nil
		}

		if !json.Valid(content) {
			addWarning(report, s.Name(), "invalid JSON in %s", rel)
			return nil
		}

		for _, dep := range npm.ParsePackageJSON(content, rel) {
			report.Dependencies = append(report.Dependencies, Dependency{
				Ecosystem:  dep.Ecosystem,
				Name:       dep.Name,
				Version:    dep.Version,
				DepType:    dep.DepType,
				SourceFile: dep.SourceFile,
			})
		}

		return nil
	})
}
