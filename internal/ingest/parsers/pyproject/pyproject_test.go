package pyproject

import (
	"testing"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

func TestParsePyprojectToml_project_dependencies(t *testing.T) {
	content := []byte(`
[project]
name = "acme-app"
version = "0.1.0"
dependencies = [
    "requests>=2.28",
    "flask",
]
`)

	got := ParsePyprojectToml(content, "pyproject.toml")

	if len(got) != 2 {
		t.Fatalf("expected 2 parsed deps, got %d: %+v", len(got), got)
	}

	byName := make(map[string]depmodel.ParsedDep)
	for _, d := range got {
		byName[d.Name] = d
	}

	requests, ok := byName["requests"]
	if !ok {
		t.Fatal("expected \"requests\" in parsed deps")
	}
	if requests.Version != ">=2.28" {
		t.Errorf("requests Version = %q, want %q", requests.Version, ">=2.28")
	}
	if requests.DepType != depmodel.Direct {
		t.Errorf("requests DepType = %q, want %q", requests.DepType, depmodel.Direct)
	}

	flask, ok := byName["flask"]
	if !ok {
		t.Fatal("expected \"flask\" in parsed deps")
	}
	if flask.Version != "" {
		t.Errorf("flask Version = %q, want empty (no constraint)", flask.Version)
	}
}

func TestParsePyprojectToml_optional_dependencies(t *testing.T) {
	content := []byte(`
[project]
name = "acme-app"
dependencies = []

[project.optional-dependencies]
test = ["pytest>=7.0.0", "coverage"]
dev = ["black>=23.0"]
`)

	got := ParsePyprojectToml(content, "pyproject.toml")

	if len(got) != 3 {
		t.Fatalf("expected 3 parsed deps, got %d: %+v", len(got), got)
	}

	byName := make(map[string]depmodel.ParsedDep)
	for _, d := range got {
		byName[d.Name] = d
	}

	for _, name := range []string{"pytest", "coverage", "black"} {
		dep, ok := byName[name]
		if !ok {
			t.Fatalf("expected %q in parsed deps", name)
		}
		if dep.DepType != depmodel.Optional {
			t.Errorf("%s DepType = %q, want %q", name, dep.DepType, depmodel.Optional)
		}
	}

	pytest := byName["pytest"]
	if pytest.Version != ">=7.0.0" {
		t.Errorf("pytest Version = %q, want %q", pytest.Version, ">=7.0.0")
	}
}

func TestParsePyprojectToml_missing_project_section_returns_empty(t *testing.T) {
	// Poetry-only pyproject.toml — no [project] table (PEP 621), only
	// [tool.poetry.dependencies]. This is a documented v1 gap, NOT an error.
	content := []byte(`
[tool.poetry]
name = "acme-app"
version = "0.1.0"

[tool.poetry.dependencies]
python = "^3.10"
requests = "^2.28"
`)

	got := ParsePyprojectToml(content, "pyproject.toml")

	if len(got) != 0 {
		t.Errorf("expected empty slice for Poetry-only pyproject.toml, got %d items: %+v", len(got), got)
	}
}

func TestParsePyprojectToml_extras_and_markers_stripped(t *testing.T) {
	content := []byte(`
[project]
name = "acme-app"
dependencies = [
    "requests[security,socks]>=2.20.0; python_version>=\"3.6\"",
]
`)

	got := ParsePyprojectToml(content, "pyproject.toml")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "requests" {
		t.Errorf("Name = %q, want %q (extras bracket must be stripped)", got[0].Name, "requests")
	}
	if got[0].Version != ">=2.20.0" {
		t.Errorf("Version = %q, want %q (marker must be stripped)", got[0].Version, ">=2.20.0")
	}
}

func TestParsePyprojectToml_marker_only_no_extras(t *testing.T) {
	content := []byte(`
[project]
name = "acme-app"
dependencies = ["certifi>=2017.4.17; sys_platform=='win32'"]
`)

	got := ParsePyprojectToml(content, "pyproject.toml")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "certifi" {
		t.Errorf("Name = %q, want %q", got[0].Name, "certifi")
	}
	if got[0].Version != ">=2017.4.17" {
		t.Errorf("Version = %q, want %q", got[0].Version, ">=2017.4.17")
	}
}

func TestParsePyprojectToml_malformed_toml_returns_empty(t *testing.T) {
	got := ParsePyprojectToml([]byte(`this is [ not valid toml`), "pyproject.toml")
	if len(got) != 0 {
		t.Errorf("expected empty slice for malformed TOML, got %d items", len(got))
	}
}

func TestParsePyprojectToml_empty_dependencies_returns_empty(t *testing.T) {
	content := []byte(`
[project]
name = "acme-app"
version = "0.1.0"
`)
	got := ParsePyprojectToml(content, "pyproject.toml")
	if len(got) != 0 {
		t.Errorf("expected empty slice for [project] with no dependencies key, got %d items", len(got))
	}
}

func TestParsePyprojectToml_ecosystem_always_pypi(t *testing.T) {
	content := []byte(`
[project]
name = "acme-app"
dependencies = ["requests>=2.28"]
`)
	got := ParsePyprojectToml(content, "pyproject.toml")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].Ecosystem != depmodel.EcosystemPyPI {
		t.Errorf("Ecosystem = %q, want %q", got[0].Ecosystem, depmodel.EcosystemPyPI)
	}
}

func TestParsePyprojectToml_source_file_path_set(t *testing.T) {
	content := []byte(`
[project]
name = "acme-app"
dependencies = ["requests>=2.28"]
`)
	got := ParsePyprojectToml(content, "services/api/pyproject.toml")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].SourceFile != "services/api/pyproject.toml" {
		t.Errorf("SourceFile = %q, want %q", got[0].SourceFile, "services/api/pyproject.toml")
	}
}

func TestParsePyprojectToml_version_operators(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		wantName    string
		wantVersion string
	}{
		{"exact pin", "django==4.1.0", "django", "==4.1.0"},
		{"minimum", "numpy>=1.23.0", "numpy", ">=1.23.0"},
		{"compatible release", "requests~=2.28", "requests", "~=2.28"},
		{"exclusion", "pytest!=7.0.0", "pytest", "!=7.0.0"},
		{"no version", "six", "six", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := []byte(`
[project]
name = "acme-app"
dependencies = ["` + tt.spec + `"]
`)
			got := ParsePyprojectToml(content, "pyproject.toml")
			if len(got) != 1 {
				t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
			}
			if got[0].Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got[0].Name, tt.wantName)
			}
			if got[0].Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", got[0].Version, tt.wantVersion)
			}
		})
	}
}
