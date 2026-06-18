package source

import (
	"context"
	"fmt"
	"os"
)

type LocalSource struct {
	Files   []string
	BaseDir string
}

func NewLocalSource(files []string, baseDir string) *LocalSource {
	return &LocalSource{Files: files, BaseDir: baseDir}
}

func (s *LocalSource) Name() string { return "local" }

func (s *LocalSource) Load(ctx context.Context) ([]Entry, error) {
	paths, err := resolveGlobs(s.Files, s.BaseDir)
	if err != nil {
		return nil, err
	}

	var allEntries []Entry
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		entries, err := Parse(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}
