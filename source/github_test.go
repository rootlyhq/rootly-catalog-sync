package source

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v72/github"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

func newTestGitHubClient(baseURL string) *github.Client {
	client := github.NewClient(nil)
	u, _ := url.Parse(baseURL + "/")
	client.BaseURL = u
	return client
}

func TestGitHubSource_SingleRepo(t *testing.T) {
	yamlContent := "name: svc-a\ntier: \"1\"\n"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(yamlContent))
	blobSHA := "abc123"

	mux := http.NewServeMux()

	// GET /repos/{owner}/{repo}
	mux.HandleFunc("GET /repos/testorg/myrepo", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"name":           "myrepo",
			"full_name":      "testorg/myrepo",
			"default_branch": "main",
			"archived":       false,
			"fork":           false,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// GET /repos/{owner}/{repo}/git/trees/{ref}?recursive=1
	mux.HandleFunc("GET /repos/testorg/myrepo/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"sha":       "treeSHA",
			"truncated": false,
			"tree": []map[string]any{
				{
					"path": "catalog.yaml",
					"mode": "100644",
					"type": "blob",
					"sha":  blobSHA,
					"size": len(yamlContent),
				},
				{
					"path": "README.md",
					"mode": "100644",
					"type": "blob",
					"sha":  "otherSHA",
					"size": 100,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// GET /repos/{owner}/{repo}/git/blobs/{sha}
	mux.HandleFunc(fmt.Sprintf("GET /repos/testorg/myrepo/git/blobs/%s", blobSHA), func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"sha":      blobSHA,
			"size":     len(yamlContent),
			"encoding": "base64",
			"content":  encodedContent,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	src := &GitHubSource{
		Owner: "testorg",
		Repos: []string{"myrepo"},
		Files: []string{"*.yaml"},
	}

	// Override the client creation by testing processRepo directly
	client := newTestGitHubClient(server.URL)
	repo := &github.Repository{
		Name:          github.Ptr("myrepo"),
		DefaultBranch: github.Ptr("main"),
		Archived:      github.Ptr(false),
		Fork:          github.Ptr(false),
	}

	entries, err := src.processRepo(context.Background(), client, repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0]["name"] != "svc-a" {
		t.Errorf("expected name=svc-a, got %v", entries[0]["name"])
	}
	if entries[0]["tier"] != "1" {
		t.Errorf("expected tier=1, got %v", entries[0]["tier"])
	}
}

func TestGitHubSource_SkipArchived(t *testing.T) {
	repos := []*github.Repository{
		{
			Name:          github.Ptr("active-repo"),
			DefaultBranch: github.Ptr("main"),
			Archived:      github.Ptr(false),
			Fork:          github.Ptr(false),
		},
		{
			Name:          github.Ptr("archived-repo"),
			DefaultBranch: github.Ptr("main"),
			Archived:      github.Ptr(true),
			Fork:          github.Ptr(false),
		},
		{
			Name:          github.Ptr("forked-repo"),
			DefaultBranch: github.Ptr("main"),
			Archived:      github.Ptr(false),
			Fork:          github.Ptr(true),
		},
	}

	src := &GitHubSource{
		Owner:           "testorg",
		IncludeArchived: false,
	}

	filtered := src.filterRepos(repos)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 repo after filtering, got %d", len(filtered))
	}
	if filtered[0].GetName() != "active-repo" {
		t.Errorf("expected active-repo, got %s", filtered[0].GetName())
	}
}

func TestGitHubSource_SkipArchived_IncludeArchived(t *testing.T) {
	repos := []*github.Repository{
		{
			Name:          github.Ptr("active-repo"),
			DefaultBranch: github.Ptr("main"),
			Archived:      github.Ptr(false),
			Fork:          github.Ptr(false),
		},
		{
			Name:          github.Ptr("archived-repo"),
			DefaultBranch: github.Ptr("main"),
			Archived:      github.Ptr(true),
			Fork:          github.Ptr(false),
		},
	}

	src := &GitHubSource{
		Owner:           "testorg",
		IncludeArchived: true,
	}

	filtered := src.filterRepos(repos)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 repos after filtering with IncludeArchived, got %d", len(filtered))
	}
}

func TestGitHubSource_EmptyRepo(t *testing.T) {
	mux := http.NewServeMux()

	// Return a tree with no entries
	mux.HandleFunc("GET /repos/testorg/empty-repo/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"sha":       "emptyTreeSHA",
			"truncated": false,
			"tree":      []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	src := &GitHubSource{
		Owner: "testorg",
		Files: []string{"*.yaml"},
	}

	client := newTestGitHubClient(server.URL)
	repo := &github.Repository{
		Name:          github.Ptr("empty-repo"),
		DefaultBranch: github.Ptr("main"),
		Archived:      github.Ptr(false),
		Fork:          github.Ptr(false),
	}

	entries, err := src.processRepo(context.Background(), client, repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for empty repo, got %d", len(entries))
	}
}

func TestGitHubSource_DoublestarMatch(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{"doublestar matches nested path", []string{"**/*.yaml"}, "foo/bar/catalog.yaml", true},
		{"doublestar matches single level", []string{"**/*.yaml"}, "catalog.yaml", true},
		{"doublestar matches deep nesting", []string{"**/catalog.yaml"}, "a/b/c/catalog.yaml", true},
		{"single star still works on basename", []string{"*.yaml"}, "services/catalog.yaml", true},
		{"no match wrong extension", []string{"**/*.yaml"}, "foo/bar/catalog.json", false},
		{"exact match via doublestar", []string{"services/**/*.yaml"}, "services/team-a/catalog.yaml", true},
		{"no match different prefix", []string{"services/**/*.yaml"}, "other/team-a/catalog.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &GitHubSource{Files: tt.patterns}
			got := src.matchesFilePatterns(tt.path)
			if got != tt.want {
				t.Errorf("matchesFilePatterns(%q) with patterns %v = %v, want %v", tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestGitHubSource_NewGitHubSource(t *testing.T) {
	cfg := &config.GitHubSourceConfig{
		Token:    "test-token",
		Owner:    "myorg",
		Repos:    []string{"repo1", "repo2"},
		Files:    []string{"*.yaml"},
		Ref:      "develop",
		Archived: true,
	}

	src := NewGitHubSource(cfg)
	if src.Token != "test-token" {
		t.Errorf("expected token=test-token, got %s", src.Token)
	}
	if src.Owner != "myorg" {
		t.Errorf("expected owner=myorg, got %s", src.Owner)
	}
	if len(src.Repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(src.Repos))
	}
	if src.Ref != "develop" {
		t.Errorf("expected ref=develop, got %s", src.Ref)
	}
	if !src.IncludeArchived {
		t.Error("expected IncludeArchived=true")
	}
	if src.Name() != "github" {
		t.Errorf("expected name=github, got %s", src.Name())
	}
}
