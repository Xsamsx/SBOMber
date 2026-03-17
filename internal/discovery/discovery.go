package discovery

import (
	"io/fs"
	"path/filepath"
	"sort"
)

// Repository describes a Git repository discovered on disk.
type Repository struct {
	Name string
	Path string
}

// FindGitRepositories walks a root directory recursively and returns every
// directory that contains a .git entry.
func FindGitRepositories(root string) ([]Repository, error) {
	repositories := make([]Repository, 0)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		if d.Name() == ".git" {
			repoPath := filepath.Dir(path)
			repositories = append(repositories, Repository{
				Name: filepath.Base(repoPath),
				Path: repoPath,
			})

			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].Path < repositories[j].Path
	})

	return repositories, nil
}
