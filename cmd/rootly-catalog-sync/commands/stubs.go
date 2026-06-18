package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
)

// SetVersionInfo is called from main to inject ldflags values.
func SetVersionInfo(version, commit string) {
	buildVersion = version
	buildCommit = commit
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("rootly-catalog-sync %s (commit: %s)\n", buildVersion, buildCommit)
	},
}
