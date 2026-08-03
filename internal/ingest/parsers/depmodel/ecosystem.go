package depmodel

// Ecosystem* constants are OSV.dev ecosystem names. They are case-sensitive
// and MUST match https://ossf.github.io/osv-schema/#affectedpackage-field
// exactly — vulnerability matching depends on exact string equality. All
// parsers MUST use these constants, never string literals, to prevent typos
// that would silently break OSV lookups for that ecosystem.
const (
	EcosystemNpm       = "npm"
	EcosystemGo        = "Go"
	EcosystemPyPI      = "PyPI"
	EcosystemMaven     = "Maven"
	EcosystemCratesIO  = "crates.io"
	EcosystemRubyGems  = "RubyGems"
	EcosystemPackagist = "Packagist"
)
