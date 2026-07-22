package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
	"github.com/rootlyhq/rootly-catalog-sync/client"
	"github.com/rootlyhq/rootly-catalog-sync/config"
	"github.com/rootlyhq/rootly-catalog-sync/mapping"
)

func jsonAPI(w http.ResponseWriter, data any) {
	jsonAPIStatus(w, 200, data)
}

func jsonAPIStatus(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func emptyList() map[string]any {
	return map[string]any{
		"data": []any{}, "links": map[string]any{"self": ""},
		"meta": map[string]any{"total_pages": 1, "current_page": 1, "total_count": 0},
	}
}

func testServer(t *testing.T, props []map[string]any, catalogs []map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/services/properties", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			jsonAPIStatus(w, 201, map[string]any{
				"data": map[string]any{"id": "new-prop", "type": "catalog_properties", "attributes": map[string]any{}},
			})
			return
		}
		data := make([]map[string]any, len(props))
		for i, p := range props {
			data[i] = map[string]any{"id": p["id"], "type": "catalog_properties", "attributes": p}
		}
		jsonAPI(w, map[string]any{
			"data": data, "links": map[string]any{"self": ""},
			"meta": map[string]any{"total_pages": 1, "current_page": 1, "total_count": len(data)},
		})
	})

	mux.HandleFunc("/v1/catalogs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			jsonAPIStatus(w, 201, map[string]any{
				"data": map[string]any{"id": "cat-new", "type": "catalogs", "attributes": map[string]any{"name": "new"}},
			})
			return
		}
		data := make([]map[string]any, len(catalogs))
		for i, c := range catalogs {
			data[i] = map[string]any{"id": c["id"], "type": "catalogs", "attributes": c}
		}
		jsonAPI(w, map[string]any{
			"data": data, "links": map[string]any{"self": ""},
			"meta": map[string]any{"total_pages": 1, "current_page": 1, "total_count": len(data)},
		})
	})

	return httptest.NewServer(mux)
}

func TestEnsureNativeOutputFields_KindMismatch(t *testing.T) {
	srv := testServer(t, []map[string]any{
		{"id": "p1", "name": "team-tier", "slug": "team-tier", "kind": "reference", "kind_catalog_id": "cat-1", "multiple": false, "position": 1, "created_at": "2024-01-01", "updated_at": "2024-01-01"},
	}, nil)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Type: "service",
		Fields: map[string]config.FieldValue{
			"team-tier": {Value: "{{ .tier }}", Kind: "text"},
		},
	}

	_, err := ensureNativeOutputFields(context.Background(), cl, out, true)
	if err == nil {
		t.Fatal("expected error for kind mismatch")
	}
	if !strings.Contains(err.Error(), "kind") || !strings.Contains(err.Error(), "reference") {
		t.Errorf("expected kind mismatch error, got: %s", err)
	}
}

func TestEnsureNativeOutputFields_CatalogMismatch(t *testing.T) {
	srv := testServer(t, []map[string]any{
		{"id": "p1", "name": "team-tier", "slug": "team-tier", "kind": "reference", "kind_catalog_id": "cat-1", "multiple": false, "position": 1, "created_at": "2024-01-01", "updated_at": "2024-01-01"},
	}, []map[string]any{
		{"id": "cat-99", "name": "Wrong Tiers", "created_at": "2024-01-01", "updated_at": "2024-01-01"},
		{"id": "cat-2", "name": "Tiers", "created_at": "2024-01-01", "updated_at": "2024-01-01"},
	})
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Type: "service",
		Fields: map[string]config.FieldValue{
			"team-tier": {Value: "{{ .tier }}", Kind: "reference", Catalog: "Tiers"},
		},
	}

	_, err := ensureNativeOutputFields(context.Background(), cl, out, true)
	if err == nil {
		t.Fatal("expected error for catalog mismatch")
	}
	if !strings.Contains(err.Error(), "references catalog") {
		t.Errorf("expected catalog mismatch error, got: %s", err)
	}
}

func TestEnsureNativeOutputFields_CatalogNotFound(t *testing.T) {
	srv := testServer(t, []map[string]any{
		{"id": "p1", "name": "team-tier", "slug": "team-tier", "kind": "reference", "kind_catalog_id": "cat-1", "multiple": false, "position": 1, "created_at": "2024-01-01", "updated_at": "2024-01-01"},
	}, []map[string]any{})
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Type: "service",
		Fields: map[string]config.FieldValue{
			"team-tier": {Value: "{{ .tier }}", Kind: "reference", Catalog: "Nonexistent"},
		},
	}

	_, err := ensureNativeOutputFields(context.Background(), cl, out, true)
	if err == nil {
		t.Fatal("expected error for missing catalog")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected catalog not found error, got: %s", err)
	}
}

func TestEnsureNativeOutputFields_AutoCreateText(t *testing.T) {
	var createCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/services/properties", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createCalled = true
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"id": "new-prop", "type": "catalog_properties", "attributes": map[string]any{}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":  []any{},
			"links": map[string]any{"self": ""},
			"meta":  map[string]any{"total_pages": 1, "current_page": 1, "total_count": 0},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Type: "service",
		Fields: map[string]config.FieldValue{
			"changelog-url": {Value: "{{ .url }}"},
		},
	}

	props, err := ensureNativeOutputFields(context.Background(), cl, out, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createCalled {
		t.Error("expected property creation API call")
	}
	found := false
	for _, p := range props {
		if p.Slug == "changelog-url" {
			found = true
		}
	}
	if !found {
		t.Error("expected changelog-url in returned props")
	}
}

func TestEnsureNativeOutputFields_RejectsNonTextMissing(t *testing.T) {
	srv := testServer(t, []map[string]any{}, nil)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Type: "service",
		Fields: map[string]config.FieldValue{
			"my-ref": {Value: "{{ .ref }}", Kind: "reference", Catalog: "Things"},
		},
	}

	_, err := ensureNativeOutputFields(context.Background(), cl, out, true)
	if err == nil {
		t.Fatal("expected error for missing non-text property")
	}
	if !strings.Contains(err.Error(), "must be created in the Rootly UI") {
		t.Errorf("expected UI creation hint, got: %s", err)
	}
}

func TestWithApplyEach_PipelineOrdering(t *testing.T) {
	var applyOrder []string

	applyFn := func(ctx context.Context, r PlanResult) error {
		name := r.Output.Catalog
		if name == "" {
			name = r.Output.Type
		}
		applyOrder = append(applyOrder, fmt.Sprintf("%s:%d", name, r.Plan.Counts.Create))
		return nil
	}

	cfg := &config.Config{
		Version: 1,
		SyncID:  "test-ordering",
		Pipelines: []config.Pipeline{
			{
				Sources: []config.SourceConfig{
					{Inline: &config.InlineSourceConfig{Entries: []map[string]any{
						{"id": "t1", "name": "Tier 1"},
					}}},
				},
				Outputs: []config.Output{
					{Catalog: "Tiers", ExternalID: "{{ .id }}", Name: "{{ .name }}"},
				},
			},
			{
				Sources: []config.SourceConfig{
					{Inline: &config.InlineSourceConfig{Entries: []map[string]any{
						{"id": "svc1", "name": "Svc 1", "description": "test"},
					}}},
				},
				Outputs: []config.Output{
					{Type: "service", ExternalID: "{{ .id }}", Name: "{{ .name }}", Fields: map[string]config.FieldValue{
						"description": {Value: "{{ .description }}"},
					}},
				},
			},
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/catalogs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			jsonAPIStatus(w, 201, map[string]any{
				"data": map[string]any{"id": "cat-new", "type": "catalogs", "attributes": map[string]any{"name": "Tiers"}},
			})
			return
		}
		jsonAPI(w, emptyList())
	})
	mux.HandleFunc("/v1/catalogs/cat-new/properties", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	mux.HandleFunc("/v1/catalogs/cat-new/entities", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	mux.HandleFunc("/v1/services/properties", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	mux.HandleFunc("/v1/services", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))

	results, err := reconcileAll(context.Background(), cfg, cl, ".", false, 0.2, withApplyEach(applyFn), withSkipSafety())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(applyOrder) != 2 {
		t.Fatalf("expected 2 apply calls, got %d", len(applyOrder))
	}
	if applyOrder[0] != "Tiers:1" {
		t.Errorf("expected Tiers applied first, got %s", applyOrder[0])
	}
	if applyOrder[1] != "service:1" {
		t.Errorf("expected service applied second, got %s", applyOrder[1])
	}
}

func TestResolveReferenceFields_LiveEntities(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/catalogs", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, map[string]any{
			"data": []map[string]any{
				{"id": "cat-tiers", "type": "catalogs", "attributes": map[string]any{"name": "Tiers"}},
			},
			"links": map[string]any{"self": ""},
			"meta":  map[string]any{"total_pages": 1, "current_page": 1, "total_count": 1},
		})
	})
	mux.HandleFunc("/v1/catalogs/cat-tiers/properties", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	mux.HandleFunc("/v1/catalogs/cat-tiers/entities", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, map[string]any{
			"data": []map[string]any{
				{"id": "ent-t1", "type": "catalog_entities", "attributes": map[string]any{"name": "Tier 1", "external_id": "tier-1"}},
				{"id": "ent-t2", "type": "catalog_entities", "attributes": map[string]any{"name": "Tier 2", "external_id": "tier-2"}},
			},
			"links": map[string]any{"self": ""},
			"meta":  map[string]any{"total_pages": 1, "current_page": 1, "total_count": 2},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Type: "service",
		Fields: map[string]config.FieldValue{
			"tier": {Value: "{{ .tier }}", Kind: "reference", Catalog: "Tiers"},
		},
	}
	desired := []catalog.DesiredEntity{
		{ExternalID: "svc-1", Name: "Svc 1", Fields: map[string]string{"tier": "Tier 1"}},
		{ExternalID: "svc-2", Name: "Svc 2", Fields: map[string]string{"tier": "Tier 2"}},
	}
	err := resolveReferenceFields(context.Background(), cl, out, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if desired[0].Fields["tier"] != "ent-t1" {
		t.Errorf("expected ent-t1, got %s", desired[0].Fields["tier"])
	}
	if desired[1].Fields["tier"] != "ent-t2" {
		t.Errorf("expected ent-t2, got %s", desired[1].Fields["tier"])
	}
}

func TestResolveReferenceFields_ValueNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/catalogs", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, map[string]any{
			"data": []map[string]any{
				{"id": "cat-tiers", "type": "catalogs", "attributes": map[string]any{"name": "Tiers"}},
			},
			"links": map[string]any{"self": ""},
			"meta":  map[string]any{"total_pages": 1, "current_page": 1, "total_count": 1},
		})
	})
	mux.HandleFunc("/v1/catalogs/cat-tiers/properties", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	mux.HandleFunc("/v1/catalogs/cat-tiers/entities", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Type: "service",
		Fields: map[string]config.FieldValue{
			"tier": {Value: "{{ .tier }}", Kind: "reference", Catalog: "Tiers"},
		},
	}
	desired := []catalog.DesiredEntity{
		{ExternalID: "svc-1", Name: "Svc 1", Fields: map[string]string{"tier": "Nonexistent"}},
	}

	err := resolveReferenceFields(context.Background(), cl, out, desired)
	if err == nil {
		t.Fatal("expected error for missing reference value")
	}
	if !strings.Contains(err.Error(), "not found in catalog") {
		t.Errorf("expected 'not found in catalog' error, got: %s", err)
	}
}

func TestResolveReferenceFields_CatalogNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/catalogs", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Type: "service",
		Fields: map[string]config.FieldValue{
			"tier": {Value: "{{ .tier }}", Kind: "reference", Catalog: "Missing"},
		},
	}
	desired := []catalog.DesiredEntity{
		{ExternalID: "svc-1", Name: "Svc 1", Fields: map[string]string{"tier": "Tier 1"}},
	}

	err := resolveReferenceFields(context.Background(), cl, out, desired)
	if err == nil {
		t.Fatal("expected error for missing catalog")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected catalog not found error, got: %s", err)
	}
}

func TestEnsureOutputFields_ReferenceCatalogNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/catalogs", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Catalog: "Services",
		Fields: map[string]config.FieldValue{
			"tier": {Value: "{{ .tier }}", Kind: "reference", Catalog: "Nonexistent"},
		},
	}

	err := ensureOutputFields(context.Background(), cl, "cat-123", out)
	if err == nil {
		t.Fatal("expected error for missing reference catalog")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %s", err)
	}
}

func TestEnsureOutputFields_UnsupportedKindReject(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/catalogs/cat-123/properties", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Catalog: "Services",
		Fields: map[string]config.FieldValue{
			"channel": {Value: "{{ .channel }}", Kind: "slack_channel"},
		},
	}

	err := ensureOutputFields(context.Background(), cl, "cat-123", out)
	if err == nil {
		t.Fatal("expected error for non-SDK-creatable kind")
	}
	if !strings.Contains(err.Error(), "cannot be auto-created") {
		t.Errorf("expected auto-create rejection, got: %s", err)
	}
}

func TestEnsureOutputFields_ExistingKindMismatch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/catalogs/cat-123/properties", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, map[string]any{
			"data": []map[string]any{
				{"id": "f1", "type": "catalog_properties", "attributes": map[string]any{
					"name": "active", "slug": "active", "kind": "text",
					"multiple": false, "position": 1,
					"created_at": "2024-01-01", "updated_at": "2024-01-01",
				}},
			},
			"links": map[string]any{"self": ""},
			"meta":  map[string]any{"total_pages": 1, "current_page": 1, "total_count": 1},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Catalog: "Services",
		Fields: map[string]config.FieldValue{
			"active": {Value: "{{ .active }}", Kind: "boolean"},
		},
	}

	err := ensureOutputFields(context.Background(), cl, "cat-123", out)
	if err == nil {
		t.Fatal("expected error for kind mismatch on existing catalog field")
	}
	if !strings.Contains(err.Error(), "kind") {
		t.Errorf("expected kind mismatch error, got: %s", err)
	}
}

func TestEnsureNativeOutputFields_DryRunNoCreate(t *testing.T) {
	srv := testServer(t, []map[string]any{}, nil)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))
	out := config.Output{
		Type: "service",
		Fields: map[string]config.FieldValue{
			"changelog-url": {Value: "{{ .url }}"},
		},
	}

	_, err := ensureNativeOutputFields(context.Background(), cl, out, false)
	if err == nil {
		t.Fatal("expected error in dry-run mode for missing property")
	}
	if !strings.Contains(err.Error(), "run sync") {
		t.Errorf("expected 'run sync' hint, got: %s", err)
	}
}

func refCatalogServer(t *testing.T, upsertedFields *[]map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/catalogs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			jsonAPIStatus(w, 201, map[string]any{
				"data": map[string]any{"id": "cat-svc", "type": "catalogs", "attributes": map[string]any{"name": "Services"}},
			})
			return
		}
		jsonAPI(w, map[string]any{
			"data": []map[string]any{
				{"id": "cat-tiers", "type": "catalogs", "attributes": map[string]any{"name": "Tiers"}},
				{"id": "cat-svc", "type": "catalogs", "attributes": map[string]any{"name": "Services"}},
			},
			"links": map[string]any{"self": ""},
			"meta":  map[string]any{"total_pages": 1, "current_page": 1, "total_count": 2},
		})
	})
	mux.HandleFunc("/v1/catalogs/cat-svc/properties", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			jsonAPIStatus(w, 201, map[string]any{
				"data": map[string]any{"id": "f-new", "type": "catalog_properties", "attributes": map[string]any{}},
			})
			return
		}
		jsonAPI(w, emptyList())
	})
	mux.HandleFunc("/v1/catalogs/cat-tiers/properties", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, emptyList())
	})
	mux.HandleFunc("/v1/catalogs/cat-tiers/entities", func(w http.ResponseWriter, r *http.Request) {
		jsonAPI(w, map[string]any{
			"data": []map[string]any{
				{"id": "ent-t1", "type": "catalog_entities", "attributes": map[string]any{"name": "Tier 1", "external_id": "tier-1"}},
				{"id": "ent-t2", "type": "catalog_entities", "attributes": map[string]any{"name": "Tier 2", "external_id": "tier-2"}},
			},
			"links": map[string]any{"self": ""},
			"meta":  map[string]any{"total_pages": 1, "current_page": 1, "total_count": 2},
		})
	})
	mux.HandleFunc("/v1/catalogs/cat-svc/entities", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if ents, ok := body["entities"].([]any); ok {
				for _, e := range ents {
					em := e.(map[string]any)
					if fields, ok := em["fields"].([]any); ok {
						for _, f := range fields {
							*upsertedFields = append(*upsertedFields, f.(map[string]any))
						}
					}
				}
			}
			data := []map[string]any{{"id": "new-1", "type": "catalog_entities", "attributes": map[string]any{}}}
			jsonAPI(w, map[string]any{"data": data})
			return
		}
		jsonAPI(w, emptyList())
	})
	return httptest.NewServer(mux)
}

func TestImport_ResolvesReferenceFields(t *testing.T) {
	var upsertedFields []map[string]any
	srv := refCatalogServer(t, &upsertedFields)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))

	cfg := &config.Config{
		Version: 1,
		SyncID:  "import-ref-test",
		Pipelines: []config.Pipeline{{
			Sources: []config.SourceConfig{{
				Inline: &config.InlineSourceConfig{Entries: []map[string]any{
					{"name": "Svc A", "tier": "Tier 1"},
				}},
			}},
			Outputs: []config.Output{{
				Catalog:    "Services",
				ExternalID: "{{ .name }}",
				Name:       "{{ .name }}",
				Fields: map[string]config.FieldValue{
					"tier": {Value: "{{ .tier }}", Kind: "reference", Catalog: "Tiers"},
				},
			}},
		}},
	}

	configPath = "fake.yaml"
	for _, pipeline := range cfg.Pipelines {
		allEntries, err := loadSources(context.Background(), pipeline, ".")
		if err != nil {
			t.Fatal(err)
		}
		for _, out := range pipeline.Outputs {
			desired, err := mapping.MapEntries(allEntries, out)
			if err != nil {
				t.Fatal(err)
			}
			catalogID, err := cl.EnsureCatalog(context.Background(), client.CatalogSpec{Name: out.Catalog})
			if err != nil {
				t.Fatal(err)
			}
			if err := ensureOutputFields(context.Background(), cl, catalogID, out); err != nil {
				t.Fatal(err)
			}
			if err := resolveReferenceFields(context.Background(), cl, out, desired); err != nil {
				t.Fatal(err)
			}

			if desired[0].Fields["tier"] != "ent-t1" {
				t.Errorf("expected tier resolved to ent-t1, got %s", desired[0].Fields["tier"])
			}
		}
	}
}

func TestAdopt_ResolvesReferenceFields(t *testing.T) {
	var upsertedFields []map[string]any
	srv := refCatalogServer(t, &upsertedFields)
	defer srv.Close()

	cl := client.New("test", client.WithBaseURL(srv.URL), client.WithMaxRetries(0))

	out := config.Output{
		Catalog:    "Services",
		ExternalID: "{{ .name }}",
		Name:       "{{ .name }}",
		Fields: map[string]config.FieldValue{
			"tier": {Value: "{{ .tier }}", Kind: "reference", Catalog: "Tiers"},
		},
	}

	desired := []catalog.DesiredEntity{
		{ExternalID: "svc-a", Name: "Svc A", Fields: map[string]string{"tier": "Tier 2"}},
	}

	if err := resolveReferenceFields(context.Background(), cl, out, desired); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if desired[0].Fields["tier"] != "ent-t2" {
		t.Errorf("expected tier resolved to ent-t2, got %s", desired[0].Fields["tier"])
	}
}
