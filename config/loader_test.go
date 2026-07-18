package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadYAML(t *testing.T) {
	content := `
version: 1
sync_id: test-sync
pipelines:
  - sources:
      - local:
          files:
            - "data/*.yaml"
    outputs:
      - catalog: services
        external_id: "{{ .name }}"
        name: "{{ .name }}"
        fields:
          tier: "{{ .tier }}"
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.SyncID != "test-sync" {
		t.Errorf("expected sync_id test-sync, got %s", cfg.SyncID)
	}
	if len(cfg.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(cfg.Pipelines))
	}
	p := cfg.Pipelines[0]
	if len(p.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(p.Sources))
	}
	if p.Sources[0].Local == nil {
		t.Fatal("expected local source")
	}
	if len(p.Sources[0].Local.Files) != 1 || p.Sources[0].Local.Files[0] != "data/*.yaml" {
		t.Errorf("unexpected local files: %v", p.Sources[0].Local.Files)
	}
	if len(p.Outputs) != 1 {
		t.Fatalf("expected 1 output, got %d", len(p.Outputs))
	}
	if p.Outputs[0].Catalog != "services" {
		t.Errorf("expected catalog services, got %s", p.Outputs[0].Catalog)
	}
	if p.Outputs[0].Fields["tier"] != "{{ .tier }}" {
		t.Errorf("unexpected fields: %v", p.Outputs[0].Fields)
	}
}

func TestEnvSubstitution(t *testing.T) {
	t.Setenv("MY_VAR", "secret-token")

	content := `
version: 1
sync_id: test-sync
pipelines:
  - sources:
      - github:
          token: "$(MY_VAR)"
          owner: myorg
          files:
            - "catalog.yaml"
    outputs:
      - catalog: services
        external_id: "{{ .name }}"
        name: "{{ .name }}"
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Pipelines[0].Sources[0].GitHub.Token != "secret-token" {
		t.Errorf("expected token secret-token, got %s", cfg.Pipelines[0].Sources[0].GitHub.Token)
	}
}

func TestLoadJsonnet(t *testing.T) {
	content := `{
  version: 1,
  sync_id: "jsonnet-sync",
  pipelines: [
    {
      sources: [
        {
          "local": {
            files: ["data/*.yaml"],
          },
        },
      ],
      outputs: [
        {
          catalog: "services",
          external_id: "{{ .name }}",
          name: "{{ .name }}",
          fields: {
            tier: "{{ .tier }}",
          },
        },
      ],
    },
  ],
}
`
	path := filepath.Join(t.TempDir(), "config.jsonnet")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.SyncID != "jsonnet-sync" {
		t.Errorf("expected sync_id jsonnet-sync, got %s", cfg.SyncID)
	}
	if len(cfg.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(cfg.Pipelines))
	}
	p := cfg.Pipelines[0]
	if len(p.Sources) != 1 || p.Sources[0].Local == nil {
		t.Fatal("expected 1 local source")
	}
	if len(p.Sources[0].Local.Files) != 1 || p.Sources[0].Local.Files[0] != "data/*.yaml" {
		t.Errorf("unexpected local files: %v", p.Sources[0].Local.Files)
	}
	if len(p.Outputs) != 1 {
		t.Fatalf("expected 1 output, got %d", len(p.Outputs))
	}
	if p.Outputs[0].Catalog != "services" {
		t.Errorf("expected catalog services, got %s", p.Outputs[0].Catalog)
	}
	if p.Outputs[0].Fields["tier"] != "{{ .tier }}" {
		t.Errorf("unexpected fields: %v", p.Outputs[0].Fields)
	}
}

func TestLoadJsonnetEnvSubstitution(t *testing.T) {
	t.Setenv("JSONNET_TOKEN", "my-secret")

	content := `{
  version: 1,
  sync_id: "jsonnet-env",
  pipelines: [
    {
      sources: [
        {
          github: {
            token: "$(JSONNET_TOKEN)",
            owner: "myorg",
            files: ["catalog.yaml"],
          },
        },
      ],
      outputs: [
        {
          catalog: "services",
          external_id: "{{ .name }}",
          name: "{{ .name }}",
        },
      ],
    },
  ],
}
`
	path := filepath.Join(t.TempDir(), "config.jsonnet")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Pipelines[0].Sources[0].GitHub.Token != "my-secret" {
		t.Errorf("expected token my-secret, got %s", cfg.Pipelines[0].Sources[0].GitHub.Token)
	}
}

func TestLoadHCL(t *testing.T) {
	content := `
version = 1
sync_id = "hcl-sync"

pipeline {
  source {
    local {
      files = ["data/*.yaml"]
    }
  }
  output {
    catalog     = "services"
    external_id = "{{ .name }}"
    name        = "{{ .name }}"
    fields = {
      tier = "{{ .tier }}"
    }
  }
}
`
	path := filepath.Join(t.TempDir(), "config.hcl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.SyncID != "hcl-sync" {
		t.Errorf("expected sync_id hcl-sync, got %s", cfg.SyncID)
	}
	if len(cfg.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(cfg.Pipelines))
	}
	p := cfg.Pipelines[0]
	if len(p.Sources) != 1 || p.Sources[0].Local == nil {
		t.Fatal("expected 1 local source")
	}
	if len(p.Sources[0].Local.Files) != 1 || p.Sources[0].Local.Files[0] != "data/*.yaml" {
		t.Errorf("unexpected local files: %v", p.Sources[0].Local.Files)
	}
	if len(p.Outputs) != 1 {
		t.Fatalf("expected 1 output, got %d", len(p.Outputs))
	}
	if p.Outputs[0].Catalog != "services" {
		t.Errorf("expected catalog services, got %s", p.Outputs[0].Catalog)
	}
	if p.Outputs[0].Fields["tier"] != "{{ .tier }}" {
		t.Errorf("unexpected fields: %v", p.Outputs[0].Fields)
	}
}

func TestValidateValid(t *testing.T) {
	cfg := &Config{
		Version: 1,
		SyncID:  "test",
		Pipelines: []Pipeline{
			{
				Sources: []SourceConfig{{Local: &LocalSourceConfig{Files: []string{"*.yaml"}}}},
				Outputs: []Output{{Catalog: "services", ExternalID: "{{ .name }}", Name: "{{ .name }}"}},
			},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateMissingSyncID(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Pipelines: []Pipeline{
			{
				Sources: []SourceConfig{{Local: &LocalSourceConfig{Files: []string{"*.yaml"}}}},
				Outputs: []Output{{Catalog: "services", ExternalID: "{{ .name }}", Name: "{{ .name }}"}},
			},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing sync_id")
	}
}

func TestValidateBadVersion(t *testing.T) {
	cfg := &Config{
		Version: 2,
		SyncID:  "test",
		Pipelines: []Pipeline{
			{
				Sources: []SourceConfig{{Local: &LocalSourceConfig{Files: []string{"*.yaml"}}}},
				Outputs: []Output{{Catalog: "services", ExternalID: "{{ .name }}", Name: "{{ .name }}"}},
			},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestValidateNoPipelines(t *testing.T) {
	cfg := &Config{
		Version: 1,
		SyncID:  "test",
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for no pipelines")
	}
}

func TestValidateMissingOutputCatalog(t *testing.T) {
	cfg := &Config{
		Version: 1,
		SyncID:  "test",
		Pipelines: []Pipeline{
			{
				Sources: []SourceConfig{{Local: &LocalSourceConfig{Files: []string{"*.yaml"}}}},
				Outputs: []Output{{ExternalID: "{{ .name }}", Name: "{{ .name }}"}},
			},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing output catalog")
	}
}

func TestValidateMissingSource(t *testing.T) {
	cfg := &Config{
		Version: 1,
		SyncID:  "test",
		Pipelines: []Pipeline{
			{
				Outputs: []Output{{Catalog: "services", ExternalID: "{{ .name }}", Name: "{{ .name }}"}},
			},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestValidateMissingOutput(t *testing.T) {
	cfg := &Config{
		Version: 1,
		SyncID:  "test",
		Pipelines: []Pipeline{
			{
				Sources: []SourceConfig{{Local: &LocalSourceConfig{Files: []string{"*.yaml"}}}},
			},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing output")
	}
}

func TestValidateNativeType(t *testing.T) {
	// A native type (e.g., "service") with no catalog field should pass validation.
	cfg := &Config{
		Version: 1,
		SyncID:  "test",
		Pipelines: []Pipeline{
			{
				Sources: []SourceConfig{{Local: &LocalSourceConfig{Files: []string{"*.yaml"}}}},
				Outputs: []Output{{
					Type:       "service",
					ExternalID: "{{ .name }}",
					Name:       "{{ .name }}",
				}},
			},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error for native type 'service', got: %v", err)
	}
}

func TestValidateInvalidType(t *testing.T) {
	// NOTE: There is currently no type validation — any non-empty Type string
	// is treated as a native type. An output with type "invalid" passes
	// validation because the Validate function only checks that catalog is
	// present when type is empty or "catalog". This is a coverage gap: invalid
	// type names are not rejected at config validation time.
	cfg := &Config{
		Version: 1,
		SyncID:  "test",
		Pipelines: []Pipeline{
			{
				Sources: []SourceConfig{{Local: &LocalSourceConfig{Files: []string{"*.yaml"}}}},
				Outputs: []Output{{
					Type:       "invalid",
					ExternalID: "{{ .name }}",
					Name:       "{{ .name }}",
				}},
			},
		},
	}
	err := Validate(cfg)
	// This currently passes — documenting that no type validation exists.
	if err != nil {
		t.Errorf("unexpected error (type validation may have been added): %v", err)
	}
}
