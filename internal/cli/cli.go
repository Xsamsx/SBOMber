package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
	"github.com/Xsamsx/SBOMber/internal/discovery"
	"github.com/Xsamsx/SBOMber/internal/ecosystem"
	"github.com/Xsamsx/SBOMber/internal/github"
	"github.com/Xsamsx/SBOMber/internal/golang"
	"github.com/Xsamsx/SBOMber/internal/health"
	"github.com/Xsamsx/SBOMber/internal/maven"
	"github.com/Xsamsx/SBOMber/internal/npm"
	"github.com/Xsamsx/SBOMber/internal/python"
	"github.com/Xsamsx/SBOMber/internal/remote"
	"github.com/Xsamsx/SBOMber/internal/ruby"
	"github.com/Xsamsx/SBOMber/internal/sbom"
	"github.com/Xsamsx/SBOMber/internal/supplychain"
	"github.com/Xsamsx/SBOMber/internal/verify"
	"github.com/Xsamsx/SBOMber/internal/vulnerability"
)

const version = "0.1.0"

const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorBlue  = "\033[34m"
	colorBold  = "\033[1m"

	formatCycloneDX = "cyclonedx"
	formatSPDX      = "spdx"
	formatBoth      = "both"
)

// Main executes the CLI and returns the exit code.
func Main(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		return runInteractive(stdin, stdout, stderr)
	}

	switch args[0] {
	case "version", "--version", "-v":
		_, _ = fmt.Fprintf(stdout, "sbomber %s\n", version)
		return 0
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "github":
		return runGitHubScan(args[1:], stdout, stderr)
	case "trace":
		return runTrace(args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr)
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
	format := fs.String("format", formatCycloneDX, "export format: cyclonedx, spdx, or both")
	includeVulnerabilities := fs.Bool("include-vulnerabilities", false, "scan for vulnerabilities using Grype")
	failOnVuln := fs.Bool("fail-on-vuln", false, "exit with code 1 when vulnerabilities are found (requires --include-vulnerabilities)")
	outputDir := fs.String("output", "", "custom output directory for SBOMs and reports")
	severityThreshold := fs.String("severity-threshold", "", "only report/fail on vulnerabilities at or above this severity (critical, high, medium, low)")

	args = reorderFlagsFirst(args, scanBoolFlags)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	threshold, err := vulnerability.NormalizeSeverityThreshold(*severityThreshold)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid severity threshold: %v\n", err)
		return 1
	}

	opts := scanOptions{
		format:                 "",
		includeVulnerabilities: *includeVulnerabilities,
		failOnVuln:             *failOnVuln,
		outputDir:              strings.TrimSpace(*outputDir),
		severityThreshold:      threshold,
	}

	// Collect all roots - supports multiple paths
	var roots []string
	if fs.NArg() > 0 {
		roots = fs.Args()
	} else {
		roots = []string{"."}
	}

	selectedFormat, err := normalizeExportFormat(*format)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid format: %v\n", err)
		return 1
	}
	opts.format = selectedFormat

	_, _ = fmt.Fprintf(stdout, "Selected SBOM export format: %s\n", selectedFormat)
	if threshold != "" {
		_, _ = fmt.Fprintf(stdout, "Severity threshold: %s\n", threshold)
	}
	if opts.outputDir != "" {
		_, _ = fmt.Fprintf(stdout, "Output directory: %s\n", opts.outputDir)
	}
	if *includeVulnerabilities {
		if vulnerability.IsGrypeAvailable() {
			_, _ = fmt.Fprintf(stdout, "Vulnerability scanning: enabled (Grype)\n")
		} else {
			_, _ = fmt.Fprintf(stderr, "WARNING: Vulnerability scanning requested but Grype not found in PATH\n")
			_, _ = fmt.Fprintf(stderr, "Install Grype from: https://github.com/anchore/grype\n")
		}
	}

	// Count total repos first to determine if we need batch mode
	var allRepos []struct {
		repo      discovery.Repository
		detection ecosystem.Detection
		rootPath  string
	}
	hadErrors := false

	for _, root := range roots {
		absoluteRoot, err := resolveScanRoot(root)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "resolve path %s: %v\n", root, err)
			hadErrors = true
			continue
		}

		repos, err := discovery.FindGitRepositories(absoluteRoot)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "scan repositories under %s: %v\n", absoluteRoot, err)
			hadErrors = true
			continue
		}

		for _, repo := range repos {
			detection, err := ecosystem.DetectRepository(repo.Path)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "detect ecosystem for %s: %v\n", repo.Path, err)
				hadErrors = true
				continue
			}
			allRepos = append(allRepos, struct {
				repo      discovery.Repository
				detection ecosystem.Detection
				rootPath  string
			}{repo, detection, absoluteRoot})
		}
	}

	if len(allRepos) == 0 {
		if hadErrors {
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "No repositories found\n")
		return 0
	}

	// Use batch mode if scanning multiple repos
	var batchDir string
	useBatchMode := len(allRepos) > 1

	if useBatchMode {
		scanName := filepath.Base(roots[0])
		if len(roots) > 1 {
			scanName = "multi-scan"
		}
		batchDir, err = sbom.ResolveBatchOutputDir(opts.outputDir, scanName)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "create batch output directory: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "\nBatch output folder: %s\n", batchDir)
	}

	_, _ = fmt.Fprintf(stdout, "\nFound %d repositories to scan\n", len(allRepos))

	foundVulns := false
	for _, item := range allRepos {
		repo := item.repo
		detection := item.detection

		stack := "unknown"
		if len(detection.Names) > 0 {
			names := make([]string, 0, len(detection.Names))
			for _, name := range detection.Names {
				names = append(names, string(name))
			}
			stack = strings.Join(names, ", ")
		}

		_, _ = fmt.Fprintf(stdout, "\n- %s  [%s]\n", repo.Name, stack)

		if useBatchMode {
			ok, vulnFound := printDependencySummaryBatch(stdout, stderr, repo.Name, repo.Path, detection, opts, batchDir)
			if !ok {
				hadErrors = true
			}
			if vulnFound {
				foundVulns = true
			}
		} else {
			ok, vulnFound := printDependencySummary(stdout, stderr, repo.Name, repo.Path, detection, opts)
			if !ok {
				hadErrors = true
			}
			if vulnFound {
				foundVulns = true
			}
		}
	}

	_, _ = fmt.Fprintf(stdout, "\nScan complete: %d repositories scanned\n", len(allRepos))
	if useBatchMode {
		_, _ = fmt.Fprintf(stdout, "All reports saved to: %s\n", batchDir)
	}
	if hadErrors || (opts.failOnVuln && foundVulns) {
		return 1
	}
	return 0
}

func runGitHubScan(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("github", flag.ContinueOnError)
	fs.SetOutput(stderr)
	includeHealth := fs.Bool("health", false, "include supply chain health metrics")
	includeVulns := fs.Bool("include-vulnerabilities", false, "scan for vulnerabilities using Grype")
	failOnVuln := fs.Bool("fail-on-vuln", false, "exit with code 1 when vulnerabilities are found (requires --include-vulnerabilities)")
	format := fs.String("format", formatCycloneDX, "export format: cyclonedx, spdx, or both")
	outputDirFlag := fs.String("output", "", "custom output directory for SBOMs and reports")
	severityThreshold := fs.String("severity-threshold", "", "only report/fail on vulnerabilities at or above this severity (critical, high, medium, low)")

	args = reorderFlagsFirst(args, githubBoolFlags)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() == 0 {
		_, _ = fmt.Fprintf(stderr, "Usage: sbomber github [--health] [--include-vulnerabilities] [--fail-on-vuln] [--output DIR] [--severity-threshold SEV] [--format FORMAT] <repo-url> [repo-url...]\n")
		_, _ = fmt.Fprintf(stderr, "\nExamples:\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber github https://github.com/expressjs/express\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber github --health https://github.com/lodash/lodash\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber github --include-vulnerabilities https://github.com/org/repo\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber github https://github.com/org/repo1 https://github.com/org/repo2\n")
		return 1
	}

	threshold, err := vulnerability.NormalizeSeverityThreshold(*severityThreshold)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid severity threshold: %v\n", err)
		return 1
	}

	opts := scanOptions{
		includeVulnerabilities: *includeVulns,
		failOnVuln:             *failOnVuln,
		outputDir:              strings.TrimSpace(*outputDirFlag),
		severityThreshold:      threshold,
	}

	token := os.Getenv("GITHUB_TOKEN")
	client := github.NewClient(token)

	if !client.HasToken() {
		_, _ = fmt.Fprintf(stderr, "WARNING: No GITHUB_TOKEN set. Rate limit is 60 requests/hour.\n")
		_, _ = fmt.Fprintf(stderr, "Set GITHUB_TOKEN for 5000 requests/hour.\n\n")
	}

	selectedFormat, err := normalizeExportFormat(*format)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid format: %v\n", err)
		return 1
	}
	opts.format = selectedFormat

	if threshold != "" {
		_, _ = fmt.Fprintf(stdout, "Severity threshold: %s\n", threshold)
	}
	if opts.outputDir != "" {
		_, _ = fmt.Fprintf(stdout, "Output directory: %s\n", opts.outputDir)
	}

	if *includeVulns {
		if vulnerability.IsGrypeAvailable() {
			_, _ = fmt.Fprintf(stdout, "Vulnerability scanning: enabled (Grype)\n")
		} else {
			_, _ = fmt.Fprintf(stderr, "WARNING: Vulnerability scanning requested but Grype not found in PATH\n")
			_, _ = fmt.Fprintf(stderr, "Install Grype from: https://github.com/anchore/grype\n\n")
		}
	}

	scanner := remote.NewScanner(client)
	scanner.SetProgress(func(msg string) {
		_, _ = fmt.Fprintf(stdout, "  %s\n", msg)
	})
	repoURLs := fs.Args()

	_, _ = fmt.Fprintf(stdout, "Scanning %d GitHub repositories...\n\n", len(repoURLs))

	outputDir, err := sbom.ResolveOutputDir(opts.outputDir, "github-scan")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "create output directory: %v\n", err)
		return 1
	}

	var healthResolver *health.Resolver
	if *includeHealth {
		healthResolver = health.NewResolver(client)
	}

	hadErrors := false
	foundVulns := false
	for _, repoURL := range repoURLs {
		_, _ = fmt.Fprintf(stdout, "Scanning: %s\n", repoURL)

		result, err := scanner.ScanRepo(repoURL)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "  Error: %v\n", err)
			hadErrors = true
			continue
		}

		_, _ = fmt.Fprintf(stdout, "  Found %d manifests: %s\n", len(result.Manifests), strings.Join(result.Manifests, ", "))
		_, _ = fmt.Fprintf(stdout, "  Dependencies: %d direct, %d transitive\n",
			len(result.Summary.Direct), len(result.Summary.Transitive))

		repoOutputDir := filepath.Join(outputDir, result.Owner+"_"+result.Repo)
		if err := os.MkdirAll(repoOutputDir, 0755); err != nil {
			_, _ = fmt.Fprintf(stderr, "  Error creating output dir: %v\n", err)
			hadErrors = true
			continue
		}

		savedPaths, err := sbom.SaveSBOMToDir(repoOutputDir, result.Repo, result.Summary, selectedFormat)
		sbomFilePath := selectSBOMPath(savedPaths)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "  Error exporting SBOM: %v\n", err)
			hadErrors = true
		} else {
			for _, p := range savedPaths {
				_, _ = fmt.Fprintf(stdout, "  Exported: %s\n", p)
			}
		}

		// Collect health metrics if requested
		var healthMetrics []*health.DependencyHealth
		if *includeHealth && healthResolver != nil {
			allDeps := append(result.Summary.Direct, result.Summary.Transitive...)
			_, _ = fmt.Fprintf(stdout, "  Fetching health metrics for %d dependencies...\n", len(allDeps))

			progressFn := func(current, total int, depName string) {
				_, _ = fmt.Fprintf(stdout, "\r  [%d/%d] Checking: %-50s", current, total, truncate(depName, 50))
			}
			healthMetrics = healthResolver.ResolveAllWithProgress(allDeps, progressFn)
			_, _ = fmt.Fprintf(stdout, "\r  %-70s\n", "Health check complete!")

			var highRisk, mediumRisk, lowRisk int
			for _, m := range healthMetrics {
				switch m.RiskLevel {
				case "high":
					highRisk++
				case "medium":
					mediumRisk++
				default:
					lowRisk++
				}
			}
			_, _ = fmt.Fprintf(stdout, "  Health: %d low risk, %d medium risk, %d high risk\n",
				lowRisk, mediumRisk, highRisk)
		}

		// Run vulnerability scan on the SBOM if requested
		ctx := context.Background()
		supplyRisks := supplychain.Analyze(ctx, result.Summary)
		var vulnResults *vulnerability.ScanResults
		if *includeVulns && vulnerability.IsGrypeAvailable() {
			if sbomFilePath == "" {
				_, _ = fmt.Fprintf(stderr, "  Error: vulnerability scan requested but no SBOM was exported\n")
				hadErrors = true
			} else {
				_, _ = fmt.Fprintf(stdout, "  Scanning SBOM for vulnerabilities...\n")
				vulnResults, err = vulnerability.ScanSBOMWithGrypeAndEnrich(ctx, sbomFilePath)
				if err != nil {
					_, _ = fmt.Fprintf(stderr, "  Vulnerability scan failed: %v\n", err)
					hadErrors = true
				} else {
					aboveThreshold := opts.vulnCountForExit(vulnResults)
					if vulnResults.TotalCount == 0 {
						_, _ = fmt.Fprintf(stdout, "  Vulnerabilities found: 0\n")
					} else if opts.severityThreshold == "" {
						_, _ = fmt.Fprintf(stdout, "  Vulnerabilities found: %d\n", vulnResults.TotalCount)
						counts := vulnResults.CountBySeverity()
						for sev, count := range counts {
							_, _ = fmt.Fprintf(stdout, "    - %s: %d\n", sev, count)
						}
					} else {
						_, _ = fmt.Fprintf(stdout, "  Vulnerabilities at or above %s: %d (total: %d)\n",
							opts.severityThreshold, aboveThreshold, vulnResults.TotalCount)
						counts := vulnResults.CountBySeverityAboveThreshold(opts.severityThreshold)
						for sev, count := range counts {
							_, _ = fmt.Fprintf(stdout, "    - %s: %d\n", sev, count)
						}
					}
					if aboveThreshold > 0 {
						foundVulns = true
					}
				}
			}
		}

		if len(supplyRisks) > 0 {
			_, _ = fmt.Fprintf(stdout, "  Supply chain risks: %d\n", len(supplyRisks))
			for _, risk := range supplyRisks {
				_, _ = fmt.Fprintf(stdout, "    - [%s] %s (%s): %s\n", risk.Type, risk.Package, risk.Severity, risk.Message)
			}
		}

		if vulnResults != nil || len(supplyRisks) > 0 {
			if findingsPath, err := vulnerability.WriteFindingsJSON(repoOutputDir, result.Repo, vulnResults, opts.severityThreshold, supplyRisks); err != nil {
				_, _ = fmt.Fprintf(stderr, "  Error writing findings.json: %v\n", err)
				hadErrors = true
			} else {
				_, _ = fmt.Fprintf(stdout, "  Findings: %s\n", filepath.Base(findingsPath))
			}
		}

		// Generate report (combined if we have both, otherwise just what we have)
		if vulnResults != nil && len(healthMetrics) > 0 {
			reportPath, err := vulnerability.GenerateFullReport(repoOutputDir, result.Repo, vulnResults, healthMetrics)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "  Error generating report: %v\n", err)
				hadErrors = true
			} else {
				_, _ = fmt.Fprintf(stdout, "  Report: %s\n", reportPath)
			}
		} else if vulnResults != nil {
			reportPath, err := vulnerability.GenerateHTMLReport(repoOutputDir, result.Repo, vulnResults)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "  Error generating report: %v\n", err)
				hadErrors = true
			} else {
				_, _ = fmt.Fprintf(stdout, "  Report: %s\n", reportPath)
			}
		} else if len(healthMetrics) > 0 {
			reportPath, err := vulnerability.GenerateHealthReport(repoOutputDir, result.Repo, healthMetrics)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "  Error generating report: %v\n", err)
				hadErrors = true
			} else {
				_, _ = fmt.Fprintf(stdout, "  Report: %s\n", reportPath)
			}
		}

		rateLimit := client.GetRateLimit()
		if rateLimit.Remaining > 0 {
			_, _ = fmt.Fprintf(stdout, "  [Rate limit: %d/%d remaining]\n", rateLimit.Remaining, rateLimit.Limit)
		}
		_, _ = fmt.Fprintf(stdout, "\n")
	}

	_, _ = fmt.Fprintf(stdout, "GitHub scan complete. Output saved to: %s\n", outputDir)
	if hadErrors || (opts.failOnVuln && foundVulns) {
		return 1
	}
	return 0
}

func runTrace(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("trace", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showTree := fs.Bool("tree", false, "show full dependency tree instead of chain")
	showList := fs.Bool("list", false, "list all dependencies with optional filters")
	showGraph := fs.Bool("graph", false, "show ASCII dependency graph")
	showDot := fs.Bool("dot", false, "output DOT format for Graphviz visualization")
	showConnections := fs.Bool("connections", false, "show detailed connection info for a package")
	showFalsePositives := fs.Bool("fp", false, "highlight potential false positives")
	filterEcosystem := fs.String("ecosystem", "", "filter by ecosystem (npm, maven, pypi, golang, rubygems)")
	filterScope := fs.String("scope", "", "filter by build-scope (runtime, dev, test, build-tooling)")
	filterType := fs.String("type", "", "filter by dependency-type (direct, transitive)")
	filterSourceFile := fs.String("source-file", "", "filter by source manifest file")
	minDepth := fs.Int("min-depth", 0, "minimum depth (0 = direct)")
	maxDepth := fs.Int("max-depth", -1, "maximum depth (-1 = no limit)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 1 {
		_, _ = fmt.Fprintf(stderr, "Usage: sbomber trace <path> [package-name] [flags]\n")
		_, _ = fmt.Fprintf(stderr, "\nVisualization Flags:\n")
		_, _ = fmt.Fprintf(stderr, "  --tree                  Show full dependency tree\n")
		_, _ = fmt.Fprintf(stderr, "  --graph                 Show ASCII dependency graph\n")
		_, _ = fmt.Fprintf(stderr, "  --dot                   Output DOT format for Graphviz\n")
		_, _ = fmt.Fprintf(stderr, "  --connections           Show detailed connection info\n")
		_, _ = fmt.Fprintf(stderr, "  --fp                    Highlight potential false positives\n")
		_, _ = fmt.Fprintf(stderr, "\nFilter Flags:\n")
		_, _ = fmt.Fprintf(stderr, "  --list                  List all dependencies with filters\n")
		_, _ = fmt.Fprintf(stderr, "  --ecosystem <name>      Filter by ecosystem (npm, maven, pypi, golang, rubygems)\n")
		_, _ = fmt.Fprintf(stderr, "  --scope <name>          Filter by build-scope (runtime, dev, test, build-tooling)\n")
		_, _ = fmt.Fprintf(stderr, "  --type <name>           Filter by dependency-type (direct, transitive)\n")
		_, _ = fmt.Fprintf(stderr, "  --source-file <path>    Filter by source manifest file\n")
		_, _ = fmt.Fprintf(stderr, "  --min-depth <n>         Minimum depth (default: 0)\n")
		_, _ = fmt.Fprintf(stderr, "  --max-depth <n>         Maximum depth (default: no limit)\n")
		_, _ = fmt.Fprintf(stderr, "\nExamples:\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber trace . lodash\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber trace . express --tree\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber trace --graph .                    # ASCII tree of all deps\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber trace --dot . > deps.dot           # DOT format for Graphviz\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber trace --connections . lodash       # Show how lodash is connected\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber trace --fp --list .                # Show potential false positives\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber trace . --list --ecosystem npm\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber trace . --list --type transitive --min-depth 2\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber trace . --list --source-file package.json\n")
		return 1
	}

	root := fs.Arg(0)
	packageName := ""
	if fs.NArg() >= 2 {
		packageName = fs.Arg(1)
	}

	absoluteRoot, err := resolveScanRoot(root)
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

	// Build filter options
	filterOpts := deps.NewFilterOptions()
	filterOpts.Ecosystem = *filterEcosystem
	filterOpts.Scope = *filterScope
	filterOpts.Type = *filterType
	filterOpts.SourceFile = *filterSourceFile
	filterOpts.MinDepth = *minDepth
	filterOpts.MaxDepth = *maxDepth
	filterOpts.NameFilter = packageName

	found := false
	for _, repo := range repos {
		detection, err := ecosystem.DetectRepository(repo.Path)
		if err != nil {
			continue
		}

		summary, err := buildRepoDependencySummary(repo.Path, detection)
		if err != nil {
			continue
		}

		// Build the dependency graph and detect false positives
		summary.BuildGraph(repo.Name)
		summary.DetectFalsePositives()

		// DOT graph output (for Graphviz)
		if *showDot {
			found = true
			dot := summary.GenerateDOTGraph(repo.Name)
			_, _ = fmt.Fprintf(stdout, "%s", dot)
			continue
		}

		// ASCII graph output
		if *showGraph {
			found = true
			_, _ = fmt.Fprintf(stdout, "\n%s[%s]%s Dependency Graph\n", colorBold, repo.Name, colorReset)
			_, _ = fmt.Fprintf(stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
			tree := summary.GenerateASCIITree(repo.Name)
			_, _ = fmt.Fprintf(stdout, "%s\n", tree)

			// Legend
			_, _ = fmt.Fprintf(stdout, "%sLegend:%s\n", colorCyan, colorReset)
			_, _ = fmt.Fprintf(stdout, "  ⚠️  = Potential false positive (test/example dependency)\n\n")
			continue
		}

		// Show false positives summary
		if *showFalsePositives {
			var fpDeps []deps.Dependency
			for _, d := range summary.AllDependencies() {
				if d.IsPotentialFalsePositive() {
					fpDeps = append(fpDeps, d)
				}
			}

			if len(fpDeps) == 0 && packageName == "" {
				continue
			}

			found = true
			_, _ = fmt.Fprintf(stdout, "\n%s[%s]%s %s\n", colorBold, repo.Name, colorReset, repo.Path)
			_, _ = fmt.Fprintf(stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

			if len(fpDeps) > 0 {
				_, _ = fmt.Fprintf(stdout, "\n%s⚠️  Potential False Positives (%d):%s\n\n", colorCyan, len(fpDeps), colorReset)
				for _, d := range fpDeps {
					_, _ = fmt.Fprintf(stdout, "  %s%-40s%s\n", colorBlue, d.Name+"@"+d.Version, colorReset)
					_, _ = fmt.Fprintf(stdout, "    Reason:  %s\n", d.FPReason)
					_, _ = fmt.Fprintf(stdout, "    Source:  %s\n", d.SourceFile)
					_, _ = fmt.Fprintf(stdout, "    Path:    %s\n\n", d.SourceLocation)
				}
			} else {
				_, _ = fmt.Fprintf(stdout, "\n%s✓ No potential false positives detected%s\n\n", colorCyan, colorReset)
			}
			continue
		}

		// List mode - show filtered dependencies
		if *showList || (packageName == "" && !*showConnections) {
			filtered := summary.Filter(filterOpts)
			if len(filtered) == 0 {
				continue
			}

			found = true
			_, _ = fmt.Fprintf(stdout, "\n%s[%s]%s %s\n", colorBold, repo.Name, colorReset, repo.Path)
			_, _ = fmt.Fprintf(stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			_, _ = fmt.Fprintf(stdout, "Found %d dependencies matching filters\n\n", len(filtered))

			// Group by source file for clarity
			bySource := make(map[string][]deps.Dependency)
			for _, d := range filtered {
				key := d.SourceFile
				if key == "" {
					key = "unknown"
				}
				bySource[key] = append(bySource[key], d)
			}

			for source, sourceDeps := range bySource {
				_, _ = fmt.Fprintf(stdout, "%sSource: %s%s\n", colorCyan, source, colorReset)
				for _, d := range sourceDeps {
					depType := "T"
					if d.IsDirect {
						depType = "D"
					}
					fpMarker := ""
					if d.IsPotentialFalsePositive() {
						fpMarker = " ⚠️"
					}
					_, _ = fmt.Fprintf(stdout, "  [%s] %-40s %s%-12s%s depth=%d%s\n",
						depType, truncate(d.Name+"@"+d.Version, 40),
						colorBlue, d.Ecosystem, colorReset, d.Depth, fpMarker)
				}
				_, _ = fmt.Fprintf(stdout, "\n")
			}
			continue
		}

		// Find specific dependency
		dep := summary.FindDependency(packageName)
		if dep == nil {
			continue
		}

		found = true
		_, _ = fmt.Fprintf(stdout, "\n%s[%s]%s %s\n", colorBold, repo.Name, colorReset, repo.Path)
		_, _ = fmt.Fprintf(stdout, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		// Show detailed connections
		if *showConnections {
			connInfo := summary.GetConnectionInfo(dep.Name)
			if connInfo != nil {
				_, _ = fmt.Fprintf(stdout, "\n%sConnection Analysis for:%s %s@%s\n", colorCyan, colorReset, dep.Name, dep.Version)
				_, _ = fmt.Fprintf(stdout, "\n%sIntroduced by:%s\n", colorCyan, colorReset)
				if len(connInfo.IntroducedBy) == 0 {
					_, _ = fmt.Fprintf(stdout, "  (unknown)\n")
				} else {
					for _, parent := range connInfo.IntroducedBy {
						_, _ = fmt.Fprintf(stdout, "  → %s\n", parent)
					}
				}

				_, _ = fmt.Fprintf(stdout, "\n%sPulls in:%s\n", colorCyan, colorReset)
				if len(connInfo.UsedBy) == 0 {
					_, _ = fmt.Fprintf(stdout, "  (no dependencies)\n")
				} else {
					for i, child := range connInfo.UsedBy {
						if i >= 15 {
							_, _ = fmt.Fprintf(stdout, "  ... and %d more\n", len(connInfo.UsedBy)-15)
							break
						}
						_, _ = fmt.Fprintf(stdout, "  ← %s\n", child)
					}
				}

				_, _ = fmt.Fprintf(stdout, "\n%sPaths to root:%s\n", colorCyan, colorReset)
				if len(connInfo.PathsToRoot) == 0 {
					_, _ = fmt.Fprintf(stdout, "  %s (direct dependency)\n", dep.Name)
				} else {
					for i, path := range connInfo.PathsToRoot {
						if i >= 5 {
							_, _ = fmt.Fprintf(stdout, "  ... and %d more paths\n", len(connInfo.PathsToRoot)-5)
							break
						}
						_, _ = fmt.Fprintf(stdout, "  %s\n", strings.Join(path, " → "))
					}
				}

				// False positive warning
				if dep.IsPotentialFalsePositive() {
					_, _ = fmt.Fprintf(stdout, "\n%s⚠️  Potential False Positive:%s\n", colorCyan, colorReset)
					_, _ = fmt.Fprintf(stdout, "  Reason: %s\n", dep.FPReason)
				}
			}
			_, _ = fmt.Fprintf(stdout, "\n")
			continue
		}

		if *showTree {
			// Show full tree
			_, _ = fmt.Fprintf(stdout, "\n%sDependency Tree:%s\n", colorCyan, colorReset)
			tree := summary.GetDependencyTree(dep.Name, "  ", nil)
			_, _ = fmt.Fprintf(stdout, "%s", tree)
		} else {
			// Show detailed chain information
			_, _ = fmt.Fprintf(stdout, "\n%sPackage:%s       %s@%s\n", colorCyan, colorReset, dep.Name, dep.Version)
			_, _ = fmt.Fprintf(stdout, "%sEcosystem:%s     %s\n", colorCyan, colorReset, dep.Ecosystem)

			depType := "transitive"
			if dep.IsDirect {
				depType = "direct"
			}
			_, _ = fmt.Fprintf(stdout, "%sType:%s          %s\n", colorCyan, colorReset, depType)
			_, _ = fmt.Fprintf(stdout, "%sDepth:%s         %d hops from root\n", colorCyan, colorReset, dep.Depth)
			_, _ = fmt.Fprintf(stdout, "%sScope:%s         %s\n", colorCyan, colorReset, dep.BuildScope())

			// False positive warning
			if dep.IsPotentialFalsePositive() {
				_, _ = fmt.Fprintf(stdout, "\n%s⚠️  Potential False Positive:%s\n", colorCyan, colorReset)
				_, _ = fmt.Fprintf(stdout, "  Reason: %s\n", dep.FPReason)
			}

			// Source information
			_, _ = fmt.Fprintf(stdout, "\n%sSource Information:%s\n", colorCyan, colorReset)
			sourceFile := dep.SourceFile
			if sourceFile == "" {
				sourceFile = "unknown"
			}
			_, _ = fmt.Fprintf(stdout, "  File:     %s\n", sourceFile)
			sourceLocation := dep.SourceLocation
			if sourceLocation == "" {
				sourceLocation = "unknown"
			}
			_, _ = fmt.Fprintf(stdout, "  Location: %s\n", sourceLocation)

			_, _ = fmt.Fprintf(stdout, "\n%sChain (path from root):%s\n", colorCyan, colorReset)
			printChain(stdout, dep.Chain)

			if len(dep.Children) > 0 {
				_, _ = fmt.Fprintf(stdout, "\n%sDepends on:%s  %d packages\n", colorCyan, colorReset, len(dep.Children))
				for i, child := range dep.Children {
					if i >= 10 {
						_, _ = fmt.Fprintf(stdout, "  ... and %d more\n", len(dep.Children)-10)
						break
					}
					_, _ = fmt.Fprintf(stdout, "  • %s\n", child)
				}
			}
		}
		_, _ = fmt.Fprintf(stdout, "\n")
	}

	if !found {
		if packageName != "" {
			_, _ = fmt.Fprintf(stderr, "Package %q not found in any repository\n", packageName)
		} else {
			_, _ = fmt.Fprintf(stderr, "No dependencies match the specified filters\n")
		}
		return 1
	}

	return 0
}

func runVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outputJSON := fs.Bool("json", false, "output results as JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 2 {
		_, _ = fmt.Fprintf(stderr, "Usage: sbomber verify <ground-truth-sbom> <generated-sbom> [--json]\n")
		_, _ = fmt.Fprintf(stderr, "\nCompare a generated SBOM against a verified ground truth SBOM.\n")
		_, _ = fmt.Fprintf(stderr, "\nSupported formats:\n")
		_, _ = fmt.Fprintf(stderr, "  - CycloneDX (XML and JSON)\n")
		_, _ = fmt.Fprintf(stderr, "  - SPDX (JSON)\n")
		_, _ = fmt.Fprintf(stderr, "\nExamples:\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber verify reference.cdx.xml my-output.cdx.xml\n")
		_, _ = fmt.Fprintf(stderr, "  sbomber verify benchmark.json generated.json --json\n")
		_, _ = fmt.Fprintf(stderr, "\nBenchmark repositories:\n")
		_, _ = fmt.Fprintf(stderr, "  - https://github.com/CycloneDX/bom-examples\n")
		_, _ = fmt.Fprintf(stderr, "  - https://github.com/sbomify/sbom-benchmarks\n")
		_, _ = fmt.Fprintf(stderr, "  - https://github.com/spdx/spdx-examples\n")
		return 1
	}

	groundTruthPath := fs.Arg(0)
	generatedPath := fs.Arg(1)

	result, err := verify.VerifyFiles(groundTruthPath, generatedPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	if *outputJSON {
		// Output as JSON for CI/CD integration
		jsonOut := struct {
			GroundTruthCount int     `json:"ground_truth_count"`
			GeneratedCount   int     `json:"generated_count"`
			MatchedCount     int     `json:"matched_count"`
			MissingCount     int     `json:"missing_count"`
			ExtraCount       int     `json:"extra_count"`
			VersionMismatch  int     `json:"version_mismatch"`
			Precision        float64 `json:"precision"`
			Recall           float64 `json:"recall"`
			F1Score          float64 `json:"f1_score"`
			VersionAccuracy  float64 `json:"version_accuracy"`
		}{
			GroundTruthCount: result.GroundTruthCount,
			GeneratedCount:   result.GeneratedCount,
			MatchedCount:     result.MatchedCount,
			MissingCount:     result.MissingCount,
			ExtraCount:       result.ExtraCount,
			VersionMismatch:  result.VersionMismatch,
			Precision:        result.Precision,
			Recall:           result.Recall,
			F1Score:          result.F1Score,
			VersionAccuracy:  result.VersionAccuracy,
		}
		data, _ := json.MarshalIndent(jsonOut, "", "  ")
		_, _ = fmt.Fprintf(stdout, "%s\n", data)
	} else {
		_, _ = fmt.Fprint(stdout, result.PrintReport())
	}

	// Return non-zero if accuracy is below threshold
	if result.F1Score < 70 {
		return 1
	}
	return 0
}

// printChain prints a dependency chain with nice formatting
func printChain(w io.Writer, chain string) {
	parts := strings.Split(chain, " > ")
	for i, part := range parts {
		indent := strings.Repeat("  ", i)
		if i == 0 {
			_, _ = fmt.Fprintf(w, "  %s%s%s (root)\n", colorBlue, part, colorReset)
		} else if i == len(parts)-1 {
			_, _ = fmt.Fprintf(w, "  %s└─ %s%s%s (target)\n", indent, colorCyan, part, colorReset)
		} else {
			_, _ = fmt.Fprintf(w, "  %s└─ %s\n", indent, part)
		}
	}
}

func runInteractive(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if stdin == os.Stdin && stdout == os.Stdout {
		for {
			result := runTUIFull()
			switch result.Action {
			case "scan":
				scanPaths := result.ScanPaths
				scanFormat := result.ScanFormat
				includeVulns := result.IncludeVulns

				// Use single path if no multi-paths specified
				if len(scanPaths) == 0 {
					scanPath := result.ScanPath
					if scanPath == "" {
						scanPath = "."
					}
					scanPaths = []string{scanPath}
				}

				if scanFormat == "" {
					scanFormat = formatCycloneDX
				}

				// Resolve all paths
				var absPaths []string
				for _, p := range scanPaths {
					absPath, err := resolveScanRoot(p)
					if err != nil {
						fmt.Fprintf(stderr, "Invalid path %s: %v\n", p, err)
						continue
					}
					absPaths = append(absPaths, absPath)
				}

				if len(absPaths) == 0 {
					fmt.Fprintf(stderr, "No valid paths to scan\n")
					continue
				}

				// Get central output folder in ~/.sbomber/reports/
				outputFolder, err := sbom.GetOutputDir(absPaths[0])
				if err != nil {
					fmt.Fprintf(stderr, "Failed to create output directory: %v\n", err)
					continue
				}

				// Show scanning message
				fmt.Print("\033[H\033[2J") // Clear screen
				fmt.Println()
				fmt.Println("  \033[1m\033[36mScanning...\033[0m")
				fmt.Println()
				if len(absPaths) == 1 {
					fmt.Printf("  Path:   %s\n", absPaths[0])
				} else {
					fmt.Printf("  Paths:  %d folders\n", len(absPaths))
					for i, p := range absPaths {
						fmt.Printf("    %d. %s\n", i+1, p)
					}
				}
				fmt.Printf("  Format: %s\n", scanFormat)
				if includeVulns {
					fmt.Println("  Vulns:  enabled (Grype)")
				}
				fmt.Println()
				fmt.Println("  \033[90mThis may take a moment...\033[0m")

				var buf bytes.Buffer
				args := []string{"--format", scanFormat}
				if includeVulns {
					args = append(args, "--include-vulnerabilities")
				}
				args = append(args, absPaths...)
				runScan(args, &buf, &buf) // Capture both stdout and stderr

				if quit := showResultsScreen(buf.String(), outputFolder); quit {
					fmt.Print("\033[H\033[2J")
					fmt.Fprint(stdout, "Goodbye!\n")
					return 0
				}
			case "github":
				// Show scanning message
				fmt.Print("\033[H\033[2J") // Clear screen
				fmt.Println()
				fmt.Println("  \033[1m\033[36mScanning GitHub repositories...\033[0m")
				fmt.Println()

				// Parse URLs (space or comma separated)
				urlInput := strings.ReplaceAll(result.GitHubURLs, ",", " ")
				urls := strings.Fields(urlInput)

				fmt.Printf("  Repos:  %d\n", len(urls))
				fmt.Printf("  Format: %s\n", result.ScanFormat)
				if result.IncludeHealth {
					fmt.Println("  Health: enabled")
				}
				fmt.Println()

				args := []string{}
				if result.ScanFormat != "" {
					args = append(args, "--format", result.ScanFormat)
				}
				if result.IncludeHealth {
					args = append(args, "--health")
				}
				if result.IncludeVulns {
					args = append(args, "--include-vulnerabilities")
				}
				args = append(args, urls...)

				// Set token in env if provided via TUI
				if result.GitHubToken != "" {
					os.Setenv("GITHUB_TOKEN", result.GitHubToken)
				}

				// Run with real-time output to stdout (not buffered)
				var buf bytes.Buffer
				// Create a writer that writes to both stdout (real-time) and buffer (for results screen)
				multiWriter := io.MultiWriter(os.Stdout, &buf)
				runGitHubScan(args, multiWriter, multiWriter)

				fmt.Println()
				fmt.Println("  \033[90mPress Enter to continue...\033[0m")
				bufio.NewReader(os.Stdin).ReadBytes('\n')

				outputFolder, _ := sbom.GetOutputDir("github-scan")
				if quit := showResultsScreen(buf.String(), outputFolder); quit {
					fmt.Print("\033[H\033[2J")
					fmt.Fprint(stdout, "Goodbye!\n")
					return 0
				}
			case "version":
				if quit := showResultsScreen(fmt.Sprintf("SBOMber %s", version), ""); quit {
					return 0
				}
			case "help":
				var buf bytes.Buffer
				printUsage(&buf)
				if quit := showResultsScreen(buf.String(), ""); quit {
					return 0
				}
			case "exit", "":
				fmt.Print("\033[H\033[2J")
				fmt.Fprint(stdout, "Goodbye!\n")
				return 0
			}
		}
	}

	printBanner(stdout)
	_, _ = fmt.Fprintf(stdout, "%sA lightweight CLI for scanning local repositories and generating SBOMs.%s\n\n", colorBlue, colorReset)
	_, _ = fmt.Fprint(stdout, "Choose an option:\n")
	_, _ = fmt.Fprint(stdout, "  1. Scan the current folder\n")
	_, _ = fmt.Fprint(stdout, "  2. Scan a custom folder\n")
	_, _ = fmt.Fprint(stdout, "  3. Show version\n")
	_, _ = fmt.Fprint(stdout, "  4. Show help\n\n")
	_, _ = fmt.Fprint(stdout, "Enter choice [1-4]: ")

	reader := bufio.NewReader(stdin)
	choice, err := reader.ReadString('\n')
	if err != nil && len(choice) == 0 {
		_, _ = fmt.Fprintf(stderr, "read choice: %v\n", err)
		return 1
	}

	switch strings.TrimSpace(choice) {
	case "", "1":
		format, exitCode := promptExportFormat(reader, stdout, stderr)
		if exitCode != 0 {
			return exitCode
		}

		includeVulns, exitCode := promptVulnerabilityScan(reader, stdout, stderr)
		if exitCode != 0 {
			return exitCode
		}

		args := []string{"--format", format}
		if includeVulns {
			args = append(args, "--include-vulnerabilities")
		}
		args = append(args, ".")

		return runScan(args, stdout, stderr)
	case "2":
		_, _ = fmt.Fprint(stdout, "Folder to scan: ")
		path, err := reader.ReadString('\n')
		if err != nil && len(path) == 0 {
			_, _ = fmt.Fprintf(stderr, "read path: %v\n", err)
			return 1
		}

		path = strings.TrimSpace(path)
		if path == "" {
			path = "."
		}

		format, exitCode := promptExportFormat(reader, stdout, stderr)
		if exitCode != 0 {
			return exitCode
		}

		includeVulns, exitCode := promptVulnerabilityScan(reader, stdout, stderr)
		if exitCode != 0 {
			return exitCode
		}

		args := []string{"--format", format}
		if includeVulns {
			args = append(args, "--include-vulnerabilities")
		}
		args = append(args, path)

		return runScan(args, stdout, stderr)
	case "3":
		_, _ = fmt.Fprintf(stdout, "sbomber %s\n", version)
		return 0
	case "4":
		printUsage(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "invalid choice %q\n", strings.TrimSpace(choice))
		return 1
	}
}

func printBanner(w io.Writer) {
	_, _ = fmt.Fprintf(w, "%s%s", colorBold, colorCyan)
	_, _ = fmt.Fprint(w, `
  ____  ____   ___  __  __ ____             
 / ___|| __ ) / _ \|  \/  | __ )  ___ _ __  
 \___ \|  _ \| | | | |\/| |  _ \ / _ \ '__| 
  ___) | |_) | |_| | |  | | |_) |  __/ |    
 |____/|____/ \___/|_|  |_|____/ \___|_|    
`)
	_, _ = fmt.Fprintf(w, "%s\n", colorReset)
}

func promptExportFormat(reader *bufio.Reader, stdout io.Writer, stderr io.Writer) (string, int) {
	_, _ = fmt.Fprint(stdout, "\nChoose SBOM export format:\n")
	_, _ = fmt.Fprint(stdout, "  1. CycloneDX\n")
	_, _ = fmt.Fprint(stdout, "  2. SPDX\n")
	_, _ = fmt.Fprint(stdout, "  3. Both\n\n")
	_, _ = fmt.Fprint(stdout, "Enter choice [1-3] (default 1): ")

	choice, err := reader.ReadString('\n')
	if err != nil && len(choice) == 0 {
		_, _ = fmt.Fprintf(stderr, "read format choice: %v\n", err)
		return "", 1
	}

	switch strings.TrimSpace(choice) {
	case "", "1":
		return formatCycloneDX, 0
	case "2":
		return formatSPDX, 0
	case "3":
		return formatBoth, 0
	default:
		_, _ = fmt.Fprintf(stderr, "invalid export format choice %q\n", strings.TrimSpace(choice))
		return "", 1
	}
}

func promptVulnerabilityScan(reader *bufio.Reader, stdout io.Writer, stderr io.Writer) (bool, int) {
	_, _ = fmt.Fprint(stdout, "\nInclude vulnerability scanning with Grype? [y/N]: ")

	choice, err := reader.ReadString('\n')
	if err != nil && len(choice) == 0 {
		_, _ = fmt.Fprintf(stderr, "read vulnerability choice: %v\n", err)
		return false, 1
	}

	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "y", "yes":
		return true, 0
	case "", "n", "no":
		return false, 0
	default:
		_, _ = fmt.Fprintf(stderr, "invalid vulnerability choice %q\n", strings.TrimSpace(choice))
		return false, 1
	}
}

func printDependencySummary(stdout io.Writer, stderr io.Writer, repoName, repoPath string, detection ecosystem.Detection, opts scanOptions) (bool, bool) {
	summary, err := buildRepoDependencySummary(repoPath, detection)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read dependencies for %s: %v\n", repoPath, err)
		return false, false
	}

	savedPaths, outputDir, err := sbom.SaveSBOMWithOutput(repoPath, repoName, summary, opts.format, opts.outputDir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "save SBOM for %s: %v\n", repoPath, err)
		return false, false
	}

	_, _ = fmt.Fprintf(stdout, "  output folder: %s\n", outputDir)
	for _, path := range savedPaths {
		_, _ = fmt.Fprintf(stdout, "  exported SBOM: %s\n", filepath.Base(path))
	}

	vulnFound := false
	if opts.includeVulnerabilities {
		scanPath := grypeScanTarget(savedPaths, repoPath)
		ok, count := generateVulnReport(stdout, stderr, scanPath, outputDir, repoName, &summary, opts)
		if !ok {
			return false, false
		}
		vulnFound = count > 0
	} else if len(summary.AllDependencies()) > 0 {
		risks := supplychain.Analyze(context.Background(), summary)
		if len(risks) > 0 {
			_, _ = fmt.Fprintf(stdout, "  supply chain risks: %d\n", len(risks))
			if path, err := vulnerability.WriteFindingsJSON(outputDir, repoName, &vulnerability.ScanResults{}, opts.severityThreshold, risks); err == nil {
				_, _ = fmt.Fprintf(stdout, "  findings: %s\n", filepath.Base(path))
			}
		}
	}

	totalDirect := summary.Count()
	totalTransitive := summary.TransitiveCount()

	if totalDirect == 0 && totalTransitive == 0 {
		return true, vulnFound
	}

	_, _ = fmt.Fprintf(stdout, "  packages:  %d direct", totalDirect)
	if totalTransitive > 0 {
		_, _ = fmt.Fprintf(stdout, ", %d transitive (%d total)", totalTransitive, summary.TotalCount())
	}
	_, _ = fmt.Fprintln(stdout)

	sourceLabel := buildSourceLabel(detection)

	_, _ = fmt.Fprintf(stdout, "  direct dependencies (%s): %d", sourceLabel, summary.Count())

	runtimeCount := summary.CountByScope(deps.ScopeRuntime)
	devCount := summary.CountByScope(deps.ScopeDev)
	peerCount := summary.CountByScope(deps.ScopePeer)
	optionalCount := summary.CountByScope(deps.ScopeOptional)

	scopeParts := make([]string, 0, 4)
	if runtimeCount > 0 {
		scopeParts = append(scopeParts, fmt.Sprintf("runtime: %d", runtimeCount))
	}
	if devCount > 0 {
		scopeParts = append(scopeParts, fmt.Sprintf("development: %d", devCount))
	}
	if peerCount > 0 {
		scopeParts = append(scopeParts, fmt.Sprintf("peer: %d", peerCount))
	}
	if optionalCount > 0 {
		scopeParts = append(scopeParts, fmt.Sprintf("optional: %d", optionalCount))
	}

	if len(scopeParts) > 0 {
		_, _ = fmt.Fprintf(stdout, " (%s)", strings.Join(scopeParts, ", "))
	}
	_, _ = fmt.Fprint(stdout, "\n")

	if summary.TransitiveCount() > 0 {
		_, _ = fmt.Fprintf(stdout, "  transitive dependencies: %d\n", summary.TransitiveCount())
		_, _ = fmt.Fprintf(stdout, "  total known dependencies: %d\n", summary.TotalCount())
	}

	preview := summary.PreviewNames(5)
	if len(preview) == 0 {
		return true, vulnFound
	}

	_, _ = fmt.Fprintf(stdout, "  sample packages: %s\n", strings.Join(preview, ", "))
	return true, vulnFound
}

func printDependencySummaryBatch(stdout io.Writer, stderr io.Writer, repoName, repoPath string, detection ecosystem.Detection, opts scanOptions, batchDir string) (bool, bool) {
	summary, err := buildRepoDependencySummary(repoPath, detection)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "  read dependencies for %s: %v\n", repoPath, err)
		return false, false
	}

	repoOutputDir, err := sbom.GetRepoOutputDir(batchDir, repoName)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "  create output dir for %s: %v\n", repoName, err)
		return false, false
	}

	savedPaths, err := sbom.SaveSBOMToDir(repoOutputDir, repoName, summary, opts.format)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "  save SBOM for %s: %v\n", repoPath, err)
		return false, false
	}

	_, _ = fmt.Fprintf(stdout, "  output: %s/\n", filepath.Base(repoOutputDir))
	for _, path := range savedPaths {
		_, _ = fmt.Fprintf(stdout, "    - %s\n", filepath.Base(path))
	}

	vulnFound := false
	if opts.includeVulnerabilities {
		scanPath := grypeScanTarget(savedPaths, repoPath)
		ok, count := generateVulnReport(stdout, stderr, scanPath, repoOutputDir, repoName, &summary, opts)
		if !ok {
			return false, false
		}
		vulnFound = count > 0
	} else if len(summary.AllDependencies()) > 0 {
		risks := supplychain.Analyze(context.Background(), summary)
		if len(risks) > 0 {
			_, _ = fmt.Fprintf(stdout, "  supply chain risks: %d\n", len(risks))
			if path, err := vulnerability.WriteFindingsJSON(repoOutputDir, repoName, &vulnerability.ScanResults{}, opts.severityThreshold, risks); err == nil {
				_, _ = fmt.Fprintf(stdout, "  findings: %s\n", filepath.Base(path))
			}
		}
	}

	totalDirect := summary.Count()
	totalTransitive := summary.TransitiveCount()

	if totalDirect == 0 && totalTransitive == 0 {
		return true, vulnFound
	}

	_, _ = fmt.Fprintf(stdout, "  packages: %d direct", totalDirect)
	if totalTransitive > 0 {
		_, _ = fmt.Fprintf(stdout, ", %d transitive", totalTransitive)
	}
	_, _ = fmt.Fprintln(stdout)
	return true, vulnFound
}

func buildRepoDependencySummary(repoPath string, _ ecosystem.Detection) (deps.Summary, error) {
	manifestRoots, err := ecosystem.FindManifestRoots(repoPath)
	if err != nil {
		return deps.Summary{}, err
	}
	if len(manifestRoots) == 0 {
		manifestRoots = []string{repoPath}
	}

	merged := deps.Summary{
		Direct:     make([]deps.Dependency, 0),
		Transitive: make([]deps.Dependency, 0),
	}

	for _, manifestRoot := range manifestRoots {
		localDetection, err := ecosystem.Detect(manifestRoot)
		if err != nil {
			return deps.Summary{}, err
		}
		part, err := buildDependencySummaryAt(manifestRoot, localDetection)
		if err != nil {
			return deps.Summary{}, err
		}
		deps.MergeSummary(&merged, part)
	}

	return merged, nil
}

func buildDependencySummaryAt(manifestRoot string, detection ecosystem.Detection) (deps.Summary, error) {
	summary := deps.Summary{
		Direct:     make([]deps.Dependency, 0),
		Transitive: make([]deps.Dependency, 0),
	}

	if containsEcosystem(detection.Names, ecosystem.NPM) {
		npmSummary, err := npm.ParseProject(manifestRoot)
		if err != nil {
			return deps.Summary{}, err
		}

		summary.Direct = append(summary.Direct, npmSummary.Direct...)
		summary.Transitive = append(summary.Transitive, npmSummary.Transitive...)
	}

	if containsEcosystem(detection.Names, ecosystem.Python) {
		pythonSummary, err := python.ParseManifests(manifestRoot)
		if err != nil {
			return deps.Summary{}, err
		}
		summary.Direct = append(summary.Direct, pythonSummary.Direct...)
		summary.Transitive = append(summary.Transitive, pythonSummary.Transitive...)
	}
	if containsEcosystem(detection.Names, ecosystem.Maven) {
		mavenSummary, err := maven.ParsePOM(manifestRoot)
		if err != nil {
			return deps.Summary{}, err
		}
		summary.Direct = append(summary.Direct, mavenSummary.Direct...)
	}

	if containsEcosystem(detection.Names, ecosystem.Ruby) {
		rubySummary, err := ruby.ParseGemfileLock(manifestRoot)
		if err != nil {
			return deps.Summary{}, err
		}
		summary.Direct = append(summary.Direct, rubySummary.Direct...)
	}

	if containsEcosystem(detection.Names, ecosystem.Go) {
		goSummary, err := golang.ParseGoMod(manifestRoot)
		if err != nil {
			return deps.Summary{}, err
		}
		summary.Direct = append(summary.Direct, goSummary.Direct...)
		summary.Transitive = append(summary.Transitive, goSummary.Transitive...)
	}

	return summary, nil
}

func generateVulnReport(stdout io.Writer, stderr io.Writer, scanPath, outputDir, repoName string, summary *deps.Summary, opts scanOptions) (bool, int) {
	if !vulnerability.IsGrypeAvailable() {
		_, _ = fmt.Fprintf(stderr, "  note: grype not available, skipping vulnerability scan\n")
		return true, 0
	}

	ctx := context.Background()
	vulnResults, err := vulnerability.ScanWithGrypeAndEnrich(ctx, scanPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "  vulnerability scan failed: %v\n", err)
		return false, 0
	}

	supplyRisks := supplychain.Analyze(ctx, *summary)
	aboveThreshold := opts.vulnCountForExit(vulnResults)

	if vulnResults.TotalCount == 0 {
		_, _ = fmt.Fprintf(stdout, "  vulnerabilities found: 0\n")
	} else if opts.severityThreshold == "" {
		_, _ = fmt.Fprintf(stdout, "  vulnerabilities found: %d\n", vulnResults.TotalCount)
		counts := vulnResults.CountBySeverity()
		for severity, count := range counts {
			_, _ = fmt.Fprintf(stdout, "    - %s: %d\n", severity, count)
		}
	} else {
		_, _ = fmt.Fprintf(stdout, "  vulnerabilities at or above %s: %d (total: %d)\n",
			opts.severityThreshold, aboveThreshold, vulnResults.TotalCount)
		counts := vulnResults.CountBySeverityAboveThreshold(opts.severityThreshold)
		for severity, count := range counts {
			_, _ = fmt.Fprintf(stdout, "    - %s: %d\n", severity, count)
		}
	}

	if len(supplyRisks) > 0 {
		_, _ = fmt.Fprintf(stdout, "  supply chain risks: %d\n", len(supplyRisks))
		for _, risk := range supplyRisks {
			_, _ = fmt.Fprintf(stdout, "    - [%s] %s (%s): %s\n", risk.Type, risk.Package, risk.Severity, risk.Message)
		}
	}

	findingsPath, err := vulnerability.WriteFindingsJSON(outputDir, repoName, vulnResults, opts.severityThreshold, supplyRisks)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "  failed to write findings.json: %v\n", err)
		return false, aboveThreshold
	}
	_, _ = fmt.Fprintf(stdout, "  findings: %s\n", filepath.Base(findingsPath))

	reportPath, err := vulnerability.GenerateHTMLReportWithDeps(outputDir, repoName, vulnResults, summary)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "  failed to generate HTML report: %v\n", err)
		return false, aboveThreshold
	}
	_, _ = fmt.Fprintf(stdout, "  HTML report: %s\n", filepath.Base(reportPath))
	return true, aboveThreshold
}

func containsEcosystem(names []ecosystem.Name, candidate ecosystem.Name) bool {
	for _, name := range names {
		if name == candidate {
			return true
		}
	}

	return false
}

func buildSourceLabel(detection ecosystem.Detection) string {
	labels := make([]string, 0, len(detection.Names))
	for _, name := range detection.Names {
		if files, ok := detection.Evidence[name]; ok && len(files) > 0 {
			labels = append(labels, strings.Join(files, " + "))
		}
	}

	if len(labels) == 0 {
		return "manifest"
	}

	return strings.Join(labels, ", ")
}

func selectSBOMPath(savedPaths []string) string {
	for _, path := range savedPaths {
		if strings.HasSuffix(path, "sbom-cyclonedx.xml") {
			return path
		}
	}
	for _, path := range savedPaths {
		if strings.HasSuffix(path, ".cdx.xml") {
			return path
		}
	}
	for _, path := range savedPaths {
		if strings.HasSuffix(path, "sbom.spdx") {
			return path
		}
	}
	return ""
}

func grypeScanTarget(savedPaths []string, repoPath string) string {
	if sbomPath := selectSBOMPath(savedPaths); sbomPath != "" {
		return "sbom:" + sbomPath
	}
	return repoPath
}

func resolveScanRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}

	root = os.ExpandEnv(root)
	if root == "~" || strings.HasPrefix(root, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		if root == "~" {
			root = home
		} else {
			root = filepath.Join(home, strings.TrimPrefix(root, "~/"))
		}
	}

	return filepath.Abs(root)
}

func normalizeExportFormat(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case formatCycloneDX:
		return formatCycloneDX, nil
	case formatSPDX:
		return formatSPDX, nil
	case formatBoth:
		return formatBoth, nil
	default:
		return "", fmt.Errorf("%q (expected cyclonedx, spdx, or both)", value)
	}
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `SBOMber scans workspaces of local Git repositories.

Usage:
  sbomber
  sbomber scan [path...] [--format cyclonedx|spdx|both] [--include-vulnerabilities] [--fail-on-vuln] [--output DIR] [--severity-threshold SEV]
  sbomber github [--health] [--include-vulnerabilities] [--fail-on-vuln] [--output DIR] [--severity-threshold SEV] [--format FORMAT] <repo-url>...
  sbomber trace <path> [package-name] [flags]
  sbomber verify <ground-truth-sbom> <generated-sbom> [--json]
  sbomber version

Scan Flags:
  --format cyclonedx|spdx|both          Export format (default: cyclonedx)
  --include-vulnerabilities             Enable vulnerability scanning with Grype
  --fail-on-vuln                        Exit with code 1 when vulnerabilities are found
  --output DIR                          Custom output directory for SBOMs and reports
  --severity-threshold SEV              Only report/fail on critical, high, medium, or low+ findings
  --health                              Include supply chain health metrics (github command)

Trace Flags:
  --tree                                Show full dependency tree
  --list                                List all dependencies with filters
  --ecosystem <name>                    Filter by ecosystem (npm, maven, pypi, golang, rubygems)
  --scope <name>                        Filter by build-scope (runtime, dev, test, build-tooling)
  --type <name>                         Filter by dependency-type (direct, transitive)
  --source-file <path>                  Filter by source manifest file
  --min-depth <n>                       Minimum depth (default: 0)
  --max-depth <n>                       Maximum depth (default: no limit)

Verify Flags:
  --json                                Output results as JSON (for CI/CD)

Examples:
  sbomber
  sbomber scan .
  sbomber scan ./repo1 ./repo2 ./repo3 --format cyclonedx
  sbomber scan ../workspace --format both --include-vulnerabilities
  sbomber github https://github.com/expressjs/express
  sbomber github --health --include-vulnerabilities https://github.com/lodash/lodash
  sbomber trace . lodash
  sbomber trace . express --tree
  sbomber trace . --list --ecosystem npm
  sbomber trace . --list --type transitive --min-depth 2
  sbomber verify reference.cdx.xml my-output.cdx.xml
  sbomber verify benchmark.json generated.json --json

Environment:
  GITHUB_TOKEN    GitHub personal access token (recommended for higher rate limits)
`)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
