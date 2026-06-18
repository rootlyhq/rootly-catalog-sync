package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

type RunOptions struct {
	Plan       *catalogsync.Plan
	ApplyFn    func(*catalogsync.Plan) error
	SourceInfo string
}

func Run(opts RunOptions) error {
	m := New(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
