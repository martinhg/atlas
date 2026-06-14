package parser

import "encoding/json"

// ParsedDep represents a single dependency entry parsed from a package.json file.
type ParsedDep struct {
	Ecosystem  string
	Name       string
	Version    string
	DepType    string // "dep", "devDep", "peer", "optional"
	SourceFile string
}

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
			Ecosystem:  "npm",
			Name:       name,
			Version:    version,
			DepType:    "dep",
			SourceFile: sourcePath,
		})
	}
	for name, version := range pkg.DevDependencies {
		if name == "" {
			continue
		}
		result = append(result, ParsedDep{
			Ecosystem:  "npm",
			Name:       name,
			Version:    version,
			DepType:    "devDep",
			SourceFile: sourcePath,
		})
	}
	for name, version := range pkg.PeerDependencies {
		if name == "" {
			continue
		}
		result = append(result, ParsedDep{
			Ecosystem:  "npm",
			Name:       name,
			Version:    version,
			DepType:    "peer",
			SourceFile: sourcePath,
		})
	}
	for name, version := range pkg.OptionalDependencies {
		if name == "" {
			continue
		}
		result = append(result, ParsedDep{
			Ecosystem:  "npm",
			Name:       name,
			Version:    version,
			DepType:    "optional",
			SourceFile: sourcePath,
		})
	}

	return result
}
