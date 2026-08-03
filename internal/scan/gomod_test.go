package scan

import (
	"path/filepath"
	"testing"
)

func TestGoModScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, root string)
		wantDepNames []string
		wantWarnings int
	}{
		{
			name: "go.mod at root",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "go.mod", "module acme\n\ngo 1.26\n\nrequire github.com/google/uuid v1.6.0\n")
			},
			wantDepNames: []string{"github.com/google/uuid"},
		},
		{
			name:         "no go.mod anywhere",
			setup:        func(t *testing.T, root string) {},
			wantDepNames: nil,
		},
		{
			name: "nested go.mod excluding vendor",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "go.mod", "module acme\n\ngo 1.26\n\nrequire github.com/google/uuid v1.6.0\n")
				writeFile(t, root, filepath.Join("modules", "billing", "go.mod"),
					"module acme/billing\n\ngo 1.26\n\nrequire github.com/go-chi/chi/v5 v5.3.0\n")
				writeFile(t, root, filepath.Join("vendor", "github.com", "foo", "bar", "go.mod"),
					"module github.com/foo/bar\n\ngo 1.26\n\nrequire github.com/should-not/appear v1.0.0\n")
			},
			wantDepNames: []string{"github.com/google/uuid", "github.com/go-chi/chi/v5"},
		},
		{
			name: "malformed go.mod produces a warning and no dependencies",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "go.mod", "this is not a valid go.mod file {{{")
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
			scanner := NewGoModScanner()

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

func TestGoModScanner_Name(t *testing.T) {
	if got := NewGoModScanner().Name(); got != "gomod" {
		t.Errorf("expected Name() to return %q, got %q", "gomod", got)
	}
}
