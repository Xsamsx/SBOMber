package canonical

import (
	"fmt"
	"strings"
)

// Component is the canonical package identity for a dependency version.
type Component struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	PURL    string `json:"purl"`
	Scope   string `json:"scope,omitempty"`
}

// Occurrence describes where a component was seen within a workspace.
type Occurrence struct {
	Workspace     string `json:"workspace"`
	ManifestPath  string `json:"manifestPath"`
	Dependency    string `json:"dependency"`
	ComponentPURL string `json:"componentPurl"`
}

// Finding is the vulnerability match for a specific component.
type Finding struct {
	VulnerabilityID string `json:"vulnerabilityId"`
	ComponentPURL   string `json:"componentPurl"`
	Severity        string `json:"severity,omitempty"`
	FixedVersion    string `json:"fixedVersion,omitempty"`
	FixState        string `json:"fixState,omitempty"`
}

// UsageEvidence records the concrete application usage site that triggered the finding.
type UsageEvidence struct {
	FindingKey      string `json:"findingKey"`
	ApplicationPath string `json:"applicationPath"`
	Symbol          string `json:"symbol"`
}

// Scan is the canonical structure for a scan result.
type Scan struct {
	SchemaVersion string          `json:"schemaVersion"`
	Workspace     string          `json:"workspace,omitempty"`
	Components    []Component     `json:"components"`
	Occurrences   []Occurrence    `json:"occurrences"`
	Findings      []Finding       `json:"findings"`
	UsageEvidence []UsageEvidence `json:"usageEvidence"`
}

func NewScan(workspace string) *Scan {
	return &Scan{
		SchemaVersion: "1.0",
		Workspace:     workspace,
		Components:    make([]Component, 0),
		Occurrences:   make([]Occurrence, 0),
		Findings:      make([]Finding, 0),
		UsageEvidence: make([]UsageEvidence, 0),
	}
}

func (c Component) Key() string {
	if c.PURL != "" {
		return c.PURL
	}
	return strings.TrimSpace(c.Name) + "@" + strings.TrimSpace(c.Version)
}

func (o Occurrence) Key() string {
	return strings.Join([]string{
		o.Workspace,
		o.ManifestPath,
		o.Dependency,
		o.ComponentPURL,
	}, "|")
}

func (f Finding) Key() string {
	return strings.Join([]string{f.VulnerabilityID, f.ComponentPURL}, "|")
}

func (u UsageEvidence) Key() string {
	return strings.Join([]string{u.FindingKey, u.ApplicationPath, u.Symbol}, "|")
}

func (s *Scan) AddComponent(name, version, purl, scope string) Component {
	component := Component{Name: name, Version: version, PURL: purl, Scope: scope}
	s.Components = append(s.Components, component)
	return component
}

func (s *Scan) AddOccurrence(workspace, manifestPath, dependency, componentPURL string) Occurrence {
	occurrence := Occurrence{
		Workspace:     workspace,
		ManifestPath:  manifestPath,
		Dependency:    dependency,
		ComponentPURL: componentPURL,
	}
	s.Occurrences = append(s.Occurrences, occurrence)
	return occurrence
}

func (s *Scan) AddFinding(vulnID, componentPURL, severity, fixedVersion, fixState string) Finding {
	finding := Finding{VulnerabilityID: vulnID, ComponentPURL: componentPURL, Severity: severity, FixedVersion: fixedVersion, FixState: fixState}
	s.Findings = append(s.Findings, finding)
	return finding
}

func (s *Scan) AddUsageEvidence(findingKey, appPath, symbol string) UsageEvidence {
	evidence := UsageEvidence{FindingKey: findingKey, ApplicationPath: appPath, Symbol: symbol}
	s.UsageEvidence = append(s.UsageEvidence, evidence)
	return evidence
}

func ParsePURL(purl string) (name, version string, ok bool) {
	if purl == "" {
		return "", "", false
	}
	if !strings.HasPrefix(purl, "pkg:") {
		return "", "", false
	}
	purl = strings.TrimPrefix(purl, "pkg:")
	if idx := strings.LastIndex(purl, "@"); idx > 0 {
		name = purl[:idx]
		version = purl[idx+1:]
		if strings.Contains(name, "/") {
			name = name[strings.LastIndex(name, "/")+1:]
		}
		return name, version, true
	}
	if idx := strings.LastIndex(purl, "/"); idx >= 0 {
		name = purl[idx+1:]
		return name, "", true
	}
	return "", "", false
}

func (s *Scan) Validate() error {
	seenComponents := make(map[string]struct{}, len(s.Components))
	for _, c := range s.Components {
		key := c.Key()
		if key == "" {
			return fmt.Errorf("component missing identity")
		}
		seenComponents[key] = struct{}{}
	}
	for _, o := range s.Occurrences {
		key := o.Key()
		if key == "" {
			return fmt.Errorf("occurrence missing identity")
		}
		if _, exists := seenComponents[o.ComponentPURL]; !exists && o.ComponentPURL != "" {
			return fmt.Errorf("occurrence points to unknown component %q", o.ComponentPURL)
		}
	}
	seenFindings := make(map[string]struct{}, len(s.Findings))
	for _, f := range s.Findings {
		key := f.Key()
		if key == "" {
			return fmt.Errorf("finding missing identity")
		}
		seenFindings[key] = struct{}{}
	}
	for _, u := range s.UsageEvidence {
		if u.Key() == "" {
			return fmt.Errorf("usage evidence missing identity")
		}
		if _, exists := seenFindings[u.FindingKey]; !exists {
			return fmt.Errorf("usage evidence references unknown finding %q", u.FindingKey)
		}
	}
	return nil
}
