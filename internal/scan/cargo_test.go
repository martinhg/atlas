package scan

import (
	"path/filepath"
	"testing"
)

func TestCargoScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, root string)
		wantDepNames []string
		wantWarnings int
	}{
		{
			name: "Cargo.toml at root",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "Cargo.toml", "[dependencies]\nserde = \"1.0\"\n")
			},
			wantDepNames: []string{"serde"},
		},
		{
			name:         "no Cargo.toml anywhere",
			setup:        func(t *testing.T, root string) {},
			wantDepNames: nil,
		},
		{
			name: "nested Cargo.toml excluding target",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "Cargo.toml", "[dependencies]\nserde = \"1.0\"\n")
				writeFile(t, root, filepath.Join("crates", "core", "Cargo.toml"),
					"[dependencies]\ntokio = \"1.35\"\n")
				writeFile(t, root, filepath.Join("target", "debug", "Cargo.toml"),
					"[dependencies]\nshould-not-appear = \"1.0.0\"\n")
			},
			wantDepNames: []string{"serde", "tokio"},
		},
		{
			name: "invalid TOML produces a warning and no dependencies",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "Cargo.toml", "this is [ not valid toml")
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
			scanner := NewCargoScanner()

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

func TestCargoScanner_Name(t *testing.T) {
	if got := NewCargoScanner().Name(); got != "cargo" {
		t.Errorf("expected Name() to return %q, got %q", "cargo", got)
	}
}
