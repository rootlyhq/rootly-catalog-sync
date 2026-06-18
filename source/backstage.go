package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

var backstageHTTPClient = &http.Client{Timeout: 30 * time.Second}

type BackstageSource struct {
	URL    string
	Token  string
	Filter string
	Kind   string
}

func NewBackstageSource(cfg *config.BackstageSourceConfig) *BackstageSource {
	return &BackstageSource{
		URL:    cfg.URL,
		Token:  cfg.Token,
		Filter: cfg.Filter,
		Kind:   cfg.Kind,
	}
}

func (s *BackstageSource) Name() string { return "backstage" }

func (s *BackstageSource) Load(ctx context.Context) ([]Entry, error) {
	const limit = 500
	var allEntries []Entry
	offset := 0

	for {
		reqURL, err := s.buildURL(offset, limit)
		if err != nil {
			return nil, fmt.Errorf("building request URL: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		if s.Token != "" {
			req.Header.Set("Authorization", "Bearer "+s.Token)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := backstageHTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching entities: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("backstage API returned status %d: %s", resp.StatusCode, string(body))
		}

		var response struct {
			Items []json.RawMessage `json:"items"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("decoding response: %w", err)
		}

		for _, raw := range response.Items {
			entry, err := flattenEntity(raw)
			if err != nil {
				return nil, fmt.Errorf("flattening entity: %w", err)
			}
			allEntries = append(allEntries, entry)
		}

		if len(response.Items) < limit {
			break
		}
		offset += limit
	}

	return allEntries, nil
}

func (s *BackstageSource) buildURL(offset, limit int) (string, error) {
	u, err := url.Parse(strings.TrimRight(s.URL, "/") + "/api/catalog/entities/by-query")
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("offset", fmt.Sprintf("%d", offset))
	q.Set("limit", fmt.Sprintf("%d", limit))

	if s.Filter != "" {
		q.Set("filter", s.Filter)
	} else if s.Kind != "" {
		q.Set("filter", "kind="+s.Kind)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func flattenEntity(raw json.RawMessage) (Entry, error) {
	var entity struct {
		Kind     string `json:"kind"`
		Metadata struct {
			Name        string            `json:"name"`
			Namespace   string            `json:"namespace"`
			Description string            `json:"description"`
			Annotations map[string]string `json:"annotations"`
			Labels      map[string]string `json:"labels"`
		} `json:"metadata"`
		Spec map[string]any `json:"spec"`
	}
	if err := json.Unmarshal(raw, &entity); err != nil {
		return nil, err
	}

	namespace := entity.Metadata.Namespace
	if namespace == "" {
		namespace = "default"
	}
	backstageID := fmt.Sprintf("%s:%s/%s", entity.Kind, namespace, entity.Metadata.Name)

	entry := Entry{
		"kind":         entity.Kind,
		"name":         entity.Metadata.Name,
		"namespace":    namespace,
		"backstage_id": backstageID,
	}

	if entity.Metadata.Description != "" {
		entry["description"] = entity.Metadata.Description
	}
	if len(entity.Metadata.Annotations) > 0 {
		entry["annotations"] = entity.Metadata.Annotations
	}
	if len(entity.Metadata.Labels) > 0 {
		entry["labels"] = entity.Metadata.Labels
	}

	for k, v := range entity.Spec {
		entry[k] = v
	}

	return entry, nil
}
