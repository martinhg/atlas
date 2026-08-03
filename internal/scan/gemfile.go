package scan

import (
	"github.com/nesbite/atlas/internal/ingest/parsers/gemfile"
)

// NewGemfileScanner returns an EcosystemScanner that discovers and parses
// Gemfile files within a directory tree, recursively, skipping any vendor/
// directory (the directory Bundler installs gems into with `bundle install
// --deployment`).
func NewGemfileScanner() WalkDirScanner {
	return NewWalkDirScanner(WalkConfig{
		Name:     "gemfile",
		FileName: "Gemfile",
		SkipDirs: []string{"vendor"},
		Parse:    gemfile.ParseGemfile,
	})
}
