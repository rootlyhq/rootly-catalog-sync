package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/rootlyhq/rootly-catalog-sync/client"
	"github.com/rootlyhq/rootly-catalog-sync/config"
	"github.com/rootlyhq/rootly-catalog-sync/oauth"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment, auth, and connectivity",
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ok := true

	// Config
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("[FAIL] Config: %v\n", err)
		ok = false
	} else if err := config.Validate(cfg); err != nil {
		fmt.Printf("[FAIL] Config validation: %v\n", err)
		ok = false
	} else {
		fmt.Printf("[OK]   Config: %s\n", configPath)
	}

	// Auth
	var opts []client.Option
	if u := os.Getenv("ROOTLY_API_URL"); u != "" {
		opts = append(opts, client.WithBaseURL(u))
	}
	if p := os.Getenv("ROOTLY_API_PATH"); p != "" {
		opts = append(opts, client.WithAPIPath(p))
	}

	if apiKey := os.Getenv("ROOTLY_API_KEY"); apiKey != "" {
		fmt.Println("[OK]   Auth: ROOTLY_API_KEY (api-key)")
	} else if oauth.HasTokens() {
		fmt.Println("[OK]   Auth: OAuth tokens (oauth)")
	} else {
		fmt.Println("[FAIL] Auth: no auth — set ROOTLY_API_KEY or run 'rootly-catalog-sync login'")
		ok = false
	}

	// Connectivity
	cl, authErr := resolveAuth(opts)
	if authErr == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		catalogs, err := cl.ListCatalogs(ctx)
		if err != nil {
			fmt.Printf("[FAIL] API connectivity: %v\n", err)
			ok = false
		} else {
			fmt.Printf("[OK]   API connectivity: %d catalogs found\n", len(catalogs))
		}
	}

	if !ok {
		return fmt.Errorf("doctor found issues")
	}
	fmt.Println("\nAll checks passed.")
	return nil
}
