package parser

import (
	"bytes"
	"strings"
)

// ParsedOwner represents one (pattern, owner) pair extracted from a CODEOWNERS file.
// A line with N owners produces N ParsedOwner entries, all sharing the same
// Pattern and LineNumber.
type ParsedOwner struct {
	Pattern    string // CODEOWNERS path pattern, e.g. "src/api/**"
	Owner      string // raw entry: "@org/team-name", "@username", or "user@example.com"
	OwnerType  string // "team", "user", or "email"
	LineNumber int    // 1-based line number for traceability
}

// utf8BOM is the UTF-8 byte order mark.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ParseCODEOWNERS parses the raw content of a CODEOWNERS file and returns
// a slice of ParsedOwner entries. One entry per (pattern, owner) pair —
// a line with 3 owners produces 3 entries. Pure function, no side effects.
// Invalid or empty content returns an empty (non-nil) slice.
func ParseCODEOWNERS(content []byte) []ParsedOwner {
	result := make([]ParsedOwner, 0)

	if len(content) == 0 {
		return result
	}

	// Strip leading UTF-8 BOM if present.
	content = bytes.TrimPrefix(content, utf8BOM)

	// Split on newlines; strings.TrimSpace per line handles \r\n (CRLF).
	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		lineNum++ // convert to 1-based

		// Trim surrounding whitespace (handles trailing \r from CRLF).
		line = strings.TrimSpace(line)

		// Skip blank lines.
		if line == "" {
			continue
		}

		// Skip full-line comments.
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Split into fields by whitespace.
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		pattern := fields[0]
		owners := fields[1:]

		for _, owner := range owners {
			// Stop at inline comment token.
			if strings.HasPrefix(owner, "#") {
				break
			}
			result = append(result, ParsedOwner{
				Pattern:    pattern,
				Owner:      owner,
				OwnerType:  detectOwnerType(owner),
				LineNumber: lineNum,
			})
		}
	}

	return result
}

// detectOwnerType determines the owner type from the raw owner string.
//
//   - "@org/team" (starts with "@" and contains "/" after the "@") → "team"
//   - "@username" (starts with "@", no "/" after) → "user"
//   - "user@example.com" (contains "@" not at position 0) → "email"
//   - anything else (bare username, empty) → "user" (fallback)
func detectOwnerType(owner string) string {
	if strings.HasPrefix(owner, "@") {
		if strings.Contains(owner[1:], "/") {
			return "team"
		}
		return "user"
	}
	if strings.Contains(owner, "@") {
		return "email"
	}
	return "user"
}
