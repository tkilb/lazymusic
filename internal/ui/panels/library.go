package panels

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/config"
	"github.com/tylerkilburn/lazymusic/internal/library"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// LibraryPanel implements Panel for Panel 1 (Library - Directories).
type LibraryPanel struct {
	focused     bool
	width       int
	height      int
	cursor      int
	offset      int
	cfg         *config.Config
	storage     *library.Storage
	dirs        []string
	filterQuery string
}

// NewLibraryPanel creates a new LibraryPanel instance displaying configured music directories.
func NewLibraryPanel(cfg *config.Config, storage *library.Storage) *LibraryPanel {
	p := &LibraryPanel{
		cursor:  0,
		cfg:     cfg,
		storage: storage,
	}
	p.Refresh()
	return p
}

// Refresh reloads directories from config.
func (p *LibraryPanel) Refresh() {
	if p.cfg == nil {
		return
	}
	p.dirs = nil
	for _, d := range p.cfg.MusicDirs {
		p.dirs = append(p.dirs, d)
	}

	if p.cursor >= len(p.dirs) {
		p.cursor = len(p.dirs) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *LibraryPanel) ID() PanelID {
	return PanelLibrary
}

func (p *LibraryPanel) Title() string {
	return "3: Dirs"
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

func (p *LibraryPanel) NextTab() {}

func (p *LibraryPanel) PrevTab() {}


func (p *LibraryPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *LibraryPanel) CursorDown() {
	if p.cursor < len(p.dirs)-1 {
		p.cursor++
	}
}

func (p *LibraryPanel) PageUp()   { p.cursor = pageMove(p.cursor, len(p.dirs), p.height, -1) }
func (p *LibraryPanel) PageDown() { p.cursor = pageMove(p.cursor, len(p.dirs), p.height, 1) }

// MoveUp moves the selected directory up one position and persists the order to config.
func (p *LibraryPanel) MoveUp() {
	if p.cursor <= 0 || p.cursor >= len(p.dirs) {
		return
	}
	p.dirs[p.cursor], p.dirs[p.cursor-1] = p.dirs[p.cursor-1], p.dirs[p.cursor]
	if p.cfg != nil {
		p.cfg.MoveMusicDir(p.cursor, p.cursor-1)
	}
	p.cursor--
}

// MoveDown moves the selected directory down one position and persists the order to config.
func (p *LibraryPanel) MoveDown() {
	if p.cursor < 0 || p.cursor >= len(p.dirs)-1 {
		return
	}
	p.dirs[p.cursor], p.dirs[p.cursor+1] = p.dirs[p.cursor+1], p.dirs[p.cursor]
	if p.cfg != nil {
		p.cfg.MoveMusicDir(p.cursor, p.cursor+1)
	}
	p.cursor++
}


// Search sets query as the active search and jumps cursor to nearest match.
func (p *LibraryPanel) Search(query string) bool {
	p.filterQuery = query
	i, ok := p.findMatch(p.cursor, 1, true)
	if ok {
		p.cursor = i
	}
	return ok
}

// ClearSearch discards the active search query.
func (p *LibraryPanel) ClearSearch() {
	p.filterQuery = ""
}

// HasActiveSearch reports whether a search query is active.
func (p *LibraryPanel) HasActiveSearch() bool {
	return p.filterQuery != ""
}

// NextMatch moves cursor to next match forward.
func (p *LibraryPanel) NextMatch() bool {
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
func (p *LibraryPanel) PrevMatch() bool {
	if p.filterQuery == "" {
		return false
	}
	i, ok := p.findMatch(p.cursor, -1, false)
	if ok {
		p.cursor = i
	}
	return ok
}

func (p *LibraryPanel) findMatch(start, direction int, includeStart bool) (int, bool) {
	n := len(p.dirs)
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
		if strings.Contains(strings.ToLower(p.dirs[i]), lowerQ) {
			return i, true
		}
		i = ((i+direction)%n + n) % n
	}
	return 0, false
}

// SelectedDir returns the currently highlighted directory path.
func (p *LibraryPanel) SelectedDir() (string, bool) {
	if p.cursor >= 0 && p.cursor < len(p.dirs) {
		return p.dirs[p.cursor], true
	}
	return "", false
}

func (p *LibraryPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *LibraryPanel) View() string {
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

	if maxOffset := len(p.dirs) - innerHeight; maxOffset >= 0 && p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
	offset := p.offset

	home, _ := os.UserHomeDir()

	if len(p.dirs) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(" [No music directories configured - press 'a' to add]"))
	} else {
		for i := offset; i < len(p.dirs) && len(lines) < innerHeight; i++ {
			item := p.dirs[i]
			displayPath := item
			if strings.HasPrefix(displayPath, home) {
				displayPath = "~" + strings.TrimPrefix(displayPath, home)
			}
			truncated := theme.TruncateLine(displayPath, innerWidth-1)

			rowBaseStyle := lipgloss.NewStyle()
			if i == p.cursor && p.focused {
				rowBaseStyle = theme.CursorRowStyle
			} else if i == p.cursor && !p.focused {
				rowBaseStyle = lipgloss.NewStyle().Bold(true)
			}

			renderedPath := theme.RenderWithMatches(truncated, p.filterQuery, rowBaseStyle, i == p.cursor)
			pad := (innerWidth - 1) - len([]rune(truncated))
			if pad < 0 {
				pad = 0
			}
			lineStr := fmt.Sprintf(" %s%s", renderedPath, rowBaseStyle.Render(strings.Repeat(" ", pad)))
			lines = append(lines, lineStr)
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}

	body := strings.Join(lines[:min(len(lines), innerHeight)], "\n")

	var footer string
	if len(p.dirs) > 0 {
		footer = fmt.Sprintf("%d/%d", p.cursor+1, len(p.dirs))
	}
	if p.filterQuery != "" {
		footer = "/" + p.filterQuery + " " + footer
	}

	return theme.Frame("[3] Dirs", body, innerWidth, innerHeight, p.focused, footer)
}
