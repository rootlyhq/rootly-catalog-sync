package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParse_JSON_SingleObject(t *testing.T) {
	data := []byte(`{"name": "svc-a", "tier": "1"}`)
	entries, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["name"] != "svc-a" {
		t.Errorf("expected name=svc-a, got %v", entries[0]["name"])
	}
}

func TestParse_JSON_Array(t *testing.T) {
	data := []byte(`[{"name": "a"}, {"name": "b"}]`)
	entries, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0]["name"] != "a" {
		t.Errorf("expected name=a, got %v", entries[0]["name"])
	}
	if entries[1]["name"] != "b" {
		t.Errorf("expected name=b, got %v", entries[1]["name"])
	}
}

func TestParse_YAML_SingleDoc(t *testing.T) {
	data := []byte("name: svc-a\ntier: \"1\"\n")
	entries, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["name"] != "svc-a" {
		t.Errorf("expected name=svc-a, got %v", entries[0]["name"])
	}
}

func TestParse_YAML_MultiDoc(t *testing.T) {
	data := []byte("name: a\n---\nname: b\n")
	entries, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0]["name"] != "a" {
		t.Errorf("expected name=a, got %v", entries[0]["name"])
	}
	if entries[1]["name"] != "b" {
		t.Errorf("expected name=b, got %v", entries[1]["name"])
	}
}

func TestInlineSource(t *testing.T) {
	raw := []map[string]any{
		{"name": "svc-a", "tier": "1"},
		{"name": "svc-b", "tier": "2"},
	}
	src := NewInlineSource(raw)
	if src.Name() != "inline" {
		t.Errorf("expected name=inline, got %s", src.Name())
	}
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0]["name"] != "svc-a" {
		t.Errorf("expected name=svc-a, got %v", entries[0]["name"])
	}
}

func TestLocalSource(t *testing.T) {
	dir := t.TempDir()

	f1 := filepath.Join(dir, "services.yaml")
	if err := os.WriteFile(f1, []byte("name: svc-a\ntier: \"1\"\n---\nname: svc-b\ntier: \"2\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	f2 := filepath.Join(dir, "extra.yaml")
	if err := os.WriteFile(f2, []byte("name: svc-c\n"), 0644); err != nil {
		t.Fatal(err)
	}

	src := NewLocalSource([]string{"*.yaml"}, dir)
	if src.Name() != "local" {
		t.Errorf("expected name=local, got %s", src.Name())
	}
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	names := map[string]bool{}
	for _, e := range entries {
		names[e["name"].(string)] = true
	}
	for _, expected := range []string{"svc-a", "svc-b", "svc-c"} {
		if !names[expected] {
			t.Errorf("missing entry %s", expected)
		}
	}
}

func TestLocalSource_NoMatch(t *testing.T) {
	dir := t.TempDir()
	src := NewLocalSource([]string{"*.json"}, dir)
	_, err := src.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for no matching files")
	}
}
