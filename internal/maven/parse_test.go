package maven

import (
	"path/filepath"
	"testing"
)

func TestParsePOM(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "fixtures", "maven-basic")
	summary, err := ParsePOM(root)
	if err != nil {
		t.Fatalf("ParsePOM failed: %v", err)
	}
	if len(summary.Direct) != 2 {
		t.Errorf("expected 2 direct dependencies, got %d", len(summary.Direct))
	}
}
