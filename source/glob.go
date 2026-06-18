package source

import (
	"fmt"
	"path/filepath"
)

func resolveGlobs(patterns []string, baseDir string) ([]string, error) {
	var paths []string
	for _, pattern := range patterns {
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(baseDir, pattern)
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		paths = append(paths, matches...)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no files matched patterns %v", patterns)
	}
	return paths, nil
}
