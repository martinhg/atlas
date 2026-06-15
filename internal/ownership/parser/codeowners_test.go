package parser

import (
	"testing"
)

func TestParseCODEOWNERS(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    []ParsedOwner
	}{
		{
			name:  "empty byte slice returns empty non-nil slice",
			input: []byte{},
			want:  []ParsedOwner{},
		},
		{
			name:  "BOM only returns empty slice",
			input: []byte{0xEF, 0xBB, 0xBF},
			want:  []ParsedOwner{},
		},
		{
			name:  "whitespace only returns empty slice",
			input: []byte("   \n  \n\t\n"),
			want:  []ParsedOwner{},
		},
		{
			name:  "comment only file returns empty slice",
			input: []byte("# This is a comment\n# Another comment\n"),
			want:  []ParsedOwner{},
		},
		{
			name:  "pattern with no owners produces zero entries",
			input: []byte("src/legacy/\n"),
			want:  []ParsedOwner{},
		},
		{
			name:  "double-star pattern with user owner",
			input: []byte("** @owner\n"),
			want: []ParsedOwner{
				{Pattern: "**", Owner: "@owner", OwnerType: "user", LineNumber: 1},
			},
		},
		{
			name:  "team owner detected by at-slash",
			input: []byte("src/ @org/backend-team\n"),
			want: []ParsedOwner{
				{Pattern: "src/", Owner: "@org/backend-team", OwnerType: "team", LineNumber: 1},
			},
		},
		{
			name:  "user owner detected by at-prefix without slash",
			input: []byte("*.go @username\n"),
			want: []ParsedOwner{
				{Pattern: "*.go", Owner: "@username", OwnerType: "user", LineNumber: 1},
			},
		},
		{
			name:  "email owner detected by at in middle",
			input: []byte("docs/ user@example.com\n"),
			want: []ParsedOwner{
				{Pattern: "docs/", Owner: "user@example.com", OwnerType: "email", LineNumber: 1},
			},
		},
		{
			name:  "multi-owner line produces one entry per owner",
			input: []byte("*.go @a @org/b user@c.com\n"),
			want: []ParsedOwner{
				{Pattern: "*.go", Owner: "@a", OwnerType: "user", LineNumber: 1},
				{Pattern: "*.go", Owner: "@org/b", OwnerType: "team", LineNumber: 1},
				{Pattern: "*.go", Owner: "user@c.com", OwnerType: "email", LineNumber: 1},
			},
		},
		{
			name:  "inline comment stripped after owners",
			input: []byte("*.go @go-owner # all Go code\n"),
			want: []ParsedOwner{
				{Pattern: "*.go", Owner: "@go-owner", OwnerType: "user", LineNumber: 1},
			},
		},
		{
			name:  "windows CRLF line endings parsed same as LF",
			input: []byte("*.go @owner\r\n"),
			want: []ParsedOwner{
				{Pattern: "*.go", Owner: "@owner", OwnerType: "user", LineNumber: 1},
			},
		},
		{
			name:  "BOM prefix stripped on valid content",
			input: append([]byte{0xEF, 0xBB, 0xBF}, []byte("src/ @team\n")...),
			want: []ParsedOwner{
				{Pattern: "src/", Owner: "@team", OwnerType: "user", LineNumber: 1},
			},
		},
		{
			name:  "mixed content comments blanks and two rules",
			input: []byte("# Owner file\n\n*.go @go-owner\n# another comment\n\ndocs/ @doc-team\n"),
			want: []ParsedOwner{
				{Pattern: "*.go", Owner: "@go-owner", OwnerType: "user", LineNumber: 3},
				{Pattern: "docs/", Owner: "@doc-team", OwnerType: "user", LineNumber: 6},
			},
		},
		{
			name:  "multi-owner BDD scenario from spec",
			input: []byte("*.go @go-team @go-lead\nsrc/ @src-team\n"),
			want: []ParsedOwner{
				{Pattern: "*.go", Owner: "@go-team", OwnerType: "user", LineNumber: 1},
				{Pattern: "*.go", Owner: "@go-lead", OwnerType: "user", LineNumber: 1},
				{Pattern: "src/", Owner: "@src-team", OwnerType: "user", LineNumber: 2},
			},
		},
		{
			name:  "tab-separated fields parsed correctly",
			input: []byte("*.go\t@owner\n"),
			want: []ParsedOwner{
				{Pattern: "*.go", Owner: "@owner", OwnerType: "user", LineNumber: 1},
			},
		},
		{
			name:  "leading and trailing whitespace on line trimmed",
			input: []byte("  *.go @owner  \n"),
			want: []ParsedOwner{
				{Pattern: "*.go", Owner: "@owner", OwnerType: "user", LineNumber: 1},
			},
		},
		{
			name:  "multiple spaces between pattern and owners handled",
			input: []byte("*.go   @owner1   @org/team2\n"),
			want: []ParsedOwner{
				{Pattern: "*.go", Owner: "@owner1", OwnerType: "user", LineNumber: 1},
				{Pattern: "*.go", Owner: "@org/team2", OwnerType: "team", LineNumber: 1},
			},
		},
		{
			name:  "BOM only content (no valid rules) returns empty slice",
			input: []byte{0xEF, 0xBB, 0xBF, '\n'},
			want:  []ParsedOwner{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCODEOWNERS(tt.input)

			// Result must never be nil.
			if got == nil {
				t.Fatal("ParseCODEOWNERS returned nil, want non-nil slice")
			}

			if len(got) != len(tt.want) {
				t.Fatalf("len(got) = %d, want %d\ngot:  %+v\nwant: %+v", len(got), len(tt.want), got, tt.want)
			}

			for i, w := range tt.want {
				g := got[i]
				if g.Pattern != w.Pattern {
					t.Errorf("[%d] Pattern = %q, want %q", i, g.Pattern, w.Pattern)
				}
				if g.Owner != w.Owner {
					t.Errorf("[%d] Owner = %q, want %q", i, g.Owner, w.Owner)
				}
				if g.OwnerType != w.OwnerType {
					t.Errorf("[%d] OwnerType = %q, want %q", i, g.OwnerType, w.OwnerType)
				}
				if g.LineNumber != w.LineNumber {
					t.Errorf("[%d] LineNumber = %d, want %d", i, g.LineNumber, w.LineNumber)
				}
			}
		})
	}
}

func TestDetectOwnerType(t *testing.T) {
	tests := []struct {
		owner string
		want  string
	}{
		{"@org/team-name", "team"},
		{"@org/backend", "team"},
		{"@username", "user"},
		{"@a", "user"},
		{"user@example.com", "email"},
		{"dev@company.io", "email"},
		{"bareusername", "user"}, // fallback
		{"", "user"},            // fallback for empty
	}

	for _, tt := range tests {
		t.Run(tt.owner, func(t *testing.T) {
			got := detectOwnerType(tt.owner)
			if got != tt.want {
				t.Errorf("detectOwnerType(%q) = %q, want %q", tt.owner, got, tt.want)
			}
		})
	}
}
