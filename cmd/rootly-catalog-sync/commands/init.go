package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/rootlyhq/rootly-catalog-sync/tui"
)

const sampleConfig = `version: 1
sync_id: services
pipelines:
  - sources:
      - local:
          files: ["catalog/*.yaml"]
    outputs:
      - catalog: "Services"
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          owner:
            value: "{{ .owner }}"
          tier:
            value: "{{ .tier }}"
`

var interactive bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a sample config file",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		msg := fmt.Sprintf("Created %s — edit it, then run `rootly-catalog-sync validate --config=%s`.\n", path, path)
		if !interactive && term.IsTerminal(int(os.Stdin.Fd())) {
			msg += "Tip: use `rootly-catalog-sync init --interactive` for a guided setup wizard.\n"
		}
		fmt.Print(msg)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&interactive, "interactive", false, "launch the guided setup wizard")
}
