package npm

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

type yarnEntry struct {
	Selectors    []string
	Name         string
	Version      string
	Dependencies []string // names of dependencies
}

// EnrichFromYarnLock reads a Yarn Berry lockfile and appends transitive
// dependency information to an existing npm summary.
func EnrichFromYarnLock(root string, summary deps.Summary) (deps.Summary, error) {
	path := filepath.Join(root, "yarn.lock")
	content, err := os.ReadFile(path)
	if err != nil {
		return summary, err
	}

	return EnrichSummaryFromYarnLock(content, path, summary)
}

// ParseYarnLockContent parses a Yarn lockfile from memory. When no direct
// dependencies are supplied, every resolved package is recorded as transitive.
func ParseYarnLockContent(content []byte, sourceFile string) (deps.Summary, error) {
	return EnrichSummaryFromYarnLock(content, sourceFile, deps.Summary{
		Direct:     make([]deps.Dependency, 0),
		Transitive: make([]deps.Dependency, 0),
	})
}

// EnrichSummaryFromYarnLock classifies yarn.lock entries against known direct deps.
func EnrichSummaryFromYarnLock(content []byte, sourceLocation string, summary deps.Summary) (deps.Summary, error) {
	entries, err := parseYarnLock(bytes.NewReader(content))
	if err != nil {
		return summary, err
	}

	sourceFile := filepath.Base(sourceLocation)
	if sourceFile == "" {
		sourceFile = "yarn.lock"
	}

	directSelectors := make(map[string]struct{}, len(summary.Direct)*2)
	directNames := make(map[string]int) // maps name to index in summary.Direct
	for i, dependency := range summary.Direct {
		directSelectors[dependency.Name+"@"+dependency.Version] = struct{}{}
		directSelectors[dependency.Name+"@npm:"+dependency.Version] = struct{}{}
		directNames[dependency.Name] = i
	}

	directLocked := make(map[string]struct{})
	transitive := make(map[string]deps.Dependency)

	// Classify lockfile entries as direct or transitive.
	for i := range entries {
		entry := &entries[i]
		if entry.Name == "" || entry.Version == "" {
			continue
		}

		key := entry.Name + "@" + entry.Version

		isDirect := false
		for _, selector := range entry.Selectors {
			if _, ok := directSelectors[selector]; ok {
				isDirect = true
				break
			}
		}

		if isDirect {
			directLocked[key] = struct{}{}
			// Update direct dependency with children
			if idx, ok := directNames[entry.Name]; ok {
				summary.Direct[idx].Children = entry.Dependencies
			}
			continue
		}

		if _, ok := directLocked[key]; ok {
			continue
		}

		transitive[key] = deps.Dependency{
			Name:           entry.Name,
			Version:        entry.Version,
			Scope:          deps.ScopeRuntime,
			Ecosystem:      "npm",
			Children:       entry.Dependencies,
			SourceFile:     sourceFile,
			SourceLocation: sourceLocation,
		}
	}

	keys := make([]string, 0, len(transitive))
	for key := range transitive {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	summary.Transitive = make([]deps.Dependency, 0, len(keys))
	for _, key := range keys {
		summary.Transitive = append(summary.Transitive, transitive[key])
	}

	return summary, nil
}

func parseYarnLock(r *bufio.Reader) ([]yarnEntry, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	entries := make([]yarnEntry, 0)
	var current *yarnEntry
	inDependencies := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch {
		case !strings.HasPrefix(line, " "):
			key := strings.TrimSuffix(line, ":")
			key = cleanYarnValue(key)
			if key == "__metadata" {
				current = nil
				inDependencies = false
				continue
			}

			entry := yarnEntry{
				Selectors:    splitSelectors(key),
				Dependencies: make([]string, 0),
			}
			entry.Name = selectorName(entry.Selectors)
			entries = append(entries, entry)
			current = &entries[len(entries)-1]
			inDependencies = false
		case current == nil:
			continue
		case strings.HasPrefix(line, "  version:"):
			current.Version = cleanYarnValue(strings.TrimSpace(strings.TrimPrefix(line, "  version:")))
			inDependencies = false
		case strings.HasPrefix(line, "  resolution:"):
			resolution := cleanYarnValue(strings.TrimSpace(strings.TrimPrefix(line, "  resolution:")))
			if name := nameFromDescriptor(resolution); name != "" {
				current.Name = name
			}
			inDependencies = false
		case strings.HasPrefix(line, "  dependencies:"):
			inDependencies = true
		case strings.HasPrefix(line, "  peerDependencies:"), strings.HasPrefix(line, "  optionalDependencies:"):
			inDependencies = false
		case inDependencies && strings.HasPrefix(line, "    "):
			// Parse dependency entry like "    lodash: ^4.17.21"
			depLine := strings.TrimSpace(line)
			if idx := strings.Index(depLine, ":"); idx > 0 {
				depName := cleanYarnValue(depLine[:idx])
				if depName != "" {
					current.Dependencies = append(current.Dependencies, depName)
				}
			}
		case strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    "):
			// Any other top-level property ends the dependencies section
			inDependencies = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func splitSelectors(value string) []string {
	parts := strings.Split(value, ",")
	selectors := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(cleanYarnValue(part))
		if part == "" {
			continue
		}

		selectors = append(selectors, part)
	}

	return selectors
}

func selectorName(selectors []string) string {
	for _, selector := range selectors {
		if name := nameFromDescriptor(selector); name != "" {
			return name
		}
	}

	return ""
}

func nameFromDescriptor(descriptor string) string {
	if idx := strings.LastIndex(descriptor, "@npm:"); idx > 0 {
		return descriptor[:idx]
	}
	if idx := strings.LastIndex(descriptor, "@"); idx > 0 {
		return descriptor[:idx]
	}

	return ""
}

func cleanYarnValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	unquoted, err := strconv.Unquote(value)
	if err == nil {
		return unquoted
	}

	return strings.Trim(value, `"`)
}
