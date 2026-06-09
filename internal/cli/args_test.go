package cli

import "testing"

func TestReorderFlagsFirst(t *testing.T) {
	t.Parallel()

	got := reorderFlagsFirst(
		[]string{"./repo", "--include-vulnerabilities", "--output", "./out", "--format", "both"},
		scanBoolFlags,
	)
	want := []string{"--include-vulnerabilities", "--output", "./out", "--format", "both", "./repo"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestReorderFlagsFirstInlineValue(t *testing.T) {
	t.Parallel()

	got := reorderFlagsFirst(
		[]string{"./repo", "--format=spdx", "--fail-on-vuln"},
		scanBoolFlags,
	)
	want := []string{"--format=spdx", "--fail-on-vuln", "./repo"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestReorderFlagsFirstPreservesURLs(t *testing.T) {
	t.Parallel()

	got := reorderFlagsFirst(
		[]string{"https://github.com/lodash/lodash", "--include-vulnerabilities"},
		githubBoolFlags,
	)
	want := []string{"--include-vulnerabilities", "https://github.com/lodash/lodash"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
