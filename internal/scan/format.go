package scan

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// FormatJSON writes report to w as indented, valid JSON. The output always
// includes the "dependencies" and "owners" arrays (empty arrays rather than
// null when there are no findings), matching the stable CLI JSON contract.
func FormatJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// FormatTable writes report to w as a human-readable table using
// text/tabwriter. It is not intended to be machine-parseable. Each section
// (dependencies, owners, warnings) is printed only when non-empty; a section
// with no findings prints a "(no data)" placeholder instead of an empty
// table.
func FormatTable(w io.Writer, report *Report) error {
	ew := &errWriter{w: w}

	ew.printf("Scan report for %s\n\n", report.Path)

	tw := tabwriter.NewWriter(ew, 0, 0, 2, ' ', 0)
	twErr := &errWriter{w: tw}

	ew.println("DEPENDENCIES")
	if len(report.Dependencies) == 0 {
		ew.println("  (no data)")
	} else {
		twErr.println("NAME\tVERSION\tTYPE\tECOSYSTEM\tSOURCE")
		for _, dep := range report.Dependencies {
			twErr.printf("%s\t%s\t%s\t%s\t%s\n", dep.Name, dep.Version, dep.DepType, dep.Ecosystem, dep.SourceFile)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		if twErr.err != nil {
			return twErr.err
		}
	}

	ew.println("\nOWNERS")
	if len(report.Owners) == 0 {
		ew.println("  (no data)")
	} else {
		twErr.println("PATTERN\tOWNER\tTYPE\tLINE")
		for _, owner := range report.Owners {
			twErr.printf("%s\t%s\t%s\t%d\n", owner.Pattern, owner.Owner, owner.OwnerType, owner.LineNumber)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		if twErr.err != nil {
			return twErr.err
		}
	}

	if len(report.Warnings) > 0 {
		ew.println("\nWARNINGS")
		for _, warning := range report.Warnings {
			ew.printf("  - %s\n", warning)
		}
	}

	return ew.err
}

// errWriter wraps an io.Writer and remembers the first write error
// encountered, so a long sequence of Fprintf/Fprintln calls can skip
// individual error checks (each one is a no-op once err is set) while still
// surfacing the failure to the caller at the end.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) Write(p []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}
	n, err := ew.w.Write(p)
	if err != nil {
		ew.err = err
	}
	return n, err
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func (ew *errWriter) println(args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintln(ew.w, args...)
}
