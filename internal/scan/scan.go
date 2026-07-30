// Package scan implements the core scanning engine behind the `atlas scan`
// CLI command. It discovers ecosystem-specific data (dependencies, ownership)
// in a single directory, offline, using pluggable EcosystemScanner
// implementations. Only Apache 2.0 code (internal/ingest/parsers/...) is
// imported here, so this package is safe for the CLI binary to statically
// link. No BSL-scoped packages may be imported from internal/scan.
package scan

import "fmt"

// EcosystemScanner scans a directory for ecosystem-specific data and appends
// its findings directly to the shared Report. A scanner is responsible for
// its own file discovery and per-file error isolation (e.g. a single
// malformed file should become a warning on the report, not an aborted
// scan). Scan should only return a non-nil error for a failure that prevents
// the scanner from doing any useful work at all (e.g. the directory itself
// cannot be walked).
type EcosystemScanner interface {
	// Name returns a short, stable identifier for the scanner (e.g. "npm",
	// "codeowners"), used to prefix warning messages.
	Name() string
	// Scan inspects dir and appends any findings to report.
	Scan(dir string, report *Report) error
}

// Report is the aggregated result of running one or more EcosystemScanners
// against a directory. It is the stable JSON contract for `atlas scan`
// output.
type Report struct {
	Path         string       `json:"path"`
	Dependencies []Dependency `json:"dependencies"`
	Owners       []Owner      `json:"owners"`
	Warnings     []string     `json:"warnings,omitempty"`
}

// Dependency is a single dependency entry discovered by an ecosystem scanner.
type Dependency struct {
	Ecosystem  string `json:"ecosystem"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	DepType    string `json:"dep_type"`
	SourceFile string `json:"source_file"`
}

// Owner is a single (pattern, owner) pair discovered by the CODEOWNERS
// scanner.
type Owner struct {
	Pattern    string `json:"pattern"`
	Owner      string `json:"owner"`
	OwnerType  string `json:"owner_type"`
	LineNumber int    `json:"line_number"`
}

// Run executes each scanner against dir, in order, and returns the
// aggregated Report. A scanner error is recorded as a warning on the Report
// rather than aborting the scan: one broken ecosystem must never prevent
// the others from reporting their findings.
func Run(dir string, scanners []EcosystemScanner) (*Report, error) {
	report := &Report{
		Path:         dir,
		Dependencies: make([]Dependency, 0),
		Owners:       make([]Owner, 0),
	}

	for _, scanner := range scanners {
		if err := scanner.Scan(dir, report); err != nil {
			addWarning(report, scanner.Name(), "%v", err)
		}
	}

	return report, nil
}

// addWarning appends a formatted, scanner-prefixed warning to report. It
// centralizes the "{scanner}: {message}" format used by every
// EcosystemScanner implementation, so an individual scanner never needs to
// hand-roll the prefix itself.
func addWarning(report *Report, scannerName, format string, args ...any) {
	report.Warnings = append(report.Warnings, fmt.Sprintf(scannerName+": "+format, args...))
}
