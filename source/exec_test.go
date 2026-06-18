package source

import (
	"context"
	"testing"
)

func TestExecSource_JSON(t *testing.T) {
	src := NewExecSource("echo", []string{`{"id":"svc-1","name":"Test"}`})
	if src.Name() != "exec" {
		t.Errorf("expected name=exec, got %s", src.Name())
	}
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["id"] != "svc-1" {
		t.Errorf("expected id=svc-1, got %v", entries[0]["id"])
	}
	if entries[0]["name"] != "Test" {
		t.Errorf("expected name=Test, got %v", entries[0]["name"])
	}
}

func TestExecSource_JSONArray(t *testing.T) {
	src := NewExecSource("echo", []string{`[{"id":"1"},{"id":"2"}]`})
	entries, err := src.Load(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0]["id"] != "1" {
		t.Errorf("expected id=1, got %v", entries[0]["id"])
	}
	if entries[1]["id"] != "2" {
		t.Errorf("expected id=2, got %v", entries[1]["id"])
	}
}

func TestExecSource_NonZeroExit(t *testing.T) {
	src := NewExecSource("sh", []string{"-c", "exit 1"})
	_, err := src.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}
}

func TestExecSource_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	src := NewExecSource("echo", []string{"hello"})
	_, err := src.Load(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}
