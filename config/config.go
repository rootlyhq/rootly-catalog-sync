package config

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version   int        `yaml:"version" json:"version" hcl:"version"`
	SyncID    string     `yaml:"sync_id" json:"sync_id" hcl:"sync_id"`
	Pipelines []Pipeline `yaml:"pipelines" json:"pipelines" hcl:"pipeline,block"`
}

type Pipeline struct {
	Sources []SourceConfig `yaml:"sources" json:"sources" hcl:"source,block"`
	Outputs []Output       `yaml:"outputs" json:"outputs" hcl:"output,block"`
}

type SourceConfig struct {
	Local     *LocalSourceConfig     `yaml:"local,omitempty" json:"local,omitempty" hcl:"local,block"`
	Inline    *InlineSourceConfig    `yaml:"inline,omitempty" json:"inline,omitempty" hcl:"inline,block"`
	GitHub    *GitHubSourceConfig    `yaml:"github,omitempty" json:"github,omitempty" hcl:"github,block"`
	Exec      *ExecSourceConfig      `yaml:"exec,omitempty" json:"exec,omitempty" hcl:"exec,block"`
	Backstage *BackstageSourceConfig `yaml:"backstage,omitempty" json:"backstage,omitempty" hcl:"backstage,block"`
	GraphQL   *GraphQLSourceConfig   `yaml:"graphql,omitempty" json:"graphql,omitempty" hcl:"graphql,block"`
	CSV       *CSVSourceConfig       `yaml:"csv,omitempty" json:"csv,omitempty" hcl:"csv,block"`
	URL       *URLSourceConfig       `yaml:"url,omitempty" json:"url,omitempty" hcl:"url,block"`
	HTTP      *HTTPSourceConfig      `yaml:"http,omitempty" json:"http,omitempty" hcl:"http,block"`
}

type URLSourceConfig struct {
	URLs    []string          `yaml:"urls" json:"urls" hcl:"urls"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty" hcl:"headers,optional"`
}

type HTTPSourceConfig struct {
	URL     string            `yaml:"url" json:"url" hcl:"url"`
	Method  string            `yaml:"method,omitempty" json:"method,omitempty" hcl:"method,optional"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty" hcl:"headers,optional"`
	Body    string            `yaml:"body,omitempty" json:"body,omitempty" hcl:"body,optional"`
	Result  string            `yaml:"result" json:"result" hcl:"result"`
}

type LocalSourceConfig struct {
	Files []string `yaml:"files" json:"files" hcl:"files"`
}

type InlineSourceConfig struct {
	Entries []map[string]any `yaml:"entries" json:"entries" hcl:"entries"`
}

type GitHubSourceConfig struct {
	Token    string   `yaml:"token,omitempty" json:"token,omitempty" hcl:"token,optional"`
	Owner    string   `yaml:"owner" json:"owner" hcl:"owner"`
	Repos    []string `yaml:"repos,omitempty" json:"repos,omitempty" hcl:"repos,optional"`
	Files    []string `yaml:"files" json:"files" hcl:"files"`
	Ref      string   `yaml:"ref,omitempty" json:"ref,omitempty" hcl:"ref,optional"`
	Archived bool     `yaml:"archived,omitempty" json:"archived,omitempty" hcl:"archived,optional"`
}

type ExecSourceConfig struct {
	Command string   `yaml:"command" json:"command" hcl:"command"`
	Args    []string `yaml:"args,omitempty" json:"args,omitempty" hcl:"args,optional"`
}

type BackstageSourceConfig struct {
	URL    string `yaml:"url" json:"url" hcl:"url"`
	Token  string `yaml:"token,omitempty" json:"token,omitempty" hcl:"token,optional"`
	Filter string `yaml:"filter,omitempty" json:"filter,omitempty" hcl:"filter,optional"`
	Kind   string `yaml:"kind,omitempty" json:"kind,omitempty" hcl:"kind,optional"`
}

type GraphQLSourceConfig struct {
	URL      string            `yaml:"url" json:"url" hcl:"url"`
	Query    string            `yaml:"query" json:"query" hcl:"query"`
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty" hcl:"headers,optional"`
	Result   string            `yaml:"result" json:"result" hcl:"result"`
	Paginate *PaginateConfig   `yaml:"paginate,omitempty" json:"paginate,omitempty" hcl:"paginate,block"`
}

type PaginateConfig struct {
	Mode       string `yaml:"mode" json:"mode" hcl:"mode"`
	PageSize   int    `yaml:"page_size" json:"page_size" hcl:"page_size"`
	CursorPath string `yaml:"cursor_path,omitempty" json:"cursor_path,omitempty" hcl:"cursor_path,optional"`
}

type CSVSourceConfig struct {
	Files     []string `yaml:"files" json:"files" hcl:"files"`
	Delimiter string   `yaml:"delimiter,omitempty" json:"delimiter,omitempty" hcl:"delimiter,optional"`
}

type Output struct {
	Type        string                `yaml:"type,omitempty" json:"type,omitempty" hcl:"type,optional"`
	Catalog     string                `yaml:"catalog" json:"catalog" hcl:"catalog"`
	ExternalID  string                `yaml:"external_id" json:"external_id" hcl:"external_id"`
	Name        string                `yaml:"name" json:"name" hcl:"name"`
	BackstageID string                `yaml:"backstage_id,omitempty" json:"backstage_id,omitempty" hcl:"backstage_id,optional"`
	Fields      map[string]FieldValue `yaml:"fields,omitempty" json:"fields,omitempty" hcl:"fields,optional"`
}

type FieldValue struct {
	Value   string `yaml:"value" json:"value" hcl:"value"`
	Kind    string `yaml:"kind,omitempty" json:"kind,omitempty" hcl:"kind,optional"`
	Catalog string `yaml:"catalog,omitempty" json:"catalog,omitempty" hcl:"catalog,optional"`
}

func (f FieldValue) EffectiveKind() string {
	if f.Kind == "" {
		return "text"
	}
	return f.Kind
}

func (f *FieldValue) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		f.Value = node.Value
		return nil
	}
	type raw FieldValue
	return node.Decode((*raw)(f))
}

func (f *FieldValue) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		f.Value = s
		return nil
	}
	type raw FieldValue
	return json.Unmarshal(data, (*raw)(f))
}

func (f FieldValue) MarshalJSON() ([]byte, error) {
	if f.Kind == "" && f.Catalog == "" {
		return json.Marshal(f.Value)
	}
	type raw FieldValue
	return json.Marshal((raw)(f))
}

func (f FieldValue) MarshalYAML() (any, error) {
	if f.Kind == "" && f.Catalog == "" {
		return f.Value, nil
	}
	return struct {
		Value   string `yaml:"value"`
		Kind    string `yaml:"kind,omitempty"`
		Catalog string `yaml:"catalog,omitempty"`
	}{f.Value, f.Kind, f.Catalog}, nil
}

func Marshal(cfg *Config) ([]byte, error) {
	if cfg.Version == 2 {
		return marshalV2(cfg)
	}
	return yaml.Marshal(cfg)
}
