package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jakoblorz/go-changesets/internal/tui"
)

// MultiSelectModel is a multi-select list component
type MultiSelectModel struct {
	items    []string
	selected map[int]bool
	cursor   int
	done     bool
}

// NewMultiSelect creates a new multi-select component
func NewMultiSelect(items []string) MultiSelectModel {
	return MultiSelectModel{
		items:    items,
		selected: make(map[int]bool),
		cursor:   0,
		done:     false,
	}
}

// Init initializes the component
func (m MultiSelectModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m MultiSelectModel) Update(msg tea.Msg) (MultiSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			// Toggle selection
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "enter":
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
func (m MultiSelectModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = tui.SelectedStyle.Render("›")
		}

		checkbox := "[ ]"
		if m.selected[i] {
			checkbox = tui.CheckedStyle.Render("[✓]")
		} else {
			checkbox = tui.UncheckedStyle.Render("[ ]")
		}

		itemStyle := lipgloss.NewStyle()
		if m.cursor == i {
			itemStyle = tui.SelectedStyle
		}

		b.WriteString(fmt.Sprintf("%s %s %s\n", cursor, checkbox, itemStyle.Render(item)))
	}

	return b.String()
}

// GetSelected returns the selected items
func (m MultiSelectModel) GetSelected() []string {
	var selected []string
	for i, item := range m.items {
		if m.selected[i] {
			selected = append(selected, item)
		}
	}
	return selected
}

// SelectedCount returns the number of selected items
func (m MultiSelectModel) SelectedCount() int {
	return len(m.GetSelected())
}

// IsDone returns whether the user finished selecting
func (m MultiSelectModel) IsDone() bool {
	return m.done
}
