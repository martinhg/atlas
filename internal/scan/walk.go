package scan

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// WalkConfig configures a WalkDirScanner for a single ecosystem.
type WalkConfig struct {
	// Name identifies this scanner in warning messages, e.g. "composer".
	Name string
	// FileName is the exact base filename this scanner looks for, e.g.
	// "composer.json". Only an exact match is considered — no globbing.
	FileName string
	// SkipDirs lists directory names to exclude from the walk entirely
	// (the walk never descends into them), e.g. []string{"vendor"}.
	SkipDirs []string
	// Parse converts the raw content of a matched file into ParsedDep
	// entries. Required.
	Parse func(data []byte, sourcePath string) []depmodel.ParsedDep
	// Validate, if set, is called on a matched file's raw content before
	// Parse. A false result produces a warning on the report and skips
	// Parse for that file. If nil, no pre-validation is performed.
	Validate func(data []byte) bool
}

// WalkDirScanner is a generic EcosystemScanner built from a WalkConfig. It
// implements the shared "walk a directory tree, match a filename, skip
// excluded directories, parse matches" pattern used by every per-ecosystem
// scanner in this package, so a new ecosystem scanner requires only a
// WalkConfig, not a hand-written filepath.WalkDir loop.
type WalkDirScanner struct {
	cfg WalkConfig
}

// NewWalkDirScanner constructs a WalkDirScanner from cfg.
func NewWalkDirScanner(cfg WalkConfig) WalkDirScanner {
	return WalkDirScanner{cfg: cfg}
}

// Name identifies this scanner in warning messages.
func (s WalkDirScanner) Name() string { return s.cfg.Name }

// Scan walks dir recursively looking for files named cfg.FileName,
// excluding any directory named in cfg.SkipDirs, parses each match with
// cfg.Parse, and appends the resulting dependencies to report.Dependencies.
// A malformed or unreadable file produces a warning on report and is
// skipped — it never aborts the whole scan.
func (s WalkDirScanner) Scan(dir string, report *Report) error {
	skip := make(map[string]bool, len(s.cfg.SkipDirs))
	for _, d := range s.cfg.SkipDirs {
		skip[d] = true
	}

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if skip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() != s.cfg.FileName {
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

		if s.cfg.Validate != nil && !s.cfg.Validate(content) {
			addWarning(report, s.Name(), "invalid content in %s", rel)
			return nil
		}

		for _, dep := range s.cfg.Parse(content, rel) {
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
