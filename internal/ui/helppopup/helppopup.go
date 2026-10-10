package helppopup

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/tylerkilburn/lazymusic/internal/ui/statusbar"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// Section is a named group of bindings shown under its own header, e.g. "Local", "Global", "Navigation".
type Section struct {
	Title    string
	Bindings []statusbar.Binding
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.FocusedColor)
	keyStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")) // Cyan key column
)

const (
	minWidth       = 40
	maxWidth       = 70
	minVisibleRows = 8
	maxVisibleRows = 30
	reservedRows   = 6
)

func VisibleRows(screenHeight int) int {
	if screenHeight <= 0 {
		return maxVisibleRows
	}
	rows := screenHeight - reservedRows
	if rows > maxVisibleRows {
		rows = maxVisibleRows
	}
	if rows < minVisibleRows {
		rows = minVisibleRows
	}
	return rows
}

func sectionHeader(title string) string {
	return "--- " + title + " ---"
}

func keyWidth(sections []Section) int {
	width := 0
	for _, sec := range sections {
		for _, b := range sec.Bindings {
			if w := ansi.StringWidth(b.Key); w > width {
				width = w
			}
		}
	}
	return width
}

func bodyLines(sections []Section) []string {
	width := keyWidth(sections)
	headerIndent := strings.Repeat(" ", width+2)
	var lines []string
	for i, sec := range sections {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, headerIndent+headerStyle.Render(sectionHeader(sec.Title)))
		for _, bind := range sec.Bindings {
			key := keyStyle.Render(padLeft(bind.Key, width))
			lines = append(lines, key+"  "+bind.Label)
		}
	}
	return lines
}

func MaxScroll(sections []Section, screenHeight int) int {
	total := len(bodyLines(sections))
	visible := VisibleRows(screenHeight)
	if total <= visible {
		return 0
	}
	return total - visible
}

// Box renders sections as a bordered, scrollable "Keybindings" box matching lazygit.
func Box(sections []Section, screenWidth, screenHeight, scrollOffset int) string {
	boxWidth := maxWidth
	if screenWidth > 0 && screenWidth-4 < boxWidth {
		boxWidth = screenWidth - 4
	}
	if boxWidth < minWidth {
		boxWidth = minWidth
	}
	contentWidth := boxWidth - 2

	lines := bodyLines(sections)
	visibleRows := VisibleRows(screenHeight)
	maxScroll := MaxScroll(sections, screenHeight)
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	end := scrollOffset + visibleRows
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[scrollOffset:end]

	footer := "esc/? close"
	if maxScroll > 0 {
		footer = fmt.Sprintf("%d-%d/%d  ↑/↓ scroll  ", scrollOffset+1, end, len(lines)) + footer
	}

	body := strings.Join(visible, "\n")
	return theme.Frame("Keybindings", body, contentWidth, len(visible), true, footer)
}

func padLeft(s string, width int) string {
	w := ansi.StringWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}
