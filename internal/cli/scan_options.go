package cli

import "github.com/Xsamsx/SBOMber/internal/vulnerability"

type scanOptions struct {
	format                 string
	includeVulnerabilities bool
	failOnVuln             bool
	outputDir              string
	severityThreshold      string
}

func (o scanOptions) vulnCountForExit(results *vulnerability.ScanResults) int {
	if results == nil {
		return 0
	}
	if o.severityThreshold == "" {
		return results.TotalCount
	}
	return results.CountAtOrAboveSeverity(o.severityThreshold)
}

func (o scanOptions) shouldFailOnVuln(results *vulnerability.ScanResults) bool {
	return o.failOnVuln && o.includeVulnerabilities && o.vulnCountForExit(results) > 0
}
