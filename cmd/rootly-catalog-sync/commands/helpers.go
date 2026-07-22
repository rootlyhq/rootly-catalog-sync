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
	if err := config.Validate(cfg); err != nil {
		return nil, nil, "", fmt.Errorf("invalid config: %w", err)
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
	canMutate  bool
	applyEach  func(ctx context.Context, r PlanResult) error
}

type reconcileOption func(*reconcileOpts)

func withSkipSafety() reconcileOption {
	return func(o *reconcileOpts) { o.skipSafety = true }
}

func withCanMutate() reconcileOption {
	return func(o *reconcileOpts) { o.canMutate = true }
}

func withApplyEach(fn func(ctx context.Context, r PlanResult) error) reconcileOption {
	return func(o *reconcileOpts) { o.applyEach = fn }
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

		var pipelineResults []PlanResult
		for _, out := range pipeline.Outputs {
			desired, err := mapping.MapEntries(allEntries, out)
			if err != nil {
				return nil, fmt.Errorf("mapping entries: %w", err)
			}

			if client.IsNativeResource(out.Type) {
				nativeProps, err := ensureNativeOutputFields(ctx, cl, out, ro.canMutate)
				if err != nil {
					return nil, fmt.Errorf("ensuring native properties: %w", err)
				}
				if err := resolveReferenceFields(ctx, cl, out, desired); err != nil {
					return nil, fmt.Errorf("resolving reference fields: %w", err)
				}
				propIDToSlug := client.NativePropertyIDToSlug(nativeProps)
				live, err := cl.ListNativeResourcesWithProps(ctx, out.Type, propIDToSlug)
				if err != nil {
					return nil, fmt.Errorf("listing %ss: %w", out.Type, err)
				}
				plan := catalogsync.Diff(out.Type, out.Type, live, desired, allowPrune)
				if !ro.skipSafety {
					minKeep := 0
					if out.Type == client.NativeEnvironment || out.Type == client.NativeTeam {
						minKeep = 1
					}
					if err := catalogsync.CheckSafety(plan, len(live), len(desired), pruneThreshold, minKeep); err != nil {
						return nil, fmt.Errorf("safety check failed: %w", err)
					}
				}
				pipelineResults = append(pipelineResults, PlanResult{Plan: plan, CatalogID: out.Type, Output: out})
			} else {
				catalogID, err := cl.EnsureCatalog(ctx, client.CatalogSpec{Name: out.Catalog})
				if err != nil {
					return nil, fmt.Errorf("ensuring catalog %q: %w", out.Catalog, err)
				}
				if err := ensureOutputFields(ctx, cl, catalogID, out); err != nil {
					return nil, fmt.Errorf("ensuring fields: %w", err)
				}
				if err := resolveReferenceFields(ctx, cl, out, desired); err != nil {
					return nil, fmt.Errorf("resolving reference fields: %w", err)
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
				pipelineResults = append(pipelineResults, PlanResult{Plan: plan, CatalogID: catalogID, Output: out})
			}
		}

		if ro.applyEach != nil {
			for _, pr := range pipelineResults {
				if err := ro.applyEach(ctx, pr); err != nil {
					return nil, err
				}
			}
		}
		results = append(results, pipelineResults...)
	}
	return results, nil
}

func ensureOutputFields(ctx context.Context, cl *client.Client, catalogID string, out config.Output) error {
	fieldSpecs := make([]client.FieldSpec, 0, len(out.Fields))
	for slug, fv := range out.Fields {
		kind := fv.EffectiveKind()
		spec := client.FieldSpec{Name: slug, Kind: kind}
		if kind == config.KindReference && fv.Catalog != "" {
			catalogs, err := cl.ListCatalogs(ctx)
			if err != nil {
				return fmt.Errorf("listing catalogs for field %q: %w", slug, err)
			}
			var refCatalogID string
			for _, c := range catalogs {
				if c.Name == fv.Catalog {
					refCatalogID = c.ID
					break
				}
			}
			if refCatalogID == "" {
				return fmt.Errorf("field %q references catalog %q which does not exist", slug, fv.Catalog)
			}
			spec.KindCatalogID = refCatalogID
		}
		fieldSpecs = append(fieldSpecs, spec)
	}
	if len(fieldSpecs) > 0 {
		return cl.EnsureFields(ctx, catalogID, fieldSpecs)
	}
	return nil
}

func ensureNativeOutputFields(ctx context.Context, cl *client.Client, out config.Output, canMutate bool) ([]client.NativePropertyInfo, error) {
	props, err := cl.ListNativeProperties(ctx, out.Type)
	if err != nil {
		return nil, err
	}

	known := client.NativeKnownAttrs(out.Type)
	var customFields []string
	for slug := range out.Fields {
		if !known[slug] {
			customFields = append(customFields, slug)
		}
	}
	if len(customFields) == 0 {
		return props, nil
	}

	propMap := client.NativePropertyMap(props)

	for _, slug := range customFields {
		fv := out.Fields[slug]
		kind := fv.EffectiveKind()
		if existing, exists := propMap[slug]; exists {
			if existing.Kind != kind {
				return nil, fmt.Errorf("property %q on %s has kind %q but config declares kind %q",
					slug, out.Type, existing.Kind, kind)
			}
			if kind == config.KindReference && fv.Catalog != "" && existing.KindCatalogID != "" {
				catalogs, err := cl.ListCatalogs(ctx)
				if err != nil {
					return nil, fmt.Errorf("listing catalogs for property %q validation: %w", slug, err)
				}
				var catalogID string
				for _, c := range catalogs {
					if c.Name == fv.Catalog {
						catalogID = c.ID
						break
					}
				}
				if catalogID == "" {
					return nil, fmt.Errorf("property %q references catalog %q which does not exist", slug, fv.Catalog)
				}
				if existing.KindCatalogID != catalogID {
					return nil, fmt.Errorf("property %q on %s references catalog %s but config declares catalog %q (%s)",
						slug, out.Type, existing.KindCatalogID, fv.Catalog, catalogID)
				}
			}
			continue
		}
		if kind != config.KindText {
			available := make([]string, 0, len(props))
			for _, p := range props {
				available = append(available, fmt.Sprintf("%s (%s)", p.Slug, p.Kind))
			}
			return nil, fmt.Errorf("property %q (kind: %s) not found on %s — must be created in the Rootly UI first\n  Available properties: %v",
				slug, kind, out.Type, available)
		}
		if !canMutate {
			return nil, fmt.Errorf("property %q does not exist on %s — run sync to auto-create it", slug, out.Type)
		}
		if err := cl.EnsureNativeProperty(ctx, out.Type, slug, config.KindText, ""); err != nil {
			return nil, fmt.Errorf("auto-creating text property %q: %w", slug, err)
		}
		props = append(props, client.NativePropertyInfo{Slug: slug, Kind: config.KindText})
	}
	return props, nil
}

func resolveReferenceFields(ctx context.Context, cl *client.Client, out config.Output, desired []catalog.DesiredEntity) error {
	refCatalogs := make(map[string]string)
	for slug, fv := range out.Fields {
		if fv.Kind == config.KindReference && fv.Catalog != "" {
			refCatalogs[slug] = fv.Catalog
		}
	}
	if len(refCatalogs) == 0 {
		return nil
	}

	catalogs, err := cl.ListCatalogs(ctx)
	if err != nil {
		return fmt.Errorf("listing catalogs: %w", err)
	}
	catalogNameToID := make(map[string]string, len(catalogs))
	for _, c := range catalogs {
		catalogNameToID[c.Name] = c.ID
	}

	catalogLookup := make(map[string]map[string]string) // catalogName → (entityName → entityID)
	for _, catalogName := range refCatalogs {
		if _, done := catalogLookup[catalogName]; done {
			continue
		}
		catalogID := catalogNameToID[catalogName]
		if catalogID == "" {
			return fmt.Errorf("referenced catalog %q not found — run sync to create it first", catalogName)
		}
		entities, err := cl.ListEntities(ctx, catalogID)
		if err != nil {
			return fmt.Errorf("listing entities in catalog %q: %w", catalogName, err)
		}
		nameToID := make(map[string]string, len(entities))
		for _, e := range entities {
			nameToID[e.Name] = e.ID
		}
		catalogLookup[catalogName] = nameToID
	}

	// Replace text values with entity UUIDs in desired entities.
	for i := range desired {
		for slug, catalogName := range refCatalogs {
			textVal := desired[i].Fields[slug]
			if textVal == "" {
				continue
			}
			nameToID := catalogLookup[catalogName]
			entityID, ok := nameToID[textVal]
			if !ok {
				available := make([]string, 0, len(nameToID))
				for name := range nameToID {
					available = append(available, name)
				}
				return fmt.Errorf("reference field %q: value %q not found in catalog %q\n  Available: %v",
					slug, textVal, catalogName, available)
			}
			desired[i].Fields[slug] = entityID
		}
	}
	return nil
}
