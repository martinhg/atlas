package scan

import (
	"path/filepath"
	"testing"
)

func TestPipScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, root string)
		wantDepNames []string
		wantWarnings int
	}{
		{
			name: "requirements.txt at root",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "requirements.txt", "requests==2.28.1\n")
			},
			wantDepNames: []string{"requests"},
		},
		{
			name:         "no requirements.txt anywhere",
			setup:        func(t *testing.T, root string) {},
			wantDepNames: nil,
		},
		{
			name: "nested requirements.txt excluding venv dirs",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "requirements.txt", "requests==2.28.1\n")
				writeFile(t, root, filepath.Join("services", "worker", "requirements.txt"),
					"flask==2.0.1\n")
				writeFile(t, root, filepath.Join(".venv", "lib", "requirements.txt"),
					"should-not-appear==1.0.0\n")
				writeFile(t, root, filepath.Join("venv", "lib", "requirements.txt"),
					"also-not-appear==1.0.0\n")
				writeFile(t, root, filepath.Join(".tox", "py311", "requirements.txt"),
					"tox-not-appear==1.0.0\n")
				writeFile(t, root, filepath.Join("__pycache__", "requirements.txt"),
					"cache-not-appear==1.0.0\n")
			},
			wantDepNames: []string{"requests", "flask"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			report := &Report{Path: root, Dependencies: make([]Dependency, 0), Owners: make([]Owner, 0)}
			scanner := NewPipScanner()

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

func TestPipScanner_Name(t *testing.T) {
	if got := NewPipScanner().Name(); got != "pip" {
		t.Errorf("expected Name() to return %q, got %q", "pip", got)
	}
}
