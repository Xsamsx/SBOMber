package supplychain

import "testing"

func TestLooksLikePrivatePackage(t *testing.T) {
	t.Parallel()

	if !looksLikePrivatePackage("acme-internal-utils") {
		t.Fatal("expected internal package name to match")
	}
	if looksLikePrivatePackage("lodash") {
		t.Fatal("expected public package name not to match")
	}
}

func TestNormalizeEcosystem(t *testing.T) {
	t.Parallel()

	if normalizeEcosystem("Python") != "python" {
		t.Fatalf("unexpected normalization: %q", normalizeEcosystem("Python"))
	}
}
