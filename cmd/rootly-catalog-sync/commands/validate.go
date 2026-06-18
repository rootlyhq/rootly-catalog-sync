package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate config file syntax and schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}
		if err := config.Validate(cfg); err != nil {
			return fmt.Errorf("validation error: %w", err)
		}
		fmt.Println("Config is valid.")
		return nil
	},
}
