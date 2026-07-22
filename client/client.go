package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oapi-codegen/nullable"
	rootly "github.com/rootlyhq/rootly-go"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

const (
	DefaultBaseURL    = "https://api.rootly.com"
	DefaultAPIPath    = "/v1"
	DefaultMaxRetries = 5
	MaxBatchSize      = 100
	DefaultPageSize   = 250
	UserAgent         = "rootly-catalog-sync/0.1.0 (+https://github.com/rootlyhq/rootly-catalog-sync)"
)

type Client struct {
	baseURL    string
	apiPath    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
	sdk        *rootly.ClientWithResponses
}

type CatalogSpec struct {
	Name        string
	Description string
	ExternalID  string
}

type FieldSpec struct {
	Name          string
	Kind          string
	ExternalID    string
	Slug          string
	KindCatalogID string
	Multiple      bool
	Required      bool
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

var sdkCreatableKinds = map[string]bool{
	"text": true, "boolean": true, "reference": true,
	"service": true, "group": true, "functionality": true, "environment": true,
	"incident_type": true, "cause": true, "user": true,
}

func sdkCreatableKind(kind string) bool {
	return sdkCreatableKinds[kind]
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

	// Build the SDK client. The SDK hardcodes "/v1/" in its operation paths,
	// so the server URL is baseURL + any prefix before "/v1" in apiPath.
	// e.g. apiPath="/v1" → serverURL=baseURL, apiPath="/api/v1" → serverURL=baseURL+"/api"
	trimmedPath := strings.TrimRight(c.apiPath, "/")
	trimmedPath = strings.TrimSuffix(trimmedPath, "/v1")
	serverURL := strings.TrimRight(c.baseURL, "/")
	if trimmedPath != "" {
		serverURL += trimmedPath
	}

	sdkOpts := []rootly.ClientOption{
		rootly.WithHTTPClient(&retryHTTPClient{
			inner:      c.httpClient,
			maxRetries: c.maxRetries,
		}),
		rootly.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			if c.apiKey != "" {
				req.Header.Set("Authorization", "Bearer "+c.apiKey)
			}
			req.Header.Set("User-Agent", UserAgent)
			return nil
		}),
	}

	sdk, err := rootly.NewClientWithResponses(serverURL, sdkOpts...)
	if err != nil {
		// This should not happen with valid inputs; panic to surface misuse early.
		panic(fmt.Sprintf("rootly-go: NewClientWithResponses: %v", err))
	}
	c.sdk = sdk

	return c
}

// ListCatalogs returns all catalogs, paginating with page[number].
func (c *Client) ListCatalogs(ctx context.Context) ([]CatalogInfo, error) {
	var catalogs []CatalogInfo
	page := 1

	for {
		pageSize := DefaultPageSize
		resp, err := c.sdk.ListCatalogsWithResponse(ctx, &rootly.ListCatalogsParams{
			PageNumber: &page,
			PageSize:   &pageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("listing catalogs: %w", err)
		}
		if resp.ApplicationVndAPIJSON200 == nil {
			return nil, fmt.Errorf("listing catalogs: unexpected status %d", resp.StatusCode())
		}

		data := resp.ApplicationVndAPIJSON200
		for _, r := range data.Data {
			catalogs = append(catalogs, CatalogInfo{
				ID:   r.ID,
				Name: r.Attributes.Name,
				// Slug is not available in the SDK's Catalog type.
			})
		}

		currentPage := 0
		if data.Meta.CurrentPage.IsSpecified() && !data.Meta.CurrentPage.IsNull() {
			currentPage = data.Meta.CurrentPage.MustGet()
		}
		if len(data.Data) == 0 || currentPage >= data.Meta.TotalPages {
			break
		}
		page++
	}

	return catalogs, nil
}

// ListEntities returns all entities for a catalog, using page-based pagination.
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
	page := 1

	for {
		pageSize := DefaultPageSize
		resp, err := c.sdk.ListCatalogEntitiesWithResponse(ctx, catalogID, &rootly.ListCatalogEntitiesParams{
			PageNumber: &page,
			PageSize:   &pageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("listing entities: %w", err)
		}
		if resp.ApplicationVndAPIJSON200 == nil {
			return nil, fmt.Errorf("listing entities: unexpected status %d", resp.StatusCode())
		}

		data := resp.ApplicationVndAPIJSON200
		for _, r := range data.Data {
			ent := sdkEntityToLive(r, fieldIDToName)
			entities = append(entities, ent)
		}

		currentPage := 0
		if data.Meta.CurrentPage.IsSpecified() && !data.Meta.CurrentPage.IsNull() {
			currentPage = data.Meta.CurrentPage.MustGet()
		}
		if len(data.Data) == 0 || currentPage >= data.Meta.TotalPages {
			break
		}
		page++
	}

	return entities, nil
}

func sdkEntityToLive(r struct {
	Attributes rootly.CatalogEntity             `json:"attributes"`
	ID         string                           `json:"id"`
	Type       rootly.CatalogEntityListDataType `json:"type"`
}, fieldIDToName map[string]string) catalog.LiveEntity {
	entityFields := make(map[string]string, len(r.Attributes.Properties))
	for _, p := range r.Attributes.Properties {
		name := fieldIDToName[p.CatalogPropertyID]
		if name != "" {
			entityFields[name] = p.Value
		}
	}

	externalID := ""
	if r.Attributes.ExternalID.IsSpecified() && !r.Attributes.ExternalID.IsNull() {
		externalID = r.Attributes.ExternalID.MustGet()
	}

	description := ""
	if r.Attributes.Description.IsSpecified() && !r.Attributes.Description.IsNull() {
		description = r.Attributes.Description.MustGet()
	}

	managedBy := ""
	if r.Attributes.ManagedBy != nil {
		managedBy = string(*r.Attributes.ManagedBy)
	}

	return catalog.LiveEntity{
		ID:          r.ID,
		ExternalID:  externalID,
		Name:        r.Attributes.Name,
		Description: description,
		ManagedBy:   managedBy,
		Fields:      entityFields,
	}
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

	body := rootly.NewCatalog{}
	body.Data.Type = rootly.NewCatalogDataTypeCatalogs
	body.Data.Attributes.Name = spec.Name
	if spec.Description != "" {
		body.Data.Attributes.Description = nullable.NewNullableWithValue(spec.Description)
	}

	resp, err := c.sdk.CreateCatalogWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, body)
	if err != nil {
		return "", fmt.Errorf("creating catalog: %w", err)
	}

	if resp.ApplicationVndAPIJSON201 == nil {
		return "", fmt.Errorf("creating catalog: unexpected status %d: %s", resp.StatusCode(), string(resp.Body))
	}

	return resp.ApplicationVndAPIJSON201.Data.ID, nil
}

// EnsureFields ensures all desired fields exist on a catalog, creating any that are missing.
func (c *Client) EnsureFields(ctx context.Context, catalogID string, fields []FieldSpec) error {
	existing, err := c.listFields(ctx, catalogID)
	if err != nil {
		return err
	}

	existingByName := make(map[string]fieldInfo, len(existing))
	for _, f := range existing {
		existingByName[f.Name] = f
	}

	for _, f := range fields {
		if ef, exists := existingByName[f.Name]; exists {
			kind := f.Kind
			if kind == "" {
				kind = "text"
			}
			if ef.Kind != kind {
				return fmt.Errorf("field %q exists with kind %q but config declares kind %q", f.Name, ef.Kind, kind)
			}
			if kind == "reference" && f.KindCatalogID != "" && ef.KindCatalogID != "" && ef.KindCatalogID != f.KindCatalogID {
				return fmt.Errorf("field %q references catalog %s but config expects %s", f.Name, ef.KindCatalogID, f.KindCatalogID)
			}
			continue
		}

		kind := f.Kind
		if kind == "" {
			kind = "text"
		}

		if !sdkCreatableKind(kind) {
			return fmt.Errorf("field %q has kind %q which cannot be auto-created — create it in the Rootly UI first", f.Name, kind)
		}

		body := rootly.NewCatalogField{}
		body.Data.Type = rootly.NewCatalogFieldDataTypeCatalogProperties
		body.Data.Attributes.Name = f.Name
		body.Data.Attributes.Kind = rootly.NewCatalogFieldDataAttributesKind(kind)
		body.Data.Attributes.Multiple = &f.Multiple
		body.Data.Attributes.Required = &f.Required
		if f.KindCatalogID != "" {
			body.Data.Attributes.KindCatalogID = nullable.NewNullableWithValue(f.KindCatalogID)
		}

		resp, err := c.sdk.CreateCatalogPropertyWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, catalogID, body)
		if err != nil {
			return fmt.Errorf("creating field %q: %w", f.Name, err)
		}

		if resp.StatusCode() != http.StatusCreated {
			return fmt.Errorf("creating field %q: unexpected status %d", f.Name, resp.StatusCode())
		}
	}

	return nil
}

type fieldInfo struct {
	ID            string
	Name          string
	Slug          string
	Kind          string
	KindCatalogID string
}

func (c *Client) listFields(ctx context.Context, catalogID string) ([]fieldInfo, error) {
	resp, err := c.sdk.ListCatalogPropertiesWithResponse(ctx, catalogID)
	if err != nil {
		return nil, fmt.Errorf("listing fields: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("listing fields: unexpected status %d", resp.StatusCode())
	}

	// ListCatalogPropertiesResponse has no typed body; parse manually.
	var parsed rootly.CatalogPropertyList
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding fields response: %w", err)
	}

	var fields []fieldInfo
	for _, r := range parsed.Data {
		slug := ""
		if r.Attributes.Slug != nil {
			slug = *r.Attributes.Slug
		}
		kindCatalogID := ""
		if r.Attributes.KindCatalogID.IsSpecified() && !r.Attributes.KindCatalogID.IsNull() {
			kindCatalogID = r.Attributes.KindCatalogID.MustGet()
		}
		fields = append(fields, fieldInfo{
			ID:            r.ID,
			Name:          r.Attributes.Name,
			Slug:          slug,
			Kind:          string(r.Attributes.Kind),
			KindCatalogID: kindCatalogID,
		})
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

		sdkEntities := make([]struct {
			BackstageID nullable.Nullable[string] `json:"backstage_id,omitempty"`
			Description nullable.Nullable[string] `json:"description,omitempty"`
			ExternalID  string                    `json:"external_id"`
			Fields      []struct {
				CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
				CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
				Value             string  `json:"value"`
			} `json:"fields,omitempty"`
			Name *string `json:"name,omitempty"`
		}, len(batch))

		for j, e := range batch {
			sdkEntities[j].ExternalID = e.ExternalID
			sdkEntities[j].Name = &e.Name

			if e.BackstageID != "" {
				sdkEntities[j].BackstageID = nullable.NewNullableWithValue(e.BackstageID)
			}

			for slug, val := range e.Fields {
				s := slug // capture for pointer
				sdkEntities[j].Fields = append(sdkEntities[j].Fields, struct {
					CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
					CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
					Value             string  `json:"value"`
				}{
					CatalogFieldID: &s,
					Value:          val,
				})
			}
		}

		body := rootly.BulkUpsertCatalogEntities{
			Entities: sdkEntities,
		}

		resp, err := c.sdk.BulkUpsertCatalogEntitiesWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, catalogID, body)
		if err != nil {
			return nil, fmt.Errorf("bulk upsert: %w", err)
		}

		br, err := parseBulkUpsertResponse(resp.StatusCode(), resp.Body)
		if err != nil {
			return nil, err
		}
		result.Succeeded += br.Succeeded
		result.Errors = append(result.Errors, br.Errors...)
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

		var body rootly.BulkDestroyCatalogEntities
		if err := body.FromBulkDestroyCatalogEntities0(rootly.BulkDestroyCatalogEntities0{
			ExternalIDs: batch,
		}); err != nil {
			return nil, fmt.Errorf("building bulk delete body: %w", err)
		}

		resp, err := c.sdk.BulkDeleteCatalogEntitiesWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, catalogID, body)
		if err != nil {
			return nil, fmt.Errorf("bulk delete: %w", err)
		}

		if resp.ApplicationVndAPIJSON200 != nil && resp.ApplicationVndAPIJSON200.Data != nil {
			result.DeletedExternalIDs = append(result.DeletedExternalIDs, resp.ApplicationVndAPIJSON200.Data.DeletedExternalIDs...)
			result.NotFoundExternalIDs = append(result.NotFoundExternalIDs, resp.ApplicationVndAPIJSON200.Data.NotFoundExternalIDs...)
		} else {
			br, err := parseBulkDeleteResponse(resp.Body)
			if err != nil {
				return nil, err
			}
			result.DeletedExternalIDs = append(result.DeletedExternalIDs, br.DeletedExternalIDs...)
			result.NotFoundExternalIDs = append(result.NotFoundExternalIDs, br.NotFoundExternalIDs...)
		}
	}

	return result, nil
}
