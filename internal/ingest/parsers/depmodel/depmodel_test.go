package depmodel

import "testing"

// TestParsedDep_field_shape verifies that ParsedDep exposes the fields every
// ecosystem parser depends on, with the expected types.
func TestParsedDep_field_shape(t *testing.T) {
	d := ParsedDep{
		Ecosystem:  EcosystemNpm,
		Name:       "react",
		Version:    "^18.2.0",
		DepType:    Direct,
		SourceFile: "package.json",
	}

	if d.Ecosystem != "npm" {
		t.Errorf("Ecosystem = %q, want %q", d.Ecosystem, "npm")
	}
	if d.Name != "react" {
		t.Errorf("Name = %q, want %q", d.Name, "react")
	}
	if d.Version != "^18.2.0" {
		t.Errorf("Version = %q, want %q", d.Version, "^18.2.0")
	}
	if d.DepType != "dep" {
		t.Errorf("DepType = %q, want %q", d.DepType, "dep")
	}
	if d.SourceFile != "package.json" {
		t.Errorf("SourceFile = %q, want %q", d.SourceFile, "package.json")
	}
}

// TestEcosystemConstants_matchOSVExactly verifies that every ecosystem
// constant matches the OSV.dev case-sensitive ecosystem name exactly.
// Vulnerability matching depends on exact string equality — a typo here
// silently breaks OSV lookups for that ecosystem.
func TestEcosystemConstants_matchOSVExactly(t *testing.T) {
	tests := []struct {
		name  string
		got   string
		want  string
	}{
		{"EcosystemNpm", EcosystemNpm, "npm"},
		{"EcosystemGo", EcosystemGo, "Go"},
		{"EcosystemPyPI", EcosystemPyPI, "PyPI"},
		{"EcosystemMaven", EcosystemMaven, "Maven"},
		{"EcosystemCratesIO", EcosystemCratesIO, "crates.io"},
		{"EcosystemRubyGems", EcosystemRubyGems, "RubyGems"},
		{"EcosystemPackagist", EcosystemPackagist, "Packagist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestEcosystemConstants_areUnique verifies no two ecosystem constants
// collide, which would cause incorrect dependency grouping.
func TestEcosystemConstants_areUnique(t *testing.T) {
	values := []string{
		EcosystemNpm,
		EcosystemGo,
		EcosystemPyPI,
		EcosystemMaven,
		EcosystemCratesIO,
		EcosystemRubyGems,
		EcosystemPackagist,
	}

	seen := make(map[string]bool, len(values))
	for _, v := range values {
		if seen[v] {
			t.Errorf("duplicate ecosystem constant value: %q", v)
		}
		seen[v] = true
	}
}

// TestDepTypeConstants_values verifies the dependency type constants carry
// the exact string values existing callers (e.g. npm parser, store layer)
// depend on.
func TestDepTypeConstants_values(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Direct", Direct, "dep"},
		{"Dev", Dev, "devDep"},
		{"Peer", Peer, "peer"},
		{"Optional", Optional, "optional"},
		{"Build", Build, "build"},
		{"Indirect", Indirect, "indirect"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}
