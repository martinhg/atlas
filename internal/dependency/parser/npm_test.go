package parser

import (
	"testing"
)

func TestParsePackageJSON_all_dep_types(t *testing.T) {
	content := []byte(`{
		"name": "my-app",
		"dependencies": {
			"react": "^18.2.0",
			"react-dom": "^18.2.0"
		},
		"devDependencies": {
			"typescript": "^5.0.0",
			"vitest": "^1.0.0"
		},
		"peerDependencies": {
			"react": ">=16.0.0"
		},
		"optionalDependencies": {
			"fsevents": "^2.3.2"
		}
	}`)

	got := ParsePackageJSON(content, "package.json")

	if len(got) != 6 {
		t.Errorf("expected 6 parsed deps, got %d", len(got))
	}

	byName := make(map[string]ParsedDep)
	for _, d := range got {
		byName[d.Name+":"+d.DepType] = d
	}

	tests := []struct {
		key        string
		wantType   string
		wantSource string
	}{
		{"react:dep", "dep", "package.json"},
		{"react-dom:dep", "dep", "package.json"},
		{"typescript:devDep", "devDep", "package.json"},
		{"vitest:devDep", "devDep", "package.json"},
		{"react:peer", "peer", "package.json"},
		{"fsevents:optional", "optional", "package.json"},
	}

	for _, tt := range tests {
		d, ok := byName[tt.key]
		if !ok {
			t.Errorf("missing dep with key %q", tt.key)
			continue
		}
		if d.DepType != tt.wantType {
			t.Errorf("dep %q: DepType = %q, want %q", tt.key, d.DepType, tt.wantType)
		}
		if d.SourceFile != tt.wantSource {
			t.Errorf("dep %q: SourceFile = %q, want %q", tt.key, d.SourceFile, tt.wantSource)
		}
	}
}

func TestParsePackageJSON_invalid_json_returns_empty(t *testing.T) {
	got := ParsePackageJSON([]byte(`{ not valid json`), "package.json")
	if len(got) != 0 {
		t.Errorf("expected empty slice for invalid JSON, got %d items", len(got))
	}
}

func TestParsePackageJSON_missing_dep_keys_returns_empty(t *testing.T) {
	content := []byte(`{
		"name": "no-deps",
		"version": "1.0.0",
		"scripts": {
			"build": "tsc"
		}
	}`)
	got := ParsePackageJSON(content, "package.json")
	if len(got) != 0 {
		t.Errorf("expected empty slice for manifest with no dep keys, got %d items", len(got))
	}
}

func TestParsePackageJSON_version_strings_preserved(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"caret", "^1.2.3"},
		{"tilde", "~2.0.0"},
		{"star", "*"},
		{"exact", "3.4.5"},
		{"range", ">=16.0.0 <17.0.0"},
		{"tag", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := []byte(`{"dependencies":{"` + tt.name + `":"` + tt.version + `"}}`)
			got := ParsePackageJSON(content, "package.json")
			if len(got) != 1 {
				t.Fatalf("expected 1 dep, got %d", len(got))
			}
			if got[0].Version != tt.version {
				t.Errorf("Version = %q, want %q", got[0].Version, tt.version)
			}
		})
	}
}

func TestParsePackageJSON_source_file_path_set(t *testing.T) {
	content := []byte(`{"dependencies":{"react":"^18.0.0"}}`)
	got := ParsePackageJSON(content, "packages/ui/package.json")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].SourceFile != "packages/ui/package.json" {
		t.Errorf("SourceFile = %q, want %q", got[0].SourceFile, "packages/ui/package.json")
	}
}

func TestParsePackageJSON_ecosystem_always_npm(t *testing.T) {
	content := []byte(`{"dependencies":{"express":"^4.18.0"}}`)
	got := ParsePackageJSON(content, "package.json")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].Ecosystem != "npm" {
		t.Errorf("Ecosystem = %q, want %q", got[0].Ecosystem, "npm")
	}
}

func TestParsePackageJSON_scoped_package_name(t *testing.T) {
	content := []byte(`{"dependencies":{"@scope/package":"^1.0.0"}}`)
	got := ParsePackageJSON(content, "package.json")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].Name != "@scope/package" {
		t.Errorf("Name = %q, want %q", got[0].Name, "@scope/package")
	}
}

func TestParsePackageJSON_empty_name_skipped(t *testing.T) {
	content := []byte(`{"dependencies":{"":"^1.0.0","react":"^18.0.0"}}`)
	got := ParsePackageJSON(content, "package.json")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep (empty name skipped), got %d", len(got))
	}
	if got[0].Name != "react" {
		t.Errorf("Name = %q, want %q", got[0].Name, "react")
	}
}

func TestParsePackageJSON_null_dependency_section(t *testing.T) {
	content := []byte(`{"dependencies":null}`)
	got := ParsePackageJSON(content, "package.json")
	if len(got) != 0 {
		t.Errorf("expected empty slice for null dependencies, got %d items", len(got))
	}
}

func TestParsePackageJSON_non_string_version_ignored(t *testing.T) {
	// json.Unmarshal into map[string]string silently ignores non-string values
	// by failing the entire unmarshal of the map, leaving it nil.
	content := []byte(`{"dependencies":{"react":123}}`)
	got := ParsePackageJSON(content, "package.json")
	// The entire dependencies map fails to unmarshal, so no deps are produced.
	if len(got) != 0 {
		t.Errorf("expected 0 deps for non-string version, got %d", len(got))
	}
}

func TestParsePackageJSON_only_some_dep_keys_present(t *testing.T) {
	content := []byte(`{
		"name": "partial",
		"devDependencies": {
			"eslint": "^8.0.0"
		}
	}`)
	got := ParsePackageJSON(content, "package.json")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep (only devDependencies present), got %d", len(got))
	}
	if got[0].Name != "eslint" {
		t.Errorf("Name = %q, want %q", got[0].Name, "eslint")
	}
	if got[0].DepType != "devDep" {
		t.Errorf("DepType = %q, want %q", got[0].DepType, "devDep")
	}
}
