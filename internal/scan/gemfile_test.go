package scan

import (
	"path/filepath"
	"testing"
)

func TestGemfileScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, root string)
		wantDepNames []string
		wantWarnings int
	}{
		{
			name: "Gemfile at root",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "Gemfile", "gem 'rails', '~> 7.0'\n")
			},
			wantDepNames: []string{"rails"},
		},
		{
			name:         "no Gemfile anywhere",
			setup:        func(t *testing.T, root string) {},
			wantDepNames: nil,
		},
		{
			name: "nested Gemfile excluding vendor",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "Gemfile", "gem 'rails'\n")
				writeFile(t, root, filepath.Join("services", "api", "Gemfile"), "gem 'sinatra'\n")
				writeFile(t, root, filepath.Join("vendor", "bundle", "Gemfile"), "gem 'should-not-appear'\n")
			},
			wantDepNames: []string{"rails", "sinatra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			report := &Report{Path: root, Dependencies: make([]Dependency, 0), Owners: make([]Owner, 0)}
			scanner := NewGemfileScanner()

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

func TestGemfileScanner_Name(t *testing.T) {
	if got := NewGemfileScanner().Name(); got != "gemfile" {
		t.Errorf("expected Name() to return %q, got %q", "gemfile", got)
	}
}
