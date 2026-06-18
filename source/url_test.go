package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

func TestURLSource_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "svc-1", "name": "Service A"},
			{"id": "svc-2", "name": "Service B"},
		})
	}))
	defer srv.Close()

	src := NewURLSource(&config.URLSourceConfig{URLs: []string{srv.URL}})
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0]["name"] != "Service A" {
		t.Errorf("expected Service A, got %v", entries[0]["name"])
	}
}

func TestURLSource_YAML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/yaml")
		_, _ = w.Write([]byte("- id: svc-1\n  name: YAML Service\n"))
	}))
	defer srv.Close()

	src := NewURLSource(&config.URLSourceConfig{URLs: []string{srv.URL}})
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["name"] != "YAML Service" {
		t.Errorf("expected YAML Service, got %v", entries[0]["name"])
	}
}

func TestURLSource_MultipleURLs(t *testing.T) {
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "a"}})
	}))
	defer srv1.Close()
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "b"}})
	}))
	defer srv2.Close()

	src := NewURLSource(&config.URLSourceConfig{URLs: []string{srv1.URL, srv2.URL}})
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries from 2 URLs, got %d", len(entries))
	}
}

func TestURLSource_CustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer my-token" {
			t.Errorf("expected auth header, got %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "1"}})
	}))
	defer srv.Close()

	src := NewURLSource(&config.URLSourceConfig{
		URLs:    []string{srv.URL},
		Headers: map[string]string{"Authorization": "Bearer my-token"},
	})
	_, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestURLSource_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	src := NewURLSource(&config.URLSourceConfig{URLs: []string{srv.URL}})
	_, err := src.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
