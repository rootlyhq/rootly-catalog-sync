package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/config"
	"github.com/rootlyhq/rootly-catalog-sync/source"
)

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "Source inspection commands",
}

var sourcesInspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Dump raw entries from sources before mapping",
	RunE:  runSourcesInspect,
}

func init() {
	sourcesCmd.AddCommand(sourcesInspectCmd)
}

func runSourcesInspect(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx := context.Background()
	baseDir := filepath.Dir(configPath)

	for i, pipeline := range cfg.Pipelines {
		for j, srcCfg := range pipeline.Sources {
			src, err := source.FromConfig(srcCfg, baseDir)
			if err != nil {
				return fmt.Errorf("pipeline[%d] source[%d]: %w", i, j, err)
			}
			entries, err := src.Load(ctx)
			if err != nil {
				return fmt.Errorf("loading %s: %w", src.Name(), err)
			}
			fmt.Printf("--- %s (%d entries) ---\n", src.Name(), len(entries))
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			for _, e := range entries {
				_ = enc.Encode(e)
			}
		}
	}
	return nil
}
