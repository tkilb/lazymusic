package theme

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	PrimaryColor   = lipgloss.Color("39")  // Cyan / Blue
	SecondaryColor = lipgloss.Color("208") // Orange / Amber
	HighlightColor = lipgloss.Color("205") // Magenta / Pink
	MutedColor     = lipgloss.Color("240") // Dark Gray
	SubtleColor    = lipgloss.Color("245") // Light Gray
	ActiveBorder   = lipgloss.Color("39")  // Cyan
	InactiveBorder = lipgloss.Color("238") // Muted Border
	TitleColor     = lipgloss.Color("229") // Cream / Light Yellow

	// Panel Styles
	BasePanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(InactiveBorder)

	FocusedPanelStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ActiveBorder)

	// Header / Tab Styles
	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(HighlightColor).
			Underline(true)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(SubtleColor)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TitleColor)

	// List item styles
	CursorItemStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(HighlightColor)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(InactiveBorder).
			Foreground(lipgloss.Color("252"))

	StatusLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(PrimaryColor)

	KeybindKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(SecondaryColor)

	KeybindDescStyle = lipgloss.NewStyle().
				Foreground(SubtleColor)
)
