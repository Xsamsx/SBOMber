package remote

import (
	"testing"

	"github.com/Xsamsx/SBOMber/internal/deps"
)

func TestDeduplicateDepsRemovesDuplicates(t *testing.T) {
	input := []deps.Dependency{
		{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"},
		{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"},
		{Name: "axios", Version: "1.6.0", Ecosystem: "npm"},
	}

	got := deduplicateDeps(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 unique dependencies, got %d", len(got))
	}
}
