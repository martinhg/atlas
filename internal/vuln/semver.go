package vuln

import (
	"log/slog"
	"strconv"
	"strings"
)

// stripRange removes semver range prefixes (^, ~, >=, <=, >, <, =) from a
// version string and returns the bare version token.
//
// When a compound range like ">=1.2.0 <2.0.0" is supplied, only the first
// space-separated token is processed. This is a known limitation: compound
// ranges may yield false negatives at sync time. Lock-file parsing with exact
// resolved versions is deferred.
func stripRange(version string) string {
	if version == "" {
		return ""
	}

	// Split on whitespace to detect compound ranges.
	parts := strings.Fields(version)
	if len(parts) > 1 {
		slog.Warn("compound semver range detected; using first token only",
			"version", version,
			"token", parts[0],
		)
	}

	token := parts[0]

	// Strip known prefix operators in longest-first order.
	for _, prefix := range []string{">=", "<=", ">", "<", "^", "~", "="} {
		if strings.HasPrefix(token, prefix) {
			token = strings.TrimPrefix(token, prefix)
			break
		}
	}

	// Strip leading whitespace that may follow the operator (e.g. ">= 1.0.0").
	token = strings.TrimSpace(token)

	// If the operator and version were separated by whitespace (e.g. ">=  1.0.0"),
	// Fields() would have placed the bare operator in parts[0] and the version
	// in parts[1]. Take parts[1] in that case.
	if token == "" && len(parts) > 1 {
		token = strings.TrimSpace(parts[1])
		// Strip any remaining prefix from the second token too.
		for _, prefix := range []string{">=", "<=", ">", "<", "^", "~", "="} {
			if strings.HasPrefix(token, prefix) {
				token = strings.TrimPrefix(token, prefix)
				break
			}
		}
		token = strings.TrimSpace(token)
	}

	// Strip a leading 'v' that Go modules use (e.g. "v1.0.0").
	token = strings.TrimPrefix(token, "v")

	return token
}

// compareSemver compares two semver strings (major.minor.patch) numerically.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Missing segments are treated as 0 (e.g. "1.0" == "1.0.0").
// Non-numeric pre-release suffixes are ignored for MVP comparisons.
func compareSemver(a, b string) int {
	aParts := parseParts(a)
	bParts := parseParts(b)

	for i := range 3 {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

// parseParts splits a semver string into a [3]int array of [major, minor, patch].
// Any non-numeric characters after the first '-' (pre-release) are ignored.
func parseParts(version string) [3]int {
	// Strip range prefixes before parsing.
	version = stripRange(version)

	// Strip pre-release suffix (e.g. "1.2.3-alpha" → "1.2.3").
	if idx := strings.IndexByte(version, '-'); idx != -1 {
		version = version[:idx]
	}

	segments := strings.Split(version, ".")
	var parts [3]int
	for i := range 3 {
		if i < len(segments) {
			n, err := strconv.Atoi(segments[i])
			if err == nil {
				parts[i] = n
			}
		}
	}
	return parts
}

// isAffected returns true when version falls within any of the provided
// AffectedRanges. The version is stripped of range prefixes before comparison.
//
// A dependency is considered affected when:
//
//	base_version >= introduced_version AND (fixed is empty OR base_version < fixed)
func isAffected(version string, ranges []AffectedRange) bool {
	base := stripRange(version)

	for _, r := range ranges {
		if r.Introduced == "" {
			continue
		}
		// base >= introduced
		if compareSemver(base, r.Introduced) < 0 {
			continue
		}
		// base < fixed (when fixed is set)
		if r.Fixed != "" && compareSemver(base, r.Fixed) >= 0 {
			continue
		}
		return true
	}
	return false
}
