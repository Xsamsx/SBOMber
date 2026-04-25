package maven

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

type pomXML struct {
	Dependencies struct {
		Dependency []pomDependency `xml:"dependency"`
	} `xml:"dependencies"`
}

type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
}

func ParsePOM(root string) (deps.Summary, error) {
	path := filepath.Join(root, "pom.xml")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return deps.Summary{Direct: make([]deps.Dependency, 0)}, nil
		}
		return deps.Summary{}, fmt.Errorf("read pom.xml: %w", err)
	}

	var pom pomXML
	if err := xml.Unmarshal(content, &pom); err != nil {
		return deps.Summary{}, fmt.Errorf("parse pom.xml: %w", err)
	}

	summary := deps.Summary{
		Direct: make([]deps.Dependency, 0, len(pom.Dependencies.Dependency)),
	}

	for _, dep := range pom.Dependencies.Dependency {
		if dep.GroupID == "" || dep.ArtifactID == "" {
			continue
		}

		scope := deps.ScopeRuntime
		if dep.Scope == "test" {
			scope = deps.ScopeDev
		}

		summary.Direct = append(summary.Direct, deps.Dependency{
			Name:    dep.GroupID + ":" + dep.ArtifactID,
			Version: dep.Version,
			Scope:   scope,
		})
	}

	sort.SliceStable(summary.Direct, func(i, j int) bool {
		return summary.Direct[i].Name < summary.Direct[j].Name
	})

	return summary, nil
}
