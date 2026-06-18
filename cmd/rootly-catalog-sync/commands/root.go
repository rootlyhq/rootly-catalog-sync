package commands

import (
	"github.com/spf13/cobra"
)

var (
	configPath     string
	dryRun         bool
	allowPrune     bool
	pruneThreshold float64
	forceApply     bool
	outputFormat   string
)

var rootCmd = &cobra.Command{
	Use:   "rootly-catalog-sync",
	Short: "Sync external catalog data into Rootly",
	Long:  "A CLI tool that reconciles external sources of truth into Rootly's Catalog, keeping services, teams, and metadata in sync.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "rootly-catalog-sync.yaml", "path to config file")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "compute and show changes without applying")
	rootCmd.PersistentFlags().BoolVar(&allowPrune, "allow-prune", false, "allow deletion of entities absent from source")
	rootCmd.PersistentFlags().Float64Var(&pruneThreshold, "prune-threshold", 0.2, "maximum fraction of live entities that can be pruned (0.0–1.0)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "text", "output format: text or json")

	rootCmd.AddCommand(planCmd)
	applyCmd.Flags().BoolVar(&forceApply, "force", false, "apply even if the plan is stale")
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(sourcesCmd)
	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(adoptCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(fmtCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
