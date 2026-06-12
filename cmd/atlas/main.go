package main

import (
	"fmt"
	"os"
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
		fmt.Println("scanning current directory...")
		fmt.Println("(not implemented yet)")
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
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
