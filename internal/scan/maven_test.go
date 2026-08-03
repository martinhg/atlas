package scan

import (
	"path/filepath"
	"testing"
)

func TestMavenScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, root string)
		wantDepNames []string
		wantWarnings int
	}{
		{
			name: "pom.xml at root",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "pom.xml", `<project><dependencies><dependency>
					<groupId>com.google.guava</groupId><artifactId>guava</artifactId><version>31.1-jre</version>
				</dependency></dependencies></project>`)
			},
			wantDepNames: []string{"com.google.guava:guava"},
		},
		{
			name:         "no pom.xml anywhere",
			setup:        func(t *testing.T, root string) {},
			wantDepNames: nil,
		},
		{
			name: "nested pom.xml excluding target and .mvn",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "pom.xml", `<project><dependencies><dependency>
					<groupId>com.google.guava</groupId><artifactId>guava</artifactId><version>31.1-jre</version>
				</dependency></dependencies></project>`)
				writeFile(t, root, filepath.Join("modules", "billing", "pom.xml"), `<project><dependencies><dependency>
					<groupId>org.apache.commons</groupId><artifactId>commons-lang3</artifactId><version>3.12.0</version>
				</dependency></dependencies></project>`)
				writeFile(t, root, filepath.Join("target", "classes", "pom.xml"), `<project><dependencies><dependency>
					<groupId>should-not</groupId><artifactId>appear</artifactId><version>1.0.0</version>
				</dependency></dependencies></project>`)
				writeFile(t, root, filepath.Join(".mvn", "wrapper", "pom.xml"), `<project><dependencies><dependency>
					<groupId>also-should-not</groupId><artifactId>appear</artifactId><version>1.0.0</version>
				</dependency></dependencies></project>`)
			},
			wantDepNames: []string{"com.google.guava:guava", "org.apache.commons:commons-lang3"},
		},
		{
			name: "malformed pom.xml produces a warning and no dependencies",
			setup: func(t *testing.T, root string) {
				writeFile(t, root, "pom.xml", "this is not valid xml {{{")
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
			scanner := NewMavenScanner()

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

func TestMavenScanner_Name(t *testing.T) {
	if got := NewMavenScanner().Name(); got != "maven" {
		t.Errorf("expected Name() to return %q, got %q", "maven", got)
	}
}
