package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitRepositories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	repos := []string{
		filepath.Join(root, "alpha", ".git"),
		filepath.Join(root, "nested", "beta", ".git"),
	}

	for _, repo := range repos {
		if err := os.MkdirAll(repo, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", repo, err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}

	found, err := FindGitRepositories(root)
	if err != nil {
		t.Fatalf("FindGitRepositories returned error: %v", err)
	}

	if len(found) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(found))
	}

	if found[0].Name != "alpha" {
		t.Fatalf("expected first repo alpha, got %s", found[0].Name)
	}

	if found[1].Name != "beta" {
		t.Fatalf("expected second repo beta, got %s", found[1].Name)
	}
}
