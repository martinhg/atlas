package scan

import "testing"

// TestRun_endToEndAgainstTestdata exercises the full scan pipeline
// (NpmScanner + ComposerScanner + CodeownersScanner via Run) against a real
// fixture directory under testdata/, rather than mocked scanners or
// t.TempDir fixtures. This is the integration test required by the PR 3
// tasks: it verifies the scanners, the parsers they depend on, and Report
// aggregation all work together correctly end-to-end.
func TestRun_endToEndAgainstTestdata(t *testing.T) {
	scanners := []EcosystemScanner{NpmScanner{}, NewComposerScanner(), CodeownersScanner{}}

	report, err := Run("testdata/sample-repo", scanners)
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	if len(report.Warnings) != 0 {
		t.Errorf("expected no warnings for a well-formed fixture, got %+v", report.Warnings)
	}

	if len(report.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies (react + vitest + monolog/monolog), got %+v", report.Dependencies)
	}
	names := map[string]Dependency{}
	for _, dep := range report.Dependencies {
		names[dep.Name] = dep
	}
	react, ok := names["react"]
	if !ok {
		t.Fatal("expected a \"react\" dependency in the report")
	}
	if react.Version != "19.0.0" || react.DepType != "dep" || react.Ecosystem != "npm" {
		t.Errorf("unexpected react dependency fields: %+v", react)
	}
	if _, ok := names["vitest"]; !ok {
		t.Error("expected a \"vitest\" devDependency in the report")
	}
	monolog, ok := names["monolog/monolog"]
	if !ok {
		t.Fatal("expected a \"monolog/monolog\" dependency in the report")
	}
	if monolog.Version != "^2.9" || monolog.DepType != "dep" || monolog.Ecosystem != "Packagist" {
		t.Errorf("unexpected monolog/monolog dependency fields: %+v", monolog)
	}

	if len(report.Owners) != 2 {
		t.Fatalf("expected 2 owner entries, got %+v", report.Owners)
	}
	if report.Owners[0].Pattern != "*" || report.Owners[0].Owner != "@nesbite/core" {
		t.Errorf("unexpected first owner entry: %+v", report.Owners[0])
	}
}
