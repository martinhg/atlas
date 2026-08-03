package cargo

import (
	"testing"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

func TestParseCargoToml_bare_string_dependency(t *testing.T) {
	content := []byte(`
[package]
name = "acme-app"
version = "0.1.0"

[dependencies]
serde = "1.0"
`)

	got := ParseCargoToml(content, "Cargo.toml")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "serde" || got[0].Version != "1.0" {
		t.Errorf("unexpected serde fields: %+v", got[0])
	}
	if got[0].DepType != depmodel.Direct {
		t.Errorf("DepType = %q, want %q", got[0].DepType, depmodel.Direct)
	}
}

func TestParseCargoToml_inline_table_dependency(t *testing.T) {
	content := []byte(`
[dependencies]
serde = { version = "1.0", features = ["derive"] }
`)

	got := ParseCargoToml(content, "Cargo.toml")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "serde" || got[0].Version != "1.0" {
		t.Errorf("unexpected serde fields: %+v", got[0])
	}
}

func TestParseCargoToml_table_dependency_multiline(t *testing.T) {
	content := []byte(`
[dependencies.tokio]
version = "1.35"
features = ["full"]
`)

	got := ParseCargoToml(content, "Cargo.toml")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "tokio" || got[0].Version != "1.35" {
		t.Errorf("unexpected tokio fields: %+v", got[0])
	}
}

func TestParseCargoToml_dev_and_build_dependencies(t *testing.T) {
	content := []byte(`
[dependencies]
serde = "1.0"

[dev-dependencies]
criterion = "0.5"

[build-dependencies]
cc = "1.0"
`)

	got := ParseCargoToml(content, "Cargo.toml")

	if len(got) != 3 {
		t.Fatalf("expected 3 parsed deps, got %d: %+v", len(got), got)
	}

	byName := make(map[string]depmodel.ParsedDep)
	for _, d := range got {
		byName[d.Name] = d
	}

	serde, ok := byName["serde"]
	if !ok {
		t.Fatal("expected \"serde\" in parsed deps")
	}
	if serde.DepType != depmodel.Direct {
		t.Errorf("serde DepType = %q, want %q", serde.DepType, depmodel.Direct)
	}

	criterion, ok := byName["criterion"]
	if !ok {
		t.Fatal("expected \"criterion\" in parsed deps")
	}
	if criterion.DepType != depmodel.Dev {
		t.Errorf("criterion DepType = %q, want %q", criterion.DepType, depmodel.Dev)
	}

	cc, ok := byName["cc"]
	if !ok {
		t.Fatal("expected \"cc\" in parsed deps")
	}
	if cc.DepType != depmodel.Build {
		t.Errorf("cc DepType = %q, want %q", cc.DepType, depmodel.Build)
	}
}

func TestParseCargoToml_workspace_inheritance_has_empty_version(t *testing.T) {
	content := []byte(`
[dependencies]
serde = { workspace = true }
tokio = { workspace = true, features = ["full"] }
`)

	got := ParseCargoToml(content, "Cargo.toml")

	if len(got) != 2 {
		t.Fatalf("expected 2 parsed deps, got %d: %+v", len(got), got)
	}
	for _, d := range got {
		if d.Version != "" {
			t.Errorf("expected empty Version for workspace-inherited dep %q, got %q", d.Name, d.Version)
		}
	}
}

func TestParseCargoToml_malformed_toml_returns_empty(t *testing.T) {
	got := ParseCargoToml([]byte(`this is [ not valid toml`), "Cargo.toml")
	if len(got) != 0 {
		t.Errorf("expected empty slice for malformed TOML, got %d items", len(got))
	}
}

func TestParseCargoToml_no_dependency_sections_returns_empty(t *testing.T) {
	content := []byte(`
[package]
name = "acme-app"
version = "0.1.0"
`)
	got := ParseCargoToml(content, "Cargo.toml")
	if len(got) != 0 {
		t.Errorf("expected empty slice for manifest with no dependency sections, got %d items", len(got))
	}
}

func TestParseCargoToml_ecosystem_always_crates_io(t *testing.T) {
	content := []byte(`
[dependencies]
serde = "1.0"
`)
	got := ParseCargoToml(content, "Cargo.toml")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].Ecosystem != depmodel.EcosystemCratesIO {
		t.Errorf("Ecosystem = %q, want %q", got[0].Ecosystem, depmodel.EcosystemCratesIO)
	}
}

func TestParseCargoToml_source_file_path_set(t *testing.T) {
	content := []byte(`
[dependencies]
serde = "1.0"
`)
	got := ParseCargoToml(content, "crates/core/Cargo.toml")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].SourceFile != "crates/core/Cargo.toml" {
		t.Errorf("SourceFile = %q, want %q", got[0].SourceFile, "crates/core/Cargo.toml")
	}
}

func TestParseCargoToml_version_strings_preserved(t *testing.T) {
	tests := []struct {
		name    string
		pkgName string
		version string
	}{
		{"caret", "caret-dep", "^2.3.1"},
		{"tilde", "tilde-dep", "~1.0.0"},
		{"star", "star-dep", "*"},
		{"exact", "exact-dep", "=1.2.3"},
		{"range", "range-dep", ">=1.0, <2.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := []byte("[dependencies]\n" + tt.pkgName + ` = "` + tt.version + `"` + "\n")
			got := ParseCargoToml(content, "Cargo.toml")
			if len(got) != 1 {
				t.Fatalf("expected 1 dep, got %d", len(got))
			}
			if got[0].Version != tt.version {
				t.Errorf("Version = %q, want %q", got[0].Version, tt.version)
			}
		})
	}
}
