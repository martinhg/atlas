package scan

import (
	"path/filepath"
	"testing"
)

func TestCodeownersScanner_Scan(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T, root string)
		wantOwnerLen  int
		wantOwnerName string
		wantWarnings  int
	}{
		{
			name: "root CODEOWNERS",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, "CODEOWNERS", "* @nesbite/core\n")
			},
			wantOwnerLen:  1,
			wantOwnerName: "@nesbite/core",
			wantWarnings:  0,
		},
		{
			name: ".github/CODEOWNERS",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, filepath.Join(".github", "CODEOWNERS"), "* @nesbite/platform\n")
			},
			wantOwnerLen:  1,
			wantOwnerName: "@nesbite/platform",
			wantWarnings:  0,
		},
		{
			name: "docs/CODEOWNERS",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, filepath.Join("docs", "CODEOWNERS"), "* @nesbite/docs\n")
			},
			wantOwnerLen:  1,
			wantOwnerName: "@nesbite/docs",
			wantWarnings:  0,
		},
		{
			name:         "no CODEOWNERS found",
			setup:        func(t *testing.T, root string) {},
			wantOwnerLen: 0,
			wantWarnings: 0,
		},
		{
			name: "root CODEOWNERS takes precedence over .github/CODEOWNERS",
			setup: func(t *testing.T, root string) {
				t.Helper()
				writeFile(t, root, "CODEOWNERS", "* @root-owner\n")
				writeFile(t, root, filepath.Join(".github", "CODEOWNERS"), "* @github-owner\n")
			},
			wantOwnerLen:  1,
			wantOwnerName: "@root-owner",
			wantWarnings:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			report := &Report{Path: root, Dependencies: make([]Dependency, 0), Owners: make([]Owner, 0)}
			scanner := CodeownersScanner{}

			if err := scanner.Scan(root, report); err != nil {
				t.Fatalf("Scan returned unexpected error: %v", err)
			}

			if len(report.Owners) != tt.wantOwnerLen {
				t.Fatalf("expected %d owners, got %d: %+v", tt.wantOwnerLen, len(report.Owners), report.Owners)
			}
			if tt.wantOwnerName != "" && report.Owners[0].Owner != tt.wantOwnerName {
				t.Errorf("expected owner %q, got %q", tt.wantOwnerName, report.Owners[0].Owner)
			}
			if len(report.Warnings) != tt.wantWarnings {
				t.Errorf("expected %d warnings, got %d: %v", tt.wantWarnings, len(report.Warnings), report.Warnings)
			}
		})
	}
}

func TestCodeownersScanner_Name(t *testing.T) {
	if got := (CodeownersScanner{}).Name(); got != "codeowners" {
		t.Errorf("expected Name() to return %q, got %q", "codeowners", got)
	}
}
