// Package depmodel defines the shared dependency model used by every
// ecosystem parser under internal/ingest/parsers/. It has no imports from
// sibling parser packages, which keeps the parser tree free of import
// cycles: each ecosystem parser depends on depmodel, never on another
// parser.
package depmodel

// ParsedDep represents a single dependency entry parsed from a manifest
// file, independent of ecosystem. Every ecosystem parser (npm, composer,
// go.mod, requirements.txt, pom.xml, Cargo.toml, pyproject.toml, Gemfile)
// returns a slice of these from its Parse function.
type ParsedDep struct {
	// Ecosystem is the OSV.dev-exact ecosystem name. Use the Ecosystem*
	// constants in ecosystem.go — never a string literal.
	Ecosystem string

	// Name is the package name as declared in the manifest, e.g. "react"
	// (npm) or "groupId:artifactId" (Maven).
	Name string

	// Version is the raw version constraint string exactly as written in
	// the manifest. It is never normalized or resolved.
	Version string

	// DepType classifies the dependency. Use the DepType constants in
	// deptype.go — never a string literal.
	DepType string

	// SourceFile is the path to the manifest file this dep was parsed
	// from, relative to the repository root.
	SourceFile string
}
