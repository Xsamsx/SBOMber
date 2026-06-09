package ecosystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindManifestRootsNestedPackageJSON(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "packages", "web")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "package.json"), []byte(`{"name":"web"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	roots, err := FindManifestRoots(root)
	if err != nil {
		t.Fatalf("FindManifestRoots returned error: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected 1 manifest root, got %d", len(roots))
	}
	if roots[0] != nested {
		t.Fatalf("expected nested path %q, got %q", nested, roots[0])
	}
}

func TestDetectRepositoryMergesNestedEvidence(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/root\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	nested := filepath.Join(root, "packages", "web")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "package.json"), []byte(`{"name":"web"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	detection, err := DetectRepository(root)
	if err != nil {
		t.Fatalf("DetectRepository returned error: %v", err)
	}

	if len(detection.Names) != 2 {
		t.Fatalf("expected 2 ecosystems, got %d", len(detection.Names))
	}
	if len(detection.Evidence[NPM]) == 0 {
		t.Fatal("expected npm evidence from nested package.json")
	}
	if len(detection.Evidence[Go]) == 0 {
		t.Fatal("expected go evidence from root go.mod")
	}
}
