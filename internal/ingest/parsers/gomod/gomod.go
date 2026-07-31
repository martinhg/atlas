// Package gomod parses go.mod manifests (Go ecosystem).
package gomod

import (
	"golang.org/x/mod/modfile"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// ParseGoMod parses the raw content of a go.mod file and returns a slice of
// ParsedDep entries. It is a pure function with no side effects. It only
// looks at require directives — replace, exclude, and retract directives do
// not describe dependencies and are ignored. Malformed go.mod content
// results in an empty (non-nil) slice.
//
// A require entry carrying a "// indirect" comment is classified as
// depmodel.Indirect; every other require entry is depmodel.Direct.
func ParseGoMod(data []byte, sourcePath string) []depmodel.ParsedDep {
	f, err := modfile.Parse(sourcePath, data, nil)
	if err != nil {
		return []depmodel.ParsedDep{}
	}

	result := make([]depmodel.ParsedDep, 0, len(f.Require))

	for _, req := range f.Require {
		if req.Mod.Path == "" {
			continue
		}
		depType := depmodel.Direct
		if req.Indirect {
			depType = depmodel.Indirect
		}
		result = append(result, depmodel.ParsedDep{
			Ecosystem:  depmodel.EcosystemGo,
			Name:       req.Mod.Path,
			Version:    req.Mod.Version,
			DepType:    depType,
			SourceFile: sourcePath,
		})
	}

	return result
}
