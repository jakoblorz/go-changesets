package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jakoblorz/go-changesets/internal/tui"
)

// ConfirmModel is a simple yes/no confirmation component
type ConfirmModel struct {
	message   string
	cursor    int
	confirmed bool
	done      bool
}

// NewConfirm creates a new confirmation component
func NewConfirm(message string) ConfirmModel {
	return ConfirmModel{
		message:   message,
		cursor:    0,
		confirmed: false,
		done:      false,
	}
}

// Init initializes the component
func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.cursor = 0
		case "right", "l":
			m.cursor = 1
		case "enter", " ":
			m.confirmed = m.cursor == 0
			m.done = true
			return m, tea.Quit
		case "y":
			m.confirmed = true
			m.done = true
			return m, tea.Quit
		case "n":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the component
func (m ConfirmModel) View() string {
	if m.done {
		return ""
	}

	yes := "Yes"
	no := "No"

	if m.cursor == 0 {
		yes = tui.SelectedStyle.Render("> " + yes)
	} else {
		yes = "  " + yes
	}

	if m.cursor == 1 {
		no = tui.SelectedStyle.Render("> " + no)
	} else {
		no = "  " + no
	}

	return fmt.Sprintf("%s\n\n%s  %s\n\n%s",
		m.message,
		yes, no,
		tui.HelpStyle.Render("←→ navigate • enter confirm • y/n quick select"))
}

// IsConfirmed returns whether the user confirmed
func (m ConfirmModel) IsConfirmed() bool {
	return m.confirmed
}

// IsDone returns whether the user finished
func (m ConfirmModel) IsDone() bool {
	return m.done
}
