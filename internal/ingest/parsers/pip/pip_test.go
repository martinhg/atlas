package pip

import (
	"testing"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

func TestParseRequirementsTxt_skips_blank_lines_and_comments(t *testing.T) {
	content := []byte(`
# this is a full-line comment
requests==2.28.1

# another comment
flask==2.0.1
`)

	got := ParseRequirementsTxt(content, "requirements.txt")

	if len(got) != 2 {
		t.Fatalf("expected 2 parsed deps, got %d: %+v", len(got), got)
	}
	byName := make(map[string]depmodel.ParsedDep)
	for _, d := range got {
		byName[d.Name] = d
	}
	if _, ok := byName["requests"]; !ok {
		t.Error("expected \"requests\" in parsed deps")
	}
	if _, ok := byName["flask"]; !ok {
		t.Error("expected \"flask\" in parsed deps")
	}
}

func TestParseRequirementsTxt_skips_recursive_and_editable_and_flag_lines(t *testing.T) {
	content := []byte(`-r base.txt
-c constraints.txt
-e git+https://github.com/acme/lib.git@main#egg=acme-lib
-e .
--index-url https://pypi.org/simple
--extra-index-url https://custom.example.com/simple
-f https://download.example.com/wheels
requests==2.28.1
`)

	got := ParseRequirementsTxt(content, "requirements.txt")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep (all option/flag lines skipped), got %d: %+v", len(got), got)
	}
	if got[0].Name != "requests" {
		t.Errorf("Name = %q, want %q", got[0].Name, "requests")
	}
}

func TestParseRequirementsTxt_version_operators(t *testing.T) {
	tests := []struct {
		name        string
		line        string
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
			got := ParseRequirementsTxt([]byte(tt.line), "requirements.txt")
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

func TestParseRequirementsTxt_extras_and_markers(t *testing.T) {
	content := []byte(`requests[security,socks]>=2.20.0; python_version>="3.6"`)

	got := ParseRequirementsTxt(content, "requirements.txt")

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

func TestParseRequirementsTxt_marker_only_no_extras(t *testing.T) {
	content := []byte(`certifi>=2017.4.17; sys_platform=='win32'`)

	got := ParseRequirementsTxt(content, "requirements.txt")

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

func TestParseRequirementsTxt_hash_pins_stripped(t *testing.T) {
	content := []byte(`flask==2.0.1 --hash=sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`)

	got := ParseRequirementsTxt(content, "requirements.txt")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "flask" || got[0].Version != "==2.0.1" {
		t.Errorf("unexpected parsed dep: %+v", got[0])
	}
}

func TestParseRequirementsTxt_inline_comment_stripped(t *testing.T) {
	content := []byte(`gunicorn==20.1.0  # WSGI server`)

	got := ParseRequirementsTxt(content, "requirements.txt")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "gunicorn" || got[0].Version != "==20.1.0" {
		t.Errorf("unexpected parsed dep: %+v", got[0])
	}
}

func TestParseRequirementsTxt_line_continuation_joined(t *testing.T) {
	content := []byte("flask==2.0.1 \\\n    --hash=sha256:abcdef0123456789\n")

	got := ParseRequirementsTxt(content, "requirements.txt")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "flask" || got[0].Version != "==2.0.1" {
		t.Errorf("unexpected parsed dep: %+v", got[0])
	}
}

func TestParseRequirementsTxt_empty_content_returns_empty(t *testing.T) {
	got := ParseRequirementsTxt([]byte(""), "requirements.txt")
	if len(got) != 0 {
		t.Errorf("expected empty slice for empty content, got %d items", len(got))
	}
}

func TestParseRequirementsTxt_all_deps_are_direct_type(t *testing.T) {
	content := []byte("requests==2.28.1\nflask==2.0.1\n")

	got := ParseRequirementsTxt(content, "requirements.txt")

	for _, d := range got {
		if d.DepType != depmodel.Direct {
			t.Errorf("DepType = %q, want %q (requirements.txt has no dev/indirect distinction)", d.DepType, depmodel.Direct)
		}
	}
}

func TestParseRequirementsTxt_ecosystem_always_pypi(t *testing.T) {
	got := ParseRequirementsTxt([]byte("requests==2.28.1"), "requirements.txt")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].Ecosystem != depmodel.EcosystemPyPI {
		t.Errorf("Ecosystem = %q, want %q", got[0].Ecosystem, depmodel.EcosystemPyPI)
	}
}

func TestParseRequirementsTxt_source_file_path_set(t *testing.T) {
	got := ParseRequirementsTxt([]byte("requests==2.28.1"), "requirements/base.txt")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].SourceFile != "requirements/base.txt" {
		t.Errorf("SourceFile = %q, want %q", got[0].SourceFile, "requirements/base.txt")
	}
}
