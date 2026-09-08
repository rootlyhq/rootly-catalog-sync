package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

func TestBackstageSource_Load(t *testing.T) {
	items := []map[string]any{
		{
			"kind": "Component",
			"metadata": map[string]any{
				"name":        "service-a",
				"namespace":   "default",
				"description": "Service A",
				"annotations": map[string]string{"backstage.io/techdocs-ref": "dir:."},
				"labels":      map[string]string{"team": "platform"},
			},
			"spec": map[string]any{
				"type":      "service",
				"lifecycle": "production",
				"owner":     "team-platform",
			},
		},
		{
			"kind": "Component",
			"metadata": map[string]any{
				"name":      "service-b",
				"namespace": "default",
			},
			"spec": map[string]any{
				"type":  "library",
				"owner": "team-core",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/catalog/entities/by-query" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept: application/json, got %s", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	}))
	defer server.Close()

	src := NewBackstageSource(&config.BackstageSourceConfig{
		URL:   server.URL,
		Token: "test-token",
	})

	if src.Name() != "backstage" {
		t.Errorf("expected name=backstage, got %s", src.Name())
	}

	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	e := entries[0]
	if e["kind"] != "Component" {
		t.Errorf("expected kind=Component, got %v", e["kind"])
	}
	if e["name"] != "service-a" {
		t.Errorf("expected name=service-a, got %v", e["name"])
	}
	if e["namespace"] != "default" {
		t.Errorf("expected namespace=default, got %v", e["namespace"])
	}
	if e["description"] != "Service A" {
		t.Errorf("expected description=Service A, got %v", e["description"])
	}
	if e["type"] != "service" {
		t.Errorf("expected type=service, got %v", e["type"])
	}
	if e["lifecycle"] != "production" {
		t.Errorf("expected lifecycle=production, got %v", e["lifecycle"])
	}
	if e["owner"] != "team-platform" {
		t.Errorf("expected owner=team-platform, got %v", e["owner"])
	}
	if e["backstage_id"] != "component:default/service-a" {
		t.Errorf("expected backstage_id=component:default/service-a, got %v", e["backstage_id"])
	}

	e2 := entries[1]
	if e2["backstage_id"] != "component:default/service-b" {
		t.Errorf("expected backstage_id=component:default/service-b, got %v", e2["backstage_id"])
	}

	annotations, ok := e["annotations"].(map[string]string)
	if !ok {
		t.Fatalf("expected annotations to be map[string]string")
	}
	if annotations["backstage.io/techdocs-ref"] != "dir:." {
		t.Errorf("unexpected annotation value: %v", annotations["backstage.io/techdocs-ref"])
	}

	labels, ok := e["labels"].(map[string]string)
	if !ok {
		t.Fatalf("expected labels to be map[string]string")
	}
	if labels["team"] != "platform" {
		t.Errorf("unexpected label value: %v", labels["team"])
	}

	e2 = entries[1]
	if e2["name"] != "service-b" {
		t.Errorf("expected name=service-b, got %v", e2["name"])
	}
	if e2["type"] != "library" {
		t.Errorf("expected type=library, got %v", e2["type"])
	}
	if _, exists := e2["description"]; exists {
		t.Error("expected no description for service-b")
	}
}

func TestFlattenEntityCanonicalBackstageID(t *testing.T) {
	for _, tc := range []struct {
		name      string
		raw       string
		wantID    string
		namespace string
	}{
		{
			name:      "mixed case",
			raw:       `{"kind":"Component","metadata":{"name":"Payments-API","namespace":"Production"}}`,
			wantID:    "component:production/payments-api",
			namespace: "Production",
		},
		{
			name:      "default namespace",
			raw:       `{"kind":"Component","metadata":{"name":"Payments-API"}}`,
			wantID:    "component:default/payments-api",
			namespace: "default",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := flattenEntity(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			if entry["backstage_id"] != tc.wantID {
				t.Errorf("backstage_id = %v, want %s", entry["backstage_id"], tc.wantID)
			}
			if entry["kind"] != "Component" || entry["name"] != "Payments-API" || entry["namespace"] != tc.namespace {
				t.Errorf("source fields should preserve their original case: %v", entry)
			}
		})
	}
}

func TestBackstageSource_WithFilter(t *testing.T) {
	var receivedFilter string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedFilter = r.URL.Query().Get("filter")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer server.Close()

	src := NewBackstageSource(&config.BackstageSourceConfig{
		URL:    server.URL,
		Filter: "kind=component,metadata.namespace=default",
	})

	_, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedFilter != "kind=component,metadata.namespace=default" {
		t.Errorf("expected filter=kind=component,metadata.namespace=default, got %s", receivedFilter)
	}
}

func TestBackstageSource_WithKind(t *testing.T) {
	var receivedFilter string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedFilter = r.URL.Query().Get("filter")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer server.Close()

	src := NewBackstageSource(&config.BackstageSourceConfig{
		URL:  server.URL,
		Kind: "API",
	})

	_, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedFilter != "kind=API" {
		t.Errorf("expected filter=kind=API, got %s", receivedFilter)
	}
}

func TestBackstageSource_Pagination(t *testing.T) {
	filters := []string{"kind=component,spec.type=service", "kind=component,spec.type=website"}
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if got := r.URL.Query()["filter"]; !slices.Equal(got, filters) {
			t.Errorf("filters = %v, want %v on every page", got, filters)
		}
		offset := r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")

		if offset == "0" || offset == "" {
			items := make([]map[string]any, 500)
			for i := range items {
				items[i] = map[string]any{
					"kind":     "Component",
					"metadata": map[string]any{"name": "svc-page1", "namespace": "default"},
					"spec":     map[string]any{},
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
		} else {
			items := []map[string]any{
				{
					"kind":     "Component",
					"metadata": map[string]any{"name": "svc-page2", "namespace": "default"},
					"spec":     map[string]any{},
				},
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
		}
	}))
	defer server.Close()

	src := NewBackstageSource(&config.BackstageSourceConfig{
		URL:     server.URL,
		Filters: filters,
	})

	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
	if len(entries) != 501 {
		t.Fatalf("expected 501 entries, got %d", len(entries))
	}
	if entries[0]["name"] != "svc-page1" {
		t.Errorf("expected first entry name=svc-page1, got %v", entries[0]["name"])
	}
	if entries[500]["name"] != "svc-page2" {
		t.Errorf("expected last entry name=svc-page2, got %v", entries[500]["name"])
	}
}

func TestBackstageSource_FilterSelection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cfg     config.BackstageSourceConfig
		want    []string
		wantErr string
	}{
		{name: "unfiltered"},
		{name: "kind fallback", cfg: config.BackstageSourceConfig{Kind: "Component"}, want: []string{"kind=Component"}},
		{name: "legacy filter overrides kind", cfg: config.BackstageSourceConfig{Kind: "Component", Filter: "kind=api"}, want: []string{"kind=api"}},
		{name: "OR filters override kind", cfg: config.BackstageSourceConfig{Kind: "API", Filters: []string{"kind=component,spec.type=service", "kind=component,spec.type=website"}}, want: []string{"kind=component,spec.type=service", "kind=component,spec.type=website"}},
		{name: "empty list keeps legacy filter", cfg: config.BackstageSourceConfig{Filters: []string{}, Filter: "kind=api"}, want: []string{"kind=api"}},
		{name: "ambiguous filters", cfg: config.BackstageSourceConfig{Filter: "kind=api", Filters: []string{"kind=component"}}, wantErr: "filter and filters cannot be used together"},
		{name: "empty filter set", cfg: config.BackstageSourceConfig{Filters: []string{"kind=component", ""}}, wantErr: "filters[1] must not be empty"},
		{name: "whitespace filter set", cfg: config.BackstageSourceConfig{Filters: []string{" \t"}}, wantErr: "filters[0] must not be empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if got := r.URL.Query()["filter"]; !slices.Equal(got, tc.want) {
					t.Errorf("filter query = %v, want %v", got, tc.want)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
			}))
			defer server.Close()
			tc.cfg.URL = server.URL
			_, err := NewBackstageSource(&tc.cfg).Load(context.Background())
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				if called {
					t.Fatal("invalid filters must fail before fetching entities")
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
