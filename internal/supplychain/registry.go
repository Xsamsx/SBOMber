package supplychain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var registryHTTPClient = &http.Client{Timeout: 15 * time.Second}

// PackageExistsOnRegistry checks whether a package name resolves on the public registry.
func PackageExistsOnRegistry(ctx context.Context, ecosystem, name string) (bool, error) {
	checkURL, err := registryCheckURL(ecosystem, name)
	if err != nil {
		return false, err
	}
	if checkURL == "" {
		return true, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "SBOMber/1.0")

	resp, err := registryHTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		return false, nil
	case http.StatusOK, http.StatusMultipleChoices:
		if normalizeEcosystem(ecosystem) == "maven" || normalizeEcosystem(ecosystem) == "java" {
			var parsed struct {
				Response struct {
					NumFound int `json:"numFound"`
				} `json:"response"`
			}
			if err := json.Unmarshal(body, &parsed); err != nil {
				return false, err
			}
			return parsed.Response.NumFound > 0, nil
		}
		return true, nil
	default:
		return false, fmt.Errorf("registry check returned %d", resp.StatusCode)
	}
}

func registryCheckURL(ecosystem, name string) (string, error) {
	switch normalizeEcosystem(ecosystem) {
	case "npm":
		escaped := url.PathEscape(name)
		if strings.HasPrefix(name, "@") {
			parts := strings.SplitN(name, "/", 2)
			if len(parts) == 2 {
				escaped = "@" + url.PathEscape(parts[0][1:]) + "%2F" + url.PathEscape(parts[1])
			}
		}
		return "https://registry.npmjs.org/" + escaped, nil
	case "pypi", "python", "pip":
		return "https://pypi.org/pypi/" + url.PathEscape(name) + "/json", nil
	case "rubygems", "ruby":
		return "https://rubygems.org/api/v1/gems/" + url.PathEscape(name) + ".json", nil
	case "maven", "java":
		parts := strings.Split(name, ":")
		if len(parts) != 2 {
			return "", nil
		}
		query := url.Values{}
		query.Set("q", fmt.Sprintf(`g:"%s" AND a:"%s"`, parts[0], parts[1]))
		query.Set("rows", "1")
		query.Set("wt", "json")
		return "https://search.maven.org/solrsearch/select?" + query.Encode(), nil
	case "go", "golang":
		module := name
		if idx := strings.Index(module, "@"); idx > 0 {
			module = module[:idx]
		}
		return "https://proxy.golang.org/" + url.PathEscape(strings.ToLower(module)) + "/@v/list", nil
	default:
		return "", nil
	}
}

func normalizeEcosystem(ecosystem string) string {
	return strings.ToLower(strings.TrimSpace(ecosystem))
}

func looksLikePrivatePackage(name string) bool {
	lower := strings.ToLower(name)
	privateHints := []string{
		"internal", "private", "corp", "company", "enterprise",
		"local", "proprietary", "intranet", "inhouse", "in-house",
	}
	for _, hint := range privateHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}
