package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Title styling
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F28C28")).
			MarginBottom(1)

	// Header styling for steps
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#F28C28")).
			Padding(0, 1)

	// Selected item styling
	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F28C28")).
			Bold(true)

	// Checkbox styling
	CheckedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).
			Bold(true)

	UncheckedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	// Help text styling
	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginTop(1)

	// Error styling
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

	// Success styling
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).
			Bold(true)

	// Border styling
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F28C28")).
			Padding(1, 2)

	// Subtle text styling
	SubtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	// Focus styling
	FocusedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F28C28"))

	// Description styling
	DescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2D73FF")).
			Italic(true)

	// Key hint styling
	KeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	KeyDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)

// NewHuhTheme wires the vivid orange/blue palette into huh forms, including key
// hint coloring.
func NewHuhTheme() *huh.Theme {
	primary := lipgloss.Color("#F28C28")
	secondary := lipgloss.Color("#2D73FF")
	success := lipgloss.Color("#22C55E")
	errorColor := lipgloss.Color("#EF4444")
	neutral := lipgloss.Color("#888888")
	text := lipgloss.Color("#FFFFFF")
	mutedBg := lipgloss.Color("#3F3F3F")

	t := huh.ThemeBase()

	t.FieldSeparator = t.FieldSeparator.SetString("\n\n")

	// t.Form.Base = t.Form.Base
	t.Form.Base = t.Form.Base.Padding(1, 2)

	t.Focused.Base = t.Focused.Base.MarginTop(1).Padding(0, 2).BorderForeground(neutral)
	t.Focused.Card = t.Focused.Card.Padding(1, 2)
	t.Focused.Title = t.Focused.Title.Foreground(primary).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(primary).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(secondary)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(errorColor)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(errorColor)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(primary)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(primary)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(primary)
	t.Focused.Option = t.Focused.Option.Foreground(text)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(primary)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(success)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(success).SetString("✓ ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(text)
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(mutedBg).SetString("• ")
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("#000000")).Background(primary).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(text).Background(mutedBg)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(primary)
	t.Focused.TextInput.CursorText = t.Focused.TextInput.CursorText.Foreground(text)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(neutral)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(primary)
	t.Focused.TextInput.Text = t.Focused.TextInput.Text.Foreground(text)
	t.Focused.Directory = t.Focused.Directory.Foreground(primary)
	t.Focused.File = t.Focused.File.Foreground(text)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder()).BorderForeground(neutral)
	t.Blurred.Card = t.Blurred.Card.BorderStyle(lipgloss.HiddenBorder()).BorderForeground(neutral)
	t.Blurred.MultiSelectSelector = t.Blurred.MultiSelectSelector.Foreground(neutral)
	t.Blurred.SelectSelector = t.Blurred.SelectSelector.Foreground(neutral)
	t.Blurred.SelectedOption = t.Blurred.SelectedOption.Foreground(success)
	t.Blurred.UnselectedOption = t.Blurred.UnselectedOption.Foreground(text)
	t.Blurred.SelectedPrefix = t.Blurred.SelectedPrefix.Foreground(neutral)
	t.Blurred.UnselectedPrefix = t.Blurred.UnselectedPrefix.Foreground(neutral)
	t.Blurred.FocusedButton = t.Blurred.FocusedButton.Foreground(text).Background(primary)
	t.Blurred.BlurredButton = t.Blurred.BlurredButton.Foreground(text).Background(mutedBg)
	t.Blurred.TextInput.Placeholder = t.Blurred.TextInput.Placeholder.Foreground(neutral)
	t.Blurred.TextInput.Text = t.Blurred.TextInput.Text.Foreground(text)
	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(neutral)
	t.Blurred.TextInput.Cursor = t.Blurred.TextInput.Cursor.Foreground(secondary)
	t.Blurred.TextInput.CursorText = t.Blurred.TextInput.CursorText.Foreground(text)

	t.Group.Title = t.Group.Title.Foreground(primary).Bold(true)
	t.Group.Description = t.Group.Description.Foreground(neutral)

	t.Help.Ellipsis = t.Help.Ellipsis.Foreground(neutral)
	t.Help.ShortKey = t.Help.ShortKey.Foreground(text).Bold(true)
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(neutral)
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(neutral)
	t.Help.FullKey = t.Help.FullKey.Foreground(text).Bold(true)
	t.Help.FullDesc = t.Help.FullDesc.Foreground(neutral)
	t.Help.FullSeparator = t.Help.FullSeparator.Foreground(neutral)

	return t
}
