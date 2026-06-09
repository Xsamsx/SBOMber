package remote

import (
	"bufio"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
	"github.com/Xsamsx/SBOMber/internal/npm"
)

func parseManifestContent(path string, content []byte) (deps.Summary, error) {
	filename := getFilename(path)

	switch filename {
	case "package.json":
		return parsePackageJSON(content)
	case "package-lock.json":
		return npm.ParsePackageLockContent(content, "package-lock.json")
	case "yarn.lock":
		return parseYarnLock(content)
	case "requirements.txt":
		return parseRequirementsTxt(content)
	case "go.mod":
		return parseGoMod(content)
	case "pom.xml":
		return parsePomXML(content)
	case "Gemfile.lock":
		return parseGemfileLock(content)
	default:
		return deps.Summary{}, nil
	}
}

type packageJSON struct {
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

func parsePackageJSON(content []byte) (deps.Summary, error) {
	var manifest packageJSON
	if err := json.Unmarshal(content, &manifest); err != nil {
		return deps.Summary{}, err
	}

	summary := deps.Summary{
		Direct: make([]deps.Dependency, 0),
	}

	addDeps := func(scope deps.Scope, values map[string]string) {
		names := sortedKeys(values)
		for _, name := range names {
			version := cleanVersion(values[name])
			summary.Direct = append(summary.Direct, deps.Dependency{
				Name:      name,
				Version:   version,
				Scope:     scope,
				Ecosystem: "npm",
			})
		}
	}

	addDeps(deps.ScopeRuntime, manifest.Dependencies)
	addDeps(deps.ScopeDev, manifest.DevDependencies)
	addDeps(deps.ScopePeer, manifest.PeerDependencies)
	addDeps(deps.ScopeOptional, manifest.OptionalDependencies)

	return summary, nil
}

func parseYarnLock(content []byte) (deps.Summary, error) {
	return npm.ParseYarnLockContent(content, "yarn.lock")
}

func parseRequirementsTxt(content []byte) (deps.Summary, error) {
	summary := deps.Summary{
		Direct: make([]deps.Dependency, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "-") {
			continue
		}

		name, version := parsePythonRequirement(line)
		if name == "" {
			continue
		}

		summary.Direct = append(summary.Direct, deps.Dependency{
			Name:      name,
			Version:   version,
			Scope:     deps.ScopeRuntime,
			Ecosystem: "pypi",
		})
	}

	return summary, nil
}

var pyReqPattern = regexp.MustCompile(`^([a-zA-Z0-9._-]+)\s*(?:\[.*?\])?\s*([<>=!~]+.+)?`)

func parsePythonRequirement(line string) (name, version string) {
	line = strings.Split(line, ";")[0]
	line = strings.Split(line, "#")[0]
	line = strings.TrimSpace(line)

	matches := pyReqPattern.FindStringSubmatch(line)
	if len(matches) >= 2 {
		name = strings.ToLower(matches[1])
		if len(matches) >= 3 {
			version = strings.TrimSpace(matches[2])
		}
	}
	return
}

var goModRequire = regexp.MustCompile(`^\s*([^\s]+)\s+([^\s]+)`)

func parseGoMod(content []byte) (deps.Summary, error) {
	summary := deps.Summary{
		Direct:     make([]deps.Dependency, 0),
		Transitive: make([]deps.Dependency, 0),
	}

	lines := strings.Split(string(content), "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if strings.HasPrefix(line, "require ") {
			line = strings.TrimPrefix(line, "require ")
		} else if !inRequire {
			continue
		}

		isIndirect := strings.Contains(line, "// indirect")
		if idx := strings.Index(line, "//"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		matches := goModRequire.FindStringSubmatch(line)
		if len(matches) >= 3 {
			dep := deps.Dependency{
				Name:      matches[1],
				Version:   matches[2],
				Scope:     deps.ScopeRuntime,
				Ecosystem: "golang",
			}
			if isIndirect {
				summary.Transitive = append(summary.Transitive, dep)
			} else {
				dep.IsDirect = true
				summary.Direct = append(summary.Direct, dep)
			}
		}
	}

	return summary, nil
}

var pomDepPattern = regexp.MustCompile(`<dependency>\s*<groupId>([^<]+)</groupId>\s*<artifactId>([^<]+)</artifactId>\s*(?:<version>([^<]*)</version>)?`)

func parsePomXML(content []byte) (deps.Summary, error) {
	summary := deps.Summary{
		Direct: make([]deps.Dependency, 0),
	}

	matches := pomDepPattern.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		if len(match) >= 3 {
			name := match[1] + ":" + match[2]
			version := ""
			if len(match) >= 4 {
				version = match[3]
			}
			summary.Direct = append(summary.Direct, deps.Dependency{
				Name:      name,
				Version:   version,
				Scope:     deps.ScopeRuntime,
				Ecosystem: "maven",
			})
		}
	}

	return summary, nil
}

func parseGemfileLock(content []byte) (deps.Summary, error) {
	summary := deps.Summary{
		Direct: make([]deps.Dependency, 0),
	}

	lines := strings.Split(string(content), "\n")
	inSpecs := false
	gemPattern := regexp.MustCompile(`^\s{4}([a-zA-Z0-9_-]+)\s+\(([^)]+)\)`)

	for _, line := range lines {
		if strings.TrimSpace(line) == "specs:" {
			inSpecs = true
			continue
		}
		if inSpecs && len(line) > 0 && line[0] != ' ' {
			inSpecs = false
			continue
		}

		if inSpecs {
			matches := gemPattern.FindStringSubmatch(line)
			if len(matches) >= 3 {
				summary.Direct = append(summary.Direct, deps.Dependency{
					Name:      matches[1],
					Version:   matches[2],
					Scope:     deps.ScopeRuntime,
					Ecosystem: "rubygems",
				})
			}
		}
	}

	return summary, nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func cleanVersion(v string) string {
	v = strings.TrimPrefix(v, "^")
	v = strings.TrimPrefix(v, "~")
	v = strings.TrimPrefix(v, ">=")
	v = strings.TrimPrefix(v, "<=")
	v = strings.TrimPrefix(v, ">")
	v = strings.TrimPrefix(v, "<")
	v = strings.TrimPrefix(v, "=")
	return strings.TrimSpace(v)
}
