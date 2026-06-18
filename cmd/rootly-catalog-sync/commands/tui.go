package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/config"
	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
	"github.com/rootlyhq/rootly-catalog-sync/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive plan/apply — review and selectively apply changes",
	Long:  "Launch an interactive terminal UI to review planned changes and selectively apply them, like git add -p for your catalog.",
	RunE:  runTUI,
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, cl, baseDir, err := initRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()

	results, err := reconcileAll(ctx, cfg, cl, baseDir, allowPrune, pruneThreshold)
	if err != nil {
		return err
	}

	sourceInfo := fmt.Sprintf("%s • %d source(s) from %d pipeline(s)", configPath, totalEntries(cfg), len(cfg.Pipelines))

	for _, r := range results {
		applier := newApplier(cl)
		applyFn := func(p *catalogsync.Plan) error {
			return applier.Apply(ctx, p)
		}

		if err := tui.Run(tui.RunOptions{
			Plan:       r.Plan,
			ApplyFn:    applyFn,
			SourceInfo: sourceInfo,
		}); err != nil {
			return fmt.Errorf("tui: %w", err)
		}
	}
	return nil
}

func totalEntries(cfg *config.Config) int {
	total := 0
	for _, p := range cfg.Pipelines {
		for range p.Sources {
			total++
		}
	}
	return total
}
