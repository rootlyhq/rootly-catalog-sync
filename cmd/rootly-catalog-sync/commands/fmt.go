package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/config"
)

var fmtCmd = &cobra.Command{
	Use:   "fmt",
	Short: "Canonicalize config file formatting",
	RunE:  runFmt,
}

func runFmt(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	data, err := config.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Formatted %s\n", configPath)
	return nil
}
