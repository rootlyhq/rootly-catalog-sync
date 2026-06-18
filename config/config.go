package config

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
	Catalog    string            `yaml:"catalog" json:"catalog" hcl:"catalog"`
	ExternalID string            `yaml:"external_id" json:"external_id" hcl:"external_id"`
	Name       string            `yaml:"name" json:"name" hcl:"name"`
	Fields     map[string]string `yaml:"fields,omitempty" json:"fields,omitempty" hcl:"fields,optional"`
}
