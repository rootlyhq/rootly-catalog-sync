package source

import "context"

type InlineSource struct {
	Entries []map[string]any
}

func NewInlineSource(entries []map[string]any) *InlineSource {
	return &InlineSource{Entries: entries}
}

func (s *InlineSource) Name() string { return "inline" }

func (s *InlineSource) Load(ctx context.Context) ([]Entry, error) {
	result := make([]Entry, len(s.Entries))
	for i, e := range s.Entries {
		result[i] = Entry(e)
	}
	return result, nil
}
