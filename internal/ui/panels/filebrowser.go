package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// FileBrowserPanel implements Panel for Panel 3 (File Browser).
type FileBrowserPanel struct {
	focused bool
	width   int
	height  int
	cursor  int
	items   []string
}

// NewFileBrowserPanel creates a new FileBrowserPanel instance.
func NewFileBrowserPanel() *FileBrowserPanel {
	return &FileBrowserPanel{
		cursor: 0,
		items: []string{
			"../",
			"Daft Punk - Discovery (2001)/",
			"Pink Floyd - Animals (1977)/",
			"Pink Floyd - Dark Side (1973)/",
			"Radiohead - OK Computer (1997)/",
			"bonus_track.mp3",
		},
	}
}

func (p *FileBrowserPanel) ID() PanelID {
	return PanelFileBrowser
}

func (p *FileBrowserPanel) Title() string {
	return "3: File Browser"
}

func (p *FileBrowserPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *FileBrowserPanel) Focus() {
	p.focused = true
}

func (p *FileBrowserPanel) Blur() {
	p.focused = false
}

func (p *FileBrowserPanel) IsFocused() bool {
	return p.focused
}

func (p *FileBrowserPanel) NextTab() {}

func (p *FileBrowserPanel) PrevTab() {}

func (p *FileBrowserPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *FileBrowserPanel) CursorDown() {
	if p.cursor < len(p.items)-1 {
		p.cursor++
	}
}

func (p *FileBrowserPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *FileBrowserPanel) View() string {
	style := theme.BasePanelStyle
	if p.focused {
		style = theme.FocusedPanelStyle
	}

	innerWidth := p.width - 2
	innerHeight := p.height - 2
	if innerWidth < 0 {
		innerWidth = 0
	}
	if innerHeight < 0 {
		innerHeight = 0
	}

	header := theme.PanelTitleStyle.Render("3: File Browser")
	var lines []string
	lines = append(lines, header)

	visibleSlots := innerHeight - 1
	if visibleSlots < 0 {
		visibleSlots = 0
	}

	for i := 0; i < len(p.items) && i < visibleSlots; i++ {
		item := p.items[i]
		if i == p.cursor && p.focused {
			lines = append(lines, theme.CursorItemStyle.Render(fmt.Sprintf(" > %s", item)))
		} else if i == p.cursor && !p.focused {
			lines = append(lines, theme.NormalItemStyle.Render(fmt.Sprintf(" > %s", item)))
		} else {
			lines = append(lines, theme.NormalItemStyle.Render(fmt.Sprintf("   %s", item)))
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:min(len(lines), innerHeight)], "\n")
	return style.Width(innerWidth).Height(innerHeight).Render(content)
}
