package source

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/google/go-github/v72/github"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

type GitHubSource struct {
	Token           string
	Owner           string
	Repos           []string
	Files           []string
	Ref             string
	IncludeArchived bool
}

func NewGitHubSource(cfg *config.GitHubSourceConfig) *GitHubSource {
	return &GitHubSource{
		Token:           cfg.Token,
		Owner:           cfg.Owner,
		Repos:           cfg.Repos,
		Files:           cfg.Files,
		Ref:             cfg.Ref,
		IncludeArchived: cfg.Archived,
	}
}

func (s *GitHubSource) Name() string { return "github" }

func (s *GitHubSource) Load(ctx context.Context) ([]Entry, error) {
	client := s.newClient()

	repos, err := s.resolveRepos(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("resolving repos: %w", err)
	}

	repos = s.filterRepos(repos)

	type result struct {
		entries []Entry
		err     error
	}

	sem := make(chan struct{}, 10)
	results := make([]result, len(repos))
	var wg sync.WaitGroup

	for i, repo := range repos {
		wg.Add(1)
		go func(i int, repo *github.Repository) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			entries, err := s.processRepo(ctx, client, repo)
			results[i] = result{entries: entries, err: err}
		}(i, repo)
	}

	wg.Wait()

	var allEntries []Entry
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		allEntries = append(allEntries, r.entries...)
	}

	return allEntries, nil
}

func (s *GitHubSource) newClient() *github.Client {
	client := github.NewClient(nil)
	if s.Token != "" {
		client = client.WithAuthToken(s.Token)
	}
	return client
}

func (s *GitHubSource) resolveRepos(ctx context.Context, client *github.Client) ([]*github.Repository, error) {
	if len(s.Repos) == 0 || (len(s.Repos) == 1 && s.Repos[0] == "*") {
		return s.listAllOrgRepos(ctx, client)
	}

	var repos []*github.Repository
	for _, name := range s.Repos {
		repo, _, err := client.Repositories.Get(ctx, s.Owner, name)
		if err != nil {
			return nil, fmt.Errorf("getting repo %s/%s: %w", s.Owner, name, err)
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

func (s *GitHubSource) listAllOrgRepos(ctx context.Context, client *github.Client) ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, s.Owner, opts)
		if err != nil {
			return nil, fmt.Errorf("listing org repos: %w", err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allRepos, nil
}

func (s *GitHubSource) filterRepos(repos []*github.Repository) []*github.Repository {
	var filtered []*github.Repository
	for _, repo := range repos {
		if repo.GetFork() {
			continue
		}
		if repo.GetArchived() && !s.IncludeArchived {
			continue
		}
		filtered = append(filtered, repo)
	}
	return filtered
}

func (s *GitHubSource) processRepo(ctx context.Context, client *github.Client, repo *github.Repository) ([]Entry, error) {
	repoName := repo.GetName()
	ref := s.Ref
	if ref == "" {
		ref = repo.GetDefaultBranch()
	}

	tree, resp, err := client.Git.GetTree(ctx, s.Owner, repoName, ref, true)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			fmt.Fprintf(os.Stderr, "warning: repo %s/%s has no tree at ref %q, skipping\n", s.Owner, repoName, ref)
			return nil, nil
		}
		return nil, fmt.Errorf("getting tree for %s/%s: %w", s.Owner, repoName, err)
	}

	var allEntries []Entry
	for _, te := range tree.Entries {
		if te.GetType() != "blob" {
			continue
		}
		path := te.GetPath()
		if !s.matchesFilePatterns(path) {
			continue
		}

		blob, _, err := client.Git.GetBlob(ctx, s.Owner, repoName, te.GetSHA())
		if err != nil {
			return nil, fmt.Errorf("getting blob %s in %s/%s: %w", path, s.Owner, repoName, err)
		}

		content, err := decodeBlob(blob)
		if err != nil {
			return nil, fmt.Errorf("decoding blob %s in %s/%s: %w", path, s.Owner, repoName, err)
		}

		entries, err := Parse(content)
		if err != nil {
			return nil, fmt.Errorf("parsing %s in %s/%s: %w", path, s.Owner, repoName, err)
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

func (s *GitHubSource) matchesFilePatterns(path string) bool {
	name := filepath.Base(path)
	for _, pattern := range s.Files {
		// doublestar.Match supports ** for recursive matching and is
		// backward-compatible with filepath.Match single-* patterns.
		if matched, _ := doublestar.Match(pattern, path); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

func decodeBlob(blob *github.Blob) ([]byte, error) {
	encoding := blob.GetEncoding()
	content := blob.GetContent()

	switch encoding {
	case "base64":
		cleaned := strings.ReplaceAll(content, "\n", "")
		return base64.StdEncoding.DecodeString(cleaned)
	case "utf-8", "":
		return []byte(content), nil
	default:
		return nil, fmt.Errorf("unsupported blob encoding: %s", encoding)
	}
}
