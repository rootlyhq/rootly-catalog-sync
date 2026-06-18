package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

func TestGraphQLSource_Simple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"services": []any{
					map[string]any{"id": "svc-1", "name": "Auth"},
					map[string]any{"id": "svc-2", "name": "Billing"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	src := NewGraphQLSource(&config.GraphQLSourceConfig{
		URL:    server.URL,
		Query:  "{ services { id name } }",
		Result: "data.services",
	})

	if src.Name() != "graphql" {
		t.Errorf("expected name=graphql, got %s", src.Name())
	}

	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0]["id"] != "svc-1" {
		t.Errorf("expected id=svc-1, got %v", entries[0]["id"])
	}
	if entries[1]["name"] != "Billing" {
		t.Errorf("expected name=Billing, got %v", entries[1]["name"])
	}
}

func TestGraphQLSource_OffsetPagination(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)

		vars, _ := req["variables"].(map[string]any)
		offset := int(vars["offset"].(float64))

		call := callCount.Add(1)

		var items []any
		if call == 1 {
			if offset != 0 {
				t.Errorf("first call: expected offset=0, got %d", offset)
			}
			items = []any{
				map[string]any{"id": "a"},
				map[string]any{"id": "b"},
			}
		} else {
			if offset != 2 {
				t.Errorf("second call: expected offset=2, got %d", offset)
			}
			items = []any{
				map[string]any{"id": "c"},
			}
		}

		resp := map[string]any{"data": map[string]any{"nodes": items}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	src := NewGraphQLSource(&config.GraphQLSourceConfig{
		URL:    server.URL,
		Query:  "query($offset: Int, $limit: Int) { nodes(offset: $offset, limit: $limit) { id } }",
		Result: "data.nodes",
		Paginate: &config.PaginateConfig{
			Mode:     "offset",
			PageSize: 2,
		},
	})

	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[2]["id"] != "c" {
		t.Errorf("expected id=c, got %v", entries[2]["id"])
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 requests, got %d", callCount.Load())
	}
}

func TestGraphQLSource_CursorPagination(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)

		vars, _ := req["variables"].(map[string]any)
		call := callCount.Add(1)

		var resp map[string]any
		if call == 1 {
			if _, hasCursor := vars["cursor"]; hasCursor {
				t.Error("first call should not have cursor variable")
			}
			resp = map[string]any{
				"data": map[string]any{
					"items":      []any{map[string]any{"id": "x"}, map[string]any{"id": "y"}},
					"nextCursor": "abc123",
				},
			}
		} else {
			cursor, _ := vars["cursor"].(string)
			if cursor != "abc123" {
				t.Errorf("expected cursor=abc123, got %s", cursor)
			}
			resp = map[string]any{
				"data": map[string]any{
					"items":      []any{map[string]any{"id": "z"}},
					"nextCursor": "",
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	src := NewGraphQLSource(&config.GraphQLSourceConfig{
		URL:    server.URL,
		Query:  "query($cursor: String, $limit: Int) { items(cursor: $cursor, limit: $limit) { id } }",
		Result: "data.items",
		Paginate: &config.PaginateConfig{
			Mode:       "cursor",
			PageSize:   2,
			CursorPath: "data.nextCursor",
		},
	})

	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0]["id"] != "x" {
		t.Errorf("expected id=x, got %v", entries[0]["id"])
	}
	if entries[2]["id"] != "z" {
		t.Errorf("expected id=z, got %v", entries[2]["id"])
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 requests, got %d", callCount.Load())
	}
}

func TestGraphQLSource_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer my-token" {
			t.Errorf("expected Authorization header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("expected X-Custom header, got %q", r.Header.Get("X-Custom"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type=application/json, got %q", r.Header.Get("Content-Type"))
		}

		resp := map[string]any{"data": map[string]any{"items": []any{}}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	src := NewGraphQLSource(&config.GraphQLSourceConfig{
		URL:   server.URL,
		Query: "{ items { id } }",
		Headers: map[string]string{
			"Authorization": "Bearer my-token",
			"X-Custom":      "value",
		},
		Result: "data.items",
	})

	_, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
