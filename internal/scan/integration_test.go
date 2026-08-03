package scan

import "testing"

// TestRun_endToEndAgainstTestdata exercises the full scan pipeline
// (NpmScanner + ComposerScanner + GoModScanner + PipScanner +
// CargoScanner + MavenScanner + PyprojectScanner + GemfileScanner +
// CodeownersScanner via Run) against a real fixture directory under
// testdata/, rather than mocked scanners or t.TempDir fixtures. This is the
// integration test required by the PR tasks: it verifies the scanners, the
// parsers they depend on, and Report aggregation all work together
// correctly end-to-end.
func TestRun_endToEndAgainstTestdata(t *testing.T) {
	scanners := []EcosystemScanner{NpmScanner{}, NewComposerScanner(), NewGoModScanner(), NewPipScanner(), NewCargoScanner(), NewMavenScanner(), NewPyprojectScanner(), NewGemfileScanner(), CodeownersScanner{}}

	report, err := Run("testdata/sample-repo", scanners)
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	if len(report.Warnings) != 0 {
		t.Errorf("expected no warnings for a well-formed fixture, got %+v", report.Warnings)
	}

	if len(report.Dependencies) != 13 {
		t.Fatalf("expected 13 dependencies (react + vitest + monolog/monolog + uuid + x/sync + requests + serde + guava + junit + click + pytest + rails + rspec-rails), got %+v", report.Dependencies)
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
	uuidDep, ok := names["github.com/google/uuid"]
	if !ok {
		t.Fatal("expected a \"github.com/google/uuid\" dependency in the report")
	}
	if uuidDep.Version != "v1.6.0" || uuidDep.DepType != "dep" || uuidDep.Ecosystem != "Go" {
		t.Errorf("unexpected github.com/google/uuid dependency fields: %+v", uuidDep)
	}
	xsync, ok := names["golang.org/x/sync"]
	if !ok {
		t.Fatal("expected a \"golang.org/x/sync\" dependency in the report")
	}
	if xsync.Version != "v0.17.0" || xsync.DepType != "indirect" || xsync.Ecosystem != "Go" {
		t.Errorf("unexpected golang.org/x/sync dependency fields: %+v", xsync)
	}
	requests, ok := names["requests"]
	if !ok {
		t.Fatal("expected a \"requests\" dependency in the report")
	}
	if requests.Version != "==2.28.1" || requests.DepType != "dep" || requests.Ecosystem != "PyPI" {
		t.Errorf("unexpected requests dependency fields: %+v", requests)
	}
	serde, ok := names["serde"]
	if !ok {
		t.Fatal("expected a \"serde\" dependency in the report")
	}
	if serde.Version != "1.0" || serde.DepType != "dep" || serde.Ecosystem != "crates.io" {
		t.Errorf("unexpected serde dependency fields: %+v", serde)
	}
	guava, ok := names["com.google.guava:guava"]
	if !ok {
		t.Fatal("expected a \"com.google.guava:guava\" dependency in the report")
	}
	if guava.Version != "31.1-jre" || guava.DepType != "dep" || guava.Ecosystem != "Maven" {
		t.Errorf("unexpected com.google.guava:guava dependency fields: %+v", guava)
	}
	junitDep, ok := names["junit:junit"]
	if !ok {
		t.Fatal("expected a \"junit:junit\" dependency in the report")
	}
	if junitDep.Version != "4.13.2" || junitDep.DepType != "devDep" || junitDep.Ecosystem != "Maven" {
		t.Errorf("unexpected junit:junit dependency fields: %+v", junitDep)
	}
	click, ok := names["click"]
	if !ok {
		t.Fatal("expected a \"click\" dependency in the report")
	}
	if click.Version != ">=8.1.0" || click.DepType != "dep" || click.Ecosystem != "PyPI" {
		t.Errorf("unexpected click dependency fields: %+v", click)
	}
	pytest, ok := names["pytest"]
	if !ok {
		t.Fatal("expected a \"pytest\" dependency in the report")
	}
	if pytest.Version != ">=7.0.0" || pytest.DepType != "optional" || pytest.Ecosystem != "PyPI" {
		t.Errorf("unexpected pytest dependency fields: %+v", pytest)
	}
	rails, ok := names["rails"]
	if !ok {
		t.Fatal("expected a \"rails\" dependency in the report")
	}
	if rails.Version != "~> 7.1" || rails.DepType != "dep" || rails.Ecosystem != "RubyGems" {
		t.Errorf("unexpected rails dependency fields: %+v", rails)
	}
	rspecRails, ok := names["rspec-rails"]
	if !ok {
		t.Fatal("expected a \"rspec-rails\" dependency in the report")
	}
	if rspecRails.Version != "" || rspecRails.DepType != "devDep" || rspecRails.Ecosystem != "RubyGems" {
		t.Errorf("unexpected rspec-rails dependency fields: %+v", rspecRails)
	}

	if len(report.Owners) != 2 {
		t.Fatalf("expected 2 owner entries, got %+v", report.Owners)
	}
	if report.Owners[0].Pattern != "*" || report.Owners[0].Owner != "@nesbite/core" {
		t.Errorf("unexpected first owner entry: %+v", report.Owners[0])
	}
}
