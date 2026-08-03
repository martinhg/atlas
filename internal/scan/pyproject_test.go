package scan

import (
	"path/filepath"
	"testing"
)

func TestPyprojectScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, root string)
		wantDepNames []string
		wantWarnings int
	}{
		{
			name: "pyproject.toml at root",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "pyproject.toml",
					"[project]\nname = \"acme-app\"\ndependencies = [\"requests>=2.28\"]\n")
			},
			wantDepNames: []string{"requests"},
		},
		{
			name:         "no pyproject.toml anywhere",
			setup:        func(t *testing.T, root string) {},
			wantDepNames: nil,
		},
		{
			name: "nested pyproject.toml excluding venv and tox dirs",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "pyproject.toml",
					"[project]\nname = \"acme-app\"\ndependencies = [\"requests>=2.28\"]\n")
				writeFile(t, root, filepath.Join("services", "api", "pyproject.toml"),
					"[project]\nname = \"api\"\ndependencies = [\"flask\"]\n")
				writeFile(t, root, filepath.Join(".venv", "lib", "pyproject.toml"),
					"[project]\nname = \"venv-pkg\"\ndependencies = [\"should-not-appear\"]\n")
				writeFile(t, root, filepath.Join("venv", "lib", "pyproject.toml"),
					"[project]\nname = \"venv-pkg2\"\ndependencies = [\"also-not-appear\"]\n")
				writeFile(t, root, filepath.Join(".tox", "py311", "pyproject.toml"),
					"[project]\nname = \"tox-pkg\"\ndependencies = [\"tox-not-appear\"]\n")
				writeFile(t, root, filepath.Join("__pycache__", "pyproject.toml"),
					"[project]\nname = \"cache-pkg\"\ndependencies = [\"cache-not-appear\"]\n")
			},
			wantDepNames: []string{"requests", "flask"},
		},
		{
			name: "invalid TOML produces a warning and no dependencies",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "pyproject.toml", "this is [ not valid toml")
			},
			wantDepNames: nil,
			wantWarnings: 1,
		},
		{
			name: "Poetry-only pyproject.toml has no [project] table and produces no deps",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "pyproject.toml",
					"[tool.poetry]\nname = \"acme-app\"\n\n[tool.poetry.dependencies]\nrequests = \"^2.28\"\n")
			},
			wantDepNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			report := &Report{Path: root, Dependencies: make([]Dependency, 0), Owners: make([]Owner, 0)}
			scanner := NewPyprojectScanner()

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

func TestPyprojectScanner_Name(t *testing.T) {
	if got := NewPyprojectScanner().Name(); got != "pyproject" {
		t.Errorf("expected Name() to return %q, got %q", "pyproject", got)
	}
}
