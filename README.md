<p align="center">
  <img src="./docs/assets/Banner.png" alt="SBOMber banner" width="100%" />
</p>

<p align="center">
  <a href="https://github.com/fluxsecurity/SBOMber/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/fluxsecurity/SBOMber/actions/workflows/ci.yml/badge.svg" /></a>
  <img alt="Go version" src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go" />
  <img alt="License" src="https://img.shields.io/badge/License-Apache--2.0-blue?style=flat-square" />
</p>

<p align="center">
  <strong>Scan repositories. Generate SBOMs. Find vulnerabilities. Verify accuracy.</strong>
</p>

---

SBOMber is a high-performance, open-source CLI that scans local and remote Git repositories, extracts dependencies, generates SBOM artifacts (CycloneDX/SPDX), finds CVEs, and provides supply chain health insights.

## Highlights

- **Remote GitHub scanning** - Scan any public/private repo without cloning
- **Parallel processing** - Fetch manifests concurrently for 10x faster scans
- **Dependency chain tracking** - Trace how any package enters your project
- **SBOM verification** - Compare output against ground truth with precision/recall metrics
- **Supply chain health** - Risk scoring based on maintainer activity, contributors, and more
- **Supports** - npm, Python, Go, Maven, Ruby ecosystems

## Quick Start

```bash
# Build
make build

# Run interactive TUI
./bin/sbomber

# Scan local repos
./bin/sbomber scan /path/to/repos --format cyclonedx --include-vulnerabilities

# Scan GitHub repos directly (no clone needed!)
./bin/sbomber github https://github.com/expressjs/express
```

**Requirements:** Go 1.26+ and [Grype](https://github.com/anchore/grype) (for vulnerability scanning)

## Features

| Feature | Status |
|---------|--------|
| Interactive TUI | done |
| Multi-repo Git scanning | done |
| npm/yarn dependencies | done |
| Python requirements.txt | done |
| Maven pom.xml | done |
| Ruby Gemfile.lock | done |
| Go go.mod | done |
| CycloneDX export | done |
| SPDX export | done |
| Grype vulnerability scan | done |
| HTML vulnerability report | done |
| Remote GitHub scanning | done |
| Parallel manifest fetching | done |
| Supply chain health metrics | done |
| Dependency chain tracing | done |
| SBOM accuracy verification | done |
| Direct/transitive filtering | done |

---

## Usage

### Interactive Mode

```bash
./bin/sbomber
```

Navigate with arrow keys, select with Enter. The TUI provides guided workflows for all features.

### Local Scanning

```bash
# Scan current directory
./bin/sbomber scan .

# Scan with both SBOM formats
./bin/sbomber scan /path/to/repos --format both

# Include vulnerability scanning
./bin/sbomber scan /path/to/repos --include-vulnerabilities
```

### GitHub Remote Scanning

Scan any GitHub repository directly via API - no cloning required.

```bash
# Basic scan
./bin/sbomber github https://github.com/expressjs/express

# With supply chain health metrics
./bin/sbomber github --health https://github.com/django/django

# With vulnerability scanning
./bin/sbomber github --include-vulnerabilities https://github.com/lodash/lodash

# Multiple repos at once
./bin/sbomber github https://github.com/org/repo1 https://github.com/org/repo2
```

**Authentication:** Set `GITHUB_TOKEN` for higher rate limits (5000 req/hr vs 60 req/hr).

```bash
export GITHUB_TOKEN=ghp_your_token_here
```

### Dependency Chain Tracing

Understand how dependencies enter your project.

```bash
# Trace a specific package
./bin/sbomber trace . lodash

# Show full dependency tree
./bin/sbomber trace . express --tree

# Filter by ecosystem
./bin/sbomber trace . --list --ecosystem npm

# Filter transitive dependencies only
./bin/sbomber trace . --list --type transitive

# Show potential false positives (test/example deps)
./bin/sbomber trace . --fp --list
```

### SBOM Verification

Compare your generated SBOM against a committed ground truth to measure accuracy.

> The repository includes a verified benchmark fixture at `testdata/benchmarks/npm-basic/ground-truth.json`. Accuracy figures are only meaningful when the reference SBOM is committed and verified in-repo.

```bash
# Compare against a committed reference SBOM
./bin/sbomber verify testdata/benchmarks/npm-basic/ground-truth.json testdata/benchmarks/npm-basic/generated.json

# Output as JSON (for CI/CD)
./bin/sbomber verify testdata/benchmarks/npm-basic/ground-truth.json testdata/benchmarks/npm-basic/generated.json --json
```

**Verified benchmark result for the committed fixture:**
```
Precision:        100.0%
Recall:           100.0%
F1 Score:         100.0%
Version Accuracy: 100.0%

Overall Grade: A+ (Excellent)
```

---

## Performance

SBOMber uses **parallel processing** for maximum speed:

| Operation | Approach | Benefit |
|-----------|----------|---------|
| Manifest fetching | Concurrent goroutines | 10x faster for multi-manifest repos |
| Health resolution | Worker pool (10 workers) | Parallel API calls per dependency |
| File parsing | Streaming parsers | Low memory footprint |

**Example:** Scanning Facebook React (157 manifest files) completes in ~2 seconds.

---

## Output

Reports are saved to `~/.sbomber/reports/`:

```
~/.sbomber/reports/
├── my-project_abc123/
│   ├── sbom-cyclonedx.xml      # CycloneDX 1.5
│   ├── sbom.spdx               # SPDX 2.3
│   └── vulnerability-report.html
└── github-scan_def456/
    └── expressjs_express/
        ├── sbom-cyclonedx.xml
        └── vulnerability-report.html
```

### HTML Vulnerability Report

Interactive report with:
- Severity breakdown (Critical/High/Medium/Low)
- Direct vs transitive dependency filtering
- Test data detection (auto-flags fixtures/examples)
- False positive marking
- CVE links to NVD/GitHub advisories

---

## Supply Chain Health

When scanning with `--health`, SBOMber evaluates each dependency:

| Metric | What it measures |
|--------|------------------|
| Last commit | Days since last activity |
| Contributors | Bus factor / maintainer risk |
| Stars | Community adoption |
| License | Compliance risk |

Risk levels: **Low** (healthy) / **Medium** (monitor) / **High** (investigate)

---

## Development

```bash
make fmt      # format code
make test     # run tests  
make vet      # run go vet
make build    # build binary
make ci       # run full CI pipeline
```

## License

[Apache-2.0](./LICENSE)
