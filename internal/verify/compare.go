package verify

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Component represents a dependency in an SBOM
type Component struct {
	Name    string
	Version string
	Purl    string
	Type    string
}

// ComparisonResult holds the verification metrics
type ComparisonResult struct {
	// Counts
	GroundTruthCount int
	GeneratedCount   int
	MatchedCount     int
	MissingCount     int
	ExtraCount       int
	VersionMismatch  int

	// Metrics (0-100)
	Precision       float64
	Recall          float64
	F1Score         float64
	VersionAccuracy float64

	// Details
	Matched  []ComponentMatch
	Missing  []Component // In ground truth but not in generated
	Extra    []Component // In generated but not in ground truth
	Mismatch []VersionMismatch
}

// ComponentMatch represents a matched component
type ComponentMatch struct {
	Name    string
	Version string
}

// VersionMismatch represents a version discrepancy
type VersionMismatch struct {
	Name            string
	ExpectedVersion string
	ActualVersion   string
}

// CycloneDX XML structures
type cycloneDXXML struct {
	XMLName    xml.Name         `xml:"bom"`
	Components cycloneDXCompXML `xml:"components"`
}

type cycloneDXCompXML struct {
	Components []cycloneDXComponentXML `xml:"component"`
}

type cycloneDXComponentXML struct {
	Type    string `xml:"type,attr"`
	Name    string `xml:"name"`
	Version string `xml:"version"`
	Purl    string `xml:"purl"`
}

// CycloneDX JSON structures
type cycloneDXJSON struct {
	Components []cycloneDXComponentJSON `json:"components"`
}

type cycloneDXComponentJSON struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Purl    string `json:"purl"`
}

// SPDX JSON structures
type spdxJSON struct {
	Packages []spdxPackage `json:"packages"`
}

type spdxPackage struct {
	Name         string            `json:"name"`
	Version      string            `json:"versionInfo"`
	ExternalRefs []spdxExternalRef `json:"externalRefs"`
}

type spdxExternalRef struct {
	ReferenceType    string `json:"referenceType"`
	ReferenceLocator string `json:"referenceLocator"`
}

// ParseSBOM reads an SBOM file and extracts components
func ParseSBOM(path string) ([]Component, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	// Try to detect format
	content := string(data)

	// CycloneDX XML
	if ext == ".xml" || strings.Contains(content, "<bom") {
		return parseCycloneDXXML(data)
	}

	// JSON formats
	if ext == ".json" {
		// Check if CycloneDX or SPDX
		if strings.Contains(content, `"bomFormat"`) || strings.Contains(content, `"components"`) {
			return parseCycloneDXJSON(data)
		}
		if strings.Contains(content, `"spdxVersion"`) || strings.Contains(content, `"packages"`) {
			return parseSPDXJSON(data)
		}
	}

	return nil, fmt.Errorf("unknown SBOM format: %s", path)
}

func parseCycloneDXXML(data []byte) ([]Component, error) {
	var bom cycloneDXXML
	if err := xml.Unmarshal(data, &bom); err != nil {
		return nil, fmt.Errorf("parse CycloneDX XML: %w", err)
	}

	components := make([]Component, 0, len(bom.Components.Components))
	for _, c := range bom.Components.Components {
		components = append(components, Component{
			Name:    c.Name,
			Version: c.Version,
			Purl:    c.Purl,
			Type:    c.Type,
		})
	}

	return components, nil
}

func parseCycloneDXJSON(data []byte) ([]Component, error) {
	var bom cycloneDXJSON
	if err := json.Unmarshal(data, &bom); err != nil {
		return nil, fmt.Errorf("parse CycloneDX JSON: %w", err)
	}

	components := make([]Component, 0, len(bom.Components))
	for _, c := range bom.Components {
		components = append(components, Component{
			Name:    c.Name,
			Version: c.Version,
			Purl:    c.Purl,
			Type:    c.Type,
		})
	}

	return components, nil
}

func parseSPDXJSON(data []byte) ([]Component, error) {
	var bom spdxJSON
	if err := json.Unmarshal(data, &bom); err != nil {
		return nil, fmt.Errorf("parse SPDX JSON: %w", err)
	}

	components := make([]Component, 0, len(bom.Packages))
	for _, p := range bom.Packages {
		purl := ""
		for _, ref := range p.ExternalRefs {
			if ref.ReferenceType == "purl" {
				purl = ref.ReferenceLocator
				break
			}
		}

		components = append(components, Component{
			Name:    p.Name,
			Version: p.Version,
			Purl:    purl,
			Type:    "library",
		})
	}

	return components, nil
}

// dedupeComponents removes duplicate name+version rows from an SBOM component list.
func dedupeComponents(components []Component) []Component {
	if len(components) == 0 {
		return components
	}

	seen := make(map[string]struct{}, len(components))
	deduped := make([]Component, 0, len(components))
	for _, component := range components {
		key := componentKey(component)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, component)
	}
	return deduped
}

func componentKey(component Component) string {
	return normalizeKey(component.Name) + "@" + normalizeVersion(component.Version)
}

func findComponentByName(components map[string]Component, skip map[string]struct{}, name string) (Component, bool) {
	target := normalizeKey(name)
	for key, component := range components {
		if _, used := skip[key]; used {
			continue
		}
		if normalizeKey(component.Name) == target {
			return component, true
		}
	}
	return Component{}, false
}

// Compare compares generated SBOM against ground truth
func Compare(groundTruth, generated []Component) *ComparisonResult {
	groundTruth = dedupeComponents(groundTruth)
	generated = dedupeComponents(generated)

	result := &ComparisonResult{
		GroundTruthCount: len(groundTruth),
		GeneratedCount:   len(generated),
		Matched:          make([]ComponentMatch, 0),
		Missing:          make([]Component, 0),
		Extra:            make([]Component, 0),
		Mismatch:         make([]VersionMismatch, 0),
	}

	truthByKey := make(map[string]Component, len(groundTruth))
	for _, component := range groundTruth {
		truthByKey[componentKey(component)] = component
	}

	genByKey := make(map[string]Component, len(generated))
	for _, component := range generated {
		genByKey[componentKey(component)] = component
	}

	matchedTruthKeys := make(map[string]struct{})
	matchedGenKeys := make(map[string]struct{})

	for key, truth := range truthByKey {
		if _, found := genByKey[key]; !found {
			continue
		}
		result.Matched = append(result.Matched, ComponentMatch{
			Name:    truth.Name,
			Version: truth.Version,
		})
		result.MatchedCount++
		matchedTruthKeys[key] = struct{}{}
		matchedGenKeys[key] = struct{}{}
	}

	for key, truth := range truthByKey {
		if _, matched := matchedTruthKeys[key]; matched {
			continue
		}
		gen, found := findComponentByName(genByKey, matchedGenKeys, truth.Name)
		if !found {
			continue
		}
		result.Mismatch = append(result.Mismatch, VersionMismatch{
			Name:            truth.Name,
			ExpectedVersion: truth.Version,
			ActualVersion:   gen.Version,
		})
		result.VersionMismatch++
		result.MatchedCount++
		matchedTruthKeys[key] = struct{}{}
		matchedGenKeys[componentKey(gen)] = struct{}{}
	}

	for key, truth := range truthByKey {
		if _, matched := matchedTruthKeys[key]; matched {
			continue
		}
		result.Missing = append(result.Missing, truth)
		result.MissingCount++
	}

	for key, gen := range genByKey {
		if _, matched := matchedGenKeys[key]; matched {
			continue
		}
		result.Extra = append(result.Extra, gen)
		result.ExtraCount++
	}

	// Calculate metrics
	if result.GeneratedCount > 0 {
		// Precision: exact name+version matches / total reported (extras count in denominator)
		exactMatches := result.MatchedCount - result.VersionMismatch
		result.Precision = float64(exactMatches) / float64(result.GeneratedCount) * 100
	}

	if result.GroundTruthCount > 0 {
		// Recall: found / total in ground truth
		result.Recall = float64(result.MatchedCount) / float64(result.GroundTruthCount) * 100
	}

	// F1 Score
	if result.Precision+result.Recall > 0 {
		result.F1Score = 2 * (result.Precision * result.Recall) / (result.Precision + result.Recall)
	}

	// Version accuracy (among matched)
	if result.MatchedCount > 0 {
		exactMatches := result.MatchedCount - result.VersionMismatch
		result.VersionAccuracy = float64(exactMatches) / float64(result.MatchedCount) * 100
	}

	// Sort results for consistent output
	sort.Slice(result.Missing, func(i, j int) bool {
		return result.Missing[i].Name < result.Missing[j].Name
	})
	sort.Slice(result.Extra, func(i, j int) bool {
		return result.Extra[i].Name < result.Extra[j].Name
	})

	return result
}

// normalizeKey creates a comparable key from a package name
func normalizeKey(name string) string {
	name = strings.ToLower(name)
	// Handle scoped npm packages
	name = strings.ReplaceAll(name, "%40", "@")
	return name
}

// normalizeVersion removes common prefixes/suffixes for comparison
func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "=")
	v = strings.TrimSpace(v)
	return v
}

// PrintReport outputs a formatted verification report
func (r *ComparisonResult) PrintReport() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║                    SBOM VERIFICATION REPORT                  ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════╝\n\n")

	// Summary
	sb.WriteString("┌─────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│ SUMMARY                                                     │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
	sb.WriteString(fmt.Sprintf("│ Ground Truth Components:  %-34d │\n", r.GroundTruthCount))
	sb.WriteString(fmt.Sprintf("│ Generated Components:     %-34d │\n", r.GeneratedCount))
	sb.WriteString(fmt.Sprintf("│ Matched:                  %-34d │\n", r.MatchedCount))
	sb.WriteString(fmt.Sprintf("│ Missing:                  %-34d │\n", r.MissingCount))
	sb.WriteString(fmt.Sprintf("│ Extra:                    %-34d │\n", r.ExtraCount))
	sb.WriteString(fmt.Sprintf("│ Version Mismatches:       %-34d │\n", r.VersionMismatch))
	sb.WriteString("└─────────────────────────────────────────────────────────────┘\n\n")

	// Metrics
	sb.WriteString("┌─────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│ ACCURACY METRICS                                            │\n")
	sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
	sb.WriteString(fmt.Sprintf("│ Precision:        %5.1f%%  (exact matches / total reported)  │\n", r.Precision))
	sb.WriteString(fmt.Sprintf("│ Recall:           %5.1f%%  (found / total in ground truth)  │\n", r.Recall))
	sb.WriteString(fmt.Sprintf("│ F1 Score:         %5.1f%%  (harmonic mean)                  │\n", r.F1Score))
	sb.WriteString(fmt.Sprintf("│ Version Accuracy: %5.1f%%  (exact version matches)          │\n", r.VersionAccuracy))
	sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")

	// Missing dependencies
	if len(r.Missing) > 0 {
		sb.WriteString("\n┌─────────────────────────────────────────────────────────────┐\n")
		sb.WriteString("│ MISSING DEPENDENCIES (in ground truth, not found)           │\n")
		sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
		for i, m := range r.Missing {
			if i >= 20 {
				sb.WriteString(fmt.Sprintf("│ ... and %d more                                             │\n", len(r.Missing)-20))
				break
			}
			line := fmt.Sprintf("│ • %-40s %-16s │\n", truncate(m.Name, 40), truncate(m.Version, 16))
			sb.WriteString(line)
		}
		sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")
	}

	// Extra dependencies
	if len(r.Extra) > 0 {
		sb.WriteString("\n┌─────────────────────────────────────────────────────────────┐\n")
		sb.WriteString("│ EXTRA DEPENDENCIES (found, not in ground truth)             │\n")
		sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
		for i, e := range r.Extra {
			if i >= 20 {
				sb.WriteString(fmt.Sprintf("│ ... and %d more                                             │\n", len(r.Extra)-20))
				break
			}
			line := fmt.Sprintf("│ • %-40s %-16s │\n", truncate(e.Name, 40), truncate(e.Version, 16))
			sb.WriteString(line)
		}
		sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")
	}

	// Version mismatches
	if len(r.Mismatch) > 0 {
		sb.WriteString("\n┌─────────────────────────────────────────────────────────────┐\n")
		sb.WriteString("│ VERSION MISMATCHES                                          │\n")
		sb.WriteString("├─────────────────────────────────────────────────────────────┤\n")
		for i, m := range r.Mismatch {
			if i >= 20 {
				sb.WriteString(fmt.Sprintf("│ ... and %d more                                             │\n", len(r.Mismatch)-20))
				break
			}
			line := fmt.Sprintf("│ • %-30s expected: %-8s got: %-8s │\n",
				truncate(m.Name, 30),
				truncate(m.ExpectedVersion, 8),
				truncate(m.ActualVersion, 8))
			sb.WriteString(line)
		}
		sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")
	}

	// Grade
	sb.WriteString("\n")
	grade := calculateGrade(r.F1Score)
	sb.WriteString(fmt.Sprintf("Overall Grade: %s\n", grade))

	return sb.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func calculateGrade(f1 float64) string {
	switch {
	case f1 >= 95:
		return "A+ (Excellent)"
	case f1 >= 90:
		return "A  (Great)"
	case f1 >= 85:
		return "B+ (Good)"
	case f1 >= 80:
		return "B  (Acceptable)"
	case f1 >= 70:
		return "C  (Needs Improvement)"
	case f1 >= 60:
		return "D  (Poor)"
	default:
		return "F  (Failing)"
	}
}

// VerifyFiles compares two SBOM files and returns the result
func VerifyFiles(groundTruthPath, generatedPath string) (*ComparisonResult, error) {
	truth, err := ParseSBOM(groundTruthPath)
	if err != nil {
		return nil, fmt.Errorf("parse ground truth: %w", err)
	}

	generated, err := ParseSBOM(generatedPath)
	if err != nil {
		return nil, fmt.Errorf("parse generated: %w", err)
	}

	return Compare(truth, generated), nil
}
