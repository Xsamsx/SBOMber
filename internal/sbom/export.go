package sbom

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

const (
	cycloneDXFilename = "sbom-cyclonedx.xml"
	spdxFilename      = "sbom.spdx"
)

type cycloneDXBom struct {
	XMLName    xml.Name          `xml:"bom"`
	Xmlns      string            `xml:"xmlns,attr"`
	Version    int               `xml:"version,attr"`
	Metadata   cycloneDXMetadata `xml:"metadata"`
	Components *componentList    `xml:"components,omitempty"`
}

type cycloneDXMetadata struct {
	Timestamp string             `xml:"timestamp"`
	Component cycloneDXComponent `xml:"component"`
}

type componentList struct {
	Components []cycloneDXComponent `xml:"component"`
}

type cycloneDXComponent struct {
	Type    string `xml:"type,attr"`
	Name    string `xml:"name"`
	Version string `xml:"version,omitempty"`
	Scope   string `xml:"scope,omitempty"`
	Purl    string `xml:"purl,omitempty"`
}

// SaveSBOM writes one or more SBOM files for the repository.
func SaveSBOM(repoDir, repoName string, summary deps.Summary, format string) ([]string, error) {
	saved := make([]string, 0, 2)
	if format == "" {
		return saved, nil
	}

	if format == "cyclonedx" || format == "both" {
		path, err := saveCycloneDX(repoDir, repoName, summary)
		if err != nil {
			return nil, err
		}
		saved = append(saved, path)
	}

	if format == "spdx" || format == "both" {
		path, err := saveSPDX(repoDir, repoName, summary)
		if err != nil {
			return nil, err
		}
		saved = append(saved, path)
	}

	return saved, nil
}

func saveCycloneDX(repoDir, repoName string, summary deps.Summary) (string, error) {
	bom := cycloneDXBom{
		Xmlns:   "http://cyclonedx.org/schema/bom/1.5",
		Version: 1,
		Metadata: cycloneDXMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Component: cycloneDXComponent{
				Type: "application",
				Name: repoName,
			},
		},
	}

	components := make([]cycloneDXComponent, 0, len(summary.Direct)+len(summary.Transitive))
	for _, dependency := range summary.Direct {
		components = append(components, cycloneDXComponent{
			Type:    "library",
			Name:    dependency.Name,
			Version: dependency.Version,
			Scope:   string(dependency.Scope),
			Purl:    buildPurl(dependency.Name, dependency.Version, "npm"),
		})
	}
	for _, dependency := range summary.Transitive {
		components = append(components, cycloneDXComponent{
			Type:    "library",
			Name:    dependency.Name,
			Version: dependency.Version,
			Scope:   "transitive",
			Purl:    buildPurl(dependency.Name, dependency.Version, "npm"),
		})
	}

	if len(components) > 0 {
		bom.Components = &componentList{Components: components}
	}

	content, err := xml.MarshalIndent(bom, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal cyclonedx sbom: %w", err)
	}

	out := xml.Header + string(content) + "\n"
	path := filepath.Join(repoDir, cycloneDXFilename)
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return "", fmt.Errorf("write cyclonedx sbom: %w", err)
	}

	return path, nil
}

func saveSPDX(repoDir, repoName string, summary deps.Summary) (string, error) {
	repoID := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(repoName)), " ", "-")
	namespace := fmt.Sprintf("http://spdx.org/spdxdocs/%s-%s", repoID, "sbom")

	lines := []string{
		"SPDXVersion: SPDX-2.3",
		"DataLicense: CC0-1.0",
		"SPDXID: SPDXRef-DOCUMENT",
		fmt.Sprintf("DocumentName: %s", repoName),
		fmt.Sprintf("DocumentNamespace: %s", namespace),
		"Creator: Tool: SBOMber",
		fmt.Sprintf("Created: %s", time.Now().UTC().Format(time.RFC3339)),
	}

	packageIndex := 1
	for _, dependency := range summary.Direct {
		packageID := fmt.Sprintf("SPDXRef-Package-%d", packageIndex)
		lines = append(lines,
			fmt.Sprintf("PackageName: %s", dependency.Name),
			fmt.Sprintf("SPDXID: %s", packageID),
			fmt.Sprintf("PackageVersion: %s", dependency.Version),
			"PackageSupplier: NOASSERTION",
			"PackageDownloadLocation: NOASSERTION",
			"FilesAnalyzed: false",
			"PackageLicenseConcluded: NOASSERTION",
			"PackageLicenseDeclared: NOASSERTION",
			"PackageCopyrightText: NONE",
			fmt.Sprintf("Relationship: SPDXRef-DOCUMENT DESCRIBES %s", packageID),
		)
		packageIndex++
	}

	for _, dependency := range summary.Transitive {
		packageID := fmt.Sprintf("SPDXRef-Package-%d", packageIndex)
		lines = append(lines,
			fmt.Sprintf("PackageName: %s", dependency.Name),
			fmt.Sprintf("SPDXID: %s", packageID),
			fmt.Sprintf("PackageVersion: %s", dependency.Version),
			"PackageSupplier: NOASSERTION",
			"PackageDownloadLocation: NOASSERTION",
			"FilesAnalyzed: false",
			"PackageLicenseConcluded: NOASSERTION",
			"PackageLicenseDeclared: NOASSERTION",
			"PackageCopyrightText: NONE",
			fmt.Sprintf("Relationship: SPDXRef-DOCUMENT DESCRIBES %s", packageID),
		)
		packageIndex++
	}

	if len(summary.Direct)+len(summary.Transitive) == 0 {
		lines = append(lines, "PackageName: NOASSERTION", "SPDXID: SPDXRef-Package-1", "PackageVersion: NOASSERTION", "PackageSupplier: NOASSERTION", "PackageDownloadLocation: NOASSERTION", "FilesAnalyzed: false", "PackageLicenseConcluded: NOASSERTION", "PackageLicenseDeclared: NOASSERTION", "PackageCopyrightText: NONE", "Relationship: SPDXRef-DOCUMENT DESCRIBES SPDXRef-Package-1")
	}

	out := strings.Join(lines, "\n") + "\n"
	path := filepath.Join(repoDir, spdxFilename)
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return "", fmt.Errorf("write spdx sbom: %w", err)
	}

	return path, nil
}

// buildPurl constructs a Package URL (purl) for a dependency.
// The purl format is: pkg:<type>/<name>@<version>
func buildPurl(name, version, purlType string) string {
	if name == "" {
		return ""
	}
	if version == "" {
		return fmt.Sprintf("pkg:%s/%s", purlType, name)
	}
	return fmt.Sprintf("pkg:%s/%s@%s", purlType, name, version)
}
