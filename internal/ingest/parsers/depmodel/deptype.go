package depmodel

// DepType classification constants shared across all ecosystem parsers.
// Parsers MUST use these constants, never string literals.
const (
	// Direct is a normal runtime/production dependency.
	Direct = "dep"
	// Dev is a development-only dependency (e.g. npm devDependencies,
	// Maven test scope, Cargo dev-dependencies).
	Dev = "devDep"
	// Peer is a peer dependency (npm peerDependencies).
	Peer = "peer"
	// Optional is an optional dependency (npm optionalDependencies,
	// Maven provided scope).
	Optional = "optional"
	// Build is a build-time-only dependency (Cargo build-dependencies).
	Build = "build"
	// Indirect is a transitive dependency pulled in by the module graph
	// (Go `// indirect` requires).
	Indirect = "indirect"
)
