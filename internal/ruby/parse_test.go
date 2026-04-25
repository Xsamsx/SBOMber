package ruby

import (
	"path/filepath"
	"testing"
)

func TestParseGemfileLock(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "fixtures", "ruby-basic")
	summary, err := ParseGemfileLock(root)
	if err != nil {
		t.Fatalf("ParseGemfileLock failed: %v", err)
	}
	if len(summary.Direct) != 2 {
		t.Errorf("expected 2 gems, got %d", len(summary.Direct))
	}

	// rack should be present
	found := false
	for _, dep := range summary.Direct {
		if dep.Name == "rack" {
			found = true
			if dep.Version != "2.2.4" {
				t.Errorf("expected rack version 2.2.4, got %s", dep.Version)
			}
		}
	}
	if !found {
		t.Error("expected to find rack in gems")
	}

	// rake should be present
	found = false
	for _, dep := range summary.Direct {
		if dep.Name == "rake" {
			found = true
			if dep.Version != "13.0.6" {
				t.Errorf("expected rake version 13.0.6, got %s", dep.Version)
			}
		}
	}
	if !found {
		t.Error("expected to find rake in gems")
	}
}

func TestParseGemfileLockNotFound(t *testing.T) {
	summary, err := ParseGemfileLock("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("expected no error for missing Gemfile.lock, got: %v", err)
	}
	if len(summary.Direct) != 0 {
		t.Errorf("expected 0 gems for missing file, got %d", len(summary.Direct))
	}
}
