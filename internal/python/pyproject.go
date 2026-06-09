package python

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

var pyprojectDepPattern = regexp.MustCompile(`^["']([^"']+)["']`)

// ParsePyProject reads [project].dependencies from pyproject.toml.
func ParsePyProject(root string) (deps.Summary, error) {
	path := filepath.Join(root, "pyproject.toml")
	f, err := os.Open(path)
	if err != nil {
		return deps.Summary{}, err
	}
	defer f.Close()

	summary := deps.Summary{
		Direct: make([]deps.Dependency, 0),
	}

	inProject := false
	inDependencies := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.Trim(line, "[]")
			inProject = section == "project"
			inDependencies = false
			continue
		}

		if !inProject {
			continue
		}

		if strings.HasPrefix(line, "dependencies") {
			if strings.Contains(line, "[") && strings.Contains(line, "]") {
				inDependencies = true
				if dep := extractInlinePyprojectDependency(line); dep != "" {
					appendPyprojectDep(&summary, dep, path)
				}
				continue
			}
			if strings.Contains(line, "=") {
				inDependencies = true
				continue
			}
		}

		if !inDependencies {
			continue
		}

		if line == "]" {
			inDependencies = false
			continue
		}

		if strings.HasPrefix(line, "[") {
			inDependencies = false
			continue
		}

		line = strings.TrimSuffix(strings.TrimSpace(line), ",")
		if dep := extractQuotedPyprojectDependency(line); dep != "" {
			appendPyprojectDep(&summary, dep, path)
		}
	}

	if err := scanner.Err(); err != nil {
		return deps.Summary{}, err
	}

	if len(summary.Direct) > 1 {
		sort.SliceStable(summary.Direct, func(i, j int) bool {
			return summary.Direct[i].Name < summary.Direct[j].Name
		})
	}

	return summary, nil
}

func extractInlinePyprojectDependency(line string) string {
	start := strings.Index(line, "[")
	end := strings.LastIndex(line, "]")
	if start == -1 || end <= start {
		return ""
	}
	return extractQuotedPyprojectDependency(strings.TrimSpace(line[start+1 : end]))
}

func extractQuotedPyprojectDependency(line string) string {
	matches := pyprojectDepPattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func appendPyprojectDep(summary *deps.Summary, requirement, sourcePath string) {
	name, version := parsePythonRequirementLine(requirement)
	if name == "" {
		return
	}
	summary.Direct = append(summary.Direct, deps.Dependency{
		Name:           name,
		Version:        version,
		Scope:          deps.ScopeRuntime,
		Ecosystem:      "pypi",
		IsDirect:       true,
		SourceFile:     "pyproject.toml",
		SourceLocation: sourcePath,
	})
}
