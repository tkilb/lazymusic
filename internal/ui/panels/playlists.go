package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// PlaylistsPanel implements Panel for Panel 2 (Playlists).
type PlaylistsPanel struct {
	focused bool
	width   int
	height  int
	cursor  int
	items   []string
}

// NewPlaylistsPanel creates a new PlaylistsPanel instance.
func NewPlaylistsPanel() *PlaylistsPanel {
	return &PlaylistsPanel{
		cursor: 0,
		items:  []string{"★ Favorites", "Night Vibes", "Coding Mix", "Road Trip", "Chillout"},
	}
}

func (p *PlaylistsPanel) ID() PanelID {
	return PanelPlaylists
}

func (p *PlaylistsPanel) Title() string {
	return "2: Playlists"
}

func (p *PlaylistsPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *PlaylistsPanel) Focus() {
	p.focused = true
}

func (p *PlaylistsPanel) Blur() {
	p.focused = false
}

func (p *PlaylistsPanel) IsFocused() bool {
	return p.focused
}

func (p *PlaylistsPanel) NextTab() {}

func (p *PlaylistsPanel) PrevTab() {}

func (p *PlaylistsPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *PlaylistsPanel) CursorDown() {
	if p.cursor < len(p.items)-1 {
		p.cursor++
	}
}

func (p *PlaylistsPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *PlaylistsPanel) View() string {
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

	header := theme.PanelTitleStyle.Render("2: Playlists")
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
