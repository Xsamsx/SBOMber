package health

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Xsamsx/SBOMber/internal/deps"
	"github.com/Xsamsx/SBOMber/internal/github"
)

// DependencyHealth combines dependency info with supply chain health metrics.
type DependencyHealth struct {
	deps.Dependency
	RepoURL        string
	RepoOwner      string
	RepoName       string
	Stars          int
	Forks          int
	Contributors   int
	LastCommit     time.Time
	CommitActivity string
	RiskLevel      string
	License        string
	Error          string
}

// Resolver fetches health metrics for dependencies.
type Resolver struct {
	ghClient   *github.Client
	httpClient *http.Client
	cache      map[string]*DependencyHealth
	cacheMu    sync.RWMutex
}

// NewResolver creates a new health resolver.
func NewResolver(ghClient *github.Client) *Resolver {
	return &Resolver{
		ghClient:   ghClient,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache:      make(map[string]*DependencyHealth),
	}
}

// ResolveHealth fetches health metrics for a single dependency.
func (r *Resolver) ResolveHealth(dep deps.Dependency) *DependencyHealth {
	key := dep.Ecosystem + ":" + dep.Name

	r.cacheMu.RLock()
	if cached, ok := r.cache[key]; ok {
		r.cacheMu.RUnlock()
		return cached
	}
	r.cacheMu.RUnlock()

	health := &DependencyHealth{Dependency: dep}

	repoURL, err := r.resolveRepoURL(dep)
	if err != nil {
		health.Error = err.Error()
		r.cacheMu.Lock()
		r.cache[key] = health
		r.cacheMu.Unlock()
		return health
	}

	health.RepoURL = repoURL
	owner, repo, err := github.ParseRepoURL(repoURL)
	if err != nil {
		health.Error = "not a GitHub repo"
		r.cacheMu.Lock()
		r.cache[key] = health
		r.cacheMu.Unlock()
		return health
	}

	health.RepoOwner = owner
	health.RepoName = repo

	metrics, err := r.ghClient.GetHealthMetrics(owner, repo)
	if err != nil {
		health.Error = err.Error()
		r.cacheMu.Lock()
		r.cache[key] = health
		r.cacheMu.Unlock()
		return health
	}

	health.Stars = metrics.Stars
	health.Forks = metrics.Forks
	health.Contributors = metrics.Contributors
	health.LastCommit = metrics.LastCommitDate
	health.CommitActivity = metrics.CommitFrequency
	health.RiskLevel = metrics.RiskLevel
	health.License = metrics.License

	r.cacheMu.Lock()
	r.cache[key] = health
	r.cacheMu.Unlock()
	return health
}

// ProgressFunc is called with progress updates during resolution.
type ProgressFunc func(current, total int, depName string)

// ResolveAll fetches health metrics for all dependencies.
func (r *Resolver) ResolveAll(dependencies []deps.Dependency) []*DependencyHealth {
	return r.ResolveAllWithProgress(dependencies, nil)
}

// ResolveAllWithProgress fetches health metrics with progress callback (parallel).
func (r *Resolver) ResolveAllWithProgress(dependencies []deps.Dependency, progress ProgressFunc) []*DependencyHealth {
	results := make([]*DependencyHealth, len(dependencies))

	// Use worker pool for parallel fetching
	const maxWorkers = 10
	jobs := make(chan int, len(dependencies))
	done := make(chan struct{})

	var completed int
	var mu sync.Mutex

	// Start workers
	for w := 0; w < maxWorkers; w++ {
		go func() {
			for i := range jobs {
				dep := dependencies[i]
				result := r.ResolveHealth(dep)
				results[i] = result

				mu.Lock()
				completed++
				if progress != nil {
					progress(completed, len(dependencies), dep.Name)
				}
				mu.Unlock()
			}
			done <- struct{}{}
		}()
	}

	// Send jobs
	for i := range dependencies {
		jobs <- i
	}
	close(jobs)

	// Wait for workers
	for w := 0; w < maxWorkers; w++ {
		<-done
	}

	return results
}

func (r *Resolver) resolveRepoURL(dep deps.Dependency) (string, error) {
	switch dep.Ecosystem {
	case "npm":
		return r.resolveNpmRepo(dep.Name)
	case "pypi":
		return r.resolvePyPIRepo(dep.Name)
	case "go", "golang":
		return r.resolveGoRepo(dep.Name)
	case "rubygems":
		return r.resolveRubyRepo(dep.Name)
	case "maven":
		return r.resolveMavenRepo(dep.Name)
	default:
		return "", fmt.Errorf("unsupported ecosystem: %s", dep.Ecosystem)
	}
}

type npmPackageInfo struct {
	Repository struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"repository"`
}

func (r *Resolver) resolveNpmRepo(name string) (string, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", name)
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("npm registry returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var pkg npmPackageInfo
	if err := json.Unmarshal(body, &pkg); err != nil {
		return "", err
	}

	repoURL := pkg.Repository.URL
	if repoURL == "" {
		return "", fmt.Errorf("no repository URL found")
	}

	return normalizeGitURL(repoURL), nil
}

type pypiPackageInfo struct {
	Info struct {
		ProjectURLs map[string]string `json:"project_urls"`
		HomePage    string            `json:"home_page"`
	} `json:"info"`
}

func (r *Resolver) resolvePyPIRepo(name string) (string, error) {
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", name)
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("PyPI returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var pkg pypiPackageInfo
	if err := json.Unmarshal(body, &pkg); err != nil {
		return "", err
	}

	for key, url := range pkg.Info.ProjectURLs {
		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "source") || strings.Contains(keyLower, "repository") || strings.Contains(keyLower, "github") {
			if strings.Contains(url, "github.com") {
				return normalizeGitURL(url), nil
			}
		}
	}

	if strings.Contains(pkg.Info.HomePage, "github.com") {
		return normalizeGitURL(pkg.Info.HomePage), nil
	}

	return "", fmt.Errorf("no GitHub repository found")
}

func (r *Resolver) resolveGoRepo(name string) (string, error) {
	if strings.HasPrefix(name, "github.com/") {
		parts := strings.Split(name, "/")
		if len(parts) >= 3 {
			return fmt.Sprintf("https://github.com/%s/%s", parts[1], parts[2]), nil
		}
	}
	return "", fmt.Errorf("cannot resolve non-GitHub Go module")
}

type rubyGemInfo struct {
	SourceCodeURI string `json:"source_code_uri"`
	HomepageURI   string `json:"homepage_uri"`
}

func (r *Resolver) resolveRubyRepo(name string) (string, error) {
	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", name)
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("RubyGems returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var gem rubyGemInfo
	if err := json.Unmarshal(body, &gem); err != nil {
		return "", err
	}

	if gem.SourceCodeURI != "" && strings.Contains(gem.SourceCodeURI, "github.com") {
		return normalizeGitURL(gem.SourceCodeURI), nil
	}
	if gem.HomepageURI != "" && strings.Contains(gem.HomepageURI, "github.com") {
		return normalizeGitURL(gem.HomepageURI), nil
	}

	return "", fmt.Errorf("no GitHub repository found")
}

func (r *Resolver) resolveMavenRepo(name string) (string, error) {
	groupID, artifactID, ok := strings.Cut(name, ":")
	if !ok || groupID == "" || artifactID == "" {
		return "", fmt.Errorf("invalid maven coordinate: %s", name)
	}

	version, err := r.lookupMavenLatestVersion(groupID, artifactID)
	if err != nil {
		return "", err
	}

	pomURL := mavenPOMURL(groupID, artifactID, version)
	resp, err := r.httpClient.Get(pomURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("maven central returned %d for %s", resp.StatusCode, pomURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if repoURL := extractGitHubURLFromPOM(string(body)); repoURL != "" {
		return repoURL, nil
	}

	return "", fmt.Errorf("no GitHub repository found in POM")
}

type mavenSearchResponse struct {
	Response struct {
		Docs []struct {
			LatestVersion string `json:"latestVersion"`
		} `json:"docs"`
	} `json:"response"`
}

func (r *Resolver) lookupMavenLatestVersion(groupID, artifactID string) (string, error) {
	query := url.QueryEscape(fmt.Sprintf(`g:"%s" AND a:"%s"`, groupID, artifactID))
	searchURL := fmt.Sprintf("https://search.maven.org/solrsearch/select?q=%s&rows=1&wt=json", query)

	resp, err := r.httpClient.Get(searchURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("maven search returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var search mavenSearchResponse
	if err := json.Unmarshal(body, &search); err != nil {
		return "", err
	}
	if len(search.Response.Docs) == 0 || search.Response.Docs[0].LatestVersion == "" {
		return "", fmt.Errorf("artifact not found on Maven Central")
	}

	return search.Response.Docs[0].LatestVersion, nil
}

func mavenPOMURL(groupID, artifactID, version string) string {
	groupPath := strings.ReplaceAll(groupID, ".", "/")
	return fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/%s/%s-%s.pom", groupPath, artifactID, version, artifactID, version)
}

var pomGitHubPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<scm>\s*<url>\s*(https?://github\.com/[^<]+)\s*</url>`),
	regexp.MustCompile(`(?i)<url>\s*(https?://github\.com/[^<]+)\s*</url>`),
}

func extractGitHubURLFromPOM(content string) string {
	for _, pattern := range pomGitHubPatterns {
		if matches := pattern.FindStringSubmatch(content); len(matches) >= 2 {
			return normalizeGitURL(strings.TrimSpace(matches[1]))
		}
	}
	return ""
}

var gitURLPattern = regexp.MustCompile(`github\.com[:/]([^/]+)/([^/\.]+)`)

func normalizeGitURL(url string) string {
	url = strings.TrimPrefix(url, "git+")
	url = strings.TrimPrefix(url, "git://")
	url = strings.TrimSuffix(url, ".git")

	matches := gitURLPattern.FindStringSubmatch(url)
	if len(matches) >= 3 {
		return fmt.Sprintf("https://github.com/%s/%s", matches[1], matches[2])
	}

	return url
}
