package npm

import (
	"encoding/json"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// ParsedDep is an alias for the shared depmodel.ParsedDep type. It is kept
// here so existing callers referencing npm.ParsedDep continue to compile
// unchanged while the codebase migrates to depmodel.ParsedDep directly.
type ParsedDep = depmodel.ParsedDep

// packageJSON is the minimal shape of a package.json file relevant to dependency parsing.
type packageJSON struct {
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

// ParsePackageJSON parses the raw content of a package.json file and returns
// a slice of ParsedDep entries. It is a pure function with no side effects.
// Invalid JSON or missing dependency sections result in an empty (non-nil) slice.
func ParsePackageJSON(content []byte, sourcePath string) []ParsedDep {
	var pkg packageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return []ParsedDep{}
	}

	result := make([]ParsedDep, 0)

	for name, version := range pkg.Dependencies {
		if name == "" {
			continue
		}
		result = append(result, ParsedDep{
			Ecosystem:  depmodel.EcosystemNpm,
			Name:       name,
			Version:    version,
			DepType:    depmodel.Direct,
			SourceFile: sourcePath,
		})
	}
	for name, version := range pkg.DevDependencies {
		if name == "" {
			continue
		}
		result = append(result, ParsedDep{
			Ecosystem:  depmodel.EcosystemNpm,
			Name:       name,
			Version:    version,
			DepType:    depmodel.Dev,
			SourceFile: sourcePath,
		})
	}
	for name, version := range pkg.PeerDependencies {
		if name == "" {
			continue
		}
		result = append(result, ParsedDep{
			Ecosystem:  depmodel.EcosystemNpm,
			Name:       name,
			Version:    version,
			DepType:    depmodel.Peer,
			SourceFile: sourcePath,
		})
	}
	for name, version := range pkg.OptionalDependencies {
		if name == "" {
			continue
		}
		result = append(result, ParsedDep{
			Ecosystem:  depmodel.EcosystemNpm,
			Name:       name,
			Version:    version,
			DepType:    depmodel.Optional,
			SourceFile: sourcePath,
		})
	}

	return result
}
