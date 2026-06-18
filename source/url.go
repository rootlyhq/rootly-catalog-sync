package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

var urlHTTPClient = &http.Client{Timeout: 30 * time.Second}

type URLSource struct {
	URLs    []string
	Headers map[string]string
}

func NewURLSource(cfg *config.URLSourceConfig) *URLSource {
	return &URLSource{
		URLs:    cfg.URLs,
		Headers: cfg.Headers,
	}
}

func (s *URLSource) Name() string { return "url" }

func (s *URLSource) Load(ctx context.Context) ([]Entry, error) {
	var allEntries []Entry
	for _, u := range s.URLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, fmt.Errorf("url: creating request for %s: %w", u, err)
		}
		for k, v := range s.Headers {
			req.Header.Set(k, v)
		}

		resp, err := urlHTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("url: fetching %s: %w", u, err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("url: reading %s: %w", u, err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("url: %s returned status %d", u, resp.StatusCode)
		}

		entries, err := Parse(body)
		if err != nil {
			return nil, fmt.Errorf("url: parsing %s: %w", u, err)
		}
		allEntries = append(allEntries, entries...)
	}
	return allEntries, nil
}
