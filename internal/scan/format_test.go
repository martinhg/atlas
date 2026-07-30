package scan

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatJSON_producesValidParseableJSON(t *testing.T) {
	report := &Report{
		Path: "/repo",
		Dependencies: []Dependency{
			{Ecosystem: "npm", Name: "react", Version: "19.0.0", DepType: "dep", SourceFile: "package.json"},
		},
		Owners: []Owner{
			{Pattern: "*", Owner: "@nesbite/core", OwnerType: "team", LineNumber: 1},
		},
	}

	var buf bytes.Buffer
	if err := FormatJSON(&buf, report); err != nil {
		t.Fatalf("FormatJSON returned unexpected error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	if _, ok := decoded["dependencies"]; !ok {
		t.Error("expected top-level \"dependencies\" key")
	}
	if _, ok := decoded["owners"]; !ok {
		t.Error("expected top-level \"owners\" key")
	}
}

func TestFormatJSON_emptyReportProducesEmptyArrays(t *testing.T) {
	report := &Report{
		Path:         "/repo",
		Dependencies: []Dependency{},
		Owners:       []Owner{},
	}

	var buf bytes.Buffer
	if err := FormatJSON(&buf, report); err != nil {
		t.Fatalf("FormatJSON returned unexpected error: %v", err)
	}

	var decoded Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(decoded.Dependencies) != 0 {
		t.Errorf("expected empty dependencies, got %+v", decoded.Dependencies)
	}
	if len(decoded.Owners) != 0 {
		t.Errorf("expected empty owners, got %+v", decoded.Owners)
	}
}

func TestFormatJSON_isIndented(t *testing.T) {
	report := &Report{Path: "/repo", Dependencies: []Dependency{}, Owners: []Owner{}}

	var buf bytes.Buffer
	if err := FormatJSON(&buf, report); err != nil {
		t.Fatalf("FormatJSON returned unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "\n  ") {
		t.Errorf("expected indented JSON output, got: %s", buf.String())
	}
}

func TestFormatTable_containsDependencyAndOwnerColumns(t *testing.T) {
	report := &Report{
		Path: "/repo",
		Dependencies: []Dependency{
			{Ecosystem: "npm", Name: "react", Version: "19.0.0", DepType: "dep", SourceFile: "package.json"},
		},
		Owners: []Owner{
			{Pattern: "*", Owner: "@nesbite/core", OwnerType: "team", LineNumber: 1},
		},
	}

	var buf bytes.Buffer
	if err := FormatTable(&buf, report); err != nil {
		t.Fatalf("FormatTable returned unexpected error: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"NAME", "VERSION", "react", "19.0.0", "PATTERN", "OWNER", "@nesbite/core"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected table output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestFormatTable_includesWarningsWhenPresent(t *testing.T) {
	report := &Report{
		Path:         "/repo",
		Dependencies: []Dependency{},
		Owners:       []Owner{},
		Warnings:     []string{"npm: invalid JSON in package.json"},
	}

	var buf bytes.Buffer
	if err := FormatTable(&buf, report); err != nil {
		t.Fatalf("FormatTable returned unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "npm: invalid JSON in package.json") {
		t.Errorf("expected warning to appear in table output, got:\n%s", buf.String())
	}
}

func TestFormatTable_emptyReportShowsNoData(t *testing.T) {
	report := &Report{
		Path:         "/repo",
		Dependencies: []Dependency{},
		Owners:       []Owner{},
	}

	var buf bytes.Buffer
	if err := FormatTable(&buf, report); err != nil {
		t.Fatalf("FormatTable returned unexpected error: %v", err)
	}

	out := strings.ToLower(buf.String())
	if !strings.Contains(out, "no data") {
		t.Errorf("expected empty report table to mention \"no data\", got:\n%s", buf.String())
	}
}
