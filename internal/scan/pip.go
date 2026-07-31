package scan

import (
	"github.com/nesbite/atlas/internal/ingest/parsers/pip"
)

// NewPipScanner returns an EcosystemScanner that discovers and parses
// requirements.txt files within a directory tree, recursively, skipping
// Python virtual environment and cache directories (.venv/, venv/, .tox/,
// __pycache__/).
func NewPipScanner() WalkDirScanner {
	return NewWalkDirScanner(WalkConfig{
		Name:     "pip",
		FileName: "requirements.txt",
		SkipDirs: []string{".venv", "venv", "__pycache__", ".tox"},
		Parse:    pip.ParseRequirementsTxt,
	})
}
