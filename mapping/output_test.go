package mapping

import (
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/config"
	"github.com/rootlyhq/rootly-catalog-sync/source"
)

func TestMapEntriesSimple(t *testing.T) {
	entries := []source.Entry{
		{"id": "svc-1", "name": "Auth Service", "tier": "critical"},
	}
	out := config.Output{
		ExternalID: "{{ .id }}",
		Name:       "{{ .name }}",
		Fields: map[string]config.FieldValue{
			"tier": {Value: "{{ .tier }}"},
		},
	}

	result, err := MapEntries(entries, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(result))
	}
	if result[0].ExternalID != "svc-1" {
		t.Fatalf("expected external_id %q, got %q", "svc-1", result[0].ExternalID)
	}
	if result[0].Name != "Auth Service" {
		t.Fatalf("expected name %q, got %q", "Auth Service", result[0].Name)
	}
	if result[0].Fields["tier"] != "critical" {
		t.Fatalf("expected field tier %q, got %q", "critical", result[0].Fields["tier"])
	}
}

func TestMapEntriesBackstageID(t *testing.T) {
	entries := []source.Entry{
		{"id": "svc-1", "name": "Auth", "backstage_id": "Component:default/auth"},
	}
	out := config.Output{
		ExternalID:  "{{ .id }}",
		Name:        "{{ .name }}",
		BackstageID: "{{ .backstage_id }}",
	}

	result, err := MapEntries(entries, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result[0].BackstageID != "Component:default/auth" {
		t.Errorf("expected backstage_id %q, got %q", "Component:default/auth", result[0].BackstageID)
	}
}

func TestMapEntriesBackstageIDEmpty(t *testing.T) {
	entries := []source.Entry{
		{"id": "svc-1", "name": "Auth"},
	}
	out := config.Output{
		ExternalID: "{{ .id }}",
		Name:       "{{ .name }}",
	}

	result, err := MapEntries(entries, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result[0].BackstageID != "" {
		t.Errorf("expected empty backstage_id, got %q", result[0].BackstageID)
	}
}

func TestMapEntriesEmptyExternalIDErrors(t *testing.T) {
	entries := []source.Entry{
		{"id": "", "name": "Auth Service"},
	}
	out := config.Output{
		ExternalID: "{{ .id }}",
		Name:       "{{ .name }}",
	}

	_, err := MapEntries(entries, out)
	if err == nil {
		t.Fatal("expected error for empty external_id, got nil")
	}
}

func TestMapEntriesMultiple(t *testing.T) {
	entries := []source.Entry{
		{"id": "svc-1", "name": "Auth"},
		{"id": "svc-2", "name": "Billing"},
		{"id": "svc-3", "name": "Gateway"},
	}
	out := config.Output{
		ExternalID: "{{ .id }}",
		Name:       "{{ .name }}",
	}

	result, err := MapEntries(entries, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(result))
	}
	for i, name := range []string{"Auth", "Billing", "Gateway"} {
		if result[i].Name != name {
			t.Fatalf("entry %d: expected name %q, got %q", i, name, result[i].Name)
		}
	}
}
