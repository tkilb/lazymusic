package panels

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/player"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// QueuePanel implements Panel for Panel 0 (Queue).
type QueuePanel struct {
	focused     bool
	width       int
	height      int
	cursor      int
	offset      int
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
	innerWidth := p.width - 2
	innerHeight := p.height - 2
	if innerWidth < 0 {
		innerWidth = 0
	}
	if innerHeight < 0 {
		innerHeight = 0
	}

	var lines []string
	filtered := p.filteredPlaylist()

	scrollMargin := 2
	if innerHeight > scrollMargin*2 {
		if p.cursor < p.offset+scrollMargin {
			p.offset = p.cursor - scrollMargin
		} else if p.cursor >= p.offset+innerHeight-scrollMargin {
			p.offset = p.cursor - innerHeight + 1 + scrollMargin
		}
	} else {
		if p.cursor < p.offset {
			p.offset = p.cursor
		} else if p.cursor >= p.offset+innerHeight {
			p.offset = p.cursor - innerHeight + 1
		}
	}

	if maxOffset := len(filtered) - innerHeight; maxOffset >= 0 && p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
	offset := p.offset

	if len(filtered) == 0 {
		if p.filterQuery != "" {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(" [No matching tracks found]"))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(" [Queue is empty - press 'a' or Enter on tracks to add]"))
		}
	} else {
		for i := offset; i < len(filtered) && len(lines) < innerHeight; i++ {
			item := filtered[i]
			title := item.Title
			if title == "" {
				title = filepath.Base(item.Filename)
			}

			playingIndicator := "  "
			if item.Current || item.Playing {
				playingIndicator = "▶ "
			}

			prefix := fmt.Sprintf("%s%2d. ", playingIndicator, i+1)
			availableWidth := innerWidth - len(prefix) - 1
			if availableWidth < 0 {
				availableWidth = 0
			}
			truncatedTitle := theme.TruncateLine(title, availableWidth)
			lineStr := fmt.Sprintf(" %s%-*s", prefix, availableWidth, truncatedTitle)

			if i == p.cursor && p.focused {
				lines = append(lines, theme.CursorRowStyle.Render(lineStr))
			} else if i == p.cursor && !p.focused {
				lines = append(lines, lipgloss.NewStyle().Bold(true).Render(lineStr))
			} else {
				lines = append(lines, lineStr)
			}
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}

	body := strings.Join(lines[:min(len(lines), innerHeight)], "\n")

	var footer string
	if len(p.playlist) > 0 {
		footer = fmt.Sprintf("%d/%d", p.cursor+1, len(p.playlist))
	}
	if p.filterQuery != "" {
		footer = "/" + p.filterQuery + " " + footer
	}

	return theme.Frame("[0] Queue", body, innerWidth, innerHeight, p.focused, footer)
}
