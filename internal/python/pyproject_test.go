package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePyProject(t *testing.T) {
	root := t.TempDir()
	content := `[project]
name = "demo"
dependencies = [
  "requests>=2.28.1",
  "flask==2.0.0",
]
`
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write pyproject.toml: %v", err)
	}

	summary, err := ParsePyProject(root)
	if err != nil {
		t.Fatalf("ParsePyProject returned error: %v", err)
	}
	if len(summary.Direct) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(summary.Direct))
	}
}

func TestParseManifestsCombinesRequirementsAndPyProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "requirements.txt"), []byte("numpy>=1.0\n"), 0o644); err != nil {
		t.Fatalf("write requirements.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(`[project]
dependencies = ["requests>=2.0"]
`), 0o644); err != nil {
		t.Fatalf("write pyproject.toml: %v", err)
	}

	summary, err := ParseManifests(root)
	if err != nil {
		t.Fatalf("ParseManifests returned error: %v", err)
	}
	if len(summary.Direct) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(summary.Direct))
	}
}
