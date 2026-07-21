package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
	"github.com/rootlyhq/rootly-catalog-sync/client"
	"github.com/rootlyhq/rootly-catalog-sync/config"
	"github.com/rootlyhq/rootly-catalog-sync/mapping"
	"github.com/rootlyhq/rootly-catalog-sync/oauth"
	"github.com/rootlyhq/rootly-catalog-sync/source"
	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

func resolveAuth(opts []client.Option) (*client.Client, error) {
	// Priority 1: API key
	if apiKey := os.Getenv("ROOTLY_API_KEY"); apiKey != "" {
		return client.New(apiKey, opts...), nil
	}

	// Priority 2: OAuth tokens
	if oauth.HasTokens() {
		apiURL := os.Getenv("ROOTLY_API_URL")
		if apiURL == "" {
			apiURL = client.DefaultBaseURL
		}
		authBaseURL := oauth.DeriveAuthBaseURL(apiURL)

		clientID, scopes := oauth.LoadCachedRegistration()
		if clientID == "" {
			return nil, fmt.Errorf("OAuth tokens found but no client registration — run 'rootly-catalog-sync login'")
		}

		oauthCfg := oauth.NewConfig(authBaseURL, clientID, scopes)
		httpClient, err := oauth.NewHTTPClient(oauthCfg, http.DefaultTransport, client.UserAgent)
		if err != nil {
			return nil, fmt.Errorf("creating OAuth client: %w", err)
		}

		opts = append(opts, client.WithHTTPClient(httpClient))
		return client.New("", opts...), nil
	}

	return nil, fmt.Errorf("no authentication — set ROOTLY_API_KEY or run 'rootly-catalog-sync login'")
}

func initRuntime() (*config.Config, *client.Client, string, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("loading config: %w", err)
	}
	var opts []client.Option
	if u := os.Getenv("ROOTLY_API_URL"); u != "" {
		opts = append(opts, client.WithBaseURL(u))
	}
	if p := os.Getenv("ROOTLY_API_PATH"); p != "" {
		opts = append(opts, client.WithAPIPath(p))
	}
	cl, err := resolveAuth(opts)
	if err != nil {
		return nil, nil, "", err
	}
	return cfg, cl, filepath.Dir(configPath), nil
}

func loadSources(ctx context.Context, pipeline config.Pipeline, baseDir string) ([]source.Entry, error) {
	var all []source.Entry
	for _, srcCfg := range pipeline.Sources {
		src, err := source.FromConfig(srcCfg, baseDir)
		if err != nil {
			return nil, fmt.Errorf("creating source: %w", err)
		}
		entries, err := src.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading source %s: %w", src.Name(), err)
		}
		all = append(all, entries...)
	}
	return all, nil
}

func checkBulkResult(result *client.BulkResult, expected int, action string) error {
	if len(result.Errors) > 0 {
		return fmt.Errorf("bulk %s: %d succeeded, %d errors: %v", action, result.Succeeded, len(result.Errors), result.Errors)
	}
	if result.Succeeded != expected {
		return fmt.Errorf("bulk %s: expected %d succeeded, got %d", action, expected, result.Succeeded)
	}
	return nil
}

func newApplier(cl *client.Client) *catalogsync.Applier {
	return &catalogsync.Applier{
		BulkUpsert: func(ctx context.Context, catalogID string, ents []catalog.DesiredEntity) error {
			result, err := cl.BulkUpsert(ctx, catalogID, ents)
			if err != nil {
				return err
			}
			return checkBulkResult(result, len(ents), "upsert")
		},
		BulkDelete: func(ctx context.Context, catalogID string, externalIDs []string) error {
			_, err := cl.BulkDelete(ctx, catalogID, externalIDs)
			return err
		},
	}
}

func newNativeApplier(cl *client.Client, resourceType string) *catalogsync.Applier {
	return &catalogsync.Applier{
		BulkUpsert: func(ctx context.Context, _ string, ents []catalog.DesiredEntity) error {
			result, err := cl.BulkUpsertNative(ctx, resourceType, ents)
			if err != nil {
				return err
			}
			return checkBulkResult(result, len(ents), "upsert")
		},
		BulkDelete: func(ctx context.Context, _ string, externalIDs []string) error {
			_, err := cl.BulkDeleteNative(ctx, resourceType, externalIDs)
			return err
		},
	}
}

func applierForOutput(cl *client.Client, out config.Output) *catalogsync.Applier {
	if client.IsNativeResource(out.Type) {
		return newNativeApplier(cl, out.Type)
	}
	return newApplier(cl)
}

type reconcileOpts struct {
	skipSafety bool
}

type reconcileOption func(*reconcileOpts)

func withSkipSafety() reconcileOption {
	return func(o *reconcileOpts) { o.skipSafety = true }
}

// PlanResult holds the reconciliation plan for a single catalog output.
type PlanResult struct {
	Plan      *catalogsync.Plan
	CatalogID string
	Output    config.Output
}

// reconcileAll runs the full pipeline: load sources, map entries, ensure catalog/fields,
// list live entities, diff, and safety check. Returns a PlanResult per output.
func reconcileAll(ctx context.Context, cfg *config.Config, cl *client.Client, baseDir string, allowPrune bool, pruneThreshold float64, opts ...reconcileOption) ([]PlanResult, error) {
	var ro reconcileOpts
	for _, o := range opts {
		o(&ro)
	}

	var results []PlanResult
	for _, pipeline := range cfg.Pipelines {
		allEntries, err := loadSources(ctx, pipeline, baseDir)
		if err != nil {
			return nil, err
		}
		for _, out := range pipeline.Outputs {
			desired, err := mapping.MapEntries(allEntries, out)
			if err != nil {
				return nil, fmt.Errorf("mapping entries: %w", err)
			}

			if client.IsNativeResource(out.Type) {
				// Native resource path: skip catalog/field setup.
				live, err := cl.ListNativeResources(ctx, out.Type)
				if err != nil {
					return nil, fmt.Errorf("listing %ss: %w", out.Type, err)
				}
				plan := catalogsync.Diff(out.Type, out.Type, live, desired, allowPrune)
				if !ro.skipSafety {
					minKeep := 0
					if out.Type == "environment" || out.Type == "team" {
						minKeep = 1
					}
					if err := catalogsync.CheckSafety(plan, len(live), len(desired), pruneThreshold, minKeep); err != nil {
						return nil, fmt.Errorf("safety check failed: %w", err)
					}
				}
				results = append(results, PlanResult{Plan: plan, CatalogID: out.Type, Output: out})
			} else {
				// Catalog entity path (original behavior).
				catalogID, err := cl.EnsureCatalog(ctx, client.CatalogSpec{Name: out.Catalog})
				if err != nil {
					return nil, fmt.Errorf("ensuring catalog %q: %w", out.Catalog, err)
				}
				if err := ensureOutputFields(ctx, cl, catalogID, out); err != nil {
					return nil, fmt.Errorf("ensuring fields: %w", err)
				}
				live, err := cl.ListEntities(ctx, catalogID)
				if err != nil {
					return nil, fmt.Errorf("listing entities: %w", err)
				}
				plan := catalogsync.Diff(out.Catalog, catalogID, live, desired, allowPrune)
				if !ro.skipSafety {
					if err := catalogsync.CheckSafety(plan, len(live), len(desired), pruneThreshold); err != nil {
						return nil, fmt.Errorf("safety check failed: %w", err)
					}
				}
				results = append(results, PlanResult{Plan: plan, CatalogID: catalogID, Output: out})
			}
		}
	}
	return results, nil
}

func ensureOutputFields(ctx context.Context, cl *client.Client, catalogID string, out config.Output) error {
	fieldSpecs := make([]client.FieldSpec, 0, len(out.Fields))
	for slug := range out.Fields {
		fieldSpecs = append(fieldSpecs, client.FieldSpec{Name: slug, Kind: "text"})
	}
	if len(fieldSpecs) > 0 {
		return cl.EnsureFields(ctx, catalogID, fieldSpecs)
	}
	return nil
}
