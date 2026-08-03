package scan

import (
	"errors"
	"testing"
)

// mockScanner is a test double for EcosystemScanner. scanFn is invoked with
// the dir and report passed to Scan, so tests can assert on both and mutate
// the report to simulate a real scanner's findings.
type mockScanner struct {
	name   string
	scanFn func(dir string, report *Report) error
}

func (m *mockScanner) Name() string { return m.name }

func (m *mockScanner) Scan(dir string, report *Report) error {
	return m.scanFn(dir, report)
}

func TestRun_callsEachScannerWithDir(t *testing.T) {
	var gotDirs []string

	scanners := []EcosystemScanner{
		&mockScanner{name: "first", scanFn: func(dir string, _ *Report) error {
			gotDirs = append(gotDirs, dir)
			return nil
		}},
		&mockScanner{name: "second", scanFn: func(dir string, _ *Report) error {
			gotDirs = append(gotDirs, dir)
			return nil
		}},
	}

	_, err := Run("/some/dir", scanners)
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	want := []string{"/some/dir", "/some/dir"}
	if len(gotDirs) != len(want) {
		t.Fatalf("expected %d scanner calls, got %d", len(want), len(gotDirs))
	}
	for i, dir := range gotDirs {
		if dir != want[i] {
			t.Errorf("call %d: got dir %q, want %q", i, dir, want[i])
		}
	}
}

func TestRun_aggregatesReportAcrossScanners(t *testing.T) {
	scanners := []EcosystemScanner{
		&mockScanner{name: "npm", scanFn: func(_ string, report *Report) error {
			report.Dependencies = append(report.Dependencies, Dependency{
				Ecosystem: "npm", Name: "react", Version: "19.0.0", DepType: "dep", SourceFile: "package.json",
			})
			return nil
		}},
		&mockScanner{name: "codeowners", scanFn: func(_ string, report *Report) error {
			report.Owners = append(report.Owners, Owner{
				Pattern: "*", Owner: "@nesbite/core", OwnerType: "team", LineNumber: 1,
			})
			return nil
		}},
	}

	report, err := Run("/repo", scanners)
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	if report.Path != "/repo" {
		t.Errorf("expected Path %q, got %q", "/repo", report.Path)
	}
	if len(report.Dependencies) != 1 || report.Dependencies[0].Name != "react" {
		t.Errorf("expected 1 dependency named react, got %+v", report.Dependencies)
	}
	if len(report.Owners) != 1 || report.Owners[0].Owner != "@nesbite/core" {
		t.Errorf("expected 1 owner @nesbite/core, got %+v", report.Owners)
	}
	if len(report.Warnings) != 0 {
		t.Errorf("expected no warnings, got %+v", report.Warnings)
	}
}

func TestRun_scannerErrorBecomesWarningAndDoesNotAbort(t *testing.T) {
	scanners := []EcosystemScanner{
		&mockScanner{name: "broken", scanFn: func(_ string, _ *Report) error {
			return errors.New("boom")
		}},
		&mockScanner{name: "ok", scanFn: func(_ string, report *Report) error {
			report.Owners = append(report.Owners, Owner{Pattern: "*", Owner: "@team", OwnerType: "team", LineNumber: 1})
			return nil
		}},
	}

	report, err := Run("/repo", scanners)
	if err != nil {
		t.Fatalf("Run should never return an error itself, got: %v", err)
	}

	if len(report.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %+v", report.Warnings)
	}
	if report.Warnings[0] != "broken: boom" {
		t.Errorf("expected warning %q, got %q", "broken: boom", report.Warnings[0])
	}
	// The second scanner must still have run despite the first one failing.
	if len(report.Owners) != 1 {
		t.Errorf("expected the second scanner to still contribute owners, got %+v", report.Owners)
	}
}

func TestRun_noScannersProducesEmptyNonNilReport(t *testing.T) {
	report, err := Run("/repo", nil)
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	if report.Dependencies == nil {
		t.Error("expected non-nil (empty) Dependencies slice")
	}
	if report.Owners == nil {
		t.Error("expected non-nil (empty) Owners slice")
	}
	if len(report.Dependencies) != 0 || len(report.Owners) != 0 {
		t.Errorf("expected empty report, got %+v", report)
	}
}
