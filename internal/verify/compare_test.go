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

func TestCompareSelfCompareNpmCleanStyleVersions(t *testing.T) {
	// Mirrors npm-clean SBOM: direct dep keeps range, lockfile resolves exact version.
	components := []Component{
		{Name: "is-number", Version: "^7.0.0"},
		{Name: "is-number", Version: "7.0.0"},
	}

	result := Compare(components, components)

	if result.Precision != 100 {
		t.Fatalf("expected precision 100, got %f", result.Precision)
	}
	if result.Recall != 100 {
		t.Fatalf("expected recall 100, got %f", result.Recall)
	}
	if result.MatchedCount != 2 {
		t.Fatalf("expected 2 matched components, got %d", result.MatchedCount)
	}
}

func TestCompareSelfCompareWithSameNameDifferentVersions(t *testing.T) {
	components := []Component{
		{Name: "is-number", Version: "7.0.0"},
		{Name: "is-number", Version: "7.0.1"},
	}

	result := Compare(components, components)

	if result.Precision != 100 {
		t.Fatalf("expected precision 100, got %f", result.Precision)
	}
	if result.Recall != 100 {
		t.Fatalf("expected recall 100, got %f", result.Recall)
	}
}

func TestCompareSelfCompareWithDuplicateRows(t *testing.T) {
	components := []Component{
		{Name: "is-number", Version: "7.0.0"},
		{Name: "is-number", Version: "7.0.0"},
	}

	result := Compare(components, components)

	if result.Precision != 100 {
		t.Fatalf("expected precision 100, got %f", result.Precision)
	}
	if result.Recall != 100 {
		t.Fatalf("expected recall 100, got %f", result.Recall)
	}
	if result.F1Score != 100 {
		t.Fatalf("expected f1 100, got %f", result.F1Score)
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
