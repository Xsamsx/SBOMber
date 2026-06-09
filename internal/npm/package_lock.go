package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

type packageLockJSON struct {
	Packages map[string]struct {
		Version string `json:"version"`
		Dev     bool   `json:"dev"`
	} `json:"packages"`
	Dependencies map[string]struct {
		Version string `json:"version"`
		Dev     bool   `json:"dev"`
	} `json:"dependencies"`
}

// ParsePackageLockJSON reads package-lock.json and returns resolved packages as transitive deps.
func ParsePackageLockJSON(root string) (deps.Summary, error) {
	path := filepath.Join(root, "package-lock.json")
	content, err := os.ReadFile(path)
	if err != nil {
		return deps.Summary{}, err
	}

	return ParsePackageLockContent(content, path)
}

// ParsePackageLockContent parses a package-lock.json payload from memory.
func ParsePackageLockContent(content []byte, sourceLocation string) (deps.Summary, error) {
	var lock packageLockJSON
	if err := json.Unmarshal(content, &lock); err != nil {
		return deps.Summary{}, fmt.Errorf("parse package-lock.json: %w", err)
	}

	sourceFile := filepath.Base(sourceLocation)
	if sourceFile == "" {
		sourceFile = "package-lock.json"
	}

	summary := deps.Summary{
		Transitive: make([]deps.Dependency, 0),
	}

	if len(lock.Packages) > 0 {
		for path, pkg := range lock.Packages {
			if path == "" {
				continue
			}
			name := strings.TrimPrefix(path, "node_modules/")
			if strings.Contains(name, "node_modules/") {
				continue
			}

			scope := deps.ScopeRuntime
			if pkg.Dev {
				scope = deps.ScopeDev
			}

			summary.Transitive = append(summary.Transitive, deps.Dependency{
				Name:           name,
				Version:        pkg.Version,
				Scope:          scope,
				Ecosystem:      "npm",
				SourceFile:     sourceFile,
				SourceLocation: sourceLocation,
			})
		}
	} else if len(lock.Dependencies) > 0 {
		for name, dep := range lock.Dependencies {
			scope := deps.ScopeRuntime
			if dep.Dev {
				scope = deps.ScopeDev
			}
			summary.Transitive = append(summary.Transitive, deps.Dependency{
				Name:           name,
				Version:        dep.Version,
				Scope:          scope,
				Ecosystem:      "npm",
				SourceFile:     sourceFile,
				SourceLocation: sourceLocation,
			})
		}
	}

	return summary, nil
}
