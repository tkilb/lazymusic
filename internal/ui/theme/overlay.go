package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Overlay composites box on top of background, centered within a
// screenWidth x screenHeight terminal area, preserving whatever of
// background falls outside the box's bounds (unlike lipgloss.Place,
// which blanks the background with whitespace). Both background and box
// may contain ANSI styling; splicing is done ANSI- and wide-rune-aware via
// charmbracelet/x/ansi so escape sequences and multi-cell runes aren't torn.
func Overlay(background, box string, screenWidth, screenHeight int) string {
	if screenWidth <= 0 || screenHeight <= 0 {
		return box
	}

	bgLines := strings.Split(background, "\n")
	for len(bgLines) < screenHeight {
		bgLines = append(bgLines, "")
	}

	boxLines := strings.Split(box, "\n")
	boxWidth := 0
	for _, l := range boxLines {
		if w := ansi.StringWidth(l); w > boxWidth {
			boxWidth = w
		}
	}
	boxHeight := len(boxLines)

	startRow := (screenHeight - boxHeight) / 2
	startCol := (screenWidth - boxWidth) / 2
	if startRow < 0 {
		startRow = 0
	}
	if startCol < 0 {
		startCol = 0
	}

	out := make([]string, len(bgLines))
	copy(out, bgLines)

	for i, boxLine := range boxLines {
		row := startRow + i
		if row < 0 || row >= len(out) {
			continue
		}
		bgLine := out[row]
		if w := ansi.StringWidth(bgLine); w < screenWidth {
			bgLine += strings.Repeat(" ", screenWidth-w)
		}

		if w := ansi.StringWidth(boxLine); w < boxWidth {
			boxLine += strings.Repeat(" ", boxWidth-w)
		}

		left := ansi.Cut(bgLine, 0, startCol)
		right := ansi.Cut(bgLine, startCol+boxWidth, screenWidth)
		out[row] = left + boxLine + right
	}

	return strings.Join(out, "\n")
}

// PopupBox renders a lazytask-style bordered modal/popup box with the title embedded
// on the left and key-binding hint embedded on the right of the top border:
// "╭─ Title ────────────────────────── <hint> ─╮"
// "│ Content                                    │"
// "╰────────────────────────────────────────────╯"
func PopupBox(title, hint, content string, screenWidth int, borderCol ...lipgloss.Color) string {
	col := FocusedColor
	if len(borderCol) > 0 && borderCol[0] != "" {
		col = borderCol[0]
	}

	const minWidth = 48
	const maxWidth = 64

	width := maxWidth
	if screenWidth > 0 && screenWidth-4 < width {
		width = screenWidth - 4
	}
	if width < minWidth {
		width = minWidth
	}
	innerWidth := width - 2
	if innerWidth < 16 {
		innerWidth = 16
	}

	border := lipgloss.RoundedBorder()
	lineStyle := lipgloss.NewStyle().Foreground(col)

	titleText := " " + lipgloss.NewStyle().Bold(true).Foreground(col).Render(title) + " "
	var hintText string
	if hint != "" {
		hintText = " " + lipgloss.NewStyle().Foreground(col).Render(hint) + " "
	}

	totalWidth := innerWidth + 2
	fillWidth := totalWidth - 2 - lipgloss.Width(titleText) - lipgloss.Width(hintText)
	if fillWidth < 1 {
		fillWidth = 1
	}

	var top strings.Builder
	top.WriteString(lineStyle.Render(border.TopLeft))
	top.WriteString(titleText)
	top.WriteString(lineStyle.Render(strings.Repeat(border.Top, fillWidth)))
	if hintText != "" {
		top.WriteString(hintText)
	}
	top.WriteString(lineStyle.Render(border.TopRight))

	bodyStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(col).
		Width(innerWidth)

	body := bodyStyle.Render(content)
	return top.String() + "\n" + body
}

// FieldBox renders a single bordered field box for multi-field forms (matching lazytask's taskform):
// e.g. "╭─ Title ────────────────────────────────╮"
func FieldBox(title, hint, content string, focused bool, screenWidth int) string {
	col := UnfocusedColor
	if focused {
		col = FocusedColor
	}
	return PopupBox(title, hint, content, screenWidth, col)
}

