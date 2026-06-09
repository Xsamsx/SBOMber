package sbom

import (
	"path/filepath"
	"testing"
)

func TestResolveOutputDirCustom(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	custom := filepath.Join(base, "reports")

	dir, err := ResolveOutputDir(custom, "/tmp/example")
	if err != nil {
		t.Fatalf("ResolveOutputDir failed: %v", err)
	}
	if dir != custom {
		t.Fatalf("expected %q, got %q", custom, dir)
	}
}

func TestResolveBatchOutputDirCustom(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	dir, err := ResolveBatchOutputDir(base, "batch")
	if err != nil {
		t.Fatalf("ResolveBatchOutputDir failed: %v", err)
	}
	if dir != base {
		t.Fatalf("expected custom batch dir %q, got %q", base, dir)
	}
}
