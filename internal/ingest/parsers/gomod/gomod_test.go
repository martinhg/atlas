package gomod

import (
	"testing"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

func TestParseGoMod_direct_and_indirect_requires(t *testing.T) {
	content := []byte(`module github.com/acme/app

go 1.26

require (
	github.com/go-chi/chi/v5 v5.3.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	golang.org/x/sync v0.17.0 // indirect
)
`)

	got := ParseGoMod(content, "go.mod")

	if len(got) != 4 {
		t.Fatalf("expected 4 parsed deps, got %d: %+v", len(got), got)
	}

	byName := make(map[string]depmodel.ParsedDep)
	for _, d := range got {
		byName[d.Name] = d
	}

	chi, ok := byName["github.com/go-chi/chi/v5"]
	if !ok {
		t.Fatal("expected \"github.com/go-chi/chi/v5\" in parsed deps")
	}
	if chi.DepType != depmodel.Direct || chi.Version != "v5.3.0" || chi.SourceFile != "go.mod" {
		t.Errorf("unexpected chi fields: %+v", chi)
	}

	uuidDep, ok := byName["github.com/google/uuid"]
	if !ok {
		t.Fatal("expected \"github.com/google/uuid\" in parsed deps")
	}
	if uuidDep.DepType != depmodel.Direct {
		t.Errorf("expected direct DepType, got %q", uuidDep.DepType)
	}

	jwt, ok := byName["github.com/golang-jwt/jwt/v4"]
	if !ok {
		t.Fatal("expected \"github.com/golang-jwt/jwt/v4\" in parsed deps")
	}
	if jwt.DepType != depmodel.Indirect || jwt.Version != "v4.5.2" {
		t.Errorf("unexpected jwt fields: %+v", jwt)
	}

	xsync, ok := byName["golang.org/x/sync"]
	if !ok {
		t.Fatal("expected \"golang.org/x/sync\" in parsed deps")
	}
	if xsync.DepType != depmodel.Indirect {
		t.Errorf("expected indirect DepType, got %q", xsync.DepType)
	}
}

func TestParseGoMod_skips_replace_exclude_retract(t *testing.T) {
	content := []byte(`module github.com/acme/app

go 1.26

require github.com/google/uuid v1.6.0

exclude github.com/foo/bar v1.0.0

replace github.com/foo/bar => github.com/foo/bar-fork v1.0.1

retract v0.0.1
`)

	got := ParseGoMod(content, "go.mod")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep (replace/exclude/retract skipped), got %d: %+v", len(got), got)
	}
	if got[0].Name != "github.com/google/uuid" {
		t.Errorf("Name = %q, want %q", got[0].Name, "github.com/google/uuid")
	}
}

func TestParseGoMod_malformed_returns_empty(t *testing.T) {
	got := ParseGoMod([]byte(`this is not a valid go.mod file {{{`), "go.mod")
	if len(got) != 0 {
		t.Errorf("expected empty slice for malformed go.mod, got %d items", len(got))
	}
}

func TestParseGoMod_no_requires_returns_empty(t *testing.T) {
	content := []byte(`module github.com/acme/app

go 1.26
`)
	got := ParseGoMod(content, "go.mod")
	if len(got) != 0 {
		t.Errorf("expected empty slice for go.mod with no requires, got %d items", len(got))
	}
}

func TestParseGoMod_ecosystem_always_go(t *testing.T) {
	content := []byte(`module github.com/acme/app

go 1.26

require github.com/google/uuid v1.6.0
`)
	got := ParseGoMod(content, "go.mod")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].Ecosystem != depmodel.EcosystemGo {
		t.Errorf("Ecosystem = %q, want %q", got[0].Ecosystem, depmodel.EcosystemGo)
	}
}

func TestParseGoMod_source_file_path_set(t *testing.T) {
	content := []byte(`module github.com/acme/app

go 1.26

require github.com/google/uuid v1.6.0
`)
	got := ParseGoMod(content, "modules/billing/go.mod")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].SourceFile != "modules/billing/go.mod" {
		t.Errorf("SourceFile = %q, want %q", got[0].SourceFile, "modules/billing/go.mod")
	}
}

func TestParseGoMod_version_strings_preserved(t *testing.T) {
	content := []byte(`module github.com/acme/app

go 1.26

require github.com/foo/bar v0.0.0-20240101000000-abcdef123456
`)
	got := ParseGoMod(content, "go.mod")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	want := "v0.0.0-20240101000000-abcdef123456"
	if got[0].Version != want {
		t.Errorf("Version = %q, want %q", got[0].Version, want)
	}
}
