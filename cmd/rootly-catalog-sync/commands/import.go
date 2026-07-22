package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
	"github.com/rootlyhq/rootly-catalog-sync/client"
	"github.com/rootlyhq/rootly-catalog-sync/mapping"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "One-shot seed/migration (no external_id, no prune)",
	Long:  "Create or update entries without stamping external_id, leaving them fully editable and never prunable.",
	RunE:  runImport,
}

func runImport(cmd *cobra.Command, args []string) error {
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

			if err := ensureOutputFields(ctx, cl, catalogID, out); err != nil {
				return fmt.Errorf("ensuring fields: %w", err)
			}

			if err := resolveReferenceFields(ctx, cl, out, desired); err != nil {
				return fmt.Errorf("resolving reference fields: %w", err)
			}

			importEnts := make([]catalog.DesiredEntity, len(desired))
			for i, d := range desired {
				importEnts[i] = catalog.DesiredEntity{
					Name:   d.Name,
					Fields: d.Fields,
				}
			}

			if dryRun {
				fmt.Printf("Catalog %q: would import %d entries (dry-run).\n", out.Catalog, len(importEnts))
				continue
			}

			if _, err := cl.BulkUpsert(ctx, catalogID, importEnts); err != nil {
				return fmt.Errorf("importing entities: %w", err)
			}

			fmt.Printf("Catalog %q: imported %d entries.\n", out.Catalog, len(importEnts))
		}
	}
	return nil
}
