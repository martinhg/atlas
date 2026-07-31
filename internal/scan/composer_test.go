package scan

import (
	"path/filepath"
	"testing"
)

func TestComposerScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, root string)
		wantDepNames []string
		wantWarnings int
	}{
		{
			name: "composer.json at root",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "composer.json", `{"require":{"monolog/monolog":"^2.0"}}`)
			},
			wantDepNames: []string{"monolog/monolog"},
		},
		{
			name:         "no composer.json anywhere",
			setup:        func(t *testing.T, root string) {},
			wantDepNames: nil,
		},
		{
			name: "nested composer.json excluding vendor",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "composer.json", `{"require":{"monolog/monolog":"^2.0"}}`)
				writeFile(t, root, filepath.Join("modules", "billing", "composer.json"),
					`{"require":{"symfony/console":"^6.0"}}`)
				writeFile(t, root, filepath.Join("vendor", "monolog", "monolog", "composer.json"),
					`{"require":{"should-not-appear/should-not-appear":"1.0.0"}}`)
			},
			wantDepNames: []string{"monolog/monolog", "symfony/console"},
		},
		{
			name: "invalid JSON produces a warning and no dependencies",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "composer.json", `{not valid json`)
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
			scanner := NewComposerScanner()

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

func TestComposerScanner_Name(t *testing.T) {
	if got := NewComposerScanner().Name(); got != "composer" {
		t.Errorf("expected Name() to return %q, got %q", "composer", got)
	}
}
