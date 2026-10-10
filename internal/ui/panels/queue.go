package panels

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tylerkilburn/lazymusic/internal/player"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// QueuePanel implements Panel for Panel 0 (Queue).
type QueuePanel struct {
	focused     bool
	width       int
	height      int
	cursor      int
	playlist    []player.PlaylistItem
	currentPos  int
	filterQuery string
}

// NewQueuePanel creates a new QueuePanel instance.
func NewQueuePanel() *QueuePanel {
	return &QueuePanel{
		cursor:     0,
		playlist:   nil,
		currentPos: -1,
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

func (p *QueuePanel) filteredPlaylist() []player.PlaylistItem {
	if p.filterQuery == "" {
		return p.playlist
	}
	query := strings.ToLower(p.filterQuery)
	var filtered []player.PlaylistItem
	for _, it := range p.playlist {
		title := it.Title
		if title == "" {
			title = filepath.Base(it.Filename)
		}
		if strings.Contains(strings.ToLower(title), query) {
			filtered = append(filtered, it)
		}
	}
	return filtered
}

func (p *QueuePanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *QueuePanel) CursorDown() {
	list := p.filteredPlaylist()
	if p.cursor < len(list)-1 {
		p.cursor++
	}
}

// Cursor returns current cursor position.
func (p *QueuePanel) Cursor() int {
	return p.cursor
}

// SetFilterQuery sets the search filter query for Queue.
func (p *QueuePanel) SetFilterQuery(query string) {
	p.filterQuery = query
	p.cursor = 0
}

// FilterQuery returns the search filter query for Queue.
func (p *QueuePanel) FilterQuery() string {
	return p.filterQuery
}

// SetPlaylist updates the queue with mpv's active playlist.
func (p *QueuePanel) SetPlaylist(items []player.PlaylistItem, currentPos int) {
	p.playlist = items
	p.currentPos = currentPos
	if p.cursor >= len(p.playlist) {
		p.cursor = len(p.playlist) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// SelectedTrack returns the selected item index and track.
func (p *QueuePanel) SelectedTrack() (int, player.PlaylistItem, bool) {
	filtered := p.filteredPlaylist()
	if p.cursor >= 0 && p.cursor < len(filtered) {
		selectedItem := filtered[p.cursor]
		for originalIdx, item := range p.playlist {
			if item.ID == selectedItem.ID {
				return originalIdx, item, true
			}
		}
	}
	return -1, player.PlaylistItem{}, false
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

	header := fmt.Sprintf("%s (%d tracks)", theme.PanelTitleStyle.Render("0: Queue"), len(p.playlist))
	if p.filterQuery != "" {
		header += fmt.Sprintf(" %s", theme.KeybindDescStyle.Render("/"+p.filterQuery))
	}

	var lines []string
	lines = append(lines, header)

	filtered := p.filteredPlaylist()

	visibleSlots := innerHeight - 1
	if visibleSlots < 0 {
		visibleSlots = 0
	}

	if len(filtered) == 0 {
		if p.filterQuery != "" {
			lines = append(lines, theme.KeybindDescStyle.Render("  [No matching tracks found]"))
		} else {
			lines = append(lines, theme.KeybindDescStyle.Render("  [Queue is empty - press 'a' or Enter on tracks to add]"))
		}
	} else {
		// Scrolling calculation
		offset := 0
		if p.cursor >= visibleSlots {
			offset = p.cursor - visibleSlots + 1
		}

		for i := offset; i < len(filtered) && len(lines) < innerHeight; i++ {
			item := filtered[i]
			title := item.Title
			if title == "" {
				title = filepath.Base(item.Filename)
			}

			playingIndicator := " "
			if item.Current || item.Playing {
				playingIndicator = "▶"
			}

			prefix := fmt.Sprintf("%s%2d. ", playingIndicator, i+1)

			if i == p.cursor && p.focused {
				lines = append(lines, theme.CursorItemStyle.Render(fmt.Sprintf(" > %s%s", prefix, title)))
			} else if i == p.cursor && !p.focused {
				lines = append(lines, theme.NormalItemStyle.Render(fmt.Sprintf(" > %s%s", prefix, title)))
			} else {
				lines = append(lines, theme.NormalItemStyle.Render(fmt.Sprintf("   %s%s", prefix, title)))
			}
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:min(len(lines), innerHeight)], "\n")
	return style.Width(innerWidth).Height(innerHeight).Render(content)
}
