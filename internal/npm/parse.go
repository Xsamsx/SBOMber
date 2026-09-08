package npm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

type packageJSON struct {
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

type packageLockJSON struct {
	Packages map[string]struct {
		Version string `json:"version"`
		Dev     bool   `json:"dev"`
	} `json:"packages"`
}

// ParsePackageJSON reads package.json and returns a normalized summary of direct
// dependencies.
func ParsePackageJSON(root string) (deps.Summary, error) {
	path := filepath.Join(root, "package.json")
	content, err := os.ReadFile(path)
	if err != nil {
		return deps.Summary{}, err
	}

	var manifest packageJSON
	if err := json.Unmarshal(content, &manifest); err != nil {
		return deps.Summary{}, err
	}

	// Get relative path for source location
	sourceFile := "package.json"
	sourceLocation := path

	summary := deps.Summary{
		Direct: make([]deps.Dependency, 0),
	}

	appendDependencies := func(scope deps.Scope, values map[string]string) {
		if len(values) == 0 {
			return
		}

		names := make([]string, 0, len(values))
		for name := range values {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			summary.Direct = append(summary.Direct, deps.Dependency{
				Name:           name,
				Version:        values[name],
				Scope:          scope,
				Ecosystem:      "npm",
				IsDirect:       true,
				SourceFile:     sourceFile,
				SourceLocation: sourceLocation,
			})
		}
	}

	appendDependencies(deps.ScopeRuntime, manifest.Dependencies)
	appendDependencies(deps.ScopeDev, manifest.DevDependencies)
	appendDependencies(deps.ScopePeer, manifest.PeerDependencies)
	appendDependencies(deps.ScopeOptional, manifest.OptionalDependencies)

	return summary, nil
}

// EnrichFromPackageLock reads package-lock.json and reconciles direct deps from
// package.json with the locked graph. It preserves nested package versions under
// different node_modules paths as distinct transitive occurrences instead of
// collapsing them by package name.
func EnrichFromPackageLock(root string, summary deps.Summary) (deps.Summary, error) {
	path := filepath.Join(root, "package-lock.json")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return summary, nil
		}
		return summary, err
	}

	var lock packageLockJSON
	if err := json.Unmarshal(content, &lock); err != nil {
		return summary, err
	}

	if len(lock.Packages) == 0 {
		return summary, nil
	}

	directByName := make(map[string]deps.Dependency, len(summary.Direct))
	for _, dep := range summary.Direct {
		directByName[dep.Name] = dep
	}

	transitive := make([]deps.Dependency, 0, len(lock.Packages))
	seenDirect := make(map[string]struct{}, len(summary.Direct))
	for pkgPath, pkg := range lock.Packages {
		if pkgPath == "" {
			continue
		}
		name := packageLockName(pkgPath)
		if name == "" || pkg.Version == "" {
			continue
		}
		if direct, ok := directByName[name]; ok && isTopLevelLockPath(pkgPath, name) {
			if _, exists := seenDirect[name]; exists {
				continue
			}
			seenDirect[name] = struct{}{}
			for i := range summary.Direct {
				if summary.Direct[i].Name == name {
					summary.Direct[i].Version = pkg.Version
					summary.Direct[i].SourceFile = "package-lock.json"
					summary.Direct[i].SourceLocation = path
					summary.Direct[i].Scope = direct.Scope
					break
				}
			}
			continue
		}

		scope := deps.ScopeRuntime
		if pkg.Dev {
			scope = deps.ScopeDev
		}
		transitive = append(transitive, deps.Dependency{
			Name:           name,
			Version:        pkg.Version,
			Scope:          scope,
			Ecosystem:      "npm",
			SourceFile:     "package-lock.json",
			SourceLocation: path,
		})
	}

	summary.Transitive = append(summary.Transitive, transitive...)
	return summary, nil
}

func packageLockName(pkgPath string) string {
	pkgPath = strings.TrimSpace(pkgPath)
	if pkgPath == "" {
		return ""
	}
	pkgPath = strings.TrimPrefix(pkgPath, "./")
	parts := strings.Split(pkgPath, "/node_modules/")
	if len(parts) == 0 {
		return ""
	}
	name := parts[len(parts)-1]
	if name == "" {
		return ""
	}
	return name
}

func isTopLevelLockPath(pkgPath, name string) bool {
	pkgPath = strings.TrimPrefix(pkgPath, "./")
	pkgPath = strings.TrimSpace(pkgPath)
	if pkgPath == "node_modules/"+name {
		return true
	}
	if strings.HasPrefix(pkgPath, "node_modules/") {
		trimmed := strings.TrimPrefix(pkgPath, "node_modules/")
		return trimmed == name && !strings.Contains(trimmed, "/node_modules/")
	}
	return false
}
