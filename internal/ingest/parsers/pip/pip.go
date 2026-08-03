// Package pip parses requirements.txt manifests (PyPI ecosystem).
package pip

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// extrasPattern matches a PEP 508 extras bracket, e.g. "[security,socks]",
// so it can be stripped from a requirement spec before the package name is
// extracted.
var extrasPattern = regexp.MustCompile(`\[[^\]]*\]`)

// versionOperators are the characters that can start a PEP 440 version
// specifier. The first occurrence of any of these characters in a
// (extras-stripped, marker-stripped) requirement spec marks the boundary
// between the package name and its raw version constraint.
const versionOperators = "=<>!~"

// ParseRequirementsTxt parses the raw content of a requirements.txt file
// and returns a slice of ParsedDep entries. It is a pure function with no
// side effects, implemented with the standard library only.
//
// Supported per line: "package==version", "package>=version",
// "package~=version", "package!=version", and a bare "package" with no
// version constraint. Extras ("package[extra1,extra2]>=version") and
// environment markers ("package>=version; python_version>=\"3.6\"") are
// stripped, keeping only the package name and raw version constraint.
//
// Skipped: blank lines, full-line comments ("#..."), inline comments
// (" #..."), option/flag lines ("-r", "-c", "-e", "-f", "--..."), and
// trailing "--hash=..." pins. Backslash line continuations are joined into
// a single logical line before parsing.
//
// Every entry uses depmodel.EcosystemPyPI and depmodel.Direct — a plain
// requirements.txt has no dev/indirect distinction (that's a documented
// v1 gap; see design.md).
func ParseRequirementsTxt(data []byte, sourcePath string) []depmodel.ParsedDep {
	result := make([]depmodel.ParsedDep, 0)

	for _, line := range joinContinuations(data) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if isOptionLine(line) {
			continue
		}

		// Strip an inline comment (a "#" preceded by whitespace).
		if idx := strings.Index(line, " #"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		// Strip trailing option flags on the same logical line, e.g.
		// "package==1.0 --hash=sha256:...".
		if idx := strings.Index(line, " --"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		// Strip the environment marker, if any ("; python_version>=...").
		if idx := strings.Index(line, ";"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		if line == "" {
			continue
		}

		// Strip the extras bracket, if any ("[security,socks]").
		line = strings.TrimSpace(extrasPattern.ReplaceAllString(line, ""))
		if line == "" {
			continue
		}

		name, version := splitNameVersion(line)
		if name == "" {
			continue
		}

		result = append(result, depmodel.ParsedDep{
			Ecosystem:  depmodel.EcosystemPyPI,
			Name:       name,
			Version:    version,
			DepType:    depmodel.Direct,
			SourceFile: sourcePath,
		})
	}

	return result
}

// joinContinuations splits data into physical lines and joins any line
// ending in a backslash with the line that follows it, producing logical
// lines. This mirrors pip's own requirements.txt line-continuation
// handling.
func joinContinuations(data []byte) []string {
	var logical []string
	var buf strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.HasSuffix(line, "\\") {
			buf.WriteString(strings.TrimSuffix(line, "\\"))
			buf.WriteString(" ")
			continue
		}
		buf.WriteString(line)
		logical = append(logical, buf.String())
		buf.Reset()
	}
	if buf.Len() > 0 {
		logical = append(logical, buf.String())
	}

	return logical
}

// isOptionLine reports whether line is a pip option/flag line rather than
// a requirement spec: a recursive include ("-r"), a constraints file
// ("-c"), an editable install ("-e"), a find-links source ("-f"), or any
// long-form flag ("--index-url", "--hash", etc.).
func isOptionLine(line string) bool {
	return strings.HasPrefix(line, "-r") ||
		strings.HasPrefix(line, "-c") ||
		strings.HasPrefix(line, "-e") ||
		strings.HasPrefix(line, "-f") ||
		strings.HasPrefix(line, "--")
}

// splitNameVersion splits a requirement spec (already stripped of extras,
// markers, and trailing flags) into a package name and a raw version
// constraint string. If no PEP 440 operator is present, version is "".
func splitNameVersion(spec string) (name, version string) {
	idx := strings.IndexAny(spec, versionOperators)
	if idx == -1 {
		return strings.TrimSpace(spec), ""
	}
	return strings.TrimSpace(spec[:idx]), strings.TrimSpace(spec[idx:])
}
