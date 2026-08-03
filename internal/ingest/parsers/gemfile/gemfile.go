// Package gemfile parses Gemfile manifests (RubyGems ecosystem) using a
// small line-based DSL parser, similar in spirit to internal/ingest/parsers/
// pip's requirements.txt parser and internal/ingest/parsers/codeowners.
// Gemfile is a real Ruby script (evaluated via `bundle install`), but for
// dependency extraction purposes only the small, well-known subset of its
// DSL needs to be understood: `gem`, `group ... do ... end`, and a handful
// of directives that carry no dependency information.
package gemfile

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

// quotedStringPattern matches a single- or double-quoted string literal and
// captures its content in either group 1 (single-quoted) or group 2
// (double-quoted).
var quotedStringPattern = regexp.MustCompile(`'([^']*)'|"([^"]*)"`)

// symbolPattern matches a Ruby symbol, e.g. ":development" or ":test".
var symbolPattern = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

// sourceOptionPattern matches the :git, :path, and :github gem options in
// either hash-rocket ("git: 'url'") or old ("=> :git") symbol form. A gem
// declared with any of these options is not resolved from RubyGems, so it
// has no meaningful version constraint (documented limitation).
var sourceOptionPattern = regexp.MustCompile(`(^|[\s,])(:git\b|git:|:path\b|path:|:github\b|github:)`)

// ParseGemfile parses the raw content of a Gemfile and returns a slice of
// ParsedDep entries. It is a pure function with no side effects,
// implemented with the standard library only (no Ruby interpreter, no
// third-party DSL parser).
//
// Recognized:
//   - `gem 'name'` — a direct dependency with no version constraint.
//   - `gem 'name', '~> 1.0'` — a direct dependency with a version
//     constraint. If more than one version-constraint argument is given
//     (e.g. `gem 'name', '>= 1.0', '< 2.0'`), only the first is kept.
//   - `group :development, :test do ... end` — every `gem` line inside the
//     block (including inside further nested `do ... end` blocks, e.g.
//     `platforms :ruby do ... end`) is classified as depmodel.Dev instead
//     of depmodel.Direct. Groups other than :development/:test (e.g.
//     :production) do not change the classification.
//   - Both single-quoted ('name') and double-quoted ("name") strings.
//   - `git:`, `path:`, and `github:` gem options (and their `:git`/`:path`/
//     `:github` symbol forms) — the gem name is still parsed, but its
//     version is left empty because it is not resolved from RubyGems
//     (documented limitation).
//
// Skipped: blank lines, full-line and inline comments ("#..."), and
// non-gem directives such as `source`, `ruby`, `git_source`, and any other
// line that is not a `gem` call, a `group ... do` block opener, a nested
// `... do` block opener, or a bare `end`.
func ParseGemfile(data []byte, sourcePath string) []depmodel.ParsedDep {
	result := make([]depmodel.ParsedDep, 0)

	// groupStack tracks whether each currently open `do ... end` block is a
	// dev group (true) or not (false). Nested blocks (e.g. `platforms :ruby
	// do ... end` inside `group :test do ... end`) inherit the state of the
	// block they are nested in.
	var groupStack []bool

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		trimmed := strings.TrimSpace(stripComment(scanner.Text()))
		if trimmed == "" {
			continue
		}

		switch {
		case trimmed == "end":
			if len(groupStack) > 0 {
				groupStack = groupStack[:len(groupStack)-1]
			}
		case strings.HasPrefix(trimmed, "group") && strings.HasSuffix(trimmed, "do"):
			groupStack = append(groupStack, isDevGroupLine(trimmed))
		case strings.HasSuffix(trimmed, " do") || trimmed == "do":
			// A non-group block (e.g. "platforms :ruby do"). Inherit the
			// enclosing block's dev/direct classification.
			groupStack = append(groupStack, currentIsDev(groupStack))
		case strings.HasPrefix(trimmed, "gem "):
			if dep, ok := parseGemLine(trimmed, sourcePath, currentIsDev(groupStack)); ok {
				result = append(result, dep)
			}
		default:
			// source, ruby, git_source, gemspec, platforms(...), and any
			// other directive carry no dependency information — skip.
		}
	}

	return result
}

// stripComment removes a trailing "#..." comment from line, respecting
// quoted strings so a '#' inside a version constraint or URL is never
// treated as a comment marker.
func stripComment(line string) string {
	var quote rune
	for i, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '#':
			return line[:i]
		}
	}
	return line
}

// currentIsDev reports whether the innermost currently open block (if any)
// is a dev group. An empty stack means "not inside any block" — direct.
func currentIsDev(stack []bool) bool {
	if len(stack) == 0 {
		return false
	}
	return stack[len(stack)-1]
}

// isDevGroupLine reports whether a `group ...` line names :development or
// :test among its group symbols.
func isDevGroupLine(trimmed string) bool {
	for _, m := range symbolPattern.FindAllStringSubmatch(trimmed, -1) {
		if m[1] == "development" || m[1] == "test" {
			return true
		}
	}
	return false
}

// parseGemLine parses a single `gem ...` line into a ParsedDep. ok is false
// when the line has no quoted gem name (malformed input) and should be
// skipped.
func parseGemLine(trimmed, sourcePath string, isDev bool) (depmodel.ParsedDep, bool) {
	matches := quotedStringPattern.FindAllStringSubmatch(trimmed, -1)
	if len(matches) == 0 {
		return depmodel.ParsedDep{}, false
	}

	name := quotedValue(matches[0])
	if name == "" {
		return depmodel.ParsedDep{}, false
	}

	version := ""
	if len(matches) > 1 {
		version = quotedValue(matches[1])
	}
	if sourceOptionPattern.MatchString(trimmed) {
		version = ""
	}

	depType := depmodel.Direct
	if isDev {
		depType = depmodel.Dev
	}

	return depmodel.ParsedDep{
		Ecosystem:  depmodel.EcosystemRubyGems,
		Name:       name,
		Version:    version,
		DepType:    depType,
		SourceFile: sourcePath,
	}, true
}

// quotedValue returns the captured content of a quotedStringPattern match,
// regardless of whether it was single- or double-quoted.
func quotedValue(m []string) string {
	if m[1] != "" {
		return m[1]
	}
	return m[2]
}
