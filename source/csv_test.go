package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCSVSource_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")
	_ = os.WriteFile(path, []byte("name,tier,owner\nsvc-a,1,team-a\nsvc-b,2,team-b\n"), 0644)

	src := NewCSVSource([]string{"data.csv"}, "", dir)
	if src.Name() != "csv" {
		t.Errorf("expected name=csv, got %s", src.Name())
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
	if entries[0]["tier"] != "1" {
		t.Errorf("expected tier=1, got %v", entries[0]["tier"])
	}
	if entries[0]["owner"] != "team-a" {
		t.Errorf("expected owner=team-a, got %v", entries[0]["owner"])
	}
	if entries[1]["name"] != "svc-b" {
		t.Errorf("expected name=svc-b, got %v", entries[1]["name"])
	}
}

func TestCSVSource_CustomDelimiter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")
	_ = os.WriteFile(path, []byte("name;tier\nsvc-a;1\nsvc-b;2\n"), 0644)

	src := NewCSVSource([]string{"data.csv"}, ";", dir)
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
	if entries[0]["tier"] != "1" {
		t.Errorf("expected tier=1, got %v", entries[0]["tier"])
	}
}

func TestCSVSource_GlobPattern(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.csv"), []byte("name\nsvc-a\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "b.csv"), []byte("name\nsvc-b\n"), 0644)

	src := NewCSVSource([]string{"*.csv"}, "", dir)
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	names := map[string]bool{}
	for _, e := range entries {
		names[e["name"].(string)] = true
	}
	for _, expected := range []string{"svc-a", "svc-b"} {
		if !names[expected] {
			t.Errorf("missing entry %s", expected)
		}
	}
}

func TestCSVSource_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "empty.csv"), []byte("name,tier\n"), 0644)

	src := NewCSVSource([]string{"empty.csv"}, "", dir)
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}
