package tmpl

import (
	"testing"
)

func TestEvalSimpleField(t *testing.T) {
	result, err := Eval("{{ .name }}", map[string]any{"name": "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "foo" {
		t.Fatalf("expected %q, got %q", "foo", result)
	}
}

func TestEvalNestedGet(t *testing.T) {
	data := map[string]any{
		"meta": map[string]any{
			"team": "platform",
		},
	}
	result, err := Eval(`{{ get .meta "team" }}`, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "platform" {
		t.Fatalf("expected %q, got %q", "platform", result)
	}
}

func TestEvalMissingFieldErrors(t *testing.T) {
	_, err := Eval("{{ .missing }}", map[string]any{"name": "foo"})
	if err == nil {
		t.Fatal("expected error for missing field, got nil")
	}
}

func TestEvalDefaultFunc(t *testing.T) {
	result, err := Eval(`{{ default .tier "unknown" }}`, map[string]any{"tier": nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "unknown" {
		t.Fatalf("expected %q, got %q", "unknown", result)
	}

	result, err = Eval(`{{ default .tier "unknown" }}`, map[string]any{"tier": "gold"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "gold" {
		t.Fatalf("expected %q, got %q", "gold", result)
	}
}

func TestEvalCaching(t *testing.T) {
	tmpl := `{{ .name }} - {{ default .tier "unknown" }}`

	data1 := map[string]any{"name": "svc-a", "tier": "gold"}
	result1, err := Eval(tmpl, data1)
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	if result1 != "svc-a - gold" {
		t.Fatalf("first call: expected %q, got %q", "svc-a - gold", result1)
	}

	data2 := map[string]any{"name": "svc-b", "tier": nil}
	result2, err := Eval(tmpl, data2)
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}
	if result2 != "svc-b - unknown" {
		t.Fatalf("second call: expected %q, got %q", "svc-b - unknown", result2)
	}
}

func TestEvalIntegerAndBoolValues(t *testing.T) {
	data := map[string]any{
		"count":   42,
		"enabled": true,
	}

	result, err := Eval("{{ .count }}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "42" {
		t.Fatalf("expected %q, got %q", "42", result)
	}

	result, err = Eval("{{ .enabled }}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "true" {
		t.Fatalf("expected %q, got %q", "true", result)
	}
}
