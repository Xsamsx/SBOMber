package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Xsamsx/SBOMber/internal/vulnerability"
)

func TestScanCleanNpmRepoExitZero(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repo := filepath.Join(root, "clean-app")
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustWriteFile(t, filepath.Join(repo, "package.json"), `{
  "dependencies": {
    "left-pad": "1.0.0"
  }
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{"scan", repo}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}
}

func TestScanMissingLockfileNoCrash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repo := filepath.Join(root, "no-lock")
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustWriteFile(t, filepath.Join(repo, "package.json"), `{"dependencies":{"ms":"2.1.3"}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{"scan", repo}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}
}

func TestScanMultiRepoUsesBatchMode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	mustMkdirAll(t, filepath.Join(repoA, ".git"))
	mustMkdirAll(t, filepath.Join(repoB, ".git"))
	mustWriteFile(t, filepath.Join(repoA, "package.json"), `{"dependencies":{"a":"1.0.0"}}`)
	mustWriteFile(t, filepath.Join(repoB, "package.json"), `{"dependencies":{"b":"1.0.0"}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{"scan", repoA, repoB}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Batch output folder:") {
		t.Fatalf("expected batch output folder in output, got %q", output)
	}
	if !strings.Contains(output, "Scan complete: 2 repositories scanned") {
		t.Fatalf("expected 2 repositories scanned, got %q", output)
	}
}

func TestScanNestedManifestFindsWorkspacePackage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repo := filepath.Join(root, "mono")
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustMkdirAll(t, filepath.Join(repo, "packages", "api"))
	mustWriteFile(t, filepath.Join(repo, "packages", "api", "package.json"), `{
  "dependencies": {
    "express": "4.18.2"
  }
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{"scan", repo}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "packages:  1 direct") {
		t.Fatalf("expected nested package dependency count in output, got %q", output)
	}
}

func TestScanFailOnVulnWithKnownVulnerablePackage(t *testing.T) {
	if !vulnerability.IsGrypeAvailable() {
		t.Skip("grype not available in PATH")
	}

	root := t.TempDir()
	repo := filepath.Join(root, "vuln-app")
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustWriteFile(t, filepath.Join(repo, "package.json"), `{
  "dependencies": {
    "lodash": "4.17.4"
  }
}`)
	mustWriteFile(t, filepath.Join(repo, "package-lock.json"), `{
  "packages": {
    "": { "version": "1.0.0" },
    "node_modules/lodash": { "version": "4.17.4" }
  }
}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{
		"scan", "--include-vulnerabilities", "--fail-on-vuln", repo,
	}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for vulnerable repo, got %d, stderr=%q stdout=%q", exitCode, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "findings.json") {
		t.Fatalf("expected findings.json in output, got %q", stdout.String())
	}
}

func TestScanInvalidSeverityThreshold(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repo := filepath.Join(root, "app")
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustWriteFile(t, filepath.Join(repo, "package.json"), `{"dependencies":{"ms":"2.1.3"}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{"scan", "--severity-threshold", "urgent", repo}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "invalid severity threshold") {
		t.Fatalf("expected severity validation error, got stderr=%q", stderr.String())
	}
}

func TestScanFlagsAfterPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repo := filepath.Join(root, "flags-after-path")
	outDir := filepath.Join(root, "custom-out")
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustWriteFile(t, filepath.Join(repo, "package.json"), `{"dependencies":{"ms":"2.1.3"}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{"scan", repo, "--output", outDir}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), outDir) {
		t.Fatalf("expected custom output dir in stdout, got %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "sbom-cyclonedx.xml")); err != nil {
		t.Fatalf("expected SBOM in custom output dir: %v", err)
	}
}

func TestScanCustomOutputDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repo := filepath.Join(root, "app")
	outDir := filepath.Join(root, "custom-out")
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	mustWriteFile(t, filepath.Join(repo, "package.json"), `{"dependencies":{"ms":"2.1.3"}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Main([]string{"scan", "--output", outDir, repo}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), outDir) {
		t.Fatalf("expected custom output dir in stdout, got %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "sbom-cyclonedx.xml")); err != nil {
		t.Fatalf("expected SBOM in custom output dir: %v", err)
	}
}
