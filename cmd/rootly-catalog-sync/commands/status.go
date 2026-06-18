package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

var failOnDrift bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status without making changes",
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&failOnDrift, "fail-on-drift", false, "exit with code 3 if drift is detected")
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, cl, baseDir, err := initRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()

	results, err := reconcileAll(ctx, cfg, cl, baseDir, true, pruneThreshold, withSkipSafety())
	if err != nil {
		return err
	}

	hasDrift := false
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
		if !r.Plan.Counts.IsNoop() {
			hasDrift = true
		}
	}

	if outputFormat != "json" {
		if hasDrift {
			fmt.Println("\nDrift detected.")
		} else {
			fmt.Println("\nIn sync.")
		}
	}
	if hasDrift && failOnDrift {
		os.Exit(3)
	}
	return nil
}
