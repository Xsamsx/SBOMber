package remote

import "testing"

func TestParseYarnLock(t *testing.T) {
	content := []byte(`__metadata:
  version: 8

"react@npm:^19.0.0":
  version: 19.1.0
  resolution: "react@npm:19.1.0"
  dependencies:
    loose-envify: "npm:^1.1.0"

"loose-envify@npm:^1.1.0":
  version: 1.4.0
  resolution: "loose-envify@npm:1.4.0"
`)

	summary, err := parseYarnLock(content)
	if err != nil {
		t.Fatalf("parseYarnLock failed: %v", err)
	}

	if len(summary.Transitive) != 2 {
		t.Fatalf("expected 2 transitive dependencies, got %d", len(summary.Transitive))
	}
}
