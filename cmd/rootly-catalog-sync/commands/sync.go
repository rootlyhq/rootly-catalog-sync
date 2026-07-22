package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Plan and apply in one step",
	RunE:  runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, cl, baseDir, err := initRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()

	applyEach := func(ctx context.Context, r PlanResult) error {
		if outputFormat == "json" {
			data, err := catalogsync.PlanToJSON(r.Plan)
			if err != nil {
				return fmt.Errorf("marshaling plan to JSON: %w", err)
			}
			fmt.Println(string(data))
		} else {
			catalogsync.FormatPlan(os.Stdout, r.Plan)
		}

		if r.Plan.Counts.IsNoop() {
			if outputFormat != "json" {
				fmt.Println("Already in sync.")
			}
			return nil
		}

		if dryRun {
			return nil
		}

		applier := applierForOutput(cl, r.Output)
		if err := applier.Apply(ctx, r.Plan); err != nil {
			return fmt.Errorf("applying: %w", err)
		}

		if outputFormat != "json" {
			fmt.Printf("Applied: %d created, %d updated, %d deleted.\n",
				r.Plan.Counts.Create, r.Plan.Counts.Update, r.Plan.Counts.Delete)
		}
		return nil
	}

	opts := []reconcileOption{withApplyEach(applyEach)}
	if !dryRun {
		opts = append(opts, withCanMutate())
	}
	_, err = reconcileAll(ctx, cfg, cl, baseDir, allowPrune, pruneThreshold, opts...)
	return err
}
