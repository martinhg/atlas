// Package cargo parses Cargo.toml manifests (crates.io ecosystem).
package cargo

import (
	"github.com/BurntSushi/toml"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// cargoManifest is the minimal shape of a Cargo.toml file relevant to
// dependency parsing. Each dependency section maps a package name to a raw
// TOML value that may be either a bare string or a table — decoded here as
// interface{} and resolved by extractVersion.
type cargoManifest struct {
	Dependencies      map[string]interface{} `toml:"dependencies"`
	DevDependencies   map[string]interface{} `toml:"dev-dependencies"`
	BuildDependencies map[string]interface{} `toml:"build-dependencies"`
}

// ParseCargoToml parses the raw content of a Cargo.toml file and returns a
// slice of ParsedDep entries. It is a pure function with no side effects.
// Malformed TOML results in an empty (non-nil) slice.
func ParseCargoToml(data []byte, sourcePath string) []depmodel.ParsedDep {
	var manifest cargoManifest
	if _, err := toml.Decode(string(data), &manifest); err != nil {
		return []depmodel.ParsedDep{}
	}

	result := make([]depmodel.ParsedDep, 0)
	result = append(result, parseSection(manifest.Dependencies, depmodel.Direct, sourcePath)...)
	result = append(result, parseSection(manifest.DevDependencies, depmodel.Dev, sourcePath)...)
	result = append(result, parseSection(manifest.BuildDependencies, depmodel.Build, sourcePath)...)
	return result
}

// parseSection converts one dependency table (e.g. [dependencies]) into
// ParsedDep entries, classifying every entry with depType.
func parseSection(section map[string]interface{}, depType, sourcePath string) []depmodel.ParsedDep {
	result := make([]depmodel.ParsedDep, 0, len(section))
	for name, raw := range section {
		if name == "" {
			continue
		}
		result = append(result, depmodel.ParsedDep{
			Ecosystem:  depmodel.EcosystemCratesIO,
			Name:       name,
			Version:    extractVersion(raw),
			DepType:    depType,
			SourceFile: sourcePath,
		})
	}
	return result
}

// extractVersion resolves a dependency's version constraint. Cargo allows
// three shapes for a dependency value:
//
//   - a bare string:        serde = "1.0"
//   - a table:               serde = { version = "1.0", features = [...] }
//   - workspace inheritance: serde = { workspace = true }
//
// Workspace inheritance means the actual version constraint lives in the
// workspace root's Cargo.toml, which this parser does not resolve — the
// same documented limitation as Maven's same-file-only ${property}
// resolution. It returns an empty version string for those entries.
func extractVersion(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return v
	case map[string]interface{}:
		if ws, ok := v["workspace"].(bool); ok && ws {
			return ""
		}
		if version, ok := v["version"].(string); ok {
			return version
		}
		return ""
	default:
		return ""
	}
}
