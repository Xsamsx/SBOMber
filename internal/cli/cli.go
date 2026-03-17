package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Xsamsx/SBOMber/internal/discovery"
)

const version = "0.1.0"

// Main executes the CLI and returns the exit code.
func Main(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 1
	}

	switch args[0] {
	case "version", "--version", "-v":
		_, _ = fmt.Fprintf(stdout, "sbomber %s\n", version)
		return 0
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func runScan(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(stderr)

	if err := fs.Parse(args); err != nil {
		return 1
	}

	root := "."
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "resolve path: %v\n", err)
		return 1
	}

	repos, err := discovery.FindGitRepositories(absoluteRoot)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "scan repositories: %v\n", err)
		return 1
	}

	if len(repos) == 0 {
		_, _ = fmt.Fprintf(stdout, "No repositories found under %s\n", absoluteRoot)
		return 0
	}

	plural := "repositories"
	if len(repos) == 1 {
		plural = "repository"
	}

	_, _ = fmt.Fprintf(stdout, "Found %d %s under %s\n", len(repos), plural, absoluteRoot)
	for _, repo := range repos {
		_, _ = fmt.Fprintf(stdout, "- %s  %s\n", repo.Name, repo.Path)
	}

	return 0
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `SBOMber scans workspaces of local Git repositories.

Usage:
  sbomber scan [path]
  sbomber version

Examples:
  sbomber scan .
  sbomber scan ../workspace
`)
}
