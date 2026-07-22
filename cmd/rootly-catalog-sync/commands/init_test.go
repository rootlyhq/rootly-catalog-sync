package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

func TestInitDemo_CreatesAllFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	err := runInitDemo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for path := range demoFiles {
		full := filepath.Join(dir, path)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", path)
		}
	}
}

func TestInitDemo_ConfigIsValidV2(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	if err := runInitDemo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.Load(filepath.Join(dir, "rootly-catalog-sync.yaml"))
	if err != nil {
		t.Fatalf("failed to load demo config: %v", err)
	}

	if cfg.Version != 2 {
		t.Errorf("expected version 2, got %d", cfg.Version)
	}

	if err := config.Validate(cfg); err != nil {
		t.Errorf("demo config should validate: %v", err)
	}

	if len(cfg.Pipelines) != 3 {
		t.Errorf("expected 3 pipelines (tiers, services, teams), got %d", len(cfg.Pipelines))
	}
}

func TestInitDemo_ServicesHaveCorrectFields(t *testing.T) {
	content := demoFiles["catalog/services.yaml"]

	if !strings.Contains(content, "api-gateway") {
		t.Error("expected api-gateway in services")
	}
	if !strings.Contains(content, "billing-engine") {
		t.Error("expected billing-engine in services")
	}
	if !strings.Contains(content, "pagerduty_id") {
		t.Error("expected pagerduty_id field in services")
	}
}

func TestInitDemo_TeamsHaveCorrectFields(t *testing.T) {
	content := demoFiles["catalog/teams.yaml"]

	if !strings.Contains(content, "platform-eng") {
		t.Error("expected platform-eng in teams")
	}
	if !strings.Contains(content, "description") {
		t.Error("expected description field in teams")
	}
}

func TestInitDemo_TiersAreValidReference(t *testing.T) {
	content := demoFiles["catalog/tiers.yml"]

	if !strings.Contains(content, "tier-1") {
		t.Error("expected tier-1 in tiers")
	}
	if !strings.Contains(content, "Tier 1") {
		t.Error("expected 'Tier 1' name in tiers")
	}
}

func TestInitDemo_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	if err := os.WriteFile("rootly-catalog-sync.yaml", []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runInitDemo()
	if err == nil {
		t.Fatal("expected error when file already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %s", err)
	}
}

func TestInitDemo_SourcesLoadSuccessfully(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	if err := runInitDemo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.Load(filepath.Join(dir, "rootly-catalog-sync.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	for i, pipeline := range cfg.Pipelines {
		src := pipeline.Sources[0]
		if src.Local == nil {
			t.Errorf("pipeline %d: expected local source", i)
			continue
		}
		for _, pattern := range src.Local.Files {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				t.Errorf("pipeline %d: bad glob %q: %v", i, pattern, err)
			}
			if len(matches) == 0 {
				t.Errorf("pipeline %d: glob %q matched no files", i, pattern)
			}
		}
	}
}
