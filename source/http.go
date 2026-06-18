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

var httpSourceClient = &http.Client{Timeout: 30 * time.Second}

type HTTPSource struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    string
	Result  string
}

func NewHTTPSource(cfg *config.HTTPSourceConfig) *HTTPSource {
	method := cfg.Method
	if method == "" {
		method = http.MethodGet
	}
	return &HTTPSource{
		URL:     cfg.URL,
		Method:  strings.ToUpper(method),
		Headers: cfg.Headers,
		Body:    cfg.Body,
		Result:  cfg.Result,
	}
}

func (s *HTTPSource) Name() string { return "http" }

func (s *HTTPSource) Load(ctx context.Context) ([]Entry, error) {
	var bodyReader io.Reader
	if s.Body != "" {
		bodyReader = bytes.NewBufferString(s.Body)
	}

	req, err := http.NewRequestWithContext(ctx, s.Method, s.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("http: creating request: %w", err)
	}
	for k, v := range s.Headers {
		req.Header.Set(k, v)
	}
	if s.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpSourceClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: request failed: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("http: reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http: %s returned status %d", s.URL, resp.StatusCode)
	}

	if s.Result == "" {
		return Parse(body)
	}

	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("http: parsing JSON response: %w", err)
	}

	items, err := extractPath(raw, s.Result)
	if err != nil {
		return nil, fmt.Errorf("http: extracting result at %q: %w", s.Result, err)
	}

	arr, ok := items.([]any)
	if !ok {
		return nil, fmt.Errorf("http: result at %q is not an array", s.Result)
	}

	entries := make([]Entry, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("http: item in result is not an object")
		}
		entries = append(entries, Entry(m))
	}

	return entries, nil
}
