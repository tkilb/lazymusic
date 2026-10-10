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
	currentPos   int
	filterQuery  string
	loopFile     bool
	loopPlaylist bool
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

func (p *QueuePanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}


func (p *QueuePanel) CursorDown() {
	if p.cursor < len(p.playlist)-1 {
		p.cursor++
	}
}

func (p *QueuePanel) PageUp()   { p.cursor = pageMove(p.cursor, len(p.playlist), p.height, -1) }
func (p *QueuePanel) PageDown() { p.cursor = pageMove(p.cursor, len(p.playlist), p.height, 1) }

// Cursor returns current cursor position.
func (p *QueuePanel) Cursor() int {
	return p.cursor
}

// Search sets query as the active search and jumps cursor to nearest match.
func (p *QueuePanel) Search(query string) bool {
	p.filterQuery = query
	i, ok := p.findMatch(p.cursor, 1, true)
	if ok {
		p.cursor = i
	}
	return ok
}

// ClearSearch discards the active search query.
func (p *QueuePanel) ClearSearch() {
	p.filterQuery = ""
}

// HasActiveSearch reports whether a search query is active.
func (p *QueuePanel) HasActiveSearch() bool {
	return p.filterQuery != ""
}

// NextMatch moves cursor to next match forward.
func (p *QueuePanel) NextMatch() bool {
	if p.filterQuery == "" {
		return false
	}
	i, ok := p.findMatch(p.cursor, 1, false)
	if ok {
		p.cursor = i
	}
	return ok
}

// PrevMatch moves cursor to previous match backward.
func (p *QueuePanel) PrevMatch() bool {
	if p.filterQuery == "" {
		return false
	}
	i, ok := p.findMatch(p.cursor, -1, false)
	if ok {
		p.cursor = i
	}
	return ok
}

func (p *QueuePanel) findMatch(start, direction int, includeStart bool) (int, bool) {
	n := len(p.playlist)
	if n == 0 || p.filterQuery == "" {
		return 0, false
	}
	i := start
	if !includeStart {
		i += direction
	}
	i = ((i % n) + n) % n
	lowerQ := strings.ToLower(p.filterQuery)
	for step := 0; step < n; step++ {
		title := p.playlist[i].Title
		if title == "" {
			title = filepath.Base(p.playlist[i].Filename)
		}
		if strings.Contains(strings.ToLower(title), lowerQ) {
			return i, true
		}
		i = ((i+direction)%n + n) % n
	}
	return 0, false
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

// SetLoop updates the loop states for song and queue.
func (p *QueuePanel) SetLoop(loopFile, loopPlaylist bool) {
	p.loopFile = loopFile
	p.loopPlaylist = loopPlaylist
}

// Playlist returns a copy of current playlist items.
func (p *QueuePanel) Playlist() []player.PlaylistItem {
	res := make([]player.PlaylistItem, len(p.playlist))
	copy(res, p.playlist)
	return res
}

// Len returns the number of tracks in the playlist.
func (p *QueuePanel) Len() int {
	return len(p.playlist)
}

// Move moves an item in the queue playlist from one index to another,
// keeping local state and currentPos in sync optimistically.
func (p *QueuePanel) Move(from, to int) {
	if from < 0 || from >= len(p.playlist) || to < 0 || to >= len(p.playlist) || from == to {
		return
	}
	item := p.playlist[from]
	p.playlist = append(p.playlist[:from], p.playlist[from+1:]...)
	p.playlist = append(p.playlist[:to], append([]player.PlaylistItem{item}, p.playlist[to:]...)...)

	if p.currentPos == from {
		p.currentPos = to
	} else if from < to && p.currentPos > from && p.currentPos <= to {
		p.currentPos--
	} else if from > to && p.currentPos >= to && p.currentPos < from {
		p.currentPos++
	}
}

// CurrentPos returns current playlist playing index.
func (p *QueuePanel) CurrentPos() int {
	return p.currentPos
}

// SelectedTrack returns the selected item index and track.
func (p *QueuePanel) SelectedTrack() (int, player.PlaylistItem, bool) {
	if p.cursor >= 0 && p.cursor < len(p.playlist) {
		return p.cursor, p.playlist[p.cursor], true
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

	if maxOffset := len(p.playlist) - innerHeight; maxOffset >= 0 && p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
	offset := p.offset

	if len(p.playlist) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(" [Queue is empty - press 'a' on files to add]"))
	} else {
		for i := offset; i < len(p.playlist) && len(lines) < innerHeight; i++ {
			item := p.playlist[i]
			title := item.Title
			if title == "" {
				title = filepath.Base(item.Filename)
			}

			playingIndicator := "  "
			if item.Current || item.Playing {
				playingIndicator = "▶ "
			}

			displayTitle := title
			if (item.Current || item.Playing) && p.loopFile {
				displayTitle += " [loop]"
			}

			prefix := fmt.Sprintf("%s%2d. ", playingIndicator, i+1)
			availableWidth := innerWidth - len(prefix) - 1
			if availableWidth < 0 {
				availableWidth = 0
			}
			truncatedTitle := theme.TruncateLine(displayTitle, availableWidth)

			rowBaseStyle := lipgloss.NewStyle()
			if i == p.cursor && p.focused {
				rowBaseStyle = theme.CursorRowStyle
			} else if i == p.cursor && !p.focused {
				rowBaseStyle = lipgloss.NewStyle().Bold(true)
			}

			renderedTitle := theme.RenderWithMatches(truncatedTitle, p.filterQuery, rowBaseStyle, i == p.cursor)
			pad := availableWidth - len([]rune(truncatedTitle))
			if pad < 0 {
				pad = 0
			}
			lineStr := fmt.Sprintf(" %s%s%s", prefix, renderedTitle, rowBaseStyle.Render(strings.Repeat(" ", pad)))
			lines = append(lines, lineStr)
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

	var loopIndicator string
	if p.loopFile {
		loopIndicator = lipgloss.NewStyle().Foreground(theme.PaletteCyan).Bold(true).Render("[Loop: Song]")
	} else if p.loopPlaylist {
		loopIndicator = lipgloss.NewStyle().Foreground(theme.PaletteCyan).Bold(true).Render("[Loop: Queue]")
	}

	return theme.FrameWithTopRight("[0] Queue", loopIndicator, body, innerWidth, innerHeight, p.focused, footer)
}
