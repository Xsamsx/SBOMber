package deps

import "sort"

// MergeSummary appends other into s, deduplicating by ecosystem, name, and version.
func MergeSummary(s *Summary, other Summary) {
	s.Direct = mergeDependencies(s.Direct, other.Direct)
	s.Transitive = mergeDependencies(s.Transitive, other.Transitive)
}

func mergeDependencies(existing, extra []Dependency) []Dependency {
	if len(extra) == 0 {
		return existing
	}

	seen := make(map[string]struct{}, len(existing)+len(extra))
	merged := make([]Dependency, 0, len(existing)+len(extra))

	for _, dep := range existing {
		key := dependencyKey(dep)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, dep)
	}

	for _, dep := range extra {
		key := dependencyKey(dep)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, dep)
	}

	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].Name == merged[j].Name {
			return merged[i].Version < merged[j].Version
		}
		return merged[i].Name < merged[j].Name
	})

	return merged
}

func dependencyKey(dep Dependency) string {
	return dep.Ecosystem + ":" + dep.Name + ":" + dep.Version
}
