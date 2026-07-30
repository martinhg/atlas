package scan

import (
	"os"
	"path/filepath"

	"github.com/nesbite/atlas/internal/ingest/parsers/codeowners"
)

// codeownersCandidatePaths are the three well-known locations where a
// CODEOWNERS file may live, checked in order. The first one found wins
// (matching the server-side discovery behavior in internal/ownership).
var codeownersCandidatePaths = []string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS"}

// CodeownersScanner discovers and parses a CODEOWNERS file within a
// directory, checking the standard candidate locations in order.
type CodeownersScanner struct{}

// Name identifies this scanner in warning messages.
func (CodeownersScanner) Name() string { return "codeowners" }

// Scan checks the standard CODEOWNERS candidate paths under dir in order,
// parses the first one found, and appends the resulting owners to
// report.Owners. No file found is not an error — it simply results in no
// owners being added. An unreadable file (e.g. a permission error) produces
// a warning on report rather than aborting the scan.
func (s CodeownersScanner) Scan(dir string, report *Report) error {
	for _, candidate := range codeownersCandidatePaths {
		path := filepath.Join(dir, candidate)

		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			addWarning(report, s.Name(), "failed to read %s: %v", candidate, err)
			return nil
		}

		for _, owner := range codeowners.ParseCODEOWNERS(content) {
			report.Owners = append(report.Owners, Owner{
				Pattern:    owner.Pattern,
				Owner:      owner.Owner,
				OwnerType:  owner.OwnerType,
				LineNumber: owner.LineNumber,
			})
		}

		return nil
	}

	return nil
}
