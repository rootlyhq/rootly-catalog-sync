package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/client"
	"github.com/rootlyhq/rootly-catalog-sync/config"
	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

var watchInterval time.Duration

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Continuous reconcile loop",
	RunE:  runWatch,
}

func init() {
	watchCmd.Flags().DurationVar(&watchInterval, "interval", 5*time.Minute, "sync interval")
}

func runWatch(cmd *cobra.Command, args []string) error {
	cfg, cl, baseDir, err := initRuntime()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("Watching every %s (ctrl-C to stop)\n", watchInterval)

	for {
		if err := runSyncOnce(ctx, cfg, cl, baseDir); err != nil {
			fmt.Fprintf(os.Stderr, "sync error: %v\n", err)
		}

		select {
		case <-ctx.Done():
			fmt.Println("\nStopping watch.")
			return nil
		case <-time.After(watchInterval):
		}
	}
}

func runSyncOnce(ctx context.Context, cfg *config.Config, cl *client.Client, baseDir string) error {
	applyEach := func(ctx context.Context, r PlanResult) error {
		catalogName := r.Output.Catalog
		if catalogName == "" {
			catalogName = r.Output.Type
		}
		if r.Plan.Counts.IsNoop() {
			fmt.Printf("[%s] %s: in sync\n", time.Now().Format("15:04:05"), catalogName)
			return nil
		}

		catalogsync.FormatPlan(os.Stdout, r.Plan)

		applier := applierForOutput(cl, r.Output)
		if err := applier.Apply(ctx, r.Plan); err != nil {
			return fmt.Errorf("applying: %w", err)
		}

		fmt.Printf("[%s] %s: %d created, %d updated, %d deleted\n",
			time.Now().Format("15:04:05"), catalogName,
			r.Plan.Counts.Create, r.Plan.Counts.Update, r.Plan.Counts.Delete)
		return nil
	}

	_, err := reconcileAll(ctx, cfg, cl, baseDir, allowPrune, pruneThreshold, withApplyEach(applyEach), withCanMutate())
	return err
}
