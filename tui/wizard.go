package tui

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	tea "github.com/charmbracelet/bubbletea"
)

var sourceTypes = []string{"inline", "local", "github", "exec", "csv", "backstage", "graphql"}
var outputFormats = []string{"yaml", "jsonnet", "hcl"}

type fieldMapping struct {
	slug     string
	template string
}

// WizardModel holds the state for the interactive config wizard.
type WizardModel struct {
	step         int // 0=source, 1=config, 2=catalog, 3=fields, 4=format, 5=preview
	sourceType   string
	sourceFiles  string
	catalogName  string
	externalID   string
	nameField    string
	fields       []fieldMapping
	currentInput string
	cursor       int
	done         bool
	result       string
	canceled     bool
	width        int
	height       int
	outputFormat string // "yaml", "jsonnet", "hcl"

	githubOwner   string
	githubRepos   string
	githubFiles   string
	csvDelimiter  string
	execCommand   string
	backstageURL  string
	graphqlURL    string
	graphqlQuery  string
	graphqlResult string

	sourceSubStep  int
	fieldSubStep   int
	enteringSlug   bool
	fieldSlugInput string
}

// WizardResult is returned by RunWizard.
type WizardResult struct {
	Content  string // generated config content
	Filename string // suggested filename (e.g. "rootly-catalog-sync.yaml")
}

func NewWizard() WizardModel {
	return WizardModel{
		step:   0,
		cursor: 0,
		fields: []fieldMapping{},
	}
}

func (m WizardModel) Init() tea.Cmd {
	return nil
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch m.step {
		case 0:
			return m.updateSelectSource(msg)
		case 1:
			return m.updateConfigureSource(msg)
		case 2:
			return m.updateCatalogName(msg)
		case 3:
			return m.updateFieldMapping(msg)
		case 4:
			return m.updateSelectFormat(msg)
		case 5:
			return m.updatePreview(msg)
		}
	}
	return m, nil
}

func (m WizardModel) updateSelectSource(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, "q":
		m.canceled = true
		m.done = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(sourceTypes)-1 {
			m.cursor++
		}
	case keyEnter:
		m.sourceType = sourceTypes[m.cursor]
		m.sourceSubStep = 0
		m.currentInput = ""
		if m.sourceType == "inline" {
			m.step = 2
		} else {
			m.step = 1
		}
	}
	return m, nil
}

func (m WizardModel) updateConfigureSource(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC:
		m.canceled = true
		m.done = true
		return m, tea.Quit
	case keyEnter:
		return m.commitSourceSubStep()
	case keyBackspace:
		if len(m.currentInput) > 0 {
			m.currentInput = m.currentInput[:len(m.currentInput)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.currentInput += msg.String()
		}
	}
	return m, nil
}

func (m WizardModel) commitSourceSubStep() (tea.Model, tea.Cmd) {
	switch m.sourceType {
	case "local":
		m.sourceFiles = m.currentInput
		if m.sourceFiles == "" {
			m.sourceFiles = "catalog/*.yaml"
		}
		m.currentInput = ""
		m.step = 2
	case "github":
		switch m.sourceSubStep {
		case 0:
			m.githubOwner = m.currentInput
			m.currentInput = ""
			m.sourceSubStep = 1
		case 1:
			m.githubRepos = m.currentInput
			m.currentInput = ""
			m.sourceSubStep = 2
		case 2:
			m.githubFiles = m.currentInput
			if m.githubFiles == "" {
				m.githubFiles = "catalog-info.yaml"
			}
			m.currentInput = ""
			m.step = 2
		}
	case "csv":
		switch m.sourceSubStep {
		case 0:
			m.sourceFiles = m.currentInput
			if m.sourceFiles == "" {
				m.sourceFiles = "*.csv"
			}
			m.currentInput = ""
			m.sourceSubStep = 1
		case 1:
			m.csvDelimiter = m.currentInput
			m.currentInput = ""
			m.step = 2
		}
	case "exec":
		m.execCommand = m.currentInput
		m.currentInput = ""
		m.step = 2
	case "backstage":
		m.backstageURL = m.currentInput
		m.currentInput = ""
		m.step = 2
	case "graphql":
		switch m.sourceSubStep {
		case 0:
			m.graphqlURL = m.currentInput
			m.currentInput = ""
			m.sourceSubStep = 1
		case 1:
			m.graphqlQuery = m.currentInput
			m.currentInput = ""
			m.sourceSubStep = 2
		case 2:
			m.graphqlResult = m.currentInput
			if m.graphqlResult == "" {
				m.graphqlResult = "data.items"
			}
			m.currentInput = ""
			m.step = 2
		}
	}
	return m, nil
}

func (m WizardModel) updateCatalogName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC:
		m.canceled = true
		m.done = true
		return m, tea.Quit
	case keyEnter:
		m.catalogName = m.currentInput
		if m.catalogName == "" {
			m.catalogName = "Services"
		}
		m.currentInput = ""
		m.fieldSubStep = 0
		m.step = 3
	case keyBackspace:
		if len(m.currentInput) > 0 {
			m.currentInput = m.currentInput[:len(m.currentInput)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.currentInput += msg.String()
		}
	}
	return m, nil
}

func (m WizardModel) updateFieldMapping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC:
		m.canceled = true
		m.done = true
		return m, tea.Quit
	case keyEnter:
		return m.commitFieldSubStep()
	case keyBackspace:
		if len(m.currentInput) > 0 {
			m.currentInput = m.currentInput[:len(m.currentInput)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.currentInput += msg.String()
		}
	}
	return m, nil
}

func (m WizardModel) commitFieldSubStep() (tea.Model, tea.Cmd) {
	switch m.fieldSubStep {
	case 0:
		m.externalID = m.currentInput
		if m.externalID == "" {
			m.externalID = "{{ .id }}"
		}
		m.currentInput = ""
		m.fieldSubStep = 1
	case 1:
		m.nameField = m.currentInput
		if m.nameField == "" {
			m.nameField = "{{ .name }}"
		}
		m.currentInput = ""
		m.fieldSubStep = 2
		m.enteringSlug = true
	default:
		if m.enteringSlug {
			if m.currentInput == "" {
				m.currentInput = ""
				m.cursor = 0
				m.step = 4
			} else {
				m.fieldSlugInput = m.currentInput
				m.currentInput = ""
				m.enteringSlug = false
			}
		} else {
			template := m.currentInput
			if template == "" {
				template = "{{ ." + m.fieldSlugInput + " }}"
			}
			m.fields = append(m.fields, fieldMapping{slug: m.fieldSlugInput, template: template})
			m.currentInput = ""
			m.fieldSlugInput = ""
			m.enteringSlug = true
		}
	}
	return m, nil
}

func (m WizardModel) updateSelectFormat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, "q":
		m.canceled = true
		m.done = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(outputFormats)-1 {
			m.cursor++
		}
	case keyEnter:
		m.outputFormat = outputFormats[m.cursor]
		m.result = m.generateConfig()
		m.step = 5
	}
	return m, nil
}

func (m WizardModel) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyCtrlC, "q":
		m.canceled = true
		m.done = true
		return m, tea.Quit
	case keyEnter:
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

var (
	wizardStepStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	wizardInputCursor = dimStyle.Render("█")

	wizardContainer = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 3)
)

func (m WizardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var outer strings.Builder

	outer.WriteString(centerBlock(renderLogo(), m.width))
	outer.WriteString("\n")

	title := titleStyle.Render("catalog-sync init")
	outer.WriteString(centerText(title, m.width))
	outer.WriteString("\n\n")

	steps := []string{"Source", "Config", "Catalog", "Fields", "Format", "Preview"}
	var progress strings.Builder
	for i, s := range steps {
		if i == m.step {
			progress.WriteString(filterActiveStyle.Render(fmt.Sprintf(" %d. %s ", i+1, s)))
		} else if i < m.step {
			progress.WriteString(createStyle.Render(fmt.Sprintf(" ✓ %s ", s)))
		} else {
			progress.WriteString(dimStyle.Render(fmt.Sprintf(" %d. %s ", i+1, s)))
		}
		if i < len(steps)-1 {
			progress.WriteString(dimStyle.Render(" › "))
		}
	}
	outer.WriteString(centerText(progress.String(), m.width))
	outer.WriteString("\n\n\n")

	var b strings.Builder
	switch m.step {
	case 0:
		b.WriteString(wizardStepStyle.Render("Select source type"))
		b.WriteString("\n\n")
		for i, st := range sourceTypes {
			if i == m.cursor {
				b.WriteString(cursorIndicator + " " + entityNameStyle.Render(st))
			} else {
				b.WriteString("  " + dimStyle.Render(st))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(helpBarStyle.Render(renderHelpItem("↑↓", "navigate") + "  " + renderHelpItem("enter", "select")))

	case 1:
		b.WriteString(wizardStepStyle.Render("Configure " + m.sourceType + " source"))
		b.WriteString("\n\n")
		b.WriteString(m.viewConfigureSource())

	case 2:
		b.WriteString(wizardStepStyle.Render("Catalog name"))
		b.WriteString("\n\n")
		b.WriteString(detailLabel.Render("Name"))
		_, _ = fmt.Fprintf(&b, "%s%s", searchInputStyle.Render(m.currentInput), wizardInputCursor)
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Default: Services  •  Press enter to accept"))

	case 3:
		b.WriteString(wizardStepStyle.Render("Field mapping"))
		b.WriteString("\n\n")
		b.WriteString(m.viewFieldMapping())

	case 4:
		b.WriteString(wizardStepStyle.Render("Output format"))
		b.WriteString("\n\n")
		for i, f := range outputFormats {
			label := f
			switch f {
			case "yaml":
				label = "YAML  — simple, zero learning curve"
			case "jsonnet":
				label = "Jsonnet — variables, functions, imports"
			case "hcl":
				label = "HCL   — Terraform-style blocks"
			}
			if i == m.cursor {
				b.WriteString(cursorIndicator + " " + entityNameStyle.Render(label))
			} else {
				b.WriteString("  " + dimStyle.Render(label))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(helpBarStyle.Render(renderHelpItem("↑↓", "navigate") + "  " + renderHelpItem("enter", "select")))

	case 5:
		ext := m.fileExtension()
		b.WriteString(wizardStepStyle.Render(fmt.Sprintf("Preview — rootly-catalog-sync%s", ext)))
		b.WriteString("\n\n")
		b.WriteString(m.result)
		b.WriteString("\n")
		b.WriteString(helpBarStyle.Render(renderHelpItem("enter", "save") + "  " + renderHelpItem("q", "cancel")))
	}

	containerWidth := m.width - 12
	if containerWidth > 80 {
		containerWidth = 80
	}
	if containerWidth < 30 {
		containerWidth = 30
	}
	container := wizardContainer.Width(containerWidth).Render(b.String())
	outer.WriteString(centerBlock(container, m.width))

	return outer.String()
}

func (m WizardModel) viewConfigureSource() string {
	var b strings.Builder
	cursor := wizardInputCursor

	switch m.sourceType {
	case "local":
		_, _ = fmt.Fprintf(&b, "  File glob pattern [catalog/*.yaml]: %s%s\n", m.currentInput, cursor)
	case "github":
		switch m.sourceSubStep {
		case 0:
			_, _ = fmt.Fprintf(&b, "  GitHub owner (org or user): %s%s\n", m.currentInput, cursor)
		case 1:
			b.WriteString(dimStyle.Render(fmt.Sprintf("  Owner: %s\n", m.githubOwner)))
			_, _ = fmt.Fprintf(&b, "  Repos (comma-separated, empty for all): %s%s\n", m.currentInput, cursor)
		case 2:
			b.WriteString(dimStyle.Render(fmt.Sprintf("  Owner: %s\n", m.githubOwner)))
			b.WriteString(dimStyle.Render(fmt.Sprintf("  Repos: %s\n", m.githubRepos)))
			_, _ = fmt.Fprintf(&b, "  File pattern [catalog-info.yaml]: %s%s\n", m.currentInput, cursor)
		}
	case "csv":
		switch m.sourceSubStep {
		case 0:
			_, _ = fmt.Fprintf(&b, "  File pattern [*.csv]: %s%s\n", m.currentInput, cursor)
		case 1:
			b.WriteString(dimStyle.Render(fmt.Sprintf("  Files: %s\n", m.sourceFiles)))
			_, _ = fmt.Fprintf(&b, "  Delimiter (empty for comma): %s%s\n", m.currentInput, cursor)
		}
	case "exec":
		_, _ = fmt.Fprintf(&b, "  Command to execute: %s%s\n", m.currentInput, cursor)
	case "backstage":
		_, _ = fmt.Fprintf(&b, "  Backstage API URL: %s%s\n", m.currentInput, cursor)
	case "graphql":
		switch m.sourceSubStep {
		case 0:
			_, _ = fmt.Fprintf(&b, "  GraphQL endpoint URL: %s%s\n", m.currentInput, cursor)
		case 1:
			b.WriteString(dimStyle.Render(fmt.Sprintf("  URL: %s\n", m.graphqlURL)))
			_, _ = fmt.Fprintf(&b, "  GraphQL query: %s%s\n", m.currentInput, cursor)
		case 2:
			b.WriteString(dimStyle.Render(fmt.Sprintf("  URL: %s\n", m.graphqlURL)))
			b.WriteString(dimStyle.Render(fmt.Sprintf("  Query: %s\n", m.graphqlQuery)))
			_, _ = fmt.Fprintf(&b, "  Result path [data.items]: %s%s\n", m.currentInput, cursor)
		}
	}

	b.WriteString("\n")
	b.WriteString(helpBarStyle.Render(renderHelpItem("enter", "next") + "  " + renderHelpItem("ctrl+c", "cancel")))
	return b.String()
}

func (m WizardModel) viewFieldMapping() string {
	var b strings.Builder
	cursor := wizardInputCursor

	switch m.fieldSubStep {
	case 0:
		_, _ = fmt.Fprintf(&b, "  external_id template [{{ .id }}]: %s%s\n", m.currentInput, cursor)
	case 1:
		b.WriteString(dimStyle.Render(fmt.Sprintf("  external_id: %s\n", m.externalID)))
		_, _ = fmt.Fprintf(&b, "  name template [{{ .name }}]: %s%s\n", m.currentInput, cursor)
	default:
		b.WriteString(dimStyle.Render(fmt.Sprintf("  external_id: %s\n", m.externalID)))
		b.WriteString(dimStyle.Render(fmt.Sprintf("  name: %s\n", m.nameField)))
		for _, f := range m.fields {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s: %s\n", f.slug, f.template)))
		}
		b.WriteString("\n")
		if m.enteringSlug {
			_, _ = fmt.Fprintf(&b, "  Field slug (empty to finish): %s%s\n", m.currentInput, cursor)
		} else {
			_, _ = fmt.Fprintf(&b, "  Template [{{ .%s }}]: %s%s\n", m.fieldSlugInput, m.currentInput, cursor)
		}
	}

	b.WriteString("\n")
	b.WriteString(helpBarStyle.Render(renderHelpItem("enter", "next") + "  " + renderHelpItem("ctrl+c", "cancel")))
	return b.String()
}

func (m WizardModel) fileExtension() string {
	switch m.outputFormat {
	case "jsonnet":
		return ".jsonnet"
	case "hcl":
		return ".hcl"
	default:
		return ".yaml"
	}
}

func (m WizardModel) generateConfig() string {
	switch m.outputFormat {
	case "jsonnet":
		return m.generateJsonnet()
	case "hcl":
		return m.generateHCL()
	default:
		return m.generateYAML()
	}
}

func (m WizardModel) generateYAML() string {
	var b strings.Builder

	b.WriteString("version: 2\n\n")
	b.WriteString("sync:\n")
	b.WriteString("  - from:\n")
	b.WriteString(m.generateSourceYAML())
	_, _ = fmt.Fprintf(&b, "    to: %q\n", m.catalogName)
	b.WriteString("    map:\n")
	_, _ = fmt.Fprintf(&b, "      external_id: %q\n", m.externalID)
	_, _ = fmt.Fprintf(&b, "      name: %q\n", m.nameField)

	for _, f := range m.fields {
		_, _ = fmt.Fprintf(&b, "      %s: %q\n", f.slug, f.template)
	}

	return b.String()
}

func (m WizardModel) generateJsonnet() string {
	syncID := slugify(m.catalogName)
	var b strings.Builder

	b.WriteString("{\n")
	b.WriteString("  version: 1,\n")
	_, _ = fmt.Fprintf(&b, "  sync_id: %q,\n", syncID)
	b.WriteString("  pipelines: [\n")
	b.WriteString("    {\n")
	b.WriteString("      sources: [\n")
	b.WriteString(m.generateSourceJsonnet())
	b.WriteString("      ],\n")
	b.WriteString("      outputs: [\n")
	b.WriteString("        {\n")
	_, _ = fmt.Fprintf(&b, "          catalog: %q,\n", m.catalogName)
	_, _ = fmt.Fprintf(&b, "          external_id: %q,\n", m.externalID)
	_, _ = fmt.Fprintf(&b, "          name: %q,\n", m.nameField)
	if len(m.fields) > 0 {
		b.WriteString("          fields: {\n")
		for _, f := range m.fields {
			_, _ = fmt.Fprintf(&b, "            %s: {value: %q},\n", f.slug, f.template)
		}
		b.WriteString("          },\n")
	}
	b.WriteString("        },\n")
	b.WriteString("      ],\n")
	b.WriteString("    },\n")
	b.WriteString("  ],\n")
	b.WriteString("}\n")

	return b.String()
}

func (m WizardModel) generateHCL() string {
	syncID := slugify(m.catalogName)
	var b strings.Builder

	b.WriteString("version = 1\n")
	_, _ = fmt.Fprintf(&b, "sync_id = %q\n\n", syncID)
	b.WriteString("pipeline {\n")
	b.WriteString("  source {\n")
	b.WriteString(m.generateSourceHCL())
	b.WriteString("  }\n")
	b.WriteString("  output {\n")
	_, _ = fmt.Fprintf(&b, "    catalog     = %q\n", m.catalogName)
	_, _ = fmt.Fprintf(&b, "    external_id = %q\n", m.externalID)
	_, _ = fmt.Fprintf(&b, "    name        = %q\n", m.nameField)
	if len(m.fields) > 0 {
		b.WriteString("    fields = {\n")
		for _, f := range m.fields {
			_, _ = fmt.Fprintf(&b, "      %s = {\n", f.slug)
			_, _ = fmt.Fprintf(&b, "        value = %q\n", f.template)
			b.WriteString("      }\n")
		}
		b.WriteString("    }\n")
	}
	b.WriteString("  }\n")
	b.WriteString("}\n")

	return b.String()
}

func (m WizardModel) generateSourceYAML() string {
	var b strings.Builder
	switch m.sourceType {
	case "inline":
		b.WriteString("      - inline:\n")
		b.WriteString("          entries:\n")
		b.WriteString("            - id: example\n")
		b.WriteString("              name: Example\n")
	case "local":
		b.WriteString("      local:\n")
		_, _ = fmt.Fprintf(&b, "        files: [%q]\n", m.sourceFiles)
	case "github":
		b.WriteString("      github:\n")
		_, _ = fmt.Fprintf(&b, "        owner: %q\n", m.githubOwner)
		m.writeReposList(&b, "        ")
		m.writeFilesList(&b, "        ", m.githubFiles)
	case "exec":
		b.WriteString("      exec:\n")
		_, _ = fmt.Fprintf(&b, "        command: %q\n", m.execCommand)
	case "csv":
		b.WriteString("      csv:\n")
		_, _ = fmt.Fprintf(&b, "        files: [%q]\n", m.sourceFiles)
		if m.csvDelimiter != "" {
			_, _ = fmt.Fprintf(&b, "        delimiter: %q\n", m.csvDelimiter)
		}
	case "backstage":
		b.WriteString("      backstage:\n")
		_, _ = fmt.Fprintf(&b, "        url: %q\n", m.backstageURL)
	case "graphql":
		b.WriteString("      graphql:\n")
		_, _ = fmt.Fprintf(&b, "        url: %q\n", m.graphqlURL)
		_, _ = fmt.Fprintf(&b, "        query: %q\n", m.graphqlQuery)
		_, _ = fmt.Fprintf(&b, "        result: %q\n", m.graphqlResult)
	}
	return b.String()
}

func (m WizardModel) generateSourceJsonnet() string {
	var b strings.Builder
	switch m.sourceType {
	case "inline":
		b.WriteString("        { inline: { entries: [{ id: \"example\", name: \"Example\" }] } },\n")
	case "local":
		_, _ = fmt.Fprintf(&b, "        { local: { files: [%q] } },\n", m.sourceFiles)
	case "github":
		b.WriteString("        {\n")
		b.WriteString("          github: {\n")
		_, _ = fmt.Fprintf(&b, "            owner: %q,\n", m.githubOwner)
		if m.githubRepos != "" {
			b.WriteString("            repos: [")
			for i, r := range splitTrim(m.githubRepos) {
				if i > 0 {
					b.WriteString(", ")
				}
				_, _ = fmt.Fprintf(&b, "%q", r)
			}
			b.WriteString("],\n")
		}
		b.WriteString("            files: [")
		for i, f := range splitTrim(m.githubFiles) {
			if i > 0 {
				b.WriteString(", ")
			}
			_, _ = fmt.Fprintf(&b, "%q", f)
		}
		b.WriteString("],\n")
		b.WriteString("          },\n")
		b.WriteString("        },\n")
	case "exec":
		_, _ = fmt.Fprintf(&b, "        { exec: { command: %q } },\n", m.execCommand)
	case "csv":
		_, _ = fmt.Fprintf(&b, "        { csv: { files: [%q]", m.sourceFiles)
		if m.csvDelimiter != "" {
			_, _ = fmt.Fprintf(&b, ", delimiter: %q", m.csvDelimiter)
		}
		b.WriteString(" } },\n")
	case "backstage":
		_, _ = fmt.Fprintf(&b, "        { backstage: { url: %q } },\n", m.backstageURL)
	case "graphql":
		b.WriteString("        {\n")
		b.WriteString("          graphql: {\n")
		_, _ = fmt.Fprintf(&b, "            url: %q,\n", m.graphqlURL)
		_, _ = fmt.Fprintf(&b, "            query: %q,\n", m.graphqlQuery)
		_, _ = fmt.Fprintf(&b, "            result: %q,\n", m.graphqlResult)
		b.WriteString("          },\n")
		b.WriteString("        },\n")
	}
	return b.String()
}

func (m WizardModel) generateSourceHCL() string {
	var b strings.Builder
	switch m.sourceType {
	case "inline":
		b.WriteString("    inline {\n")
		b.WriteString("      entries = [{ id = \"example\", name = \"Example\" }]\n")
		b.WriteString("    }\n")
	case "local":
		b.WriteString("    local {\n")
		_, _ = fmt.Fprintf(&b, "      files = [%q]\n", m.sourceFiles)
		b.WriteString("    }\n")
	case "github":
		b.WriteString("    github {\n")
		_, _ = fmt.Fprintf(&b, "      owner = %q\n", m.githubOwner)
		if m.githubRepos != "" {
			b.WriteString("      repos = [")
			for i, r := range splitTrim(m.githubRepos) {
				if i > 0 {
					b.WriteString(", ")
				}
				_, _ = fmt.Fprintf(&b, "%q", r)
			}
			b.WriteString("]\n")
		}
		b.WriteString("      files = [")
		for i, f := range splitTrim(m.githubFiles) {
			if i > 0 {
				b.WriteString(", ")
			}
			_, _ = fmt.Fprintf(&b, "%q", f)
		}
		b.WriteString("]\n")
		b.WriteString("    }\n")
	case "exec":
		b.WriteString("    exec {\n")
		_, _ = fmt.Fprintf(&b, "      command = %q\n", m.execCommand)
		b.WriteString("    }\n")
	case "csv":
		b.WriteString("    csv {\n")
		_, _ = fmt.Fprintf(&b, "      files = [%q]\n", m.sourceFiles)
		if m.csvDelimiter != "" {
			_, _ = fmt.Fprintf(&b, "      delimiter = %q\n", m.csvDelimiter)
		}
		b.WriteString("    }\n")
	case "backstage":
		b.WriteString("    backstage {\n")
		_, _ = fmt.Fprintf(&b, "      url = %q\n", m.backstageURL)
		b.WriteString("    }\n")
	case "graphql":
		b.WriteString("    graphql {\n")
		_, _ = fmt.Fprintf(&b, "      url   = %q\n", m.graphqlURL)
		_, _ = fmt.Fprintf(&b, "      query = %q\n", m.graphqlQuery)
		_, _ = fmt.Fprintf(&b, "      result = %q\n", m.graphqlResult)
		b.WriteString("    }\n")
	}
	return b.String()
}

func (m WizardModel) writeReposList(b *strings.Builder, indent string) {
	if m.githubRepos != "" {
		repos := splitTrim(m.githubRepos)
		b.WriteString(indent + "repos:\n")
		for _, r := range repos {
			_, _ = fmt.Fprintf(b, "%s  - %q\n", indent, r)
		}
	}
}

func (m WizardModel) writeFilesList(b *strings.Builder, indent, files string) {
	parts := splitTrim(files)
	b.WriteString(indent + "files:\n")
	for _, f := range parts {
		_, _ = fmt.Fprintf(b, "%s  - %q\n", indent, f)
	}
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	var result strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result.WriteRune(c)
		}
	}
	return result.String()
}

// RunWizard runs the wizard and returns the result.
func RunWizard() (*WizardResult, error) {
	m := NewWizard()
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard error: %w", err)
	}

	result := finalModel.(WizardModel)
	if result.canceled {
		return nil, fmt.Errorf("wizard canceled")
	}
	if !result.done {
		return nil, fmt.Errorf("wizard did not complete")
	}

	return &WizardResult{
		Content:  result.result,
		Filename: "rootly-catalog-sync" + result.fileExtension(),
	}, nil
}
