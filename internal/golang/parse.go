package golang

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

// ParseGoMod reads a go.mod file and returns a normalized dependency summary.
// Modules marked with "// indirect" are recorded as transitive dependencies.
// All other entries in the require block are recorded as direct dependencies.
func ParseGoMod(root string) (deps.Summary, error) {
	path := filepath.Join(root, "go.mod")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return deps.Summary{
				Direct:     make([]deps.Dependency, 0),
				Transitive: make([]deps.Dependency, 0),
			}, nil
		}
		return deps.Summary{}, fmt.Errorf("read go.mod: %w", err)
	}
	defer func() { _ = f.Close() }()

	summary := deps.Summary{
		Direct:     make([]deps.Dependency, 0),
		Transitive: make([]deps.Dependency, 0),
	}

	inRequireBlock := false
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Opening of a require block: "require ("
		if trimmed == "require (" {
			inRequireBlock = true
			continue
		}

		// Closing of a require block: ")"
		if trimmed == ")" && inRequireBlock {
			inRequireBlock = false
			continue
		}

		// Single-line require outside a block: "require module/path v1.2.3"
		if strings.HasPrefix(trimmed, "require ") && !inRequireBlock {
			remainder := strings.TrimPrefix(trimmed, "require ")
			isIndirect := strings.Contains(remainder, "// indirect")
			if idx := strings.Index(remainder, "//"); idx != -1 {
				remainder = strings.TrimSpace(remainder[:idx])
			}
			parts := strings.Fields(remainder)
			if len(parts) >= 2 {
				dep := deps.Dependency{
					Name:    parts[0],
					Version: parts[1],
					Scope:   deps.ScopeRuntime,
				}
				if isIndirect {
					summary.Transitive = append(summary.Transitive, dep)
				} else {
					summary.Direct = append(summary.Direct, dep)
				}
			}
			continue
		}

		if !inRequireBlock {
			continue
		}

		// Skip blank lines and pure comments inside require block
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Inside a require block: "    module/path v1.2.3 // indirect"
		isIndirect := strings.Contains(line, "// indirect")

		// Strip the inline comment before splitting into parts
		clean := trimmed
		if idx := strings.Index(clean, "//"); idx != -1 {
			clean = strings.TrimSpace(clean[:idx])
		}

		parts := strings.Fields(clean)
		if len(parts) < 2 {
			continue
		}

		dep := deps.Dependency{
			Name:    parts[0],
			Version: parts[1],
			Scope:   deps.ScopeRuntime,
		}

		if isIndirect {
			summary.Transitive = append(summary.Transitive, dep)
		} else {
			summary.Direct = append(summary.Direct, dep)
		}
	}

	if err := scanner.Err(); err != nil {
		return deps.Summary{}, fmt.Errorf("scan go.mod: %w", err)
	}

	sort.SliceStable(summary.Direct, func(i, j int) bool {
		return summary.Direct[i].Name < summary.Direct[j].Name
	})
	sort.SliceStable(summary.Transitive, func(i, j int) bool {
		return summary.Transitive[i].Name < summary.Transitive[j].Name
	})

	return summary, nil
}
