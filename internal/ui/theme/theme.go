package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Standard lazygit palette definitions matching lazygit's default theme:
// - activeBorderColor: green (ANSI 2)
// - inactiveBorderColor: default (terminal foreground / "")
// - optionsTextColor: blue (ANSI 4)
// - selectedLineBgColor: blue (ANSI 4)
// - inactiveTabColor: default (terminal foreground / "")
// - activeTabColor: blue (ANSI 4)
// - searchMatchColor: cyan (ANSI 6)
// - status dim label: 245
const (
	PaletteDefault = lipgloss.Color("")
	PaletteGreen   = lipgloss.Color("2")
	PaletteBlue    = lipgloss.Color("4")
	PaletteRed     = lipgloss.Color("1")
	PaletteYellow  = lipgloss.Color("3")
	PaletteCyan    = lipgloss.Color("6")
	PaletteDim     = lipgloss.Color("245")
)

var (
	// Border / Focus colors
	UnfocusedColor = PaletteDefault
	FocusedColor   = PaletteGreen

	// Selection & Active colors
	SelectedRowBg  = PaletteBlue
	ActiveTabColor = PaletteBlue
	KeyColor       = PaletteBlue

	// Styles
	BasePanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(UnfocusedColor)

	FocusedPanelStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(FocusedColor)

	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ActiveTabColor)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(PaletteDefault)

	PanelTitleFocused = lipgloss.NewStyle().
				Bold(true).
				Foreground(FocusedColor)

	PanelTitleUnfocused = lipgloss.NewStyle().
				Bold(true).
				Foreground(UnfocusedColor)

	// List cursor item style: full row highlight with blue background like lazygit
	CursorRowStyle = lipgloss.NewStyle().
			Background(SelectedRowBg).
			Bold(true)

	NormalRowStyle = lipgloss.NewStyle()

	// Status bar / Bottom help bar
	StatusBarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(UnfocusedColor)

	StatusKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(KeyColor)

	StatusDescStyle = lipgloss.NewStyle().
			Foreground(PaletteDim)

	ModalBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(FocusedColor).
			Padding(1, 2)
)

// BorderColor returns the border color based on focus.
func BorderColor(focused bool) lipgloss.Color {
	if focused {
		return FocusedColor
	}
	return UnfocusedColor
}

// TitleStyle returns the title style based on focus.
func TitleStyle(focused bool) lipgloss.Style {
	if focused {
		return PanelTitleFocused
	}
	return PanelTitleUnfocused
}

// Frame renders a lazygit-style panel with title embedded in the top border:
// e.g. "╭─[1] Library─────╮"
func Frame(title, body string, width, height int, focused bool, footer ...string) string {
	lineStyle := lipgloss.NewStyle().Foreground(BorderColor(focused))
	top := topBorder(title, width, focused, lineStyle)
	return frameBody(top, firstFooter(footer), body, width, height, lineStyle)
}

// Tab represents a tab element in FrameTabs.
type Tab struct {
	Label  string
	Active bool
}

// FrameTabs renders a lazygit-style panel with tabs embedded in the top border:
// e.g. "╭─[1] Dirs - Artists - Albums - Tracks────╮"
func FrameTabs(titlePrefix string, tabs []Tab, body string, width, height int, focused bool, footer ...string) string {
	lineStyle := lipgloss.NewStyle().Foreground(BorderColor(focused))
	top := topBorderTabs(titlePrefix, tabs, width, focused, lineStyle)
	return frameBody(top, firstFooter(footer), body, width, height, lineStyle)
}

func firstFooter(footer []string) string {
	if len(footer) == 0 {
		return ""
	}
	return footer[0]
}

func frameBody(top, footer, body string, width, height int, lineStyle lipgloss.Style) string {
	bottom := bottomBorder(footer, width, lineStyle)

	content := lipgloss.NewStyle().Width(width).Height(height).Render(body)

	var b strings.Builder
	b.WriteString(top)
	b.WriteString("\n")
	side := lineStyle.Render("│")
	lines := strings.Split(content, "\n")
	for i := 0; i < height; i++ {
		b.WriteString(side)
		if i < len(lines) {
			b.WriteString(lines[i])
		} else {
			b.WriteString(strings.Repeat(" ", width))
		}
		b.WriteString(side)
		b.WriteString("\n")
	}
	b.WriteString(bottom)
	return b.String()
}

func topBorder(title string, width int, focused bool, lineStyle lipgloss.Style) string {
	const leadDashes = 2
	label := title
	if len(label) > width-4 {
		label = label[:width-4]
	}

	fill := width - leadDashes - len(label)
	if fill < 0 {
		fill = 0
	}

	var b strings.Builder
	b.WriteString(lineStyle.Render("╭" + strings.Repeat("─", leadDashes)))
	b.WriteString(TitleStyle(focused).Render(label))
	b.WriteString(lineStyle.Render(strings.Repeat("─", fill) + "╮"))
	return b.String()
}

func topBorderTabs(prefix string, tabs []Tab, width int, focused bool, lineStyle lipgloss.Style) string {
	const leadDashes = 2

	var b strings.Builder
	b.WriteString(lineStyle.Render("╭" + strings.Repeat("─", leadDashes)))
	b.WriteString(TitleStyle(focused).Render(prefix + " "))

	renderedLen := leadDashes + len(prefix) + 1

	for i, t := range tabs {
		if i > 0 {
			b.WriteString(lineStyle.Render(" - "))
			renderedLen += 3
		}
		if t.Active {
			b.WriteString(TabActiveStyle.Render(t.Label))
		} else {
			b.WriteString(TabInactiveStyle.Render(t.Label))
		}
		renderedLen += len(t.Label)
	}

	fill := width - renderedLen
	if fill < 0 {
		fill = 0
	}
	b.WriteString(lineStyle.Render(strings.Repeat("─", fill) + "╮"))
	return b.String()
}

func bottomBorder(footer string, width int, lineStyle lipgloss.Style) string {
	if footer == "" || len(footer) > width {
		return lineStyle.Render("╰" + strings.Repeat("─", width) + "╯")
	}

	const trailDashes = 1
	fill := width - trailDashes - len(footer)
	if fill < 0 {
		fill = 0
	}

	var b strings.Builder
	b.WriteString(lineStyle.Render("╰" + strings.Repeat("─", fill)))
	b.WriteString(lipgloss.NewStyle().Foreground(PaletteDim).Render(footer))
	b.WriteString(lineStyle.Render(strings.Repeat("─", trailDashes) + "╯"))
	return b.String()
}
