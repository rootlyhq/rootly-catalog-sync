package source

import (
	"fmt"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

func FromConfig(cfg config.SourceConfig, baseDir string) (Source, error) {
	count := 0
	if cfg.Inline != nil {
		count++
	}
	if cfg.Local != nil {
		count++
	}
	if cfg.GitHub != nil {
		count++
	}
	if cfg.Exec != nil {
		count++
	}
	if cfg.Backstage != nil {
		count++
	}
	if cfg.GraphQL != nil {
		count++
	}
	if cfg.CSV != nil {
		count++
	}
	if cfg.URL != nil {
		count++
	}
	if cfg.HTTP != nil {
		count++
	}

	if count == 0 {
		return nil, fmt.Errorf("no source type configured")
	}
	if count > 1 {
		return nil, fmt.Errorf("multiple source types configured; exactly one is required")
	}

	switch {
	case cfg.Inline != nil:
		return NewInlineSource(cfg.Inline.Entries), nil
	case cfg.Local != nil:
		return NewLocalSource(cfg.Local.Files, baseDir), nil
	case cfg.GitHub != nil:
		return NewGitHubSource(cfg.GitHub), nil
	case cfg.Exec != nil:
		return NewExecSource(cfg.Exec.Command, cfg.Exec.Args), nil
	case cfg.Backstage != nil:
		return NewBackstageSource(cfg.Backstage), nil
	case cfg.GraphQL != nil:
		return NewGraphQLSource(cfg.GraphQL), nil
	case cfg.CSV != nil:
		return NewCSVSource(cfg.CSV.Files, cfg.CSV.Delimiter, baseDir), nil
	case cfg.URL != nil:
		return NewURLSource(cfg.URL), nil
	case cfg.HTTP != nil:
		return NewHTTPSource(cfg.HTTP), nil
	default:
		return nil, fmt.Errorf("source type not yet supported")
	}
}
