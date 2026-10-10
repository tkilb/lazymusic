package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// LibraryPanel implements Panel for Panel 1 (Library).
type LibraryPanel struct {
	focused bool
	width   int
	height  int
	tabs    []string
	curTab  int
	cursor  int
	items   map[int][]string
}

// NewLibraryPanel creates a new LibraryPanel instance.
func NewLibraryPanel() *LibraryPanel {
	return &LibraryPanel{
		tabs:   []string{"Dirs", "Artists", "Albums", "Tracks"},
		curTab: 0,
		cursor: 0,
		items: map[int][]string{
			0: {"~/Music", "~/Music/Daft Punk", "~/Music/Pink Floyd", "~/Music/Radiohead"},
			1: {"Daft Punk", "Pink Floyd", "Radiohead"},
			2: {"Discovery", "The Dark Side of the Moon", "Wish You Were Here", "OK Computer", "Kid A"},
			3: {"Airbag", "Paranoid Android", "Subterranean Homesick Alien", "Exit Music (For a Film)", "Let Down", "Karma Police"},
		},
	}
}

func (p *LibraryPanel) ID() PanelID {
	return PanelLibrary
}

func (p *LibraryPanel) Title() string {
	return "1: Library"
}

func (p *LibraryPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *LibraryPanel) Focus() {
	p.focused = true
}

func (p *LibraryPanel) Blur() {
	p.focused = false
}

func (p *LibraryPanel) IsFocused() bool {
	return p.focused
}

func (p *LibraryPanel) NextTab() {
	p.curTab = (p.curTab + 1) % len(p.tabs)
	p.cursor = 0
}

func (p *LibraryPanel) PrevTab() {
	p.curTab = (p.curTab - 1 + len(p.tabs)) % len(p.tabs)
	p.cursor = 0
}

func (p *LibraryPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *LibraryPanel) CursorDown() {
	list := p.items[p.curTab]
	if p.cursor < len(list)-1 {
		p.cursor++
	}
}

func (p *LibraryPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *LibraryPanel) View() string {
	// Frame styling
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

	// Render tabs
	var tabStrings []string
	for i, tab := range p.tabs {
		if i == p.curTab {
			tabStrings = append(tabStrings, theme.TabActiveStyle.Render(fmt.Sprintf("[%s]", tab)))
		} else {
			tabStrings = append(tabStrings, theme.TabInactiveStyle.Render(tab))
		}
	}
	header := fmt.Sprintf("%s %s", theme.PanelTitleStyle.Render("1: Library"), strings.Join(tabStrings, " "))

	var lines []string
	lines = append(lines, header)

	currentItems := p.items[p.curTab]
	visibleSlots := innerHeight - 1
	if visibleSlots < 0 {
		visibleSlots = 0
	}

	for i := 0; i < len(currentItems) && i < visibleSlots; i++ {
		item := currentItems[i]
		if i == p.cursor && p.focused {
			lines = append(lines, theme.CursorItemStyle.Render(fmt.Sprintf(" > %s", item)))
		} else if i == p.cursor && !p.focused {
			lines = append(lines, theme.NormalItemStyle.Render(fmt.Sprintf(" > %s", item)))
		} else {
			lines = append(lines, theme.NormalItemStyle.Render(fmt.Sprintf("   %s", item)))
		}
	}

	// Pad lines to innerHeight
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:min(len(lines), innerHeight)], "\n")
	return style.Width(innerWidth).Height(innerHeight).Render(content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
