// Package pyproject parses pyproject.toml manifests using the PEP 621
// [project] table (PyPI ecosystem). Poetry-only manifests that declare
// dependencies under [tool.poetry.dependencies] instead of [project] are a
// documented v1 gap — see design.md.
package pyproject

import (
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// pyprojectManifest is the minimal shape of a pyproject.toml file relevant
// to PEP 621 dependency parsing.
type pyprojectManifest struct {
	Project *pyprojectTable `toml:"project"`
}

// pyprojectTable mirrors the [project] table's dependency-related keys.
type pyprojectTable struct {
	Dependencies         []string            `toml:"dependencies"`
	OptionalDependencies map[string][]string `toml:"optional-dependencies"`
}

// extrasPattern matches a PEP 508 extras bracket, e.g. "[security,socks]",
// so it can be stripped from a requirement spec before the package name is
// extracted.
var extrasPattern = regexp.MustCompile(`\[[^\]]*\]`)

// versionOperators are the characters that can start a PEP 440 version
// specifier. The first occurrence of any of these characters in an
// (extras-stripped, marker-stripped) requirement spec marks the boundary
// between the package name and its raw version constraint.
const versionOperators = "=<>!~"

// ParsePyprojectToml parses the raw content of a pyproject.toml file and
// returns a slice of ParsedDep entries. It is a pure function with no side
// effects.
//
// Only the PEP 621 [project] table is supported: [project].dependencies
// (depmodel.Direct) and [project.optional-dependencies] (depmodel.Optional,
// one group per array). Each PEP 508 requirement string
// ("name[extras]>=version; markers") is reduced to its package name and raw
// version constraint — extras brackets and environment markers are
// stripped, never resolved.
//
// If the [project] table is absent (a Poetry-only pyproject.toml using
// [tool.poetry.dependencies]) or malformed TOML is given, an empty
// (non-nil) slice is returned — never an error. This is a documented v1
// gap; see design.md.
func ParsePyprojectToml(data []byte, sourcePath string) []depmodel.ParsedDep {
	result := make([]depmodel.ParsedDep, 0)

	var manifest pyprojectManifest
	if _, err := toml.Decode(string(data), &manifest); err != nil {
		return result
	}
	if manifest.Project == nil {
		return result
	}

	for _, spec := range manifest.Project.Dependencies {
		if dep, ok := parseDependencySpec(spec, depmodel.Direct, sourcePath); ok {
			result = append(result, dep)
		}
	}

	for _, specs := range manifest.Project.OptionalDependencies {
		for _, spec := range specs {
			if dep, ok := parseDependencySpec(spec, depmodel.Optional, sourcePath); ok {
				result = append(result, dep)
			}
		}
	}

	return result
}

// parseDependencySpec reduces a single PEP 508 requirement string to a
// ParsedDep, stripping environment markers and extras brackets. It reports
// false when the spec has no usable package name (e.g. an empty string).
func parseDependencySpec(spec string, depType, sourcePath string) (depmodel.ParsedDep, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return depmodel.ParsedDep{}, false
	}

	// Strip the environment marker, if any ("; python_version>=...").
	if idx := strings.Index(spec, ";"); idx != -1 {
		spec = strings.TrimSpace(spec[:idx])
	}

	// Strip the extras bracket, if any ("[security,socks]").
	spec = strings.TrimSpace(extrasPattern.ReplaceAllString(spec, ""))
	if spec == "" {
		return depmodel.ParsedDep{}, false
	}

	name, version := splitNameVersion(spec)
	if name == "" {
		return depmodel.ParsedDep{}, false
	}

	return depmodel.ParsedDep{
		Ecosystem:  depmodel.EcosystemPyPI,
		Name:       name,
		Version:    version,
		DepType:    depType,
		SourceFile: sourcePath,
	}, true
}

// splitNameVersion splits a requirement spec (already stripped of extras,
// markers, and surrounding whitespace) into a package name and a raw
// version constraint string. If no PEP 440 operator is present, version is
// "".
func splitNameVersion(spec string) (name, version string) {
	idx := strings.IndexAny(spec, versionOperators)
	if idx == -1 {
		return strings.TrimSpace(spec), ""
	}
	return strings.TrimSpace(spec[:idx]), strings.TrimSpace(spec[idx:])
}
