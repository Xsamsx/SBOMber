package npm

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

// ParseProject reads npm manifest and lockfiles from a repository root.
// Missing package.json is tolerated when yarn.lock or package-lock.json is present.
func ParseProject(root string) (deps.Summary, error) {
	summary := deps.Summary{
		Direct:     make([]deps.Dependency, 0),
		Transitive: make([]deps.Dependency, 0),
	}

	pkgSummary, err := ParsePackageJSONIfPresent(root)
	if err != nil {
		return deps.Summary{}, err
	}
	summary.Direct = append(summary.Direct, pkgSummary.Direct...)

	yarnPath := filepath.Join(root, "yarn.lock")
	if _, statErr := os.Stat(yarnPath); statErr == nil {
		if len(summary.Direct) > 0 {
			enriched, enrichErr := EnrichFromYarnLock(root, summary)
			if enrichErr != nil {
				return deps.Summary{}, enrichErr
			}
			summary = enriched
		} else {
			content, readErr := os.ReadFile(yarnPath)
			if readErr != nil {
				return deps.Summary{}, readErr
			}
			lockSummary, parseErr := ParseYarnLockContent(content, yarnPath)
			if parseErr != nil {
				return deps.Summary{}, parseErr
			}
			summary.Transitive = append(summary.Transitive, lockSummary.Transitive...)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return deps.Summary{}, statErr
	}

	if summary.TransitiveCount() == 0 {
		lockSummary, lockErr := ParsePackageLockJSON(root)
		if lockErr == nil {
			summary.Transitive = append(summary.Transitive, lockSummary.Transitive...)
		} else if !errors.Is(lockErr, os.ErrNotExist) {
			return deps.Summary{}, lockErr
		}
	}

	if summary.Count() == 0 && summary.TransitiveCount() == 0 {
		return summary, nil
	}

	return summary, nil
}

// ParsePackageJSONIfPresent reads package.json when it exists.
func ParsePackageJSONIfPresent(root string) (deps.Summary, error) {
	path := filepath.Join(root, "package.json")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return deps.Summary{
				Direct:     make([]deps.Dependency, 0),
				Transitive: make([]deps.Dependency, 0),
			}, nil
		}
		return deps.Summary{}, err
	}
	return ParsePackageJSON(root)
}
