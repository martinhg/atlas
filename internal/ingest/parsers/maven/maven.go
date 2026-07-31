// Package maven parses pom.xml manifests (Maven ecosystem).
package maven

import (
	"encoding/xml"
	"regexp"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// propertyPlaceholder matches a Maven property placeholder that spans the
// entire version string, e.g. "${spring.version}". Partial interpolation
// (e.g. "1.0.${suffix}") is not resolved — a documented v1 limitation.
var propertyPlaceholder = regexp.MustCompile(`^\$\{([^}]+)\}$`)

// pomProject is the minimal shape of a pom.xml file relevant to dependency
// parsing.
//
// The Dependencies field uses the path tag "dependencies>dependency", so it
// matches only <project><dependencies><dependency> elements —
// <dependencyManagement><dependencies><dependency> is a different XML path
// and is never matched. dependencyManagement entries are version hints, not
// actual dependencies, so this skip is structural rather than a runtime
// filter.
type pomProject struct {
	XMLName      xml.Name        `xml:"project"`
	Properties   pomProperties   `xml:"properties"`
	Dependencies []pomDependency `xml:"dependencies>dependency"`
}

// pomProperties captures a pom.xml's <properties> section, whose child
// element names are arbitrary (user-defined), so xml:",any" is required to
// capture them generically instead of naming each field.
type pomProperties struct {
	Entries []pomProperty `xml:",any"`
}

// pomProperty is a single <properties> child element, e.g.
// <junit.version>4.13.2</junit.version> becomes XMLName.Local ==
// "junit.version" and Value == "4.13.2".
type pomProperty struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
}

// ParsePomXML parses the raw content of a pom.xml file and returns a slice
// of ParsedDep entries. It is a pure function with no side effects.
// Malformed XML results in an empty (non-nil) slice.
//
// Only <project><dependencies><dependency> entries are parsed —
// <dependencyManagement> entries are version hints, not actual
// dependencies, and are skipped by construction (see pomProject).
//
// Each dependency's Name is "groupId:artifactId". A "${property}" version
// placeholder is resolved against the same file's <properties> section; if
// the property is not defined there (e.g. it is inherited from a parent
// POM, which this parser does not fetch), the raw "${...}" string is kept
// as-is — a documented v1 limitation.
func ParsePomXML(data []byte, sourcePath string) []depmodel.ParsedDep {
	var pom pomProject
	if err := xml.Unmarshal(data, &pom); err != nil {
		return []depmodel.ParsedDep{}
	}

	props := make(map[string]string, len(pom.Properties.Entries))
	for _, p := range pom.Properties.Entries {
		props[p.XMLName.Local] = p.Value
	}

	result := make([]depmodel.ParsedDep, 0, len(pom.Dependencies))
	for _, dep := range pom.Dependencies {
		if dep.GroupID == "" || dep.ArtifactID == "" {
			continue
		}
		result = append(result, depmodel.ParsedDep{
			Ecosystem:  depmodel.EcosystemMaven,
			Name:       dep.GroupID + ":" + dep.ArtifactID,
			Version:    resolveVersion(dep.Version, props),
			DepType:    depTypeForScope(dep.Scope),
			SourceFile: sourcePath,
		})
	}

	return result
}

// resolveVersion resolves a "${property}" placeholder against props, the
// pom.xml's own <properties> section. If version is not a full property
// placeholder, or the referenced property is undefined in this file (e.g.
// it comes from a parent POM this parser never fetches), version is
// returned unchanged.
func resolveVersion(version string, props map[string]string) string {
	m := propertyPlaceholder.FindStringSubmatch(version)
	if m == nil {
		return version
	}
	if resolved, ok := props[m[1]]; ok {
		return resolved
	}
	return version
}

// depTypeForScope maps a Maven <scope> value to a depmodel.DepType
// constant. "test" is a development-only dependency; "compile" (the Maven
// default, used when <scope> is omitted), "provided", "runtime", and
// "system" are all treated as depmodel.Direct.
func depTypeForScope(scope string) string {
	if scope == "test" {
		return depmodel.Dev
	}
	return depmodel.Direct
}
