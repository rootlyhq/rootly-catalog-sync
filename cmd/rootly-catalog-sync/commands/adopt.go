package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
	"github.com/rootlyhq/rootly-catalog-sync/client"
	"github.com/rootlyhq/rootly-catalog-sync/mapping"
)

var adoptMatch string

var adoptCmd = &cobra.Command{
	Use:   "adopt",
	Short: "Claim existing UI entries under sync management",
	Long:  "Match existing catalog entries by name and stamp them with external_id so the next sync updates instead of duplicating.",
	RunE:  runAdopt,
}

func init() {
	adoptCmd.Flags().StringVar(&adoptMatch, "match", "name", "match strategy: name")
}

func runAdopt(cmd *cobra.Command, args []string) error {
	cfg, cl, baseDir, err := initRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, pipeline := range cfg.Pipelines {
		allEntries, err := loadSources(ctx, pipeline, baseDir)
		if err != nil {
			return err
		}

		for _, out := range pipeline.Outputs {
			desired, err := mapping.MapEntries(allEntries, out)
			if err != nil {
				return fmt.Errorf("mapping entries: %w", err)
			}

			catalogID, err := cl.EnsureCatalog(ctx, client.CatalogSpec{Name: out.Catalog})
			if err != nil {
				return fmt.Errorf("ensuring catalog %q: %w", out.Catalog, err)
			}

			live, err := cl.ListEntities(ctx, catalogID)
			if err != nil {
				return fmt.Errorf("listing entities: %w", err)
			}

			liveByName := make(map[string]catalog.LiveEntity, len(live))
			for _, e := range live {
				liveByName[strings.ToLower(e.Name)] = e
			}

			var toAdopt []catalog.DesiredEntity
			for _, d := range desired {
				existing, found := liveByName[strings.ToLower(d.Name)]
				if !found {
					continue
				}
				if existing.ExternalID != "" {
					fmt.Printf("  skip  %q (already has external_id %q)\n", d.Name, existing.ExternalID)
					continue
				}
				toAdopt = append(toAdopt, catalog.DesiredEntity{
					ExternalID: d.ExternalID,
					Name:       d.Name,
					Fields:     d.Fields,
				})
				fmt.Printf("  adopt  %q → external_id=%q\n", d.Name, d.ExternalID)
			}

			if len(toAdopt) == 0 {
				fmt.Printf("Catalog %q: nothing to adopt.\n", out.Catalog)
				continue
			}

			if dryRun {
				fmt.Printf("Catalog %q: would adopt %d entries (dry-run).\n", out.Catalog, len(toAdopt))
				continue
			}

			if _, err := cl.BulkUpsert(ctx, catalogID, toAdopt); err != nil {
				return fmt.Errorf("adopting entities: %w", err)
			}

			fmt.Printf("Catalog %q: adopted %d entries.\n", out.Catalog, len(toAdopt))
		}
	}
	return nil
}
