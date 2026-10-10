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

func (p *LibraryPanel) filteredDirs() []string {
	if p.filterQuery == "" {
		return p.dirs
	}
	query := strings.ToLower(p.filterQuery)
	var filtered []string
	for _, d := range p.dirs {
		if strings.Contains(strings.ToLower(d), query) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func (p *LibraryPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *LibraryPanel) CursorDown() {
	list := p.filteredDirs()
	if p.cursor < len(list)-1 {
		p.cursor++
	}
}

// SetFilterQuery sets the fuzzy filter string.
func (p *LibraryPanel) SetFilterQuery(query string) {
	p.filterQuery = query
	p.cursor = 0
}

// FilterQuery returns the current filter query.
func (p *LibraryPanel) FilterQuery() string {
	return p.filterQuery
}

// SelectedDir returns the currently highlighted directory path.
func (p *LibraryPanel) SelectedDir() (string, bool) {
	filtered := p.filteredDirs()
	if p.cursor >= 0 && p.cursor < len(filtered) {
		return filtered[p.cursor], true
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

	currentItems := p.filteredDirs()
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

	if maxOffset := len(currentItems) - innerHeight; maxOffset >= 0 && p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
	offset := p.offset

	home, _ := os.UserHomeDir()

	if len(currentItems) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(" [No music directories configured - press 'a' to add]"))
	} else {
		for i := offset; i < len(currentItems) && len(lines) < innerHeight; i++ {
			item := currentItems[i]
			displayPath := item
			if strings.HasPrefix(displayPath, home) {
				displayPath = "~" + strings.TrimPrefix(displayPath, home)
			}
			truncated := theme.TruncateLine(displayPath, innerWidth-1)
			lineStr := fmt.Sprintf(" %-*s", innerWidth-1, truncated)

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
	if len(currentItems) > 0 {
		footer = fmt.Sprintf("%d/%d", p.cursor+1, len(currentItems))
	}
	if p.filterQuery != "" {
		footer = "/" + p.filterQuery + " " + footer
	}

	return theme.Frame("[3] Dirs", body, innerWidth, innerHeight, p.focused, footer)
}
