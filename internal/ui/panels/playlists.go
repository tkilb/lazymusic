package panels

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// PlaylistItem represents a playlist file (.m3u or .m3u8).
type PlaylistItemFile struct {
	Name string
	Path string
}

// PlaylistsPanel implements Panel for Panel 2 (Playlists).
type PlaylistsPanel struct {
	focused      bool
	width        int
	height       int
	cursor       int
	offset       int
	playlistsDir string
	items        []PlaylistItemFile
}

// NewPlaylistsPanel creates a new PlaylistsPanel reading from playlistsDir.
func NewPlaylistsPanel(playlistsDir string) *PlaylistsPanel {
	p := &PlaylistsPanel{
		cursor:       0,
		playlistsDir: playlistsDir,
	}
	p.Refresh()
	return p
}

// Refresh scans playlistsDir for .m3u / .m3u8 files.
func (p *PlaylistsPanel) Refresh() {
	p.items = nil
	if p.playlistsDir == "" {
		return
	}

	entries, err := os.ReadDir(p.playlistsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".m3u" || ext == ".m3u8" {
			p.items = append(p.items, PlaylistItemFile{
				Name: strings.TrimSuffix(name, ext),
				Path: filepath.Join(p.playlistsDir, name),
			})
		}
	}
	if p.cursor >= len(p.items) {
		p.cursor = len(p.items) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *PlaylistsPanel) ID() PanelID {
	return PanelPlaylists
}

func (p *PlaylistsPanel) Title() string {
	return "1: Playlists"
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

// SelectedPlaylist returns the currently selected playlist file.
func (p *PlaylistsPanel) SelectedPlaylist() (PlaylistItemFile, bool) {
	if p.cursor >= 0 && p.cursor < len(p.items) {
		return p.items[p.cursor], true
	}
	return PlaylistItemFile{}, false
}

// ParsePlaylistFile reads an M3U file and returns the list of file paths.
func ParsePlaylistFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tracks []string
	scanner := bufio.NewScanner(file)
	dir := filepath.Dir(filePath)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(dir, line)
		}
		tracks = append(tracks, line)
	}

	return tracks, scanner.Err()
}

func (p *PlaylistsPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *PlaylistsPanel) View() string {
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

	if maxOffset := len(p.items) - innerHeight; maxOffset >= 0 && p.offset > maxOffset {
		p.offset = maxOffset
	}
	if p.offset < 0 {
		p.offset = 0
	}
	offset := p.offset

	if len(p.items) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(" [No playlists in ~/.config/lazymusic/playlists]"))
	} else {
		for i := offset; i < len(p.items) && len(lines) < innerHeight; i++ {
			item := p.items[i]
			truncated := theme.TruncateLine(item.Name, innerWidth-1)
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
	if len(p.items) > 0 {
		footer = fmt.Sprintf("%d/%d", p.cursor+1, len(p.items))
	}

	return theme.Frame("[1] Playlists", body, innerWidth, innerHeight, p.focused, footer)
}
