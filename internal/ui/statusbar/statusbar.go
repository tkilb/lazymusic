package statusbar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// keyStyle uses ANSI blue (SGR 34), matching lazygit's default options bar
	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("4"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	separator = "  "
)

// Binding describes a single keybinding entry.
type Binding struct {
	Key   string
	Label string
}

// Render lays out bindings as a lazygit-style line, e.g. "Space pause  > next  q detach"
func Render(bindings []Binding) string {
	if len(bindings) == 0 {
		return ""
	}

	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		parts = append(parts, keyStyle.Render(b.Key)+" "+labelStyle.Render(b.Label))
	}
	return strings.Join(parts, separator)
}
