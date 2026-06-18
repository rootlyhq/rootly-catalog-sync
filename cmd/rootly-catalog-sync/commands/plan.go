package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Compute a sync plan showing what would change",
	RunE:  runPlan,
}

func runPlan(cmd *cobra.Command, args []string) error {
	cfg, cl, baseDir, err := initRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()

	results, err := reconcileAll(ctx, cfg, cl, baseDir, allowPrune, pruneThreshold)
	if err != nil {
		return err
	}

	for _, r := range results {
		if outputFormat == "json" {
			data, err := catalogsync.PlanToJSON(r.Plan)
			if err != nil {
				return fmt.Errorf("marshaling plan to JSON: %w", err)
			}
			fmt.Println(string(data))
		} else {
			catalogsync.FormatPlan(os.Stdout, r.Plan)
		}

		planPath := fmt.Sprintf(".rootly-catalog-sync-plan-%s.json", r.Output.Catalog)
		if err := catalogsync.SavePlan(r.Plan, planPath); err != nil {
			return fmt.Errorf("saving plan: %w", err)
		}
		if outputFormat != "json" {
			_, _ = fmt.Fprintf(os.Stdout, "\nPlan saved to %s\n", planPath)
		}
	}
	return nil
}
