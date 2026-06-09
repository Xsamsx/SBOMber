package python

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

// ParseManifests reads all supported Python manifest files from a repository root.
func ParseManifests(root string) (deps.Summary, error) {
	summary := deps.Summary{
		Direct:     make([]deps.Dependency, 0),
		Transitive: make([]deps.Dependency, 0),
	}

	reqSummary, err := ParseRequirements(root)
	if err != nil {
		return deps.Summary{}, err
	}
	summary.Direct = append(summary.Direct, reqSummary.Direct...)

	pyprojectPath := filepath.Join(root, "pyproject.toml")
	if _, statErr := os.Stat(pyprojectPath); statErr == nil {
		pySummary, parseErr := ParsePyProject(root)
		if parseErr != nil {
			return deps.Summary{}, parseErr
		}
		summary.Direct = mergePythonDirect(summary.Direct, pySummary.Direct)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return deps.Summary{}, statErr
	}

	return summary, nil
}

func mergePythonDirect(existing, extra []deps.Dependency) []deps.Dependency {
	seen := make(map[string]struct{}, len(existing))
	merged := append([]deps.Dependency{}, existing...)
	for _, dep := range existing {
		seen[dep.Name] = struct{}{}
	}
	for _, dep := range extra {
		if _, ok := seen[dep.Name]; ok {
			continue
		}
		merged = append(merged, dep)
		seen[dep.Name] = struct{}{}
	}
	return merged
}
