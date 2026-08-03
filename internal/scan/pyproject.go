package scan

import (
	"github.com/nesbite/atlas/internal/ingest/parsers/pyproject"
)

// NewPyprojectScanner returns an EcosystemScanner that discovers and parses
// pyproject.toml files within a directory tree, recursively, skipping
// Python virtual environment and cache directories (.venv/, venv/, .tox/,
// __pycache__/). See pyproject.ParsePyprojectToml for the PEP 621 parsing
// rules (Poetry-only manifests without a [project] table yield no deps).
func NewPyprojectScanner() WalkDirScanner {
	return NewWalkDirScanner(WalkConfig{
		Name:     "pyproject",
		FileName: "pyproject.toml",
		SkipDirs: []string{".venv", "venv", "__pycache__", ".tox"},
		Parse:    pyproject.ParsePyprojectToml,
		Validate: isValidTOML,
	})
}
