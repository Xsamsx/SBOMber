package ecosystem

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

var skipManifestWalkDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"vendor":       {},
	"dist":         {},
	"build":        {},
	"target":       {},
	".next":        {},
	"__pycache__":  {},
	".venv":        {},
	"venv":         {},
}

// FindManifestRoots returns directories under root that contain supported manifest files.
func FindManifestRoots(root string) ([]string, error) {
	roots := make([]string, 0)
	seen := make(map[string]struct{})

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}

		if path != root {
			if _, skip := skipManifestWalkDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}
		}

		if !directoryHasManifestMarker(path) {
			return nil
		}

		if _, ok := seen[path]; ok {
			return nil
		}
		seen[path] = struct{}{}
		roots = append(roots, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(roots)
	return roots, nil
}

// DetectRepository discovers ecosystems across the repo root and nested manifest directories.
func DetectRepository(root string) (Detection, error) {
	manifestRoots, err := FindManifestRoots(root)
	if err != nil {
		return Detection{}, err
	}
	if len(manifestRoots) == 0 {
		return Detect(root)
	}

	merged := Detection{
		Names:    make([]Name, 0),
		Evidence: make(map[Name][]string),
	}
	nameSet := make(map[Name]struct{})
	evidenceSet := make(map[Name]map[string]struct{})

	for _, manifestRoot := range manifestRoots {
		local, err := Detect(manifestRoot)
		if err != nil {
			return Detection{}, err
		}

		for _, name := range local.Names {
			if _, ok := nameSet[name]; !ok {
				nameSet[name] = struct{}{}
				merged.Names = append(merged.Names, name)
			}
			if evidenceSet[name] == nil {
				evidenceSet[name] = make(map[string]struct{})
			}
			for _, file := range local.Evidence[name] {
				rel, relErr := filepath.Rel(root, filepath.Join(manifestRoot, file))
				if relErr != nil {
					rel = filepath.Join(filepath.Base(manifestRoot), file)
				}
				if _, ok := evidenceSet[name][rel]; ok {
					continue
				}
				evidenceSet[name][rel] = struct{}{}
				merged.Evidence[name] = append(merged.Evidence[name], rel)
			}
		}
	}

	for name := range merged.Evidence {
		sort.Strings(merged.Evidence[name])
	}
	sort.Slice(merged.Names, func(i, j int) bool {
		return merged.Names[i] < merged.Names[j]
	})

	return merged, nil
}

func directoryHasManifestMarker(dir string) bool {
	for _, filenames := range markers {
		for _, filename := range filenames {
			if _, err := os.Stat(filepath.Join(dir, filename)); err == nil {
				return true
			}
		}
	}

	gemspecs, err := filepath.Glob(filepath.Join(dir, "*.gemspec"))
	return err == nil && len(gemspecs) > 0
}
