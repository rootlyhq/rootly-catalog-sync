package source

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func Parse(data []byte) ([]Entry, error) {
	var raw any
	if err := json.Unmarshal(data, &raw); err == nil {
		return entriesFromAny(raw)
	}

	entries, err := parseYAML(data)
	if err == nil && len(entries) > 0 {
		return entries, nil
	}

	return nil, fmt.Errorf("failed to parse data as JSON or YAML")
}

func parseYAML(data []byte) ([]Entry, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var entries []Entry
	for {
		var doc any
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if doc == nil {
			continue
		}
		parsed, err := entriesFromAny(doc)
		if err != nil {
			return nil, err
		}
		entries = append(entries, parsed...)
	}
	return entries, nil
}

func entriesFromAny(v any) ([]Entry, error) {
	switch val := v.(type) {
	case map[string]any:
		return []Entry{Entry(val)}, nil
	case []any:
		var entries []Entry
		for i, item := range val {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("element %d is not a map", i)
			}
			entries = append(entries, Entry(m))
		}
		return entries, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}
