package cli

import "testing"

func TestSelectSBOMPathPrefersCycloneDX(t *testing.T) {
	paths := []string{
		"/tmp/repo/sbom.spdx",
		"/tmp/repo/sbom-cyclonedx.xml",
	}

	got := selectSBOMPath(paths)
	want := "/tmp/repo/sbom-cyclonedx.xml"
	if got != want {
		t.Fatalf("selectSBOMPath() = %q, want %q", got, want)
	}
}

func TestGrypeScanTargetUsesExportedSBOM(t *testing.T) {
	paths := []string{"/tmp/repo/sbom-cyclonedx.xml"}
	got := grypeScanTarget(paths, "/tmp/repo")
	want := "sbom:/tmp/repo/sbom-cyclonedx.xml"
	if got != want {
		t.Fatalf("grypeScanTarget() = %q, want %q", got, want)
	}
}
