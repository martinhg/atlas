// Package composer parses composer.json manifests (Packagist ecosystem).
package composer

import (
	"encoding/json"
	"strings"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// composerJSON is the minimal shape of a composer.json file relevant to
// dependency parsing.
type composerJSON struct {
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}

// ParseComposerJSON parses the raw content of a composer.json file and
// returns a slice of ParsedDep entries. It is a pure function with no side
// effects. Invalid JSON or missing require sections result in an empty
// (non-nil) slice.
//
// Platform requirements ("php" and "ext-*" entries) are not real packages —
// they describe the PHP runtime and its extensions — so they are skipped.
func ParseComposerJSON(data []byte, sourcePath string) []depmodel.ParsedDep {
	var pkg composerJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return []depmodel.ParsedDep{}
	}

	result := make([]depmodel.ParsedDep, 0)

	for name, version := range pkg.Require {
		if name == "" || isPlatformRequirement(name) {
			continue
		}
		result = append(result, depmodel.ParsedDep{
			Ecosystem:  depmodel.EcosystemPackagist,
			Name:       name,
			Version:    version,
			DepType:    depmodel.Direct,
			SourceFile: sourcePath,
		})
	}
	for name, version := range pkg.RequireDev {
		if name == "" || isPlatformRequirement(name) {
			continue
		}
		result = append(result, depmodel.ParsedDep{
			Ecosystem:  depmodel.EcosystemPackagist,
			Name:       name,
			Version:    version,
			DepType:    depmodel.Dev,
			SourceFile: sourcePath,
		})
	}

	return result
}

// isPlatformRequirement reports whether name is a Composer platform
// requirement (the PHP runtime itself, or a PHP extension) rather than a
// real Packagist package.
func isPlatformRequirement(name string) bool {
	return name == "php" || strings.HasPrefix(name, "ext-")
}
