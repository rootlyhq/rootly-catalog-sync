package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadBackstageFilters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
	}{
		{"v1.yaml", `version: 1
sync_id: backstage
pipelines:
  - sources:
      - backstage:
          url: https://backstage.example
          filters: ["kind=component,spec.type=service", "kind=component,spec.type=website"]
    outputs:
      - type: service
        external_id: "{{ .name }}"
        name: "{{ .name }}"
`},
		{"v2.yaml", `version: 2
sync:
  - from:
      backstage:
        url: https://backstage.example
        filters: ["kind=component,spec.type=service", "kind=component,spec.type=website"]
    to: service
    map:
      external_id: "{{ .name }}"
      name: "{{ .name }}"
`},
		{"v2.jsonnet", `{
  version: 2,
  sync: [{
    from: {backstage: {
      url: "https://backstage.example",
      filters: ["kind=component,spec.type=service", "kind=component,spec.type=website"],
    }},
    to: "service",
    map: {external_id: "{{ .name }}", name: "{{ .name }}"},
  }],
}`},
		{"v2.hcl", `version = 2
sync {
  from {
    backstage {
      url = "https://backstage.example"
      filters = ["kind=component,spec.type=service", "kind=component,spec.type=website"]
    }
  }
  to = "service"
  map = { external_id = "{{ .name }}", name = "{{ .name }}" }
}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.name)
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			assertFilters := func(cfg *Config) {
				t.Helper()
				if len(cfg.Pipelines) != 1 || len(cfg.Pipelines[0].Sources) != 1 {
					t.Fatal("filter sets must stay in one source and pipeline")
				}
				bs := cfg.Pipelines[0].Sources[0].Backstage
				want := []string{"kind=component,spec.type=service", "kind=component,spec.type=website"}
				if bs == nil || !slices.Equal(bs.Filters, want) {
					t.Fatalf("backstage source = %+v, want filters %v", bs, want)
				}
			}
			assertFilters(cfg)

			data, err := Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			roundTrip := filepath.Join(t.TempDir(), "roundtrip.yaml")
			if err := os.WriteFile(roundTrip, data, 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err = Load(roundTrip)
			if err != nil {
				t.Fatal(err)
			}
			assertFilters(cfg)
		})
	}
}
