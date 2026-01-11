package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jakoblorz/go-changesets/internal/tui"
)

// RadioOption represents a single radio option
type RadioOption struct {
	Value       string
	Label       string
	Description string
}

// RadioModel is a radio button component
type RadioModel struct {
	options  []RadioOption
	cursor   int
	selected int
	done     bool
}

// NewRadio creates a new radio button component
func NewRadio(options []RadioOption) RadioModel {
	return RadioModel{
		options:  options,
		cursor:   0,
		selected: -1,
		done:     false,
	}
}

// Init initializes the component
func (m RadioModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m RadioModel) Update(msg tea.Msg) (RadioModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "q", "esc":
			m.done = false
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the component
func (m RadioModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	for i, option := range m.options {
		cursor := " "
		if m.cursor == i {
			cursor = tui.SelectedStyle.Render("›")
		}

		radio := "( )"
		if m.selected == i {
			radio = tui.CheckedStyle.Render("(•)")
		} else {
			radio = tui.UncheckedStyle.Render("( )")
		}

		labelStyle := lipgloss.NewStyle()
		descStyle := tui.DescStyle
		if m.cursor == i {
			labelStyle = tui.SelectedStyle
		}

		label := labelStyle.Render(option.Label)
		if option.Description != "" {
			label += "  " + descStyle.Render(option.Description)
		}

		b.WriteString(fmt.Sprintf("%s %s %s\n", cursor, radio, label))
	}

	return b.String()
}

// GetSelected returns the selected option value
func (m RadioModel) GetSelected() string {
	if m.selected >= 0 && m.selected < len(m.options) {
		return m.options[m.selected].Value
	}
	return ""
}

// IsDone returns whether the user finished selecting
func (m RadioModel) IsDone() bool {
	return m.done
}

// HasSelection returns whether a selection has been made
func (m RadioModel) HasSelection() bool {
	return m.selected >= 0
}
