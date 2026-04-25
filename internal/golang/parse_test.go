package golang

import (
	"path/filepath"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "fixtures", "go-basic")
	summary, err := ParseGoMod(root)
	if err != nil {
		t.Fatalf("ParseGoMod failed: %v", err)
	}

	if len(summary.Direct) != 1 {
		t.Errorf("expected 1 direct dependency, got %d", len(summary.Direct))
	}
	if len(summary.Transitive) != 1 {
		t.Errorf("expected 1 transitive dependency, got %d", len(summary.Transitive))
	}

	if summary.Direct[0].Name != "github.com/gin-gonic/gin" {
		t.Errorf("expected gin as direct dep, got %s", summary.Direct[0].Name)
	}
	if summary.Direct[0].Version != "v1.9.1" {
		t.Errorf("expected gin version v1.9.1, got %s", summary.Direct[0].Version)
	}

	if summary.Transitive[0].Name != "golang.org/x/crypto" {
		t.Errorf("expected crypto as transitive dep, got %s", summary.Transitive[0].Name)
	}
	if summary.Transitive[0].Version != "v0.14.0" {
		t.Errorf("expected crypto version v0.14.0, got %s", summary.Transitive[0].Version)
	}
}

func TestParseGoModNotFound(t *testing.T) {
	summary, err := ParseGoMod("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("expected no error for missing go.mod, got: %v", err)
	}
	if len(summary.Direct) != 0 {
		t.Errorf("expected 0 direct deps, got %d", len(summary.Direct))
	}
	if len(summary.Transitive) != 0 {
		t.Errorf("expected 0 transitive deps, got %d", len(summary.Transitive))
	}
}
