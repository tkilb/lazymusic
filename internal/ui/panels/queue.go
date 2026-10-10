package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// QueuePanel implements Panel for Panel 0 (Queue).
type QueuePanel struct {
	focused bool
	width   int
	height  int
	cursor  int
	items   []string
}

// NewQueuePanel creates a new QueuePanel instance.
func NewQueuePanel() *QueuePanel {
	return &QueuePanel{
		cursor: 1,
		items: []string{
			"Airbag - Radiohead",
			"Paranoid Android - Radiohead",
			"Subterranean Homesick Alien - Radiohead",
			"Exit Music (For a Film) - Radiohead",
			"Let Down - Radiohead",
			"Karma Police - Radiohead",
			"Time - Pink Floyd",
			"Money - Pink Floyd",
			"Us and Them - Pink Floyd",
			"Brain Damage - Pink Floyd",
		},
	}
}

func (p *QueuePanel) ID() PanelID {
	return PanelQueue
}

func (p *QueuePanel) Title() string {
	return "0: Queue"
}

func (p *QueuePanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *QueuePanel) Focus() {
	p.focused = true
}

func (p *QueuePanel) Blur() {
	p.focused = false
}

func (p *QueuePanel) IsFocused() bool {
	return p.focused
}

func (p *QueuePanel) NextTab() {}

func (p *QueuePanel) PrevTab() {}

func (p *QueuePanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *QueuePanel) CursorDown() {
	if p.cursor < len(p.items)-1 {
		p.cursor++
	}
}

func (p *QueuePanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *QueuePanel) View() string {
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

	header := fmt.Sprintf("%s (%d tracks)", theme.PanelTitleStyle.Render("0: Queue"), len(p.items))
	var lines []string
	lines = append(lines, header)

	visibleSlots := innerHeight - 1
	if visibleSlots < 0 {
		visibleSlots = 0
	}

	for i := 0; i < len(p.items) && i < visibleSlots; i++ {
		item := p.items[i]
		prefix := fmt.Sprintf("%2d. ", i+1)
		if i == p.cursor && p.focused {
			lines = append(lines, theme.CursorItemStyle.Render(fmt.Sprintf(" > %s%s", prefix, item)))
		} else if i == p.cursor && !p.focused {
			lines = append(lines, theme.NormalItemStyle.Render(fmt.Sprintf(" > %s%s", prefix, item)))
		} else {
			lines = append(lines, theme.NormalItemStyle.Render(fmt.Sprintf("   %s%s", prefix, item)))
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:min(len(lines), innerHeight)], "\n")
	return style.Width(innerWidth).Height(innerHeight).Render(content)
}
