package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNpmScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, root string)
		wantDepNames []string
		wantWarnings int
	}{
		{
			name: "package.json at root",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, "package.json", `{"dependencies":{"react":"19.0.0"}}`)
			},
			wantDepNames: []string{"react"},
			wantWarnings: 0,
		},
		{
			name:         "no package.json anywhere",
			setup:        func(t *testing.T, root string) {},
			wantDepNames: nil,
			wantWarnings: 0,
		},
		{
			name: "nested package.json excluding node_modules",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, "package.json", `{"dependencies":{"react":"19.0.0"}}`)
				writeFile(t, root, filepath.Join("packages", "api", "package.json"),
					`{"dependencies":{"chi":"5.0.0"}}`)
				writeFile(t, root, filepath.Join("node_modules", "some-lib", "package.json"),
					`{"dependencies":{"should-not-appear":"1.0.0"}}`)
			},
			wantDepNames: []string{"react", "chi"},
			wantWarnings: 0,
		},
		{
			name: "invalid JSON produces a warning and no dependencies",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, "package.json", `{not valid json`)
			},
			wantDepNames: nil,
			wantWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			report := &Report{Path: root, Dependencies: make([]Dependency, 0), Owners: make([]Owner, 0)}
			scanner := NpmScanner{}

			if err := scanner.Scan(root, report); err != nil {
				t.Fatalf("Scan returned unexpected error: %v", err)
			}

			gotNames := make([]string, 0, len(report.Dependencies))
			for _, dep := range report.Dependencies {
				gotNames = append(gotNames, dep.Name)
			}

			if len(gotNames) != len(tt.wantDepNames) {
				t.Fatalf("expected dep names %v, got %v", tt.wantDepNames, gotNames)
			}
			seen := make(map[string]bool)
			for _, n := range gotNames {
				seen[n] = true
			}
			for _, want := range tt.wantDepNames {
				if !seen[want] {
					t.Errorf("expected dependency %q in %v", want, gotNames)
				}
			}

			if len(report.Warnings) != tt.wantWarnings {
				t.Errorf("expected %d warnings, got %d: %v", tt.wantWarnings, len(report.Warnings), report.Warnings)
			}
		})
	}
}

func TestNpmScanner_Name(t *testing.T) {
	if got := (NpmScanner{}).Name(); got != "npm" {
		t.Errorf("expected Name() to return %q, got %q", "npm", got)
	}
}

// writeFile creates relPath (and any parent directories) under root with the
// given content. It fails the test on any error.
func writeFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	full := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", relPath, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", relPath, err)
	}
}
