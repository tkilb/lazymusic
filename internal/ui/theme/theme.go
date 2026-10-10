package theme

import "github.com/charmbracelet/lipgloss"

type ThemeType int

const (
	ThemeCatppuccin ThemeType = iota
	ThemeGruvbox
	ThemeDracula
)

type ThemePalette struct {
	Name           string
	PrimaryColor   lipgloss.Color
	SecondaryColor lipgloss.Color
	HighlightColor lipgloss.Color
	MutedColor     lipgloss.Color
	SubtleColor    lipgloss.Color
	ActiveBorder   lipgloss.Color
	InactiveBorder lipgloss.Color
	TitleColor     lipgloss.Color
}

var Palettes = map[ThemeType]ThemePalette{
	ThemeCatppuccin: {
		Name:           "Catppuccin Mocha",
		PrimaryColor:   lipgloss.Color("#89b4fa"), // Blue
		SecondaryColor: lipgloss.Color("#fab387"), // Peach
		HighlightColor: lipgloss.Color("#f38ba8"), // Red/Pink
		MutedColor:     lipgloss.Color("#45475a"), // Surface0
		SubtleColor:    lipgloss.Color("#a6adc8"), // Subtext0
		ActiveBorder:   lipgloss.Color("#89b4fa"), // Blue
		InactiveBorder: lipgloss.Color("#313244"), // Surface1
		TitleColor:     lipgloss.Color("#cdd6f4"), // Text
	},
	ThemeGruvbox: {
		Name:           "Gruvbox Dark",
		PrimaryColor:   lipgloss.Color("#83a598"), // Blue
		SecondaryColor: lipgloss.Color("#fe8019"), // Orange
		HighlightColor: lipgloss.Color("#fb4934"), // Red
		MutedColor:     lipgloss.Color("#504945"), // Dark3
		SubtleColor:    lipgloss.Color("#a89984"), // Gray
		ActiveBorder:   lipgloss.Color("#fabd2f"), // Yellow
		InactiveBorder: lipgloss.Color("#3c3836"), // Dark1
		TitleColor:     lipgloss.Color("#ebdbb2"), // Light1
	},
	ThemeDracula: {
		Name:           "Dracula",
		PrimaryColor:   lipgloss.Color("#8be9fd"), // Cyan
		SecondaryColor: lipgloss.Color("#ffb86c"), // Orange
		HighlightColor: lipgloss.Color("#ff79c6"), // Pink
		MutedColor:     lipgloss.Color("#44475a"), // Current line
		SubtleColor:    lipgloss.Color("#6272a4"), // Comment
		ActiveBorder:   lipgloss.Color("#bd93f9"), // Purple
		InactiveBorder: lipgloss.Color("#282a36"), // Background
		TitleColor:     lipgloss.Color("#f8f8f2"), // Foreground
	},
}

// Active Palette
var CurrentTheme = ThemeCatppuccin

func SetTheme(t ThemeType) {
	CurrentTheme = t
	p := Palettes[t]

	PrimaryColor = p.PrimaryColor
	SecondaryColor = p.SecondaryColor
	HighlightColor = p.HighlightColor
	MutedColor = p.MutedColor
	SubtleColor = p.SubtleColor
	ActiveBorder = p.ActiveBorder
	InactiveBorder = p.InactiveBorder
	TitleColor = p.TitleColor

	updateStyles()
}

func CycleTheme() string {
	CurrentTheme = (CurrentTheme + 1) % 3
	SetTheme(CurrentTheme)
	return Palettes[CurrentTheme].Name
}

var (
	PrimaryColor   = Palettes[ThemeCatppuccin].PrimaryColor
	SecondaryColor = Palettes[ThemeCatppuccin].SecondaryColor
	HighlightColor = Palettes[ThemeCatppuccin].HighlightColor
	MutedColor     = Palettes[ThemeCatppuccin].MutedColor
	SubtleColor    = Palettes[ThemeCatppuccin].SubtleColor
	ActiveBorder   = Palettes[ThemeCatppuccin].ActiveBorder
	InactiveBorder = Palettes[ThemeCatppuccin].InactiveBorder
	TitleColor     = Palettes[ThemeCatppuccin].TitleColor

	BasePanelStyle    lipgloss.Style
	FocusedPanelStyle lipgloss.Style
	TabActiveStyle    lipgloss.Style
	TabInactiveStyle  lipgloss.Style
	PanelTitleStyle   lipgloss.Style
	CursorItemStyle   lipgloss.Style
	NormalItemStyle   lipgloss.Style
	StatusBarStyle    lipgloss.Style
	StatusLabelStyle  lipgloss.Style
	KeybindKeyStyle   lipgloss.Style
	KeybindDescStyle  lipgloss.Style
	ModalBoxStyle     lipgloss.Style
)

func updateStyles() {
	BasePanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(InactiveBorder)

	FocusedPanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ActiveBorder)

	TabActiveStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(HighlightColor).
		Underline(true)

	TabInactiveStyle = lipgloss.NewStyle().
		Foreground(SubtleColor)

	PanelTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(TitleColor)

	CursorItemStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(HighlightColor)

	NormalItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

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

	ModalBoxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(ActiveBorder).
		Padding(1, 2)
}

func init() {
	updateStyles()
}
