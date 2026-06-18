package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

const (
	DefaultBaseURL    = "https://api.rootly.com"
	DefaultAPIPath    = "/v1"
	DefaultMaxRetries = 5
	MaxBatchSize      = 100
	DefaultPageSize   = 250
	UserAgent         = "rootly-catalog-sync/0.1.0 (+https://github.com/rootlyhq/rootly-catalog-sync)"

	jData       = "data"
	jType       = "type"
	jAttributes = "attributes"
	jName       = "name"
	jSlug       = "slug"
	jExtID      = "external_id"
	jDesc       = "description"
)

type Client struct {
	baseURL    string
	apiPath    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
}

type CatalogSpec struct {
	Name        string
	Description string
	ExternalID  string
}

type FieldSpec struct {
	Name       string
	Kind       string
	ExternalID string
	Slug       string
	Multiple   bool
	Required   bool
}

type BulkResult struct {
	Succeeded           int
	DeletedExternalIDs  []string
	NotFoundExternalIDs []string
	Errors              []BulkError
}

type BulkError struct {
	Index      int    `json:"index"`
	ExternalID string `json:"external_id"`
	Errors     any    `json:"errors"`
}

type ClientInterface interface {
	ListEntities(ctx context.Context, catalogID string) ([]catalog.LiveEntity, error)
	BulkUpsert(ctx context.Context, catalogID string, ents []catalog.DesiredEntity) (*BulkResult, error)
	BulkDelete(ctx context.Context, catalogID string, externalIDs []string) (*BulkResult, error)
	EnsureCatalog(ctx context.Context, spec CatalogSpec) (string, error)
	EnsureFields(ctx context.Context, catalogID string, fields []FieldSpec) error
	ListCatalogs(ctx context.Context) ([]CatalogInfo, error)
}

type CatalogInfo struct {
	ID   string
	Name string
	Slug string
}

type Option func(*Client)

func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

func WithAPIPath(p string) Option {
	return func(c *Client) { c.apiPath = p }
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

func WithMaxRetries(n int) Option {
	return func(c *Client) { c.maxRetries = n }
}

func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		apiPath:    DefaultAPIPath,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxRetries: DefaultMaxRetries,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	u := strings.TrimRight(c.baseURL, "/") + strings.TrimRight(c.apiPath, "/") + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")

	return req, nil
}

type jsonAPIListResponse struct {
	Data []jsonAPIResource `json:"data"`
	Meta struct {
		NextCursor  string `json:"next_cursor"`
		TotalCount  int    `json:"total_count"`
		TotalPages  int    `json:"total_pages"`
		CurrentPage int    `json:"current_page"`
	} `json:"meta"`
}

type jsonAPIResource struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes json.RawMessage `json:"attributes"`
}

type jsonAPISingleResponse struct {
	Data jsonAPIResource `json:"data"`
}

// ListCatalogs returns all catalogs, paginating with page[number].
func (c *Client) ListCatalogs(ctx context.Context) ([]CatalogInfo, error) {
	var catalogs []CatalogInfo
	page := 1

	for {
		path := fmt.Sprintf("/catalogs?page[number]=%d&page[size]=%d", page, DefaultPageSize)
		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.doWithRetry(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("listing catalogs: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("listing catalogs: unexpected status %d", resp.StatusCode)
		}

		var parsed jsonAPIListResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("decoding catalogs response: %w", err)
		}
		_ = resp.Body.Close()

		for _, r := range parsed.Data {
			var attrs struct {
				Name string `json:"name"`
				Slug string `json:"slug"`
			}
			if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
				return nil, fmt.Errorf("decoding catalog attributes: %w", err)
			}
			catalogs = append(catalogs, CatalogInfo{
				ID:   r.ID,
				Name: attrs.Name,
				Slug: attrs.Slug,
			})
		}

		if len(parsed.Data) == 0 || parsed.Meta.CurrentPage >= parsed.Meta.TotalPages {
			break
		}
		page++
	}

	return catalogs, nil
}

// ListEntities returns all entities for a catalog, using cursor-based pagination.
// It also fetches catalog fields to resolve property IDs to field names.
func (c *Client) ListEntities(ctx context.Context, catalogID string) ([]catalog.LiveEntity, error) {
	fields, err := c.listFields(ctx, catalogID)
	if err != nil {
		return nil, fmt.Errorf("listing fields for property resolution: %w", err)
	}
	fieldIDToName := make(map[string]string, len(fields))
	for _, f := range fields {
		fieldIDToName[f.ID] = f.Name
	}

	var entities []catalog.LiveEntity
	var cursor string

	for {
		path := fmt.Sprintf("/catalogs/%s/entities?page[size]=%d", catalogID, DefaultPageSize)
		if cursor != "" {
			path += "&page[after]=" + url.QueryEscape(cursor)
		}

		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.doWithRetry(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("listing entities: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("listing entities: unexpected status %d", resp.StatusCode)
		}

		var parsed jsonAPIListResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("decoding entities response: %w", err)
		}
		_ = resp.Body.Close()

		for _, r := range parsed.Data {
			ent, err := parseEntity(r, fieldIDToName)
			if err != nil {
				return nil, err
			}
			entities = append(entities, ent)
		}

		if parsed.Meta.NextCursor == "" || len(parsed.Data) == 0 {
			break
		}
		cursor = parsed.Meta.NextCursor
	}

	return entities, nil
}

func parseEntity(r jsonAPIResource, fieldIDToName map[string]string) (catalog.LiveEntity, error) {
	var attrs struct {
		Name        string `json:"name"`
		ExternalID  string `json:"external_id"`
		Description string `json:"description"`
		ManagedBy   string `json:"managed_by"`
		Properties  []struct {
			CatalogPropertyID string `json:"catalog_property_id"`
			Value             string `json:"value"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
		return catalog.LiveEntity{}, fmt.Errorf("decoding entity attributes: %w", err)
	}

	entityFields := make(map[string]string, len(attrs.Properties))
	for _, p := range attrs.Properties {
		name := fieldIDToName[p.CatalogPropertyID]
		if name != "" {
			entityFields[name] = p.Value
		}
	}

	return catalog.LiveEntity{
		ID:          r.ID,
		ExternalID:  attrs.ExternalID,
		Name:        attrs.Name,
		Description: attrs.Description,
		ManagedBy:   attrs.ManagedBy,
		Fields:      entityFields,
	}, nil
}

// EnsureCatalog finds a catalog by name or creates it, returning the catalog ID.
func (c *Client) EnsureCatalog(ctx context.Context, spec CatalogSpec) (string, error) {
	catalogs, err := c.ListCatalogs(ctx)
	if err != nil {
		return "", err
	}

	for _, cat := range catalogs {
		if cat.Name == spec.Name {
			return cat.ID, nil
		}
	}

	body := map[string]any{
		jData: map[string]any{
			jType: "catalogs",
			jAttributes: map[string]any{
				"name": spec.Name,
				jDesc:  spec.Description,
			},
		},
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/catalogs", body)
	if err != nil {
		return "", err
	}

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("creating catalog: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return "", fmt.Errorf("creating catalog: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed jsonAPISingleResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		_ = resp.Body.Close()
		return "", fmt.Errorf("decoding created catalog: %w", err)
	}
	_ = resp.Body.Close()

	return parsed.Data.ID, nil
}

// EnsureFields ensures all desired fields exist on a catalog, creating any that are missing.
func (c *Client) EnsureFields(ctx context.Context, catalogID string, fields []FieldSpec) error {
	existing, err := c.listFields(ctx, catalogID)
	if err != nil {
		return err
	}

	existingByName := make(map[string]bool, len(existing))
	for _, f := range existing {
		existingByName[f.Name] = true
	}

	for _, f := range fields {
		if existingByName[f.Name] {
			continue
		}

		kind := f.Kind
		if kind == "" {
			kind = "text"
		}

		body := map[string]any{
			jData: map[string]any{
				jType: "catalog_fields",
				jAttributes: map[string]any{
					"name":     f.Name,
					"kind":     kind,
					"slug":     f.Slug,
					"multiple": f.Multiple,
					"required": f.Required,
				},
			},
		}

		req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/catalogs/%s/fields", catalogID), body)
		if err != nil {
			return err
		}

		resp, err := c.doWithRetry(ctx, req)
		if err != nil {
			return fmt.Errorf("creating field %q: %w", f.Name, err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("creating field %q: unexpected status %d", f.Name, resp.StatusCode)
		}
	}

	return nil
}

type fieldInfo struct {
	ID   string
	Name string
	Slug string
}

func (c *Client) listFields(ctx context.Context, catalogID string) ([]fieldInfo, error) {
	var fields []fieldInfo
	page := 1

	for {
		path := fmt.Sprintf("/catalogs/%s/fields?page[number]=%d&page[size]=%d", catalogID, page, DefaultPageSize)
		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.doWithRetry(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("listing fields: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("listing fields: unexpected status %d", resp.StatusCode)
		}

		var parsed jsonAPIListResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("decoding fields response: %w", err)
		}
		_ = resp.Body.Close()

		for _, r := range parsed.Data {
			var attrs struct {
				Name string `json:"name"`
				Slug string `json:"slug"`
			}
			if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
				return nil, fmt.Errorf("decoding field attributes: %w", err)
			}
			fields = append(fields, fieldInfo{ID: r.ID, Name: attrs.Name, Slug: attrs.Slug})
		}

		if len(parsed.Data) == 0 || parsed.Meta.CurrentPage >= parsed.Meta.TotalPages {
			break
		}
		page++
	}

	return fields, nil
}

// BulkUpsert creates or updates entities in batches of MaxBatchSize.
func (c *Client) BulkUpsert(ctx context.Context, catalogID string, ents []catalog.DesiredEntity) (*BulkResult, error) {
	result := &BulkResult{}

	for i := 0; i < len(ents); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(ents) {
			end = len(ents)
		}
		batch := ents[i:end]

		entities := make([]map[string]any, len(batch))
		for j, e := range batch {
			var fields []map[string]string
			for slug, val := range e.Fields {
				fields = append(fields, map[string]string{
					"catalog_field_id": slug,
					"value":            val,
				})
			}
			ent := map[string]any{
				jExtID:   e.ExternalID,
				"name":   e.Name,
				"fields": fields,
			}
			if e.BackstageID != "" {
				ent["backstage_id"] = e.BackstageID
			}
			entities[j] = ent
		}

		body := map[string]any{"entities": entities}
		req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/catalogs/%s/entities/bulk_upsert", catalogID), body)
		if err != nil {
			return nil, err
		}

		resp, err := c.doWithRetry(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("bulk upsert: %w", err)
		}

		var batchResult struct {
			Succeeded int         `json:"succeeded"`
			Errors    []BulkError `json:"errors"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&batchResult); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("decoding bulk upsert response: %w", err)
		}
		_ = resp.Body.Close()

		result.Succeeded += batchResult.Succeeded
		result.Errors = append(result.Errors, batchResult.Errors...)
	}

	return result, nil
}

// BulkDelete deletes entities by external ID in batches of MaxBatchSize.
func (c *Client) BulkDelete(ctx context.Context, catalogID string, externalIDs []string) (*BulkResult, error) {
	result := &BulkResult{}

	for i := 0; i < len(externalIDs); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(externalIDs) {
			end = len(externalIDs)
		}
		batch := externalIDs[i:end]

		body := map[string]any{"external_ids": batch}
		req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/catalogs/%s/entities/bulk_delete", catalogID), body)
		if err != nil {
			return nil, err
		}

		resp, err := c.doWithRetry(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("bulk delete: %w", err)
		}

		var batchResult struct {
			DeletedExternalIDs  []string `json:"deleted_external_ids"`
			NotFoundExternalIDs []string `json:"not_found_external_ids"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&batchResult); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("decoding bulk delete response: %w", err)
		}
		_ = resp.Body.Close()

		result.DeletedExternalIDs = append(result.DeletedExternalIDs, batchResult.DeletedExternalIDs...)
		result.NotFoundExternalIDs = append(result.NotFoundExternalIDs, batchResult.NotFoundExternalIDs...)
	}

	return result, nil
}
