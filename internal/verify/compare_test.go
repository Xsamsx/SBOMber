package verify

import "testing"

func TestComparePrecisionExcludesVersionMismatches(t *testing.T) {
	groundTruth := []Component{
		{Name: "lodash", Version: "4.17.21"},
	}
	generated := []Component{
		{Name: "lodash", Version: "4.17.20"},
	}

	result := Compare(groundTruth, generated)

	if result.MatchedCount != 1 {
		t.Fatalf("expected matched count 1, got %d", result.MatchedCount)
	}
	if result.VersionMismatch != 1 {
		t.Fatalf("expected version mismatch 1, got %d", result.VersionMismatch)
	}
	if result.Precision != 0 {
		t.Fatalf("expected precision 0, got %f", result.Precision)
	}
	if result.Recall != 100 {
		t.Fatalf("expected recall 100, got %f", result.Recall)
	}
}

func TestComparePrecisionIncludesExtrasInDenominator(t *testing.T) {
	groundTruth := []Component{
		{Name: "express", Version: "4.18.2"},
	}
	generated := []Component{
		{Name: "express", Version: "4.18.2"},
		{Name: "phantom", Version: "1.0.0"},
	}

	result := Compare(groundTruth, generated)

	if result.Precision != 50 {
		t.Fatalf("expected precision 50, got %f", result.Precision)
	}
	if result.ExtraCount != 1 {
		t.Fatalf("expected 1 extra component, got %d", result.ExtraCount)
	}
}
