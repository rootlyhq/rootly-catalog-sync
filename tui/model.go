package tui

import (
	"fmt"
	"sort"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	tea "github.com/charmbracelet/bubbletea"

	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

type state int

const (
	stateBrowse state = iota
	stateConfirm
	stateApplied
	stateSearch
)

const (
	keyCtrlC     = "ctrl+c"
	keyEnter     = "enter"
	keyEsc       = "esc"
	keyBackspace = "backspace"
)

type Model struct {
	plan        *catalogsync.Plan
	selected    map[int]bool
	cursor      int
	expanded    map[int]bool
	width       int
	height      int
	offset      int
	state       state
	applyErr    error
	applyFn     func(plan *catalogsync.Plan) error
	appliedPlan *catalogsync.Plan
	sourceInfo  string

	// Filter by op type: "" = all, "create"/"update"/"delete"/"noop"
	filter string
	// Filtered indices into plan.Changes
	filtered []int

	// Search
	searchQuery string
}

func New(opts RunOptions) Model {
	sel := make(map[int]bool)
	for i, c := range opts.Plan.Changes {
		if c.Op != catalogsync.OpNoop {
			sel[i] = true
		}
	}
	m := Model{
		plan:       opts.Plan,
		selected:   sel,
		cursor:     0,
		expanded:   make(map[int]bool),
		applyFn:    opts.ApplyFn,
		sourceInfo: opts.SourceInfo,
	}
	m.refilter()
	return m
}

func (m *Model) refilter() {
	m.filtered = nil
	for i, c := range m.plan.Changes {
		if m.filter != "" && c.Op != m.filter {
			continue
		}
		if m.searchQuery != "" {
			name := strings.ToLower(catalogsync.EntityName(c))
			eid := strings.ToLower(c.ExternalID)
			q := strings.ToLower(m.searchQuery)
			if !strings.Contains(name, q) && !strings.Contains(eid, q) {
				continue
			}
		}
		m.filtered = append(m.filtered, i)
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.offset = 0
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampOffset()
		return m, nil
	case tea.KeyMsg:
		switch m.state {
		case stateConfirm:
			return m.updateConfirm(msg)
		case stateApplied:
			return m.updateApplied(msg)
		case stateSearch:
			return m.updateSearch(msg)
		default:
			return m.updateBrowse(msg)
		}
	}
	return m, nil
}

func (m Model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.filtered)
	if n == 0 {
		switch msg.String() {
		case "q", keyCtrlC:
			return m, tea.Quit
		case "0":
			m.filter = ""
			m.refilter()
		case "/":
			m.state = stateSearch
			m.searchQuery = ""
		}
		return m, nil
	}

	switch msg.String() {
	case "q", keyCtrlC:
		return m, tea.Quit
	case "j", "down":
		if m.cursor < n-1 {
			m.cursor++
			m.ensureVisible()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
		}
	case " ":
		idx := m.filtered[m.cursor]
		m.selected[idx] = !m.selected[idx]
	case keyEnter:
		idx := m.filtered[m.cursor]
		m.expanded[idx] = !m.expanded[idx]
		m.clampOffset()
	case "a":
		for _, idx := range m.filtered {
			m.selected[idx] = true
		}
	case "n":
		for _, idx := range m.filtered {
			m.selected[idx] = false
		}
	case "A":
		if m.selectedCount() > 0 {
			m.state = stateConfirm
		}
	case "c":
		m.toggleFilter(catalogsync.OpCreate)
	case "u":
		m.toggleFilter(catalogsync.OpUpdate)
	case "d":
		m.toggleFilter(catalogsync.OpDelete)
	case "0":
		m.filter = ""
		m.refilter()
	case "/":
		m.state = stateSearch
		m.searchQuery = ""
	}
	return m, nil
}

func (m *Model) toggleFilter(op string) {
	if m.filter == op {
		m.filter = ""
	} else {
		m.filter = op
	}
	m.refilter()
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		filtered := m.buildFilteredPlan()
		m.applyErr = m.applyFn(filtered)
		m.appliedPlan = filtered
		m.state = stateApplied
		return m, nil
	case "n", "N", "q", keyEsc, keyCtrlC:
		m.state = stateBrowse
		return m, nil
	}
	return m, nil
}

func (m Model) updateApplied(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", keyCtrlC, keyEnter, keyEsc:
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, keyEsc:
		m.searchQuery = ""
		m.state = stateBrowse
		m.refilter()
		return m, nil
	case keyEnter:
		m.state = stateBrowse
		return m, nil
	case keyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.refilter()
		}
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
			m.refilter()
		}
	}
	return m, nil
}

func (m Model) viewportHeight() int {
	h := m.height - 14
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content strings.Builder

	// Logo (centered)
	content.WriteString(centerBlock(renderLogo(), m.width))

	// Title bar (centered)
	title := titleStyle.Render("catalog-sync")
	catalog := catalogNameStyle.Render(m.plan.Catalog)
	id := catalogIDStyle.Render(m.plan.CatalogID)
	titleLine := fmt.Sprintf("%s  %s  %s", title, catalog, id)
	content.WriteString(centerText(titleLine, m.width))
	content.WriteString("\n")

	// Source info (centered)
	if m.sourceInfo != "" {
		content.WriteString(centerText(sourceInfoStyle.Render(m.sourceInfo), m.width))
		content.WriteString("\n")
	}

	// Count summary + filter indicator (centered)
	var badges []string
	if m.plan.Counts.Create > 0 {
		badge := fmt.Sprintf("● %d create", m.plan.Counts.Create)
		if m.filter == catalogsync.OpCreate {
			badges = append(badges, filterActiveStyle.Render(badge))
		} else {
			badges = append(badges, createStyle.Render(badge))
		}
	}
	if m.plan.Counts.Update > 0 {
		badge := fmt.Sprintf("● %d update", m.plan.Counts.Update)
		if m.filter == catalogsync.OpUpdate {
			badges = append(badges, filterActiveStyle.Render(badge))
		} else {
			badges = append(badges, updateStyle.Render(badge))
		}
	}
	if m.plan.Counts.Delete > 0 {
		badge := fmt.Sprintf("● %d delete", m.plan.Counts.Delete)
		if m.filter == catalogsync.OpDelete {
			badges = append(badges, filterActiveStyle.Render(badge))
		} else {
			badges = append(badges, deleteStyle.Render(badge))
		}
	}
	if m.plan.Counts.Noop > 0 {
		badges = append(badges, noopStyle.Render(fmt.Sprintf("○ %d unchanged", m.plan.Counts.Noop)))
	}
	selected := m.selectedCount()
	badges = append(badges, dimStyle.Render(fmt.Sprintf("│ %d selected", selected)))
	content.WriteString(centerText(strings.Join(badges, "  "), m.width))
	content.WriteString("\n")

	// Search bar
	if m.state == stateSearch {
		searchBar := searchPromptStyle.Render("/") + searchInputStyle.Render(m.searchQuery) + dimStyle.Render("_")
		content.WriteString(centerText(searchBar, m.width))
		content.WriteString("\n")
	} else if m.searchQuery != "" {
		searchInfo := dimStyle.Render(fmt.Sprintf("search: %q (%d matches)", m.searchQuery, len(m.filtered)))
		content.WriteString(centerText(searchInfo, m.width))
		content.WriteString("\n")
	}

	content.WriteString("\n")

	// Main layout: list + detail pane side by side
	listWidth := m.width - 8
	detailWidth := 0
	if m.width > 100 {
		listWidth = (m.width - 12) * 6 / 10
		detailWidth = (m.width - 12) * 4 / 10
	}
	if listWidth > 100 {
		listWidth = 100
	}
	if listWidth < 30 {
		listWidth = 30
	}

	changesContent := m.viewChanges()

	// Overlay dialogs
	if m.state == stateConfirm {
		changesContent += "\n" + confirmStyle.Render(fmt.Sprintf(" Apply %d changes? [y/n] ", selected))
	} else if m.state == stateApplied {
		if m.applyErr != nil {
			changesContent += "\n" + errorMsgStyle.Render(fmt.Sprintf(" ✖ Error: %v ", m.applyErr))
		} else if m.appliedPlan != nil {
			changesContent += "\n" + successMsgStyle.Render(fmt.Sprintf(
				" ✓ Applied: %d created, %d updated, %d deleted ",
				m.appliedPlan.Counts.Create, m.appliedPlan.Counts.Update, m.appliedPlan.Counts.Delete))
		}
		changesContent += "\n" + dimStyle.Render("Press q to quit")
	}

	listView := listContainer.Width(listWidth).Render(changesContent)

	if detailWidth > 0 && len(m.filtered) > 0 {
		detailContent := m.renderDetail()
		detailView := detailContainer.Width(detailWidth).Render(detailContent)
		joined := lipgloss.JoinHorizontal(lipgloss.Top, listView, "  ", detailView)
		content.WriteString(centerBlock(joined, m.width))
	} else {
		content.WriteString(centerBlock(listView, m.width))
	}

	// Help bar (centered)
	if m.state == stateBrowse || m.state == stateSearch {
		var helpItems []string
		if m.state == stateSearch {
			helpItems = []string{
				renderHelpItem("type", "search"),
				renderHelpItem("enter", "confirm"),
				renderHelpItem("esc", "cancel"),
			}
		} else {
			helpItems = []string{
				renderHelpItem("↑↓", "navigate"),
				renderHelpItem("space", "toggle"),
				renderHelpItem("enter", "expand"),
				renderHelpItem("c/u/d", "filter"),
				renderHelpItem("0", "all"),
				renderHelpItem("/", "search"),
				renderHelpItem("A", "apply"),
				renderHelpItem("q", "quit"),
			}
		}
		content.WriteString("\n")
		content.WriteString(centerText(helpBarStyle.Render(strings.Join(helpItems, "  ")), m.width))
		content.WriteString("\n")
	}

	return content.String()
}

func (m Model) renderDetail() string {
	if len(m.filtered) == 0 {
		return dimStyle.Render("No entry selected")
	}

	idx := m.filtered[m.cursor]
	c := m.plan.Changes[idx]
	name := catalogsync.EntityName(c)

	var b strings.Builder
	b.WriteString(detailTitle.Render(name))
	b.WriteString("\n\n")

	row := func(label, value string) {
		b.WriteString(detailLabel.Render(label))
		b.WriteString(detailValue.Render(value))
		b.WriteString("\n")
	}

	row("External ID", c.ExternalID)
	row("Operation", c.Op)

	// Show all field values
	switch c.Op {
	case catalogsync.OpCreate:
		if c.After != nil && len(c.After.Fields) > 0 {
			b.WriteString("\n")
			b.WriteString(detailTitle.Render("Fields (new)"))
			b.WriteString("\n")
			for _, k := range sortedKeys(c.After.Fields) {
				row(k, c.After.Fields[k])
			}
		}
	case catalogsync.OpUpdate:
		if len(c.FieldDiffs) > 0 {
			b.WriteString("\n")
			b.WriteString(detailTitle.Render("Changes"))
			b.WriteString("\n")
			for _, k := range sortedDiffKeys(c.FieldDiffs) {
				d := c.FieldDiffs[k]
				b.WriteString(detailLabel.Render(k))
				b.WriteString(diffOldStyle.Render(d[0]))
				b.WriteString(diffArrow)
				b.WriteString(diffNewStyle.Render(d[1]))
				b.WriteString("\n")
			}
		}
		if c.After != nil && len(c.After.Fields) > 0 {
			b.WriteString("\n")
			b.WriteString(detailTitle.Render("All fields"))
			b.WriteString("\n")
			for _, k := range sortedKeys(c.After.Fields) {
				row(k, c.After.Fields[k])
			}
		}
	case catalogsync.OpDelete:
		if c.Before != nil {
			row("Managed by", c.Before.ManagedBy)
			if len(c.Before.Fields) > 0 {
				b.WriteString("\n")
				b.WriteString(detailTitle.Render("Fields (will be deleted)"))
				b.WriteString("\n")
				for _, k := range sortedKeys(c.Before.Fields) {
					row(k, c.Before.Fields[k])
				}
			}
		}
	case catalogsync.OpNoop:
		if c.Before != nil && len(c.Before.Fields) > 0 {
			b.WriteString("\n")
			b.WriteString(detailTitle.Render("Current fields"))
			b.WriteString("\n")
			for _, k := range sortedKeys(c.Before.Fields) {
				row(k, c.Before.Fields[k])
			}
		}
	}

	return b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedDiffKeys(m map[string][2]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func centerText(text string, width int) string {
	textLen := lipgloss.Width(text)
	if textLen >= width {
		return text
	}
	pad := (width - textLen) / 2
	return strings.Repeat(" ", pad) + text
}

func centerBlock(block string, width int) string {
	lines := strings.Split(block, "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(centerText(line, width))
		b.WriteString("\n")
	}
	return b.String()
}

func hasFieldData(c catalogsync.Change) bool {
	switch c.Op {
	case catalogsync.OpCreate:
		return c.After != nil && len(c.After.Fields) > 0
	case catalogsync.OpUpdate:
		return len(c.FieldDiffs) > 0 || (c.After != nil && len(c.After.Fields) > 0)
	case catalogsync.OpDelete:
		return c.Before != nil && len(c.Before.Fields) > 0
	case catalogsync.OpNoop:
		return c.Before != nil && len(c.Before.Fields) > 0
	}
	return false
}

func (m Model) viewChanges() string {
	if len(m.filtered) == 0 {
		if m.filter != "" || m.searchQuery != "" {
			return dimStyle.Render("  No matching changes.")
		}
		return dimStyle.Render("  No changes.")
	}

	var lines []string
	for ci, idx := range m.filtered {
		c := m.plan.Changes[idx]
		lines = append(lines, m.renderChangeLine(ci, idx, c))
		if m.expanded[idx] {
			lines = append(lines, m.renderExpandedContent(idx, c)...)
		}
	}

	vh := m.viewportHeight()
	if m.offset > len(lines)-vh {
		m.offset = len(lines) - vh
	}
	if m.offset < 0 {
		m.offset = 0
	}

	end := m.offset + vh
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	for _, l := range lines[m.offset:end] {
		b.WriteString(l)
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderChangeLine(cursorIdx, planIdx int, c catalogsync.Change) string {
	name := catalogsync.EntityName(c)

	check := checkboxOff
	if m.selected[planIdx] {
		check = checkboxOn
	}

	indicator := "  "
	if cursorIdx == m.cursor {
		indicator = cursorIndicator + " "
	}

	badge := opBadge(c.Op)
	eid := entityIDStyle.Render(c.ExternalID)
	nameStr := entityNameStyle.Render(name)

	expandHint := ""
	if hasFieldData(c) {
		if m.expanded[planIdx] {
			expandHint = dimStyle.Render(" ▾")
		} else {
			count := fieldCount(c)
			expandHint = dimStyle.Render(fmt.Sprintf(" ▸ %d fields", count))
		}
	}

	content := fmt.Sprintf("%s%s %s  %s  %s%s", indicator, check, badge, eid, nameStr, expandHint)

	if cursorIdx == m.cursor {
		return cursorStyle.Render(content)
	}
	if !m.selected[planIdx] && c.Op == catalogsync.OpNoop {
		return dimStyle.Render(content)
	}
	return content
}

func fieldCount(c catalogsync.Change) int {
	switch c.Op {
	case catalogsync.OpUpdate:
		return len(c.FieldDiffs)
	case catalogsync.OpCreate:
		if c.After != nil {
			return len(c.After.Fields)
		}
	case catalogsync.OpDelete:
		if c.Before != nil {
			return len(c.Before.Fields)
		}
	case catalogsync.OpNoop:
		if c.Before != nil {
			return len(c.Before.Fields)
		}
	}
	return 0
}

func (m Model) renderExpandedContent(planIdx int, c catalogsync.Change) []string {
	highlighted := m.cursor < len(m.filtered) && m.filtered[m.cursor] == planIdx

	switch c.Op {
	case catalogsync.OpUpdate:
		return m.renderFieldDiffs(c.FieldDiffs, highlighted)
	case catalogsync.OpCreate:
		if c.After != nil {
			return renderFieldValues(c.After.Fields, createStyle, highlighted)
		}
	case catalogsync.OpDelete:
		if c.Before != nil {
			return renderFieldValues(c.Before.Fields, deleteStyle, highlighted)
		}
	case catalogsync.OpNoop:
		if c.Before != nil {
			return renderFieldValues(c.Before.Fields, noopStyle, highlighted)
		}
	}
	return nil
}

func renderFieldValues(fields map[string]string, style lipgloss.Style, highlighted bool) []string {
	keys := sortedKeys(fields)
	lines := make([]string, 0, len(keys))
	for i, k := range keys {
		var connector string
		if i == len(keys)-1 {
			connector = diffTreeStyle.Render("          └─ ")
		} else {
			connector = diffTreeStyle.Render("          ├─ ")
		}
		line := connector + diffKeyStyle.Render(k) + ": " + style.Render(fields[k])
		if highlighted {
			line = cursorStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

func (m Model) renderFieldDiffs(diffs map[string][2]string, highlighted bool) []string {
	keys := sortedDiffKeys(diffs)
	lines := make([]string, 0, len(keys))
	for i, k := range keys {
		d := diffs[k]
		var connector string
		if i == len(keys)-1 {
			connector = diffTreeStyle.Render("          └─ ")
		} else {
			connector = diffTreeStyle.Render("          ├─ ")
		}
		line := connector + diffKeyStyle.Render(k) + ": " + diffOldStyle.Render(d[0]) + diffArrow + diffNewStyle.Render(d[1])
		if highlighted {
			line = cursorStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

func (m *Model) ensureVisible() {
	lineIdx := 0
	for i := 0; i < m.cursor && i < len(m.filtered); i++ {
		lineIdx++
		idx := m.filtered[i]
		if m.expanded[idx] {
			lineIdx += fieldCount(m.plan.Changes[idx])
		}
	}

	vh := m.viewportHeight()
	if lineIdx < m.offset {
		m.offset = lineIdx
	}
	if lineIdx >= m.offset+vh {
		m.offset = lineIdx - vh + 1
	}
}

func (m *Model) clampOffset() {
	total := m.totalLines()
	vh := m.viewportHeight()
	if m.offset > total-vh {
		m.offset = total - vh
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m Model) totalLines() int {
	n := len(m.filtered)
	for _, idx := range m.filtered {
		if m.expanded[idx] {
			n += fieldCount(m.plan.Changes[idx])
		}
	}
	return n
}

func (m Model) selectedCount() int {
	count := 0
	for i, sel := range m.selected {
		if sel && m.plan.Changes[i].Op != catalogsync.OpNoop {
			count++
		}
	}
	return count
}

func (m Model) buildFilteredPlan() *catalogsync.Plan {
	var changes []catalogsync.Change
	var counts catalogsync.Counts

	for i, c := range m.plan.Changes {
		if !m.selected[i] || c.Op == catalogsync.OpNoop {
			continue
		}
		changes = append(changes, c)
		switch c.Op {
		case catalogsync.OpCreate:
			counts.Create++
		case catalogsync.OpUpdate:
			counts.Update++
		case catalogsync.OpDelete:
			counts.Delete++
		}
	}

	return &catalogsync.Plan{
		Catalog:   m.plan.Catalog,
		CatalogID: m.plan.CatalogID,
		Changes:   changes,
		Counts:    counts,
	}
}
