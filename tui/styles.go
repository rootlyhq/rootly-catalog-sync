package tui

import (
	"image/color"
	"strings"

	lipgloss "charm.land/lipgloss/v2"

	catalogsync "github.com/rootlyhq/rootly-catalog-sync/sync"
)

// Rootly brand colors (matching rootly-tui)
var (
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorHighlight = lipgloss.Color("#8B5CF6")
	colorSuccess   = lipgloss.Color("#10B981")
	colorWarning   = lipgloss.Color("#F59E0B")
	colorDanger    = lipgloss.Color("#EF4444")
	colorInfo      = lipgloss.Color("#4D96FF")
	colorMuted     = lipgloss.Color("#6B7280")
	colorText      = lipgloss.Color("#F9FAFB")
	colorTextDim   = lipgloss.Color("#9CA3AF")
	colorBorder    = lipgloss.Color("#374151")
	colorBlack     = lipgloss.Color("#000000")
)

// Op-specific colors
var (
	createColor = colorSuccess
	updateColor = colorWarning
	deleteColor = colorDanger
	noopColor   = colorMuted
)

// Text styles
var (
	createStyle = lipgloss.NewStyle().Foreground(createColor)
	updateStyle = lipgloss.NewStyle().Foreground(updateColor)
	deleteStyle = lipgloss.NewStyle().Foreground(deleteColor)
	noopStyle   = lipgloss.NewStyle().Foreground(noopColor)
	dimStyle    = lipgloss.NewStyle().Foreground(colorTextDim)
)

// Logo gradient colors (purple → indigo → blue)
var logoGradient = []color.Color{
	lipgloss.Color("#7C3AED"), lipgloss.Color("#6D5AED"), lipgloss.Color("#6366F1"),
	lipgloss.Color("#5B7FF1"), lipgloss.Color("#4D96FF"),
}

var asciiLogo = []string{
	`  ██████╗  ██████╗  ██████╗ ████████╗██╗  ██╗   ██╗`,
	`  ██╔══██╗██╔═══██╗██╔═══██╗╚══██╔══╝██║  ╚██╗ ██╔╝`,
	`  ██████╔╝██║   ██║██║   ██║   ██║   ██║   ╚████╔╝ `,
	`  ██╔══██╗██║   ██║██║   ██║   ██║   ██║    ╚██╔╝  `,
	`  ██║  ██║╚██████╔╝╚██████╔╝   ██║   ███████╗██║   `,
	`  ╚═╝  ╚═╝ ╚═════╝  ╚═════╝    ╚═╝   ╚══════╝╚═╝   `,
}

// Layout styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorPrimary).
			Padding(0, 1)

	catalogNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText)

	catalogIDStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Italic(true)

	listContainer = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHighlight)
)

// Help bar
var (
	helpBarStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			MarginTop(1)

	helpKey = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	helpDesc = lipgloss.NewStyle().
			Foreground(colorTextDim)
)

// Entity display styles (hoisted from render loop)
var (
	entityIDStyle   = lipgloss.NewStyle().Foreground(colorInfo)
	entityNameStyle = lipgloss.NewStyle().Foreground(colorText).Bold(true)
)

// Diff styles
var (
	diffKeyStyle  = lipgloss.NewStyle().Foreground(colorInfo)
	diffOldStyle  = lipgloss.NewStyle().Foreground(colorDanger).Strikethrough(true)
	diffNewStyle  = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	diffTreeStyle = lipgloss.NewStyle().Foreground(colorBorder)
	diffArrow     = dimStyle.Render(" → ")
)

// Detail pane
var (
	detailContainer = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	detailTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	detailLabel = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Width(15)

	detailValue = lipgloss.NewStyle().
			Foreground(colorText)

	searchInputStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Bold(true)

	searchPromptStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	filterActiveStyle = lipgloss.NewStyle().
				Foreground(colorBlack).
				Background(colorPrimary).
				Padding(0, 1).
				Bold(true)

	sourceInfoStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Italic(true)
)

// Dialog styles
var (
	confirmStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBlack).
			Background(colorWarning).
			Padding(0, 1)

	successMsgStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBlack).
			Background(colorSuccess).
			Padding(0, 1)

	errorMsgStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorDanger).
			Padding(0, 1)
)

// Checkboxes and indicators
var (
	checkboxOn      = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("◉")
	checkboxOff     = lipgloss.NewStyle().Foreground(colorMuted).Render("○")
	cursorIndicator = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("▶")
)

// Op metadata for lookup-based dispatch
type opMeta struct {
	color color.Color
	fg    color.Color
	icon  string
	label string
}

var opRegistry = map[string]opMeta{
	catalogsync.OpCreate: {createColor, colorBlack, "+", " + CREATE "},
	catalogsync.OpUpdate: {updateColor, colorBlack, "◆", " ◆ UPDATE "},
	catalogsync.OpDelete: {deleteColor, colorText, "✖", " ✖ DELETE "},
	catalogsync.OpNoop:   {colorBorder, colorTextDim, "─", "   NOOP   "},
}

var badgeBase = lipgloss.NewStyle().Padding(0, 1).Bold(true)

func opBadge(op string) string {
	m, ok := opRegistry[op]
	if !ok {
		return "?"
	}
	return badgeBase.Foreground(m.fg).Background(m.color).Render(m.label)
}

func renderHelpItem(key, desc string) string {
	return helpKey.Render(key) + " " + helpDesc.Render(desc)
}

func renderLogo() string {
	var b strings.Builder
	for _, line := range asciiLogo {
		runes := []rune(line)
		width := len(runes)
		for i, r := range runes {
			if r == ' ' {
				b.WriteRune(r)
				continue
			}
			idx := i * (len(logoGradient) - 1) / max(width, 1)
			style := lipgloss.NewStyle().Foreground(logoGradient[idx]).Bold(true)
			b.WriteString(style.Render(string(r)))
		}
		b.WriteRune('\n')
	}
	return b.String()
}
