package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

func TestHTTPSource_SimpleGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"services": []map[string]any{
					{"id": "svc-1", "name": "Service A"},
					{"id": "svc-2", "name": "Service B"},
				},
			},
		})
	}))
	defer srv.Close()

	src := NewHTTPSource(&config.HTTPSourceConfig{
		URL:    srv.URL,
		Result: "data.services",
	})
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

func TestHTTPSource_POST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{{"id": "1"}},
		})
	}))
	defer srv.Close()

	src := NewHTTPSource(&config.HTTPSourceConfig{
		URL:    srv.URL,
		Method: "POST",
		Body:   `{"query": "services"}`,
		Result: "items",
	})
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestHTTPSource_NoResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "svc-1", "name": "Direct"},
		})
	}))
	defer srv.Close()

	src := NewHTTPSource(&config.HTTPSourceConfig{
		URL: srv.URL,
	})
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["name"] != "Direct" {
		t.Errorf("expected Direct, got %v", entries[0]["name"])
	}
}

func TestHTTPSource_CustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "secret" {
			t.Errorf("expected X-API-Key header")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{{"id": "1"}},
		})
	}))
	defer srv.Close()

	src := NewHTTPSource(&config.HTTPSourceConfig{
		URL:     srv.URL,
		Headers: map[string]string{"X-API-Key": "secret"},
		Result:  "results",
	})
	_, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPSource_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	src := NewHTTPSource(&config.HTTPSourceConfig{
		URL:    srv.URL,
		Result: "data",
	})
	_, err := src.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestHTTPSource_NestedResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"body": map[string]any{
					"items": []map[string]any{
						{"id": "deep-1"},
						{"id": "deep-2"},
						{"id": "deep-3"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	src := NewHTTPSource(&config.HTTPSourceConfig{
		URL:    srv.URL,
		Result: "response.body.items",
	})
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}
