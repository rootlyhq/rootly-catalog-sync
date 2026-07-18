package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

func TestListNativeResources_Services(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/services", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":   "svc-1",
						"type": "services",
						"attributes": map[string]any{
							"name":        "Auth Service",
							"external_id": "ext-auth",
							"description": "Handles authentication",
							"color":       "#ff0000",
							"created_at":  "2024-01-01T00:00:00Z",
							"updated_at":  "2024-01-01T00:00:00Z",
						},
					},
					{
						"id":   "svc-2",
						"type": "services",
						"attributes": map[string]any{
							"name":         "API Gateway",
							"external_id":  "ext-api",
							"description":  "Routes requests",
							"pagerduty_id": "PD123",
							"created_at":   "2024-01-01T00:00:00Z",
							"updated_at":   "2024-01-01T00:00:00Z",
						},
					},
				},
				"links": map[string]any{"first": "", "self": ""},
				"meta":  map[string]any{"total_pages": 2, "current_page": 1, "total_count": 3},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":   "svc-3",
					"type": "services",
					"attributes": map[string]any{
						"name":        "Billing",
						"external_id": "ext-billing",
						"description": "",
						"created_at":  "2024-01-01T00:00:00Z",
						"updated_at":  "2024-01-01T00:00:00Z",
					},
				},
			},
			"links": map[string]any{"first": "", "self": ""},
			"meta":  map[string]any{"total_pages": 2, "current_page": 2, "total_count": 3},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))
	entities, err := c.ListNativeResources(context.Background(), "service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entities) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(entities))
	}
	if entities[0].Name != "Auth Service" {
		t.Errorf("expected Auth Service, got %s", entities[0].Name)
	}
	if entities[0].ExternalID != "ext-auth" {
		t.Errorf("expected ext-auth, got %s", entities[0].ExternalID)
	}
	if entities[0].Description != "Handles authentication" {
		t.Errorf("expected description, got %s", entities[0].Description)
	}
	if entities[0].Fields["color"] != "#ff0000" {
		t.Errorf("expected color=#ff0000, got %s", entities[0].Fields["color"])
	}
	if entities[1].Fields["pagerduty_id"] != "PD123" {
		t.Errorf("expected pagerduty_id=PD123, got %s", entities[1].Fields["pagerduty_id"])
	}
	if entities[2].ExternalID != "ext-billing" {
		t.Errorf("expected ext-billing, got %s", entities[2].ExternalID)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
}

func TestBulkUpsertNative_Services(t *testing.T) {
	var requestBodies []map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/services/bulk_upsert", func(w http.ResponseWriter, r *http.Request) {
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
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))

	ents := []catalog.DesiredEntity{
		{
			ExternalID: "ext-svc-1",
			Name:       "Service One",
			Fields: map[string]string{
				"description":  "A test service",
				"color":        "#00ff00",
				"pagerduty_id": "PD456",
				"custom_field": "custom_value",
			},
		},
		{
			ExternalID:  "ext-svc-2",
			Name:        "Service Two",
			BackstageID: "ns:Component:svc-two",
			Fields: map[string]string{
				"opsgenie_id": "OG789",
			},
		},
	}

	result, err := c.BulkUpsertNative(context.Background(), "service", ents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(requestBodies) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(requestBodies))
	}

	if result.Succeeded != 2 {
		t.Errorf("expected 2 succeeded, got %d", result.Succeeded)
	}

	entities := requestBodies[0]["entities"].([]any)
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities in batch, got %d", len(entities))
	}

	first := entities[0].(map[string]any)
	if first["external_id"] != "ext-svc-1" {
		t.Errorf("expected ext-svc-1, got %v", first["external_id"])
	}
	if first["description"] != "A test service" {
		t.Errorf("expected known attr 'description' set directly, got %v", first["description"])
	}
	if first["color"] != "#00ff00" {
		t.Errorf("expected known attr 'color' set directly, got %v", first["color"])
	}
	if first["pagerduty_id"] != "PD456" {
		t.Errorf("expected known attr 'pagerduty_id' set directly, got %v", first["pagerduty_id"])
	}

	// custom_field should be in the fields array, not as a top-level attr
	fieldsArr, ok := first["fields"].([]any)
	if !ok || len(fieldsArr) == 0 {
		t.Fatal("expected fields array with custom_field")
	}
	found := false
	for _, f := range fieldsArr {
		fm := f.(map[string]any)
		if fm["catalog_field_id"] == "custom_field" && fm["value"] == "custom_value" {
			found = true
		}
	}
	if !found {
		t.Error("expected custom_field in fields array")
	}

	// Second entity should have backstage_id set
	second := entities[1].(map[string]any)
	if second["backstage_id"] != "ns:Component:svc-two" {
		t.Errorf("expected backstage_id, got %v", second["backstage_id"])
	}
}

func TestBulkDeleteNative_Services(t *testing.T) {
	var requestBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/services/bulk_delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&requestBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"deleted_external_ids":   []string{"ext-1", "ext-2"},
			"not_found_external_ids": []string{"ext-3"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New("test-key", WithBaseURL(srv.URL), WithMaxRetries(0))
	result, err := c.BulkDeleteNative(context.Background(), "service", []string{"ext-1", "ext-2", "ext-3"})
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

func TestIsNativeResource(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"service", true},
		{"functionality", true},
		{"environment", true},
		{"team", true},
		{"catalog", false},
		{"", false},
		{"custom_thing", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsNativeResource(tt.input)
			if got != tt.want {
				t.Errorf("IsNativeResource(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
