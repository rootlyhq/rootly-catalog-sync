package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

var graphqlHTTPClient = &http.Client{Timeout: 30 * time.Second}

type GraphQLSource struct {
	URL      string
	Query    string
	Headers  map[string]string
	Result   string
	Paginate *config.PaginateConfig
}

func NewGraphQLSource(cfg *config.GraphQLSourceConfig) *GraphQLSource {
	return &GraphQLSource{
		URL:      cfg.URL,
		Query:    cfg.Query,
		Headers:  cfg.Headers,
		Result:   cfg.Result,
		Paginate: cfg.Paginate,
	}
}

func (s *GraphQLSource) Name() string { return "graphql" }

func (s *GraphQLSource) Load(ctx context.Context) ([]Entry, error) {
	mode := "none"
	if s.Paginate != nil && s.Paginate.Mode != "" {
		mode = s.Paginate.Mode
	}

	switch mode {
	case "none":
		return s.loadSingle(ctx, nil)
	case "offset":
		return s.loadOffset(ctx)
	case "cursor":
		return s.loadCursor(ctx)
	default:
		return nil, fmt.Errorf("unsupported pagination mode: %s", mode)
	}
}

func (s *GraphQLSource) loadSingle(ctx context.Context, variables map[string]any) ([]Entry, error) {
	body, err := s.execute(ctx, variables)
	if err != nil {
		return nil, err
	}
	return s.extractEntries(body)
}

func (s *GraphQLSource) loadOffset(ctx context.Context) ([]Entry, error) {
	pageSize := s.Paginate.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	var all []Entry
	offset := 0

	for {
		variables := map[string]any{
			"offset": offset,
			"limit":  pageSize,
		}

		body, err := s.execute(ctx, variables)
		if err != nil {
			return nil, err
		}

		entries, err := s.extractEntries(body)
		if err != nil {
			return nil, err
		}

		all = append(all, entries...)

		if len(entries) < pageSize {
			break
		}

		offset += pageSize
	}

	return all, nil
}

func (s *GraphQLSource) loadCursor(ctx context.Context) ([]Entry, error) {
	pageSize := s.Paginate.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	var all []Entry
	var cursor any

	for {
		variables := map[string]any{
			"limit": pageSize,
		}
		if cursor != nil {
			variables["cursor"] = cursor
		}

		body, err := s.execute(ctx, variables)
		if err != nil {
			return nil, err
		}

		entries, err := s.extractEntries(body)
		if err != nil {
			return nil, err
		}

		all = append(all, entries...)

		if s.Paginate.CursorPath == "" {
			break
		}

		nextCursor, err := extractPath(body, s.Paginate.CursorPath)
		if err != nil || nextCursor == nil || nextCursor == "" {
			break
		}

		cursor = nextCursor

		if len(entries) < pageSize {
			break
		}
	}

	return all, nil
}

func (s *GraphQLSource) execute(ctx context.Context, variables map[string]any) (any, error) {
	payload := map[string]any{
		"query": s.Query,
	}
	if len(variables) > 0 {
		payload["variables"] = variables
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("graphql: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("graphql: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range s.Headers {
		req.Header.Set(k, v)
	}

	resp, err := graphqlHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("graphql: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("graphql: unmarshal response: %w", err)
	}

	return result, nil
}

func (s *GraphQLSource) extractEntries(data any) ([]Entry, error) {
	items, err := extractPath(data, s.Result)
	if err != nil {
		return nil, fmt.Errorf("graphql: extract result at %q: %w", s.Result, err)
	}

	arr, ok := items.([]any)
	if !ok {
		return nil, fmt.Errorf("graphql: result at %q is not an array", s.Result)
	}

	entries := make([]Entry, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("graphql: item in result is not an object")
		}
		entries = append(entries, Entry(m))
	}

	return entries, nil
}

func extractPath(data any, path string) (any, error) {
	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot traverse into %q: not an object", part)
		}
		current, ok = m[part]
		if !ok {
			return nil, fmt.Errorf("key %q not found", part)
		}
	}
	return current, nil
}
