package python

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

// ParseRequirements reads Python requirements files and returns a normalized dependency summary.
// It currently supports requirements.txt and requirements-dev.txt.
func ParseRequirements(root string) (deps.Summary, error) {
	summary := deps.Summary{
		Direct: make([]deps.Dependency, 0),
	}

	parseFile := func(filename string, scope deps.Scope) error {
		path := filepath.Join(root, filename)
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		defer func() { _ = f.Close() }()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "-r ") || strings.HasPrefix(line, "--requirement") || strings.HasPrefix(line, "--") {
				continue
			}

			name, version := parsePythonRequirementLine(line)
			if name == "" {
				continue
			}

			summary.Direct = append(summary.Direct, deps.Dependency{
				Name:           name,
				Version:        version,
				Scope:          scope,
				Ecosystem:      "pypi",
				IsDirect:       true,
				SourceFile:     filename,
				SourceLocation: path,
			})
		}

		return scanner.Err()
	}

	if err := parseFile("requirements.txt", deps.ScopeRuntime); err != nil {
		return deps.Summary{}, err
	}
	if err := parseFile("requirements-dev.txt", deps.ScopeDev); err != nil {
		return deps.Summary{}, err
	}

	if len(summary.Direct) > 1 {
		sort.SliceStable(summary.Direct, func(i, j int) bool {
			return summary.Direct[i].Name < summary.Direct[j].Name
		})
	}

	return summary, nil
}

func parsePythonRequirementLine(line string) (name, version string) {
	sepIndex := strings.IndexAny(line, "=<>~!")
	if sepIndex == -1 {
		name = strings.TrimSpace(line)
		return name, ""
	}

	name = strings.TrimSpace(line[:sepIndex])
	version = strings.TrimSpace(line[sepIndex:])
	return name, version
}
