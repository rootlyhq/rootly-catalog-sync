package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
	"github.com/rootlyhq/rootly-catalog-sync/client"
	"github.com/rootlyhq/rootly-catalog-sync/mapping"
	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

var explainCmd = &cobra.Command{
	Use:   "explain <external_id>",
	Short: "Trace one entry through source → mapping → diff",
	Args:  cobra.ExactArgs(1),
	RunE:  runExplain,
}

func runExplain(cmd *cobra.Command, args []string) error {
	targetID := args[0]

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

			var found *catalog.DesiredEntity
			for i, d := range desired {
				if d.ExternalID == targetID {
					found = &desired[i]
					break
				}
			}

			if found == nil {
				continue
			}

			if err := resolveReferenceFields(ctx, cl, out, desired); err != nil {
				return fmt.Errorf("resolving reference fields: %w", err)
			}

			fmt.Printf("Entry: %s\n", targetID)
			if client.IsNativeResource(out.Type) {
				fmt.Printf("  Type: %s\n", out.Type)
			} else {
				fmt.Printf("  Catalog: %s\n", out.Catalog)
			}
			fmt.Printf("  Name:    %s\n", found.Name)
			for k, v := range found.Fields {
				fmt.Printf("  Field %s: %s\n", k, v)
			}

			var live []catalog.LiveEntity
			var targetLabel string
			if client.IsNativeResource(out.Type) {
				live, err = cl.ListNativeResources(ctx, out.Type)
				if err != nil {
					return fmt.Errorf("listing %ss: %w", out.Type, err)
				}
				targetLabel = out.Type
			} else {
				catalogID, err := cl.EnsureCatalog(ctx, client.CatalogSpec{Name: out.Catalog})
				if err != nil {
					return fmt.Errorf("ensuring catalog %q: %w", out.Catalog, err)
				}
				live, err = cl.ListEntities(ctx, catalogID)
				if err != nil {
					return fmt.Errorf("listing entities: %w", err)
				}
				targetLabel = out.Catalog
			}

			reconciledDesired := []catalog.DesiredEntity{{
				ExternalID: found.ExternalID,
				Name:       found.Name,
				Fields:     found.Fields,
			}}

			plan := catalogsync.Diff(targetLabel, targetLabel, live, reconciledDesired, true)

			for _, ch := range plan.Changes {
				if ch.ExternalID != targetID {
					continue
				}
				fmt.Printf("\n  Status: %s\n", ch.Op)
				if len(ch.FieldDiffs) > 0 {
					fmt.Println("  Diffs:")
					for field, diff := range ch.FieldDiffs {
						fmt.Printf("    %s: %q → %q\n", field, diff[0], diff[1])
					}
				}
			}
			return nil
		}
	}

	fmt.Fprintf(os.Stderr, "external_id %q not found in any source.\n", targetID)
	return nil
}
