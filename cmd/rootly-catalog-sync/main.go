package main

import (
	"fmt"
	"os"

	"github.com/rootlyhq/rootly-catalog-sync/cmd/rootly-catalog-sync/commands"
)

var (
	version = "dev"
	commit  = "none"
)

func init() {
	commands.SetVersionInfo(version, commit)
}

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
