package tui

import "github.com/charmbracelet/lipgloss"

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
