package composer

import (
	"testing"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

func TestParseComposerJSON_require_and_require_dev(t *testing.T) {
	content := []byte(`{
		"name": "acme/app",
		"require": {
			"php": "^8.2",
			"monolog/monolog": "^2.0",
			"symfony/console": "^6.0"
		},
		"require-dev": {
			"phpunit/phpunit": "^10.0"
		}
	}`)

	got := ParseComposerJSON(content, "composer.json")

	if len(got) != 3 {
		t.Fatalf("expected 3 parsed deps (php skipped), got %d: %+v", len(got), got)
	}

	byName := make(map[string]depmodel.ParsedDep)
	for _, d := range got {
		byName[d.Name] = d
	}

	if _, ok := byName["php"]; ok {
		t.Error("expected \"php\" platform requirement to be skipped")
	}

	monolog, ok := byName["monolog/monolog"]
	if !ok {
		t.Fatal("expected \"monolog/monolog\" in parsed deps")
	}
	if monolog.DepType != depmodel.Direct || monolog.Version != "^2.0" || monolog.SourceFile != "composer.json" {
		t.Errorf("unexpected monolog/monolog fields: %+v", monolog)
	}

	console, ok := byName["symfony/console"]
	if !ok {
		t.Fatal("expected \"symfony/console\" in parsed deps")
	}
	if console.DepType != depmodel.Direct {
		t.Errorf("expected symfony/console DepType %q, got %q", depmodel.Direct, console.DepType)
	}

	phpunit, ok := byName["phpunit/phpunit"]
	if !ok {
		t.Fatal("expected \"phpunit/phpunit\" in parsed deps")
	}
	if phpunit.DepType != depmodel.Dev {
		t.Errorf("expected phpunit/phpunit DepType %q, got %q", depmodel.Dev, phpunit.DepType)
	}
}

func TestParseComposerJSON_skips_php_and_ext_entries(t *testing.T) {
	content := []byte(`{
		"require": {
			"php": ">=8.1",
			"ext-json": "*",
			"ext-mbstring": "*",
			"guzzlehttp/guzzle": "^7.0"
		}
	}`)

	got := ParseComposerJSON(content, "composer.json")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep (php + ext-* skipped), got %d: %+v", len(got), got)
	}
	if got[0].Name != "guzzlehttp/guzzle" {
		t.Errorf("Name = %q, want %q", got[0].Name, "guzzlehttp/guzzle")
	}
}

func TestParseComposerJSON_invalid_json_returns_empty(t *testing.T) {
	got := ParseComposerJSON([]byte(`{ not valid json`), "composer.json")
	if len(got) != 0 {
		t.Errorf("expected empty slice for invalid JSON, got %d items", len(got))
	}
}

func TestParseComposerJSON_missing_require_keys_returns_empty(t *testing.T) {
	content := []byte(`{"name": "acme/no-deps", "type": "library"}`)
	got := ParseComposerJSON(content, "composer.json")
	if len(got) != 0 {
		t.Errorf("expected empty slice for manifest with no require keys, got %d items", len(got))
	}
}

func TestParseComposerJSON_empty_name_skipped(t *testing.T) {
	content := []byte(`{"require":{"":"^1.0.0","monolog/monolog":"^2.0"}}`)
	got := ParseComposerJSON(content, "composer.json")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep (empty name skipped), got %d", len(got))
	}
	if got[0].Name != "monolog/monolog" {
		t.Errorf("Name = %q, want %q", got[0].Name, "monolog/monolog")
	}
}

func TestParseComposerJSON_ecosystem_always_packagist(t *testing.T) {
	content := []byte(`{"require":{"monolog/monolog":"^2.0"}}`)
	got := ParseComposerJSON(content, "composer.json")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].Ecosystem != depmodel.EcosystemPackagist {
		t.Errorf("Ecosystem = %q, want %q", got[0].Ecosystem, depmodel.EcosystemPackagist)
	}
}

func TestParseComposerJSON_source_file_path_set(t *testing.T) {
	content := []byte(`{"require":{"monolog/monolog":"^2.0"}}`)
	got := ParseComposerJSON(content, "modules/billing/composer.json")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].SourceFile != "modules/billing/composer.json" {
		t.Errorf("SourceFile = %q, want %q", got[0].SourceFile, "modules/billing/composer.json")
	}
}

func TestParseComposerJSON_version_strings_preserved(t *testing.T) {
	tests := []struct {
		name    string
		pkgName string
		version string
	}{
		{"caret", "vendor/caret", "^2.3.1"},
		{"tilde", "vendor/tilde", "~1.0.0"},
		{"star", "vendor/star", "*"},
		{"exact", "vendor/exact", "1.2.3"},
		{"range", "vendor/range", ">=1.0 <2.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := []byte(`{"require":{"` + tt.pkgName + `":"` + tt.version + `"}}`)
			got := ParseComposerJSON(content, "composer.json")
			if len(got) != 1 {
				t.Fatalf("expected 1 dep, got %d", len(got))
			}
			if got[0].Version != tt.version {
				t.Errorf("Version = %q, want %q", got[0].Version, tt.version)
			}
		})
	}
}
