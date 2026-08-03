package scan

import (
	"github.com/BurntSushi/toml"

	"github.com/nesbite/atlas/internal/ingest/parsers/cargo"
)

// NewCargoScanner returns an EcosystemScanner that discovers and parses
// Cargo.toml files within a directory tree, recursively, skipping any
// target/ directory (Cargo's build output directory).
func NewCargoScanner() WalkDirScanner {
	return NewWalkDirScanner(WalkConfig{
		Name:     "cargo",
		FileName: "Cargo.toml",
		SkipDirs: []string{"target"},
		Parse:    cargo.ParseCargoToml,
		Validate: isValidTOML,
	})
}

// isValidTOML reports whether data parses as well-formed TOML.
func isValidTOML(data []byte) bool {
	var v map[string]interface{}
	_, err := toml.Decode(string(data), &v)
	return err == nil
}
