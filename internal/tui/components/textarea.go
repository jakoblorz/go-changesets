package components

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// TextAreaModel wraps the bubbles textarea for our use
type TextAreaModel struct {
	textarea textarea.Model
	done     bool
}

// NewTextArea creates a new textarea component
func NewTextArea(placeholder string) TextAreaModel {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.Focus()
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(5)

	return TextAreaModel{
		textarea: ta,
		done:     false,
	}
}

// Init initializes the component
func (m TextAreaModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages
func (m TextAreaModel) Update(msg tea.Msg) (TextAreaModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlD, tea.KeyCtrlS:
			// Ctrl+D or Ctrl+S to finish
			m.done = true
			return m, tea.Quit
		case tea.KeyEsc:
			// Esc to cancel
			m.done = false
			m.textarea.SetValue("")
			return m, tea.Quit
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the component
func (m TextAreaModel) View() string {
	if m.done {
		return ""
	}
	return m.textarea.View()
}

// GetValue returns the text area value
func (m TextAreaModel) GetValue() string {
	return m.textarea.Value()
}

// IsDone returns whether the user finished editing
func (m TextAreaModel) IsDone() bool {
	return m.done
}

// SetValue sets the text area value
func (m *TextAreaModel) SetValue(value string) {
	m.textarea.SetValue(value)
}

// Focus focuses the textarea
func (m *TextAreaModel) Focus() tea.Cmd {
	return m.textarea.Focus()
}

// Blur unfocuses the textarea
func (m *TextAreaModel) Blur() {
	m.textarea.Blur()
}
