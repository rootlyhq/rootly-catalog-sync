package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if e["backstage_id"] != "Component:default/service-a" {
		t.Errorf("expected backstage_id=Component:default/service-a, got %v", e["backstage_id"])
	}

	e2 := entries[1]
	if e2["backstage_id"] != "Component:default/service-b" {
		t.Errorf("expected backstage_id=Component:default/service-b, got %v", e2["backstage_id"])
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
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
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
		URL: server.URL,
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
