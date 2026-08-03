package scan

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// fakeParse is a stand-in Parse function used to exercise WalkDirScanner
// without depending on any real ecosystem parser. It returns one
// ParsedDep per call, encoding sourcePath into the dependency name so
// tests can assert exactly which files were discovered.
func fakeParse(_ []byte, sourcePath string) []depmodel.ParsedDep {
	return []depmodel.ParsedDep{
		{Ecosystem: "fake", Name: "dep-from-" + sourcePath, Version: "1.0.0", DepType: depmodel.Direct, SourceFile: sourcePath},
	}
}

func TestWalkDirScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		cfg          WalkConfig
		setup        func(t *testing.T, root string)
		wantDepNames []string
		wantWarnings int
	}{
		{
			name: "matches configured filename at root",
			cfg:  WalkConfig{Name: "fake", FileName: "manifest.json", Parse: fakeParse},
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "manifest.json", `{}`)
			},
			wantDepNames: []string{"dep-from-manifest.json"},
		},
		{
			name: "ignores non-matching filenames",
			cfg:  WalkConfig{Name: "fake", FileName: "manifest.json", Parse: fakeParse},
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "other.json", `{}`)
			},
			wantDepNames: nil,
		},
		{
			name: "discovers nested matches",
			cfg:  WalkConfig{Name: "fake", FileName: "manifest.json", Parse: fakeParse},
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "manifest.json", `{}`)
				writeFile(t, root, filepath.Join("a", "b", "manifest.json"), `{}`)
			},
			wantDepNames: []string{
				"dep-from-manifest.json",
				"dep-from-" + filepath.Join("a", "b", "manifest.json"),
			},
		},
		{
			name: "skips configured directories entirely",
			cfg:  WalkConfig{Name: "fake", FileName: "manifest.json", SkipDirs: []string{"vendor"}, Parse: fakeParse},
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "manifest.json", `{}`)
				writeFile(t, root, filepath.Join("vendor", "pkg", "manifest.json"), `{}`)
			},
			wantDepNames: []string{"dep-from-manifest.json"},
		},
		{
			name: "no matching files anywhere",
			cfg:  WalkConfig{Name: "fake", FileName: "manifest.json", Parse: fakeParse},
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "unrelated.txt", `hello`)
			},
			wantDepNames: nil,
		},
		{
			name: "Validate rejects invalid content and produces a warning",
			cfg:  WalkConfig{Name: "fake", FileName: "manifest.json", Parse: fakeParse, Validate: json.Valid},
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "manifest.json", `{not valid`)
			},
			wantDepNames: nil,
			wantWarnings: 1,
		},
		{
			name: "Validate accepts valid content",
			cfg:  WalkConfig{Name: "fake", FileName: "manifest.json", Parse: fakeParse, Validate: json.Valid},
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "manifest.json", `{}`)
			},
			wantDepNames: []string{"dep-from-manifest.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			report := &Report{Path: root, Dependencies: make([]Dependency, 0), Owners: make([]Owner, 0)}
			scanner := NewWalkDirScanner(tt.cfg)

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

func TestWalkDirScanner_Name(t *testing.T) {
	scanner := NewWalkDirScanner(WalkConfig{Name: "fake"})
	if got := scanner.Name(); got != "fake" {
		t.Errorf("expected Name() to return %q, got %q", "fake", got)
	}
}
