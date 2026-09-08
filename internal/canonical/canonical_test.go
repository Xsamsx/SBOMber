package canonical

import "testing"

func TestOccurrenceIdentityPreservesDistinctOccurrences(t *testing.T) {
	componentA := Component{Name: "ansi-styles", Version: "4.3.0", PURL: "pkg:npm/ansi-styles@4.3.0"}
	componentB := Component{Name: "ansi-styles", Version: "3.2.0", PURL: "pkg:npm/ansi-styles@3.2.0"}

	occA := Occurrence{Workspace: "/repo/app-a", ManifestPath: "package.json", Dependency: "dependencies.ansi-styles", ComponentPURL: componentA.PURL}
	occB := Occurrence{Workspace: "/repo/app-b", ManifestPath: "package.json", Dependency: "dependencies.ansi-styles", ComponentPURL: componentA.PURL}
	occC := Occurrence{Workspace: "/repo/app-a", ManifestPath: "package.json", Dependency: "dependencies.ansi-styles", ComponentPURL: componentB.PURL}

	if occA.Key() == occB.Key() {
		t.Fatalf("occurrence keys should differ by workspace")
	}
	if occA.Key() == occC.Key() {
		t.Fatalf("occurrence keys should differ by component version")
	}

	scan := NewScan("/repo")
	scan.Components = append(scan.Components, componentA, componentB)
	scan.Occurrences = append(scan.Occurrences, occA, occB, occC)

	if err := scan.Validate(); err != nil {
		t.Fatalf("validate canonical scan: %v", err)
	}
}

func TestPURLParsing(t *testing.T) {
	name, version, ok := ParsePURL("pkg:npm/chalk@5.0.0")
	if !ok {
		t.Fatal("expected valid purl")
	}
	if name != "chalk" || version != "5.0.0" {
		t.Fatalf("unexpected parse result: name=%q version=%q", name, version)
	}
}
