package vuln

import (
	"testing"
)

// TestStripRange verifies prefix stripping for semver range notation.
func TestStripRange(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"caret prefix", "^4.17.21", "4.17.21"},
		{"tilde prefix", "~2.0.0", "2.0.0"},
		{"gte prefix", ">=1.2.3", "1.2.3"},
		{"lte prefix", "<=3.0.0", "3.0.0"},
		{"gt prefix", ">0.9.0", "0.9.0"},
		{"lt prefix", "<5.0.0", "5.0.0"},
		{"eq prefix", "=1.0.0", "1.0.0"},
		{"no prefix clean", "1.0.0", "1.0.0"},
		{"compound range takes first token", ">=1.2.0 <2.0.0", "1.2.0"},
		{"compound with caret first", "^1.0.0 ^2.0.0", "1.0.0"},
		{"empty string", "", ""},
		{"major only", "1", "1"},
		{"major.minor only", "2.1", "2.1"},
		{"tilde with pre-release", "~1.2.3-alpha", "1.2.3-alpha"},
		{"gte with space after op", ">=  1.0.0", "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripRange(tt.input)
			if got != tt.want {
				t.Errorf("stripRange(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestCompareSemver verifies numeric major.minor.patch comparison.
func TestCompareSemver(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{"equal versions", "1.0.0", "1.0.0", 0},
		{"a less than b major", "1.0.0", "2.0.0", -1},
		{"a greater than b major", "2.0.0", "1.0.0", 1},
		{"a less than b minor", "1.0.0", "1.1.0", -1},
		{"a greater than b minor", "1.2.0", "1.1.0", 1},
		{"a less than b patch", "1.0.0", "1.0.1", -1},
		{"a greater than b patch", "1.0.2", "1.0.1", 1},
		{"missing patch a", "1.0", "1.0.0", 0},
		{"missing patch b", "1.0.0", "1.0", 0},
		{"major only equal", "1", "1.0.0", 0},
		{"major only less", "1", "2.0.0", -1},
		{"large patch numbers", "1.0.100", "1.0.99", 1},
		{"zero versions", "0.0.0", "0.0.0", 0},
		{"v prefix stripped", "v1.0.0", "1.0.0", 0},
		{"empty a treated as 0", "", "0.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareSemver(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareSemver(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestIsAffected verifies version range membership logic.
func TestIsAffected(t *testing.T) {
	tests := []struct {
		name    string
		version string
		ranges  []AffectedRange
		want    bool
	}{
		{
			name:    "within range",
			version: "4.17.21",
			ranges:  []AffectedRange{{Introduced: "4.0.0", Fixed: "4.18.0"}},
			want:    true,
		},
		{
			name:    "below introduced",
			version: "3.9.9",
			ranges:  []AffectedRange{{Introduced: "4.0.0", Fixed: "4.18.0"}},
			want:    false,
		},
		{
			name:    "equal to fixed is NOT affected",
			version: "4.18.0",
			ranges:  []AffectedRange{{Introduced: "4.0.0", Fixed: "4.18.0"}},
			want:    false,
		},
		{
			name:    "above fixed",
			version: "4.18.1",
			ranges:  []AffectedRange{{Introduced: "4.0.0", Fixed: "4.18.0"}},
			want:    false,
		},
		{
			name:    "open upper bound (no fixed)",
			version: "5.0.0",
			ranges:  []AffectedRange{{Introduced: "4.0.0", Fixed: ""}},
			want:    true,
		},
		{
			name:    "exactly at introduced",
			version: "4.0.0",
			ranges:  []AffectedRange{{Introduced: "4.0.0", Fixed: "5.0.0"}},
			want:    true,
		},
		{
			name:    "empty ranges slice",
			version: "1.0.0",
			ranges:  []AffectedRange{},
			want:    false,
		},
		{
			name:    "nil ranges slice",
			version: "1.0.0",
			ranges:  nil,
			want:    false,
		},
		{
			name:    "affected by second range when first does not match",
			version: "2.5.0",
			ranges: []AffectedRange{
				{Introduced: "1.0.0", Fixed: "1.5.0"},
				{Introduced: "2.0.0", Fixed: "3.0.0"},
			},
			want: true,
		},
		{
			name:    "not in any range",
			version: "4.0.0",
			ranges: []AffectedRange{
				{Introduced: "1.0.0", Fixed: "2.0.0"},
				{Introduced: "2.5.0", Fixed: "3.0.0"},
			},
			want: false,
		},
		{
			name:    "version with caret prefix stripped",
			version: "^4.17.21",
			ranges:  []AffectedRange{{Introduced: "4.0.0", Fixed: "4.18.0"}},
			want:    true,
		},
		{
			name:    "version with tilde prefix stripped",
			version: "~1.2.3",
			ranges:  []AffectedRange{{Introduced: "1.0.0", Fixed: "2.0.0"}},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAffected(tt.version, tt.ranges)
			if got != tt.want {
				t.Errorf("isAffected(%q, %v) = %v, want %v", tt.version, tt.ranges, got, tt.want)
			}
		})
	}
}
