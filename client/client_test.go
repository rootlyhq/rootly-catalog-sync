package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

func TestListEntities(t *testing.T) {
	entityCallCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/catalogs/cat-1/fields", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "f1", "type": "catalog_fields", "attributes": map[string]any{"name": "owner", "slug": "owner"}},
			},
			"meta": map[string]any{"total_pages": 1, "current_page": 1},
		})
	})
	mux.HandleFunc("/v1/catalogs/cat-1/entities", func(w http.ResponseWriter, r *http.Request) {
		entityCallCount++
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if entityCallCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":   "1",
						"type": "catalog_entities",
						"attributes": map[string]any{
							"name":        "Service A",
							"external_id": "ext-a",
							"description": "First",
							"managed_by":  "api",
							"properties": []map[string]any{
								{"catalog_property_id": "f1", "value": "team-a"},
							},
						},
					},
					{
						"id":   "2",
						"type": "catalog_entities",
						"attributes": map[string]any{
							"name":        "Service B",
							"external_id": "ext-b",
							"description": "Second",
							"managed_by":  "api",
							"properties":  []map[string]any{},
						},
					},
				},
				"meta": map[string]any{"next_cursor": "cursor-page2"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":   "3",
					"type": "catalog_entities",
					"attributes": map[string]any{
						"name":        "Service C",
						"external_id": "ext-c",
						"description": "Third",
						"managed_by":  "api",
						"properties":  []map[string]any{},
					},
				},
			},
			"meta": map[string]any{"next_cursor": ""},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))
	entities, err := c.ListEntities(context.Background(), "cat-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entities) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(entities))
	}
	if entities[0].Name != "Service A" {
		t.Errorf("expected Service A, got %s", entities[0].Name)
	}
	if entities[0].Fields["owner"] != "team-a" {
		t.Errorf("expected owner=team-a, got %s", entities[0].Fields["owner"])
	}
	if entities[2].ExternalID != "ext-c" {
		t.Errorf("expected ext-c, got %s", entities[2].ExternalID)
	}
	if entityCallCount != 2 {
		t.Errorf("expected 2 entity API calls for pagination, got %d", entityCallCount)
	}
}

func TestBulkUpsert(t *testing.T) {
	var requestBodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		requestBodies = append(requestBodies, body)

		entities := body["entities"].([]any)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"succeeded": len(entities),
			"errors":    []any{},
		})
	}))
	defer srv.Close()

	c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))

	ents := make([]catalog.DesiredEntity, 150)
	for i := range ents {
		ents[i] = catalog.DesiredEntity{
			ExternalID: fmt.Sprintf("ext-%d", i),
			Name:       fmt.Sprintf("Entity %d", i),
			Fields:     map[string]string{"owner": "team-x"},
		}
	}

	result, err := c.BulkUpsert(context.Background(), "cat-1", ents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(requestBodies) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(requestBodies))
	}

	batch1 := requestBodies[0]["entities"].([]any)
	batch2 := requestBodies[1]["entities"].([]any)
	if len(batch1) != 100 {
		t.Errorf("expected batch 1 to have 100 entities, got %d", len(batch1))
	}
	if len(batch2) != 50 {
		t.Errorf("expected batch 2 to have 50 entities, got %d", len(batch2))
	}

	if result.Succeeded != 150 {
		t.Errorf("expected 150 succeeded, got %d", result.Succeeded)
	}

	first := batch1[0].(map[string]any)
	if first["external_id"] != "ext-0" {
		t.Errorf("expected ext-0, got %v", first["external_id"])
	}
	if first["fields"] == nil {
		t.Error("expected fields to be present")
	}
}

func TestBulkDelete(t *testing.T) {
	var requestBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&requestBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"deleted_external_ids":   []string{"ext-1", "ext-2"},
			"not_found_external_ids": []string{"ext-3"},
		})
	}))
	defer srv.Close()

	c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))
	result, err := c.BulkDelete(context.Background(), "cat-1", []string{"ext-1", "ext-2", "ext-3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ids := requestBody["external_ids"].([]any)
	if len(ids) != 3 {
		t.Errorf("expected 3 external_ids in request, got %d", len(ids))
	}

	if len(result.DeletedExternalIDs) != 2 {
		t.Errorf("expected 2 deleted, got %d", len(result.DeletedExternalIDs))
	}
	if len(result.NotFoundExternalIDs) != 1 {
		t.Errorf("expected 1 not found, got %d", len(result.NotFoundExternalIDs))
	}
}

func TestRetryOn429(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": "rate limited"}`))
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{},
			"meta": map[string]any{"total_pages": 1, "current_page": 1},
		})
	}))
	defer srv.Close()

	c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(3))
	_, err := c.ListCatalogs(context.Background())
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts (2 retries + 1 success), got %d", attempts)
	}
}

func TestNewRequestURLConstruction(t *testing.T) {
	// Regression: url.JoinPath was encoding ? as %3F in query strings.
	// The fix uses string concatenation instead.
	c := New("test-key", WithBaseURL("https://example.com"), WithMaxRetries(0))
	req, err := c.newRequest(context.Background(), http.MethodGet, "/catalogs?page[number]=1&page[size]=250", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "https://example.com/v1/catalogs?page[number]=1&page[size]=250"
	if req.URL.String() != expected {
		t.Errorf("expected URL %q, got %q", expected, req.URL.String())
	}

	// Verify the ? is not percent-encoded
	if strings.Contains(req.URL.String(), "%3F") {
		t.Error("URL contains encoded '?' (%3F) — query params are broken")
	}
}

func TestListEntities_ResolvesPropertyIDs(t *testing.T) {
	// Regression: the API returns entity properties with catalog_property_id (UUID),
	// not field names. ListEntities must fetch fields first and map IDs to names.
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/catalogs/cat-1/fields", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "field-uuid-1", "type": "catalog_fields", "attributes": map[string]any{"name": "owner", "slug": "owner"}},
			},
			"meta": map[string]any{"total_pages": 1, "current_page": 1},
		})
	})
	mux.HandleFunc("/v1/catalogs/cat-1/entities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":   "ent-1",
					"type": "catalog_entities",
					"attributes": map[string]any{
						"name":        "My Service",
						"external_id": "ext-1",
						"description": "",
						"managed_by":  "api",
						"properties": []map[string]any{
							{"catalog_property_id": "field-uuid-1", "value": "team-x"},
						},
					},
				},
			},
			"meta": map[string]any{"next_cursor": ""},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))
	entities, err := c.ListEntities(context.Background(), "cat-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if entities[0].Fields["owner"] != "team-x" {
		t.Errorf("expected Fields[\"owner\"]=\"team-x\", got %q", entities[0].Fields["owner"])
	}
	// Ensure the UUID key is NOT present — only the resolved name should be.
	if _, ok := entities[0].Fields["field-uuid-1"]; ok {
		t.Error("Fields should use resolved name 'owner', not raw UUID 'field-uuid-1'")
	}
}

func TestWithAPIPath(t *testing.T) {
	var requestedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{},
			"meta": map[string]any{"total_pages": 1, "current_page": 1},
		})
	}))
	defer srv.Close()

	c := New("test-key", WithBaseURL(srv.URL), WithAPIPath("/api/v1"), WithMaxRetries(0))
	_, err := c.ListCatalogs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(requestedPath, "/api/v1/catalogs") {
		t.Errorf("expected path to start with /api/v1/catalogs, got %q", requestedPath)
	}
}

func TestEnsureCatalog(t *testing.T) {
	t.Run("existing catalog", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":   "existing-id",
						"type": "catalogs",
						"attributes": map[string]any{
							"name": "My Catalog",
							"slug": "my-catalog",
						},
					},
				},
				"meta": map[string]any{
					"total_pages":  1,
					"current_page": 1,
				},
			})
		}))
		defer srv.Close()

		c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))
		id, err := c.EnsureCatalog(context.Background(), CatalogSpec{Name: "My Catalog"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "existing-id" {
			t.Errorf("expected existing-id, got %s", id)
		}
	})

	t.Run("create new catalog", func(t *testing.T) {
		callCount := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/vnd.api+json")

			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []any{},
					"meta": map[string]any{
						"total_pages":  1,
						"current_page": 1,
					},
				})
				return
			}

			if r.Method == http.MethodPost {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)

				data := body["data"].(map[string]any)
				if data["type"] != "catalogs" {
					t.Errorf("expected type=catalogs, got %v", data["type"])
				}

				attrs := data["attributes"].(map[string]any)
				if attrs["name"] != "New Catalog" {
					t.Errorf("expected name=New Catalog, got %v", attrs["name"])
				}

				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"id":   "new-id",
						"type": "catalogs",
						"attributes": map[string]any{
							"name": "New Catalog",
							"slug": "new-catalog",
						},
					},
				})
				return
			}
		}))
		defer srv.Close()

		c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))
		id, err := c.EnsureCatalog(context.Background(), CatalogSpec{
			Name:        "New Catalog",
			Description: "A new catalog",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "new-id" {
			t.Errorf("expected new-id, got %s", id)
		}
	})
}
