package supplychain

import (
	"context"
	"strings"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

// Analyze runs registry and malware checks for dependencies in summary.
func Analyze(ctx context.Context, summary deps.Summary) []RiskFinding {
	seen := make(map[string]struct{})
	findings := make([]RiskFinding, 0)

	for _, dep := range summary.AllDependencies() {
		key := dep.Ecosystem + "|" + dep.Name
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		if finding, ok := checkDependency(ctx, dep); ok {
			findings = append(findings, finding)
		}
	}

	return findings
}

func checkDependency(ctx context.Context, dep deps.Dependency) (RiskFinding, bool) {
	if dep.Name == "" || dep.Ecosystem == "" {
		return RiskFinding{}, false
	}

	if finding, ok, err := CheckMalware(ctx, dep.Ecosystem, dep.Name); err == nil && ok {
		finding.Version = dep.Version
		finding.IsDirect = dep.IsDirect
		return finding, true
	}

	exists, err := PackageExistsOnRegistry(ctx, dep.Ecosystem, dep.Name)
	if err != nil || exists {
		if err == nil && exists && looksLikePrivatePackage(dep.Name) {
			return RiskFinding{
				Type:      RiskDependencyConfusion,
				Package:   dep.Name,
				Version:   dep.Version,
				Ecosystem: dep.Ecosystem,
				Severity:  "high",
				Message:   "Package name suggests a private dependency but resolves on a public registry",
				Source:    "registry",
				IsDirect:  dep.IsDirect,
			}, true
		}
		return RiskFinding{}, false
	}

	if !dep.IsDirect {
		return RiskFinding{}, false
	}

	return RiskFinding{
		Type:      RiskRegistryMissing,
		Package:   dep.Name,
		Version:   dep.Version,
		Ecosystem: dep.Ecosystem,
		Severity:  "medium",
		Message:   "Direct dependency was not found on the public registry",
		Source:    "registry",
		IsDirect:  true,
	}, true
}

// HasHighSeverityRisk reports whether any finding is high or critical.
func HasHighSeverityRisk(findings []RiskFinding) bool {
	for _, finding := range findings {
		switch strings.ToLower(finding.Severity) {
		case "critical", "high":
			return true
		}
	}
	return false
}
