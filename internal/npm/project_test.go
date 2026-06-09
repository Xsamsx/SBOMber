package npm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProjectYarnLockOnly(t *testing.T) {
	root := t.TempDir()
	lockfile := `__metadata:
  version: 8

"react@npm:^19.0.0":
  version: 19.1.0
  resolution: "react@npm:19.1.0"
`
	if err := os.WriteFile(filepath.Join(root, "yarn.lock"), []byte(lockfile), 0o644); err != nil {
		t.Fatalf("write yarn.lock: %v", err)
	}

	summary, err := ParseProject(root)
	if err != nil {
		t.Fatalf("ParseProject returned error: %v", err)
	}
	if summary.TransitiveCount() != 1 {
		t.Fatalf("expected 1 transitive dependency, got %d", summary.TransitiveCount())
	}
}

func TestParseProjectPackageLockTransitive(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{
  "dependencies": {
    "lodash": "^4.17.21"
  }
}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	lockfile := `{
  "packages": {
    "": { "version": "1.0.0" },
    "node_modules/lodash": { "version": "4.17.21" },
    "node_modules/axios": { "version": "1.6.0" }
  }
}`
	if err := os.WriteFile(filepath.Join(root, "package-lock.json"), []byte(lockfile), 0o644); err != nil {
		t.Fatalf("write package-lock.json: %v", err)
	}

	summary, err := ParseProject(root)
	if err != nil {
		t.Fatalf("ParseProject returned error: %v", err)
	}
	if summary.Count() != 1 {
		t.Fatalf("expected 1 direct dependency, got %d", summary.Count())
	}
	if summary.TransitiveCount() != 2 {
		t.Fatalf("expected 2 transitive dependencies from lockfile, got %d", summary.TransitiveCount())
	}
}
