package config

import (
	"encoding/json"
	"fmt"

	"github.com/rootlyhq/rootly-catalog-sync/client"
	"gopkg.in/yaml.v3"
)

type configV2 struct {
	Version int      `json:"version"`
	Sync    []syncV2 `json:"sync"`
}

type syncV2 struct {
	From sourceConfigV2       `json:"from"`
	To   string               `json:"to"`
	Map  map[string]mapEntryV2 `json:"map"`
}

type sourceConfigV2 = SourceConfig

type mapEntryV2 struct {
	Value     string `json:"value"`
	Reference string `json:"reference,omitempty"`
	Kind      string `json:"kind,omitempty"`
}

func (m *mapEntryV2) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		m.Value = s
		return nil
	}
	type raw mapEntryV2
	return json.Unmarshal(data, (*raw)(m))
}

func loadV2(data []byte) (*Config, error) {
	var raw configV2
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing v2 config: %w", err)
	}

	if len(raw.Sync) == 0 {
		return nil, fmt.Errorf("v2 config: at least one sync entry is required")
	}

	cfg := &Config{
		Version:   2,
		SyncID:    deriveSyncID(raw.Sync),
		Pipelines: make([]Pipeline, 0, len(raw.Sync)),
	}

	for i, s := range raw.Sync {
		if s.To == "" {
			return nil, fmt.Errorf("sync[%d]: 'to' is required", i)
		}

		externalID, ok := s.Map["external_id"]
		if !ok || externalID.Value == "" {
			return nil, fmt.Errorf("sync[%d]: map.external_id is required", i)
		}
		name, ok := s.Map["name"]
		if !ok || name.Value == "" {
			return nil, fmt.Errorf("sync[%d]: map.name is required", i)
		}

		out := Output{
			ExternalID: externalID.Value,
			Name:       name.Value,
			Fields:     make(map[string]FieldValue),
		}

		if client.IsNativeResource(s.To) {
			out.Type = s.To
		} else {
			out.Catalog = s.To
		}

		if backstageID, ok := s.Map["backstage_id"]; ok {
			out.BackstageID = backstageID.Value
		}

		for key, entry := range s.Map {
			if key == "external_id" || key == "name" || key == "backstage_id" {
				continue
			}
			fv := FieldValue{Value: entry.Value}
			if entry.Reference != "" {
				fv.Kind = "reference"
				fv.Catalog = entry.Reference
			} else if entry.Kind != "" {
				fv.Kind = entry.Kind
			}
			out.Fields[key] = fv
		}

		cfg.Pipelines = append(cfg.Pipelines, Pipeline{
			Sources: []SourceConfig{s.From},
			Outputs: []Output{out},
		})
	}

	return cfg, nil
}

func deriveSyncID(syncs []syncV2) string {
	if len(syncs) > 0 {
		return syncs[0].To
	}
	return "sync"
}

func marshalV2(cfg *Config) ([]byte, error) {
	raw := configV2YAML{Version: 2}

	for _, p := range cfg.Pipelines {
		if len(p.Sources) == 0 || len(p.Outputs) == 0 {
			continue
		}
		out := p.Outputs[0]

		target := out.Catalog
		if out.Type != "" {
			target = out.Type
		}

		m := make(map[string]mapEntryV2YAML)
		m["external_id"] = mapEntryV2YAML{scalar: out.ExternalID}
		m["name"] = mapEntryV2YAML{scalar: out.Name}
		if out.BackstageID != "" {
			m["backstage_id"] = mapEntryV2YAML{scalar: out.BackstageID}
		}
		for slug, fv := range out.Fields {
			if fv.Kind == "reference" {
				m[slug] = mapEntryV2YAML{Value: fv.Value, Reference: fv.Catalog}
			} else if fv.Kind != "" && fv.Kind != "text" {
				m[slug] = mapEntryV2YAML{Value: fv.Value, Kind: fv.Kind}
			} else {
				m[slug] = mapEntryV2YAML{scalar: fv.Value}
			}
		}

		raw.Sync = append(raw.Sync, syncV2YAML{
			From: p.Sources[0],
			To:   target,
			Map:  m,
		})
	}

	return yaml.Marshal(raw)
}

type configV2YAML struct {
	Version int          `yaml:"version"`
	Sync    []syncV2YAML `yaml:"sync"`
}

type syncV2YAML struct {
	From SourceConfig             `yaml:"from"`
	To   string                   `yaml:"to"`
	Map  map[string]mapEntryV2YAML `yaml:"map"`
}

type mapEntryV2YAML struct {
	scalar    string
	Value     string `yaml:"value,omitempty"`
	Reference string `yaml:"reference,omitempty"`
	Kind      string `yaml:"kind,omitempty"`
}

func (m mapEntryV2YAML) MarshalYAML() (any, error) {
	if m.scalar != "" {
		return m.scalar, nil
	}
	return struct {
		Value     string `yaml:"value"`
		Reference string `yaml:"reference,omitempty"`
		Kind      string `yaml:"kind,omitempty"`
	}{m.Value, m.Reference, m.Kind}, nil
}
