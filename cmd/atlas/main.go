package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/nesbite/atlas/internal/scan"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("atlas %s\n", version)
	case "scan":
		os.Exit(runScan(os.Args[2:], os.Stdout, os.Stderr))
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// runScan parses the `scan` subcommand's flags, runs the scan engine, and
// writes the formatted report to stdout. It returns the process exit code
// rather than calling os.Exit directly, so it can be unit tested without
// spawning the compiled binary.
//
// Only truly fatal conditions (an invalid --path, or an invalid --format)
// produce a non-zero exit code. A malformed package.json or an unreadable
// CODEOWNERS file is recorded as a warning on the report — the scan still
// completes and exits 0.
func runScan(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("path", ".", "directory to scan (default: current directory)")
	format := fs.String("format", "json", `output format: "json" or "table"`)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	info, err := os.Stat(*path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "directory not found: %s\n", *path)
		return 1
	}
	if !info.IsDir() {
		_, _ = fmt.Fprintf(stderr, "not a directory: %s\n", *path)
		return 1
	}

	scanners := []scan.EcosystemScanner{scan.NpmScanner{}, scan.NewComposerScanner(), scan.NewGoModScanner(), scan.NewPipScanner(), scan.NewCargoScanner(), scan.NewMavenScanner(), scan.NewPyprojectScanner(), scan.NewGemfileScanner(), scan.CodeownersScanner{}}
	report, err := scan.Run(*path, scanners)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "scan failed: %v\n", err)
		return 1
	}

	switch *format {
	case "json":
		if err := scan.FormatJSON(stdout, report); err != nil {
			_, _ = fmt.Fprintf(stderr, "failed to format output: %v\n", err)
			return 1
		}
	case "table":
		if err := scan.FormatTable(stdout, report); err != nil {
			_, _ = fmt.Fprintf(stderr, "failed to format output: %v\n", err)
			return 1
		}
	default:
		_, _ = fmt.Fprintf(stderr, "invalid format: %s (must be \"json\" or \"table\")\n", *format)
		return 1
	}

	return 0
}

func printUsage() {
	fmt.Println(`Atlas CLI - Engineering Intelligence

Usage:
  atlas <command>

Commands:
  scan       Scan the current directory for dependencies
  version    Print the CLI version
  help       Show this help message

Learn more: https://github.com/nesbite/atlas`)
}
