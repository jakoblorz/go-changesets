package add

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/tui"
	"github.com/jakoblorz/go-changesets/internal/tui/components"
	"github.com/jakoblorz/go-changesets/internal/workspace"
)

// State represents the current state of the add flow
type State int

const (
	StateSelectProjects State = iota
	StateSelectBump
	StateInputMessage
	StateDone
	StateError
)

// Model is the bubbletea model for the add command
type Model struct {
	state     State
	fs        filesystem.FileSystem
	workspace *workspace.Workspace
	csManager *changeset.Manager

	// Data
	projects         []string
	selectedProjects []string
	bumpType         models.BumpType
	message          string
	createdFiles     []string
	err              error

	// Components
	multiSelect components.MultiSelectModel
	radio       components.RadioModel
	textarea    components.TextAreaModel
}

// NewModel creates a new add command model
func NewModel(fs filesystem.FileSystem, ws *workspace.Workspace) Model {
	projects := ws.GetProjectNames()
	multiSelect := components.NewMultiSelect(projects)

	return Model{
		state:       StateSelectProjects,
		fs:          fs,
		workspace:   ws,
		csManager:   changeset.NewManager(fs, ws.ChangesetDir()),
		projects:    projects,
		multiSelect: multiSelect,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.multiSelect.Init()
}

// Update handles messages and state transitions
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.state {
	case StateSelectProjects:
		return m.updateSelectProjects(msg)
	case StateSelectBump:
		return m.updateSelectBump(msg)
	case StateInputMessage:
		return m.updateInputMessage(msg)
	case StateDone, StateError:
		return m, tea.Quit
	}

	return m, nil
}

// updateSelectProjects handles the project selection state
func (m Model) updateSelectProjects(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.multiSelect, cmd = m.multiSelect.Update(msg)

	if m.multiSelect.IsDone() {
		m.selectedProjects = m.multiSelect.GetSelected()

		if len(m.selectedProjects) == 0 {
			m.err = fmt.Errorf("no projects selected")
			m.state = StateError
			return m, tea.Quit
		}

		// Transition to bump selection
		m.state = StateSelectBump
		m.radio = components.NewRadio([]components.RadioOption{
			{Value: string(models.BumpPatch), Label: "patch", Description: "Bug fixes, no breaking changes"},
			{Value: string(models.BumpMinor), Label: "minor", Description: "New features, backward compatible"},
			{Value: string(models.BumpMajor), Label: "major", Description: "Breaking changes"},
		})
		return m, m.radio.Init()
	}

	return m, cmd
}

// updateSelectBump handles the bump type selection state
func (m Model) updateSelectBump(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.radio, cmd = m.radio.Update(msg)

	if m.radio.IsDone() {
		bumpStr := m.radio.GetSelected()
		if bumpStr == "" {
			m.err = fmt.Errorf("no bump type selected")
			m.state = StateError
			return m, tea.Quit
		}

		var err error
		m.bumpType, err = models.ParseBumpType(bumpStr)
		if err != nil {
			m.err = err
			m.state = StateError
			return m, tea.Quit
		}

		// Transition to message input
		m.state = StateInputMessage
		m.textarea = components.NewTextArea("Describe your changes...")
		return m, m.textarea.Init()
	}

	return m, cmd
}

// updateInputMessage handles the message input state
func (m Model) updateInputMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	if m.textarea.IsDone() {
		m.message = strings.TrimSpace(m.textarea.GetValue())

		if m.message == "" {
			m.err = fmt.Errorf("message cannot be empty")
			m.state = StateError
			return m, tea.Quit
		}

		// Create changesets
		if err := m.createChangesets(); err != nil {
			m.err = err
			m.state = StateError
			return m, tea.Quit
		}

		m.state = StateDone
		return m, tea.Quit
	}

	return m, cmd
}

// createChangesets creates one changeset file per selected project
func (m *Model) createChangesets() error {
	for _, projectName := range m.selectedProjects {
		// Generate unique ID for this project
		id, err := m.csManager.GenerateID()
		if err != nil {
			return fmt.Errorf("failed to generate changeset ID for %s: %w", projectName, err)
		}

		// Create changeset with single project
		singleProjectBumps := map[string]models.BumpType{
			projectName: m.bumpType,
		}
		cs := models.NewChangeset(id, singleProjectBumps, m.message)

		// Write changeset to disk
		if err := m.csManager.Write(cs); err != nil {
			return fmt.Errorf("failed to write changeset for %s: %w", projectName, err)
		}

		m.createdFiles = append(m.createdFiles, fmt.Sprintf("%s.md", id))
	}

	return nil
}

// View renders the current state
func (m Model) View() string {
	switch m.state {
	case StateSelectProjects:
		return m.viewSelectProjects()
	case StateSelectBump:
		return m.viewSelectBump()
	case StateInputMessage:
		return m.viewInputMessage()
	case StateDone:
		return m.viewDone()
	case StateError:
		return m.viewError()
	}
	return ""
}

// viewSelectProjects renders the project selection screen
func (m Model) viewSelectProjects() string {
	var b strings.Builder

	b.WriteString(tui.TitleStyle.Render("📦 Create Changeset (1/3)"))
	b.WriteString("\n\n")
	b.WriteString("Select projects:\n\n")
	b.WriteString(m.multiSelect.View())
	b.WriteString("\n")

	count := m.multiSelect.SelectedCount()
	if count > 0 {
		b.WriteString(tui.SubtleStyle.Render(fmt.Sprintf("%d selected • ", count)))
	}
	b.WriteString(tui.HelpStyle.Render("↑↓ navigate • space toggle • enter next"))

	return b.String()
}

// viewSelectBump renders the bump type selection screen
func (m Model) viewSelectBump() string {
	var b strings.Builder

	b.WriteString(tui.TitleStyle.Render("📦 Create Changeset (2/3)"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Select bump type (applies to %s):\n\n",
		tui.SelectedStyle.Render(strings.Join(m.selectedProjects, ", "))))
	b.WriteString(m.radio.View())
	b.WriteString("\n")
	b.WriteString(tui.HelpStyle.Render("↑↓ navigate • enter next"))

	return b.String()
}

// viewInputMessage renders the message input screen
func (m Model) viewInputMessage() string {
	var b strings.Builder

	b.WriteString(tui.TitleStyle.Render("📦 Create Changeset (3/3)"))
	b.WriteString("\n\n")
	b.WriteString("Describe your changes:\n\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n\n")
	b.WriteString(tui.HelpStyle.Render("ctrl+d done • esc cancel"))

	return b.String()
}

// viewDone renders the success screen
func (m Model) viewDone() string {
	var b strings.Builder

	b.WriteString(tui.SuccessStyle.Render("✓ Changeset Created"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Created %d changeset file(s):\n", len(m.createdFiles)))
	for i, file := range m.createdFiles {
		b.WriteString(fmt.Sprintf("  %d. %s (%s: %s)\n",
			i+1, file, m.selectedProjects[i], m.bumpType))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Message: %s\n", m.message))

	return b.String()
}

// viewError renders the error screen
func (m Model) viewError() string {
	return tui.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
}

// GetError returns any error that occurred during execution
func (m Model) GetError() error {
	return m.err
}
