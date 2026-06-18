package source

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

type CSVSource struct {
	Files     []string
	Delimiter rune
	BaseDir   string
}

func NewCSVSource(files []string, delimiter string, baseDir string) *CSVSource {
	d := ','
	if len(strings.TrimSpace(delimiter)) > 0 {
		d = rune(delimiter[0])
	}
	return &CSVSource{Files: files, Delimiter: d, BaseDir: baseDir}
}

func (s *CSVSource) Name() string { return "csv" }

func (s *CSVSource) Load(ctx context.Context) ([]Entry, error) {
	paths, err := resolveGlobs(s.Files, s.BaseDir)
	if err != nil {
		return nil, err
	}

	var allEntries []Entry
	for _, path := range paths {
		entries, err := s.readFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

func (s *CSVSource) readFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.Comma = s.Delimiter
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		entry := make(Entry, len(header))
		for i, col := range header {
			if i < len(row) {
				entry[col] = row[i]
			}
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
