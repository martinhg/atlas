package scan

import (
	"encoding/json"

	"github.com/nesbite/atlas/internal/ingest/parsers/composer"
)

// NewComposerScanner returns an EcosystemScanner that discovers and parses
// composer.json files within a directory tree, recursively, skipping any
// vendor/ directory (Composer's dependency install directory).
func NewComposerScanner() WalkDirScanner {
	return NewWalkDirScanner(WalkConfig{
		Name:     "composer",
		FileName: "composer.json",
		SkipDirs: []string{"vendor"},
		Parse:    composer.ParseComposerJSON,
		Validate: json.Valid,
	})
}
