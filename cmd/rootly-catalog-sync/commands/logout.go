package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/oauth"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored OAuth tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := oauth.ClearTokens(); err != nil {
			return fmt.Errorf("clearing tokens: %w", err)
		}
		fmt.Println("Logged out. OAuth tokens cleared.")
		return nil
	},
}
