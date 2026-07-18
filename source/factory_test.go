package source

import (
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

func TestFromConfig_Local(t *testing.T) {
	cfg := config.SourceConfig{
		Local: &config.LocalSourceConfig{
			Files: []string{"data/*.yaml"},
		},
	}
	src, err := FromConfig(cfg, "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ls, ok := src.(*LocalSource)
	if !ok {
		t.Fatalf("expected *LocalSource, got %T", src)
	}
	if ls.Name() != "local" {
		t.Errorf("Name() = %q, want %q", ls.Name(), "local")
	}
	if len(ls.Files) != 1 || ls.Files[0] != "data/*.yaml" {
		t.Errorf("Files = %v, want [data/*.yaml]", ls.Files)
	}
	if ls.BaseDir != "/tmp" {
		t.Errorf("BaseDir = %q, want %q", ls.BaseDir, "/tmp")
	}
}

func TestFromConfig_Inline(t *testing.T) {
	entries := []map[string]any{
		{"name": "svc-a", "tier": "1"},
		{"name": "svc-b", "tier": "2"},
	}
	cfg := config.SourceConfig{
		Inline: &config.InlineSourceConfig{
			Entries: entries,
		},
	}
	src, err := FromConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	is, ok := src.(*InlineSource)
	if !ok {
		t.Fatalf("expected *InlineSource, got %T", src)
	}
	if is.Name() != "inline" {
		t.Errorf("Name() = %q, want %q", is.Name(), "inline")
	}
	if len(is.Entries) != 2 {
		t.Errorf("len(Entries) = %d, want 2", len(is.Entries))
	}
}

func TestFromConfig_None(t *testing.T) {
	cfg := config.SourceConfig{}
	_, err := FromConfig(cfg, "")
	if err == nil {
		t.Fatal("expected error for empty config, got nil")
	}
	if err.Error() != "no source type configured" {
		t.Errorf("error = %q, want %q", err.Error(), "no source type configured")
	}
}

func TestFromConfig_Multiple(t *testing.T) {
	cfg := config.SourceConfig{
		Local: &config.LocalSourceConfig{
			Files: []string{"*.yaml"},
		},
		Inline: &config.InlineSourceConfig{
			Entries: []map[string]any{{"name": "x"}},
		},
	}
	_, err := FromConfig(cfg, "")
	if err == nil {
		t.Fatal("expected error for multiple source types, got nil")
	}
	if err.Error() != "multiple source types configured; exactly one is required" {
		t.Errorf("error = %q, want %q", err.Error(), "multiple source types configured; exactly one is required")
	}
}

func TestFromConfig_GitHub(t *testing.T) {
	cfg := config.SourceConfig{
		GitHub: &config.GitHubSourceConfig{
			Owner: "myorg",
			Files: []string{"catalog.yaml"},
			Token: "ghp_test",
		},
	}
	src, err := FromConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gs, ok := src.(*GitHubSource)
	if !ok {
		t.Fatalf("expected *GitHubSource, got %T", src)
	}
	if gs.Name() != "github" {
		t.Errorf("Name() = %q, want %q", gs.Name(), "github")
	}
}

func TestFromConfig_URL(t *testing.T) {
	cfg := config.SourceConfig{
		URL: &config.URLSourceConfig{
			URLs: []string{"https://example.com/data.yaml"},
		},
	}
	src, err := FromConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	us, ok := src.(*URLSource)
	if !ok {
		t.Fatalf("expected *URLSource, got %T", src)
	}
	if us.Name() != "url" {
		t.Errorf("Name() = %q, want %q", us.Name(), "url")
	}
	if len(us.URLs) != 1 || us.URLs[0] != "https://example.com/data.yaml" {
		t.Errorf("URLs = %v, want [https://example.com/data.yaml]", us.URLs)
	}
}

func TestFromConfig_HTTP(t *testing.T) {
	cfg := config.SourceConfig{
		HTTP: &config.HTTPSourceConfig{
			URL:    "https://api.example.com/items",
			Method: "POST",
			Result: "data.items",
		},
	}
	src, err := FromConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hs, ok := src.(*HTTPSource)
	if !ok {
		t.Fatalf("expected *HTTPSource, got %T", src)
	}
	if hs.Name() != "http" {
		t.Errorf("Name() = %q, want %q", hs.Name(), "http")
	}
	if hs.Method != "POST" {
		t.Errorf("Method = %q, want %q", hs.Method, "POST")
	}
}

func TestFromConfig_Exec(t *testing.T) {
	cfg := config.SourceConfig{
		Exec: &config.ExecSourceConfig{
			Command: "echo",
			Args:    []string{"hello"},
		},
	}
	src, err := FromConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.Name() != "exec" {
		t.Errorf("Name() = %q, want %q", src.Name(), "exec")
	}
}

func TestFromConfig_CSV(t *testing.T) {
	cfg := config.SourceConfig{
		CSV: &config.CSVSourceConfig{
			Files:     []string{"data.csv"},
			Delimiter: ",",
		},
	}
	src, err := FromConfig(cfg, "/base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.Name() != "csv" {
		t.Errorf("Name() = %q, want %q", src.Name(), "csv")
	}
}

func TestFromConfig_Backstage(t *testing.T) {
	cfg := config.SourceConfig{
		Backstage: &config.BackstageSourceConfig{
			URL: "https://backstage.example.com",
		},
	}
	src, err := FromConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.Name() != "backstage" {
		t.Errorf("Name() = %q, want %q", src.Name(), "backstage")
	}
}

func TestFromConfig_GraphQL(t *testing.T) {
	cfg := config.SourceConfig{
		GraphQL: &config.GraphQLSourceConfig{
			URL:    "https://api.example.com/graphql",
			Query:  "{ items { id name } }",
			Result: "data.items",
		},
	}
	src, err := FromConfig(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.Name() != "graphql" {
		t.Errorf("Name() = %q, want %q", src.Name(), "graphql")
	}
}
