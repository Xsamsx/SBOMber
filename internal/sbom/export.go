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
	XMLName      xml.Name          `xml:"bom"`
	Xmlns        string            `xml:"xmlns,attr"`
	Version      int               `xml:"version,attr"`
	Metadata     cycloneDXMetadata `xml:"metadata"`
	Components   *componentList    `xml:"components,omitempty"`
	Dependencies *dependenciesList `xml:"dependencies,omitempty"`
}

type cycloneDXMetadata struct {
	Timestamp string             `xml:"timestamp"`
	Component cycloneDXComponent `xml:"component"`
}

type componentList struct {
	Components []cycloneDXComponent `xml:"component"`
}

type dependenciesList struct {
	Dependencies []cycloneDXDependency `xml:"dependency"`
}

type cycloneDXDependency struct {
	Ref          string          `xml:"ref,attr"`
	Dependencies []dependencyRef `xml:"dependency,omitempty"`
}

type dependencyRef struct {
	Ref string `xml:"ref,attr"`
}

type cycloneDXComponent struct {
	Type       string          `xml:"type,attr"`
	BomRef     string          `xml:"bom-ref,attr,omitempty"`
	Name       string          `xml:"name"`
	Version    string          `xml:"version,omitempty"`
	Scope      string          `xml:"scope,omitempty"`
	Purl       string          `xml:"purl,omitempty"`
	Properties *propertiesList `xml:"properties,omitempty"`
}

type propertiesList struct {
	Properties []cycloneDXProperty `xml:"property"`
}

type cycloneDXProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

// SaveSBOM writes one or more SBOM files for the repository.
// Returns the list of saved file paths and the output directory path.
// Output is stored in ~/.sbomber/reports/<project-name>/ unless outputDir is provided.
func SaveSBOM(repoDir, repoName string, summary deps.Summary, format string) ([]string, string, error) {
	return SaveSBOMWithOutput(repoDir, repoName, summary, format, "")
}

// SaveSBOMWithOutput writes SBOM files to outputDir when non-empty, otherwise the default location.
func SaveSBOMWithOutput(repoDir, repoName string, summary deps.Summary, format, outputDir string) ([]string, string, error) {
	saved := make([]string, 0, 2)
	if format == "" {
		return saved, "", nil
	}

	var err error
	if outputDir == "" {
		outputDir, err = GetOutputDir(repoDir)
	} else if err = os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, "", fmt.Errorf("create output directory: %w", err)
	}
	if err != nil {
		return nil, "", err
	}

	if format == "cyclonedx" || format == "both" {
		path, err := saveCycloneDX(outputDir, repoName, summary)
		if err != nil {
			return nil, "", err
		}
		saved = append(saved, path)
	}

	if format == "spdx" || format == "both" {
		path, err := saveSPDX(outputDir, repoName, summary)
		if err != nil {
			return nil, "", err
		}
		saved = append(saved, path)
	}

	return saved, outputDir, nil
}

// SaveSBOMToDir writes SBOM files directly to the specified directory.
// Unlike SaveSBOM, this does not create a subdirectory.
func SaveSBOMToDir(outputDir, repoName string, summary deps.Summary, format string) ([]string, error) {
	saved := make([]string, 0, 2)
	if format == "" {
		return saved, nil
	}

	if format == "cyclonedx" || format == "both" {
		path, err := saveCycloneDX(outputDir, repoName, summary)
		if err != nil {
			return nil, err
		}
		saved = append(saved, path)
	}

	if format == "spdx" || format == "both" {
		path, err := saveSPDX(outputDir, repoName, summary)
		if err != nil {
			return nil, err
		}
		saved = append(saved, path)
	}

	return saved, nil
}

func saveCycloneDX(repoDir, repoName string, summary deps.Summary) (string, error) {
	// Build dependency graph if not already built
	if summary.DependencyGraph == nil {
		summary.BuildGraph(repoName)
	}

	rootPurl := fmt.Sprintf("pkg:generic/%s", strings.ToLower(repoName))

	bom := cycloneDXBom{
		Xmlns:   "http://cyclonedx.org/schema/bom/1.5",
		Version: 1,
		Metadata: cycloneDXMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Component: cycloneDXComponent{
				Type:   "application",
				BomRef: rootPurl,
				Name:   repoName,
			},
		},
	}

	components := make([]cycloneDXComponent, 0, len(summary.Direct)+len(summary.Transitive))
	bomRefMap := make(map[string]string) // maps package name to bom-ref

	// Process direct dependencies
	for _, dependency := range summary.Direct {
		purl := dependency.Purl()
		bomRefMap[dependency.Name] = purl

		props := buildProperties(dependency, true)
		components = append(components, cycloneDXComponent{
			Type:       "library",
			BomRef:     purl,
			Name:       dependency.Name,
			Version:    dependency.Version,
			Scope:      mapScopeToRequired(dependency.Scope),
			Purl:       purl,
			Properties: props,
		})
	}

	// Process transitive dependencies
	for _, dependency := range summary.Transitive {
		purl := dependency.Purl()
		bomRefMap[dependency.Name] = purl

		props := buildProperties(dependency, false)
		components = append(components, cycloneDXComponent{
			Type:       "library",
			BomRef:     purl,
			Name:       dependency.Name,
			Version:    dependency.Version,
			Scope:      "optional", // CycloneDX uses optional for transitive
			Purl:       purl,
			Properties: props,
		})
	}

	if len(components) > 0 {
		bom.Components = &componentList{Components: components}
	}

	// Build dependencies section
	dependencies := buildDependenciesSection(rootPurl, summary, bomRefMap)
	if len(dependencies) > 0 {
		bom.Dependencies = &dependenciesList{Dependencies: dependencies}
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

// buildProperties creates the supply chain properties for a component.
func buildProperties(dep deps.Dependency, isDirect bool) *propertiesList {
	depType := "transitive"
	if isDirect {
		depType = "direct"
	}

	ecosystem := dep.Ecosystem
	if ecosystem == "" {
		ecosystem = "unknown"
	}

	depth := dep.Depth
	chain := dep.Chain
	if chain == "" {
		chain = dep.Name
	}

	sourceFile := dep.SourceFile
	if sourceFile == "" {
		sourceFile = "unknown"
	}

	sourceLocation := dep.SourceLocation
	if sourceLocation == "" {
		sourceLocation = "unknown"
	}

	props := []cycloneDXProperty{
		{Name: "supplychain:dependency-type", Value: depType},
		{Name: "supplychain:ecosystem", Value: ecosystem},
		{Name: "supplychain:build-scope", Value: dep.BuildScope()},
		{Name: "supplychain:depth", Value: fmt.Sprintf("%d", depth)},
		{Name: "supplychain:chain", Value: chain},
		{Name: "supplychain:source-file", Value: sourceFile},
		{Name: "supplychain:source-location", Value: sourceLocation},
	}

	return &propertiesList{Properties: props}
}

// buildDependenciesSection creates the CycloneDX dependencies mapping.
func buildDependenciesSection(rootRef string, summary deps.Summary, bomRefMap map[string]string) []cycloneDXDependency {
	dependencies := make([]cycloneDXDependency, 0)

	// Root depends on all direct dependencies
	rootDeps := make([]dependencyRef, 0, len(summary.Direct))
	for _, d := range summary.Direct {
		if ref, ok := bomRefMap[d.Name]; ok {
			rootDeps = append(rootDeps, dependencyRef{Ref: ref})
		}
	}
	if len(rootDeps) > 0 {
		dependencies = append(dependencies, cycloneDXDependency{
			Ref:          rootRef,
			Dependencies: rootDeps,
		})
	}

	// Add dependency relationships for all packages
	allDeps := append(summary.Direct, summary.Transitive...)
	for _, dep := range allDeps {
		if len(dep.Children) == 0 {
			continue
		}

		ref := bomRefMap[dep.Name]
		if ref == "" {
			continue
		}

		childRefs := make([]dependencyRef, 0, len(dep.Children))
		for _, childName := range dep.Children {
			if childRef, ok := bomRefMap[childName]; ok {
				childRefs = append(childRefs, dependencyRef{Ref: childRef})
			}
		}

		if len(childRefs) > 0 {
			dependencies = append(dependencies, cycloneDXDependency{
				Ref:          ref,
				Dependencies: childRefs,
			})
		}
	}

	return dependencies
}

// mapScopeToRequired maps internal scope to CycloneDX scope (required/optional).
func mapScopeToRequired(scope deps.Scope) string {
	switch scope {
	case deps.ScopeRuntime:
		return "required"
	case deps.ScopeDev, deps.ScopeTest, deps.ScopeBuild:
		return "optional"
	default:
		return "required"
	}
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
