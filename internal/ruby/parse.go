package ruby

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

// ParseGemfileLock reads a Gemfile.lock file and returns a normalized dependency summary.
// All gems in the GEM specs: section are returned as direct dependencies.
// Distinguishing direct from transitive requires cross-referencing Gemfile and
// is deferred to Semester 2.
func ParseGemfileLock(root string) (deps.Summary, error) {
	path := filepath.Join(root, "Gemfile.lock")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return deps.Summary{Direct: make([]deps.Dependency, 0)}, nil
		}
		return deps.Summary{}, fmt.Errorf("read Gemfile.lock: %w", err)
	}
	defer func() { _ = f.Close() }()

	summary := deps.Summary{
		Direct: make([]deps.Dependency, 0),
	}

	inGemSection := false
	inSpecsSection := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect start of the GEM section
		if trimmed == "GEM" {
			inGemSection = true
			inSpecsSection = false
			continue
		}

		// Detect start of the specs: block inside GEM
		if trimmed == "specs:" && inGemSection {
			inSpecsSection = true
			continue
		}

		// Any non-indented line that is not "specs:" ends the GEM block
		if inGemSection && len(line) > 0 && line[0] != ' ' && trimmed != "specs:" {
			inGemSection = false
			inSpecsSection = false
			continue
		}

		if !inSpecsSection {
			continue
		}

		// Gem entries are indented exactly 4 spaces: "    gemname (version)"
		// Sub-dependencies are indented 6+ spaces — skip those
		if !strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "      ") {
			continue
		}

		// Parse "    gemname (version)"
		entry := strings.TrimSpace(line)
		openParen := strings.Index(entry, " (")
		if openParen == -1 {
			continue
		}

		name := strings.TrimSpace(entry[:openParen])
		rest := entry[openParen+2:]
		closeParen := strings.Index(rest, ")")
		if closeParen == -1 {
			continue
		}
		version := strings.TrimSpace(rest[:closeParen])

		if name == "" {
			continue
		}

		summary.Direct = append(summary.Direct, deps.Dependency{
			Name:    name,
			Version: version,
			Scope:   deps.ScopeRuntime,
		})
	}

	if err := scanner.Err(); err != nil {
		return deps.Summary{}, fmt.Errorf("scan Gemfile.lock: %w", err)
	}

	sort.SliceStable(summary.Direct, func(i, j int) bool {
		return summary.Direct[i].Name < summary.Direct[j].Name
	})

	return summary, nil
}
