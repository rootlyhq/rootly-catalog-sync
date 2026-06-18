package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/client"
	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

var applyCmd = &cobra.Command{
	Use:   "apply [plan-file]",
	Short: "Apply a saved plan",
	RunE:  runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	var planPath string
	if len(args) > 0 {
		planPath = args[0]
	} else {
		return fmt.Errorf("plan file path required (run `plan` first)")
	}

	plan, err := catalogsync.LoadPlan(planPath)
	if err != nil {
		return fmt.Errorf("loading plan: %w", err)
	}

	if plan.Counts.IsNoop() {
		fmt.Println("Nothing to do — already in sync.")
		return nil
	}

	ctx := context.Background()
	var opts []client.Option
	if u := os.Getenv("ROOTLY_API_URL"); u != "" {
		opts = append(opts, client.WithBaseURL(u))
	}
	if p := os.Getenv("ROOTLY_API_PATH"); p != "" {
		opts = append(opts, client.WithAPIPath(p))
	}
	cl, err := resolveAuth(opts)
	if err != nil {
		return err
	}

	// Validate that the plan still matches live state.
	live, err := cl.ListEntities(ctx, plan.CatalogID)
	if err != nil {
		return fmt.Errorf("fetching live entities for freshness check: %w", err)
	}
	if stale := catalogsync.ValidatePlanFreshness(plan, live); len(stale) > 0 {
		fmt.Fprintf(os.Stderr, "Plan is stale — live state has changed since the plan was created:\n")
		for _, s := range stale {
			fmt.Fprintf(os.Stderr, "  • %s: %s\n", s.ExternalID, s.Reason)
		}
		if !forceApply {
			return fmt.Errorf("plan is stale (re-run `plan` to get a fresh plan, or use --force to apply anyway)")
		}
		fmt.Fprintf(os.Stderr, "Proceeding anyway (--force).\n")
	}

	if dryRun {
		catalogsync.FormatPlan(os.Stdout, plan)
		return nil
	}

	applier := newApplier(cl)

	if err := applier.Apply(ctx, plan); err != nil {
		return fmt.Errorf("applying plan: %w", err)
	}

	fmt.Printf("Applied: %d created, %d updated, %d deleted.\n",
		plan.Counts.Create, plan.Counts.Update, plan.Counts.Delete)
	return nil
}
