package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestRunScan_defaultFormatProducesValidJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app","dependencies":{"react":"19.0.0"}}`)
	writeFile(t, filepath.Join(dir, "CODEOWNERS"), "* @nesbite/core\n")

	var stdout, stderr bytes.Buffer
	code := runScan([]string{"--path", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}

	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\noutput: %s", err, stdout.String())
	}
	deps, ok := decoded["dependencies"].([]any)
	if !ok || len(deps) != 1 {
		t.Errorf("expected 1 dependency in output, got %+v", decoded["dependencies"])
	}
}

func TestRunScan_tableFormatProducesReadableOutput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app","dependencies":{"react":"19.0.0"}}`)

	var stdout, stderr bytes.Buffer
	code := runScan([]string{"--path", dir, "--format", "table"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "react") {
		t.Errorf("expected table output to mention \"react\", got:\n%s", stdout.String())
	}
}

func TestRunScan_defaultPathIsCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app","dependencies":{"react":"19.0.0"}}`)

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir into fixture dir: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := runScan(nil, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "react") {
		t.Errorf("expected default scan (no --path) to scan cwd, got:\n%s", stdout.String())
	}
}

func TestRunScan_nonexistentPathExitsNonZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runScan([]string{"--path", "/definitely/does/not/exist/atlas-scan-test"}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("expected non-zero exit code for nonexistent path")
	}
	if stderr.Len() == 0 {
		t.Error("expected an error message on stderr")
	}
}

func TestRunScan_pathIsAFileExitsNonZero(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir.txt")
	writeFile(t, filePath, "hello")

	var stdout, stderr bytes.Buffer
	code := runScan([]string{"--path", filePath}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("expected non-zero exit code when --path is a file, not a directory")
	}
	if stderr.Len() == 0 {
		t.Error("expected an error message on stderr")
	}
}

func TestRunScan_malformedPackageJSONWarnsButExitsZero(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{not valid json`)

	var stdout, stderr bytes.Buffer
	code := runScan([]string{"--path", dir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0 for malformed package.json (warning, not fatal), got %d", code)
	}

	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	warnings, ok := decoded["warnings"].([]any)
	if !ok || len(warnings) != 1 {
		t.Errorf("expected 1 warning in output, got %+v", decoded["warnings"])
	}
}
