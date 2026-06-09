package sbom

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	SBOMberDir = ".sbomber"
	ReportsDir = "reports"
)

// GetOutputDir returns the central output directory for a project.
// Output is stored at ~/.sbomber/reports/<sanitized-project-name>/
func GetOutputDir(projectPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	projectName := sanitizeProjectName(projectPath)
	outputDir := filepath.Join(home, SBOMberDir, ReportsDir, projectName)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	return outputDir, nil
}

// GetSBOMberHome returns the SBOMber home directory (~/.sbomber)
func GetSBOMberHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	sbomberHome := filepath.Join(home, SBOMberDir)
	if err := os.MkdirAll(sbomberHome, 0o755); err != nil {
		return "", fmt.Errorf("create sbomber home: %w", err)
	}

	return sbomberHome, nil
}

// GetBatchOutputDir creates a timestamped directory for batch scans.
// Output is stored at ~/.sbomber/reports/<scan-name>_<timestamp>/
// Returns the batch directory path.
func GetBatchOutputDir(scanName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	safeName := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(scanName, "_")
	if safeName == "" {
		safeName = "batch-scan"
	}

	timestamp := strings.ReplaceAll(strings.ReplaceAll(
		strings.Split(fmt.Sprintf("%s", time.Now().Format(time.RFC3339)), "T")[0]+
			"_"+strings.Split(fmt.Sprintf("%s", time.Now().Format(time.RFC3339)), "T")[1][:8],
		":", "-"), "Z", "")

	batchDir := filepath.Join(home, SBOMberDir, ReportsDir, fmt.Sprintf("%s_%s", strings.ToLower(safeName), timestamp))

	if err := os.MkdirAll(batchDir, 0o755); err != nil {
		return "", fmt.Errorf("create batch output directory: %w", err)
	}

	return batchDir, nil
}

// ResolveOutputDir returns the output directory for a scan.
// When customOutput is empty, the default ~/.sbomber/reports/<project> path is used.
func ResolveOutputDir(customOutput, projectPath string) (string, error) {
	if strings.TrimSpace(customOutput) == "" {
		return GetOutputDir(projectPath)
	}

	absPath, err := filepath.Abs(customOutput)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	return absPath, nil
}

// ResolveBatchOutputDir returns the directory for multi-repo scans.
// When customOutput is set it is used directly; otherwise a timestamped batch dir is created.
func ResolveBatchOutputDir(customOutput, scanName string) (string, error) {
	if strings.TrimSpace(customOutput) == "" {
		return GetBatchOutputDir(scanName)
	}
	return ResolveOutputDir(customOutput, scanName)
}

// GetRepoOutputDir creates a subdirectory for a specific repo within a batch scan.
func GetRepoOutputDir(batchDir, repoName string) (string, error) {
	safeName := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(repoName, "_")
	if safeName == "" {
		safeName = "repo"
	}

	repoDir := filepath.Join(batchDir, strings.ToLower(safeName))
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return "", fmt.Errorf("create repo output directory: %w", err)
	}

	return repoDir, nil
}

// sanitizeProjectName creates a safe directory name from a project path
func sanitizeProjectName(projectPath string) string {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		absPath = projectPath
	}

	baseName := filepath.Base(absPath)
	if baseName == "." || baseName == "/" {
		baseName = "root"
	}

	safeName := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(baseName, "_")
	if safeName == "" {
		safeName = "project"
	}

	hash := sha256.Sum256([]byte(absPath))
	shortHash := hex.EncodeToString(hash[:])[:8]

	return fmt.Sprintf("%s_%s", strings.ToLower(safeName), shortHash)
}
