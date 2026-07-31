package scan

import (
	"golang.org/x/mod/modfile"

	"github.com/nesbite/atlas/internal/ingest/parsers/gomod"
)

// NewGoModScanner returns an EcosystemScanner that discovers and parses
// go.mod files within a directory tree, recursively, skipping any vendor/
// directory (Go's vendored dependency directory).
func NewGoModScanner() WalkDirScanner {
	return NewWalkDirScanner(WalkConfig{
		Name:     "gomod",
		FileName: "go.mod",
		SkipDirs: []string{"vendor"},
		Parse:    gomod.ParseGoMod,
		Validate: isValidGoMod,
	})
}

// isValidGoMod reports whether data parses as a well-formed go.mod file.
func isValidGoMod(data []byte) bool {
	_, err := modfile.Parse("go.mod", data, nil)
	return err == nil
}
