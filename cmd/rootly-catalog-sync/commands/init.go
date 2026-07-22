package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rootlyhq/rootly-catalog-sync/tui"
)

const sampleConfig = `version: 2

sync:
  - from:
      local:
        files: ["catalog/*.yaml"]
    to: Services
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"
      owner: "{{ .owner }}"
      tier: "{{ .tier }}"
`

var demoFiles = map[string]string{
	"rootly-catalog-sync.yaml": `version: 2

sync:
  # 1. Sync tier definitions as a reference catalog
  - from:
      local:
        files: ["catalog/tiers.yml"]
    to: Tiers
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"

  # 2. Sync services with a tier reference
  - from:
      local:
        files: ["catalog/services.yaml"]
    to: service
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"
      description: "{{ .description }}"
      pagerduty_id: "{{ .pagerduty_id }}"

  # 3. Sync teams
  - from:
      local:
        files: ["catalog/teams.yaml"]
    to: team
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"
      description: "{{ .description }}"
`,
	"catalog/tiers.yml": `- id: tier-1
  name: "Tier 1"
- id: tier-2
  name: "Tier 2"
- id: tier-3
  name: "Tier 3"
`,
	"catalog/services.yaml": `- id: api-gateway
  name: API Gateway
  description: Edge proxy handling authentication, rate limiting, and request routing
  pagerduty_id: PABC123

- id: billing-engine
  name: Billing Engine
  description: Processes subscriptions, invoices, and payment reconciliation
  pagerduty_id: PDEF456

- id: feature-flags
  name: Feature Flags
  description: Runtime feature flag evaluation and audience targeting
  pagerduty_id: PGHI789
`,
	"catalog/teams.yaml": `- id: platform-eng
  name: Platform Engineering
  description: Owns infrastructure, CI/CD, and developer experience

- id: revenue-systems
  name: Revenue Systems
  description: Owns billing, payments, and subscription lifecycle

- id: growth-eng
  name: Growth Engineering
  description: Owns onboarding, activation, and experimentation
`,
}

var (
	interactive bool
	demo        bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a sample config file",
	Long:  "Create a sample config file. Use --demo to scaffold a complete working example with sample data files.",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	if demo {
		return runInitDemo()
	}

	var path, content string

	if interactive {
		result, err := tui.RunWizard()
		if err != nil {
			return err
		}
		path = result.Filename
		content = result.Content
	} else {
		path = "rootly-catalog-sync.yaml"
		content = sampleConfig
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	msg := fmt.Sprintf("Created %s — edit it, then run `rootly-catalog-sync validate`.\n", path)
	if !interactive && term.IsTerminal(int(os.Stdin.Fd())) {
		msg += "Tip: use `rootly-catalog-sync init --interactive` for a guided setup wizard.\n"
		msg += "     or `rootly-catalog-sync init --demo` for a complete working example.\n"
	}
	fmt.Print(msg)
	return nil
}

func runInitDemo() error {
	for path := range demoFiles {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists — remove it first or run from an empty directory", path)
		}
	}

	for path, content := range demoFiles {
		dir := filepath.Dir(path)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", dir, err)
			}
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		fmt.Printf("  created  %s\n", path)
	}

	fmt.Println()
	fmt.Println("Demo project ready! Try these commands:")
	fmt.Println()
	fmt.Println("  export ROOTLY_API_KEY=rootly_...")
	fmt.Println("  rootly-catalog-sync plan          # preview what would change")
	fmt.Println("  rootly-catalog-sync sync --dry-run # dry run (no changes)")
	fmt.Println("  rootly-catalog-sync sync           # apply changes")
	fmt.Println()
	fmt.Println("Edit the files in catalog/ to add your own data.")
	return nil
}

func init() {
	initCmd.Flags().BoolVar(&interactive, "interactive", false, "launch the guided setup wizard")
	initCmd.Flags().BoolVar(&demo, "demo", false, "scaffold a complete working example with sample data")
}
