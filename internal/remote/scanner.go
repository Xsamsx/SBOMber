package remote

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Xsamsx/SBOMber/internal/deps"
	"github.com/Xsamsx/SBOMber/internal/github"
)

// KnownManifests maps manifest filenames to their ecosystems.
var KnownManifests = map[string]string{
	"package.json":      "npm",
	"package-lock.json": "npm",
	"yarn.lock":         "npm",
	"requirements.txt":  "pypi",
	"Pipfile.lock":      "pypi",
	"go.mod":            "go",
	"go.sum":            "go",
	"pom.xml":           "maven",
	"Gemfile.lock":      "rubygems",
}

// ProgressFunc is called to report scanning progress.
type ProgressFunc func(message string)

// Scanner scans remote GitHub repositories for dependencies.
type Scanner struct {
	client   *github.Client
	progress ProgressFunc
}

// ScanResult contains the results from scanning a remote repository.
type ScanResult struct {
	RepoURL    string
	Owner      string
	Repo       string
	Summary    deps.Summary
	Manifests  []string
	RepoHealth *github.HealthMetrics
	Error      error
}

// NewScanner creates a new remote repository scanner.
func NewScanner(client *github.Client) *Scanner {
	return &Scanner{client: client}
}

// SetProgress sets the progress callback function.
func (s *Scanner) SetProgress(fn ProgressFunc) {
	s.progress = fn
}

func (s *Scanner) log(msg string) {
	if s.progress != nil {
		s.progress(msg)
	}
}

// ScanRepo scans a single GitHub repository and extracts dependencies.
func (s *Scanner) ScanRepo(repoURL string) (*ScanResult, error) {
	owner, repo, err := github.ParseRepoURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub URL: %w", err)
	}

	s.log(fmt.Sprintf("Parsing URL: %s/%s", owner, repo))

	result := &ScanResult{
		RepoURL:   repoURL,
		Owner:     owner,
		Repo:      repo,
		Manifests: make([]string, 0),
		Summary: deps.Summary{
			Direct:     make([]deps.Dependency, 0),
			Transitive: make([]deps.Dependency, 0),
		},
	}

	s.log("Fetching repository tree...")
	tree, err := s.client.GetRepoTree(owner, repo, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get repo tree: %w", err)
	}
	s.log(fmt.Sprintf("Found %d files in repository", len(tree)))

	manifestFiles := findManifestFiles(tree)
	if len(manifestFiles) == 0 {
		s.log("No manifest files found")
		return result, nil
	}

	s.log(fmt.Sprintf("Found %d manifest files", len(manifestFiles)))
	for _, mf := range manifestFiles {
		s.log(fmt.Sprintf("  - %s (%s)", mf.Path, mf.Ecosystem))
	}

	// Fetch and parse all manifests in parallel
	type fetchResult struct {
		manifest manifestFile
		content  []byte
		parsed   *deps.Summary
		err      error
		fetchErr error
	}

	results := make(chan fetchResult, len(manifestFiles))
	var wg sync.WaitGroup

	s.log("Fetching manifests in parallel...")
	for _, mf := range manifestFiles {
		wg.Add(1)
		go func(mf manifestFile) {
			defer wg.Done()
			fr := fetchResult{manifest: mf}

			content, err := s.client.GetFileContent(owner, repo, mf.Path)
			if err != nil {
				fr.fetchErr = err
				results <- fr
				return
			}
			fr.content = content.Content

			parsed, err := parseManifestContent(mf.Path, content.Content)
			if err != nil {
				fr.err = err
				results <- fr
				return
			}
			fr.parsed = &parsed
			results <- fr
		}(mf)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for fr := range results {
		if fr.fetchErr != nil {
			s.log(fmt.Sprintf("  Error fetching %s: %v", fr.manifest.Path, fr.fetchErr))
			continue
		}
		if fr.err != nil {
			s.log(fmt.Sprintf("  Error parsing %s: %v", fr.manifest.Path, fr.err))
			continue
		}

		depCount := len(fr.parsed.Direct) + len(fr.parsed.Transitive)
		s.log(fmt.Sprintf("  %s: %d dependencies", fr.manifest.Path, depCount))

		result.Manifests = append(result.Manifests, fr.manifest.Path)
		result.Summary.Direct = append(result.Summary.Direct, fr.parsed.Direct...)
		result.Summary.Transitive = append(result.Summary.Transitive, fr.parsed.Transitive...)
	}

	result.Summary.Direct = deduplicateDeps(result.Summary.Direct)
	result.Summary.Transitive = deduplicateDeps(result.Summary.Transitive)
	s.log(fmt.Sprintf("Total: %d direct, %d transitive dependencies", len(result.Summary.Direct), len(result.Summary.Transitive)))

	return result, nil
}

// ScanRepos scans multiple GitHub repositories concurrently.
func (s *Scanner) ScanRepos(repoURLs []string) []*ScanResult {
	results := make([]*ScanResult, len(repoURLs))

	for i, url := range repoURLs {
		result, err := s.ScanRepo(url)
		if err != nil {
			results[i] = &ScanResult{
				RepoURL: url,
				Error:   err,
			}
		} else {
			results[i] = result
		}
	}

	return results
}

// FetchHealthMetrics fetches health metrics for a scan result.
func (s *Scanner) FetchHealthMetrics(result *ScanResult) error {
	metrics, err := s.client.GetHealthMetrics(result.Owner, result.Repo)
	if err != nil {
		return err
	}
	result.RepoHealth = metrics
	return nil
}

type manifestFile struct {
	Path      string
	Ecosystem string
}

func findManifestFiles(tree []github.TreeEntry) []manifestFile {
	var manifests []manifestFile

	for _, entry := range tree {
		if entry.Type != "blob" {
			continue
		}

		filename := getFilename(entry.Path)
		if ecosystem, ok := KnownManifests[filename]; ok {
			manifests = append(manifests, manifestFile{
				Path:      entry.Path,
				Ecosystem: ecosystem,
			})
		}
	}

	return manifests
}

func getFilename(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func deduplicateDeps(depsSlice []deps.Dependency) []deps.Dependency {
	seen := make(map[string]bool)
	unique := make([]deps.Dependency, 0, len(depsSlice))

	for _, dep := range depsSlice {
		key := dep.Ecosystem + ":" + dep.Name + ":" + dep.Version
		if !seen[key] {
			seen[key] = true
			unique = append(unique, dep)
		}
	}

	return unique
}
