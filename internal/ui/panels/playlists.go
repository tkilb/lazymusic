package panels

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/player"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// PlaylistItem represents a playlist file (.m3u or .m3u8).
type PlaylistItemFile struct {
	Name string
	Path string
}

// PlaylistViewMode indicates whether Panel 1 is showing the list of playlists or inspecting a playlist's tracks.
type PlaylistViewMode int

const (
	PlaylistViewList PlaylistViewMode = iota
	PlaylistViewTracks
)

// PlaylistsPanel implements Panel for Panel 1 (Playlists).
type PlaylistsPanel struct {
	focused      bool
	width        int
	height       int
	cursor       int
	offset       int
	playlistsDir string
	items        []PlaylistItemFile
	searchQuery  string
	depWarning   string

	// Drill-down track view state
	viewMode       PlaylistViewMode
	activePlaylist PlaylistItemFile
	tracks         []M3UEntry
	trackCursor    int
	trackOffset    int
	trackSearch    string
}

// NewPlaylistsPanel creates a new PlaylistsPanel reading from playlistsDir.
func NewPlaylistsPanel(playlistsDir string) *PlaylistsPanel {
	p := &PlaylistsPanel{
		cursor:       0,
		playlistsDir: playlistsDir,
		viewMode:     PlaylistViewList,
	}
	p.Refresh()
	return p
}

// Refresh scans playlistsDir for .m3u / .m3u8 files.
func (p *PlaylistsPanel) Refresh() {
	var savedPlaylistName string
	if p.cursor >= 0 && p.cursor < len(p.items) {
		savedPlaylistName = p.items[p.cursor].Name
	}

	p.items = nil
	if p.playlistsDir == "" {
		return
	}

	entries, err := os.ReadDir(p.playlistsDir)
	if err != nil {
		return
	}

	var diskItems []PlaylistItemFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".m3u" || ext == ".m3u8" {
			diskItems = append(diskItems, PlaylistItemFile{
				Name: strings.TrimSuffix(name, ext),
				Path: filepath.Join(p.playlistsDir, name),
			})
		}
	}

	order := ReadPlaylistOrder(p.playlistsDir)
	if len(order) > 0 {
		itemMap := make(map[string]PlaylistItemFile, len(diskItems))
		for _, item := range diskItems {
			itemMap[item.Name] = item
		}

		var ordered []PlaylistItemFile
		for _, name := range order {
			if item, exists := itemMap[name]; exists {
				ordered = append(ordered, item)
				delete(itemMap, name)
			}
		}
		// Append any remaining/new playlists not yet in .order
		for _, item := range diskItems {
			if _, exists := itemMap[item.Name]; exists {
				ordered = append(ordered, item)
			}
		}
		p.items = ordered
	} else {
		p.items = diskItems
	}

	foundCursor := -1
	if savedPlaylistName != "" {
		for i, item := range p.items {
			if item.Name == savedPlaylistName {
				foundCursor = i
				break
			}
		}
	}
	if foundCursor >= 0 {
		p.cursor = foundCursor
	} else {
		if p.cursor >= len(p.items) {
			p.cursor = len(p.items) - 1
		}
		if p.cursor < 0 {
			p.cursor = 0
		}
	}

	// If currently inspecting tracks in an expanded playlist, refresh its tracks from disk
	if p.viewMode == PlaylistViewTracks && p.activePlaylist.Path != "" {
		savedTrackCursor := p.trackCursor
		var savedTrackPath string
		if p.trackCursor >= 0 && p.trackCursor < len(p.tracks) {
			savedTrackPath = p.tracks[p.trackCursor].Path
		}

		// Verify active playlist still exists on disk
		var stillExists bool
		for _, item := range p.items {
			if item.Path == p.activePlaylist.Path {
				stillExists = true
				p.activePlaylist = item
				break
			}
		}

		if !stillExists {
			p.viewMode = PlaylistViewList
			p.tracks = nil
			p.trackCursor = 0
			p.trackOffset = 0
		} else {
			tracks, _ := ReadM3U(p.activePlaylist.Path)
			p.tracks = tracks

			foundTrackIdx := -1
			if savedTrackPath != "" {
				for i, tr := range p.tracks {
					if tr.Path == savedTrackPath {
						foundTrackIdx = i
						break
					}
				}
			}

			if foundTrackIdx >= 0 {
				p.trackCursor = foundTrackIdx
			} else {
				p.trackCursor = savedTrackCursor
				if p.trackCursor >= len(p.tracks) {
					p.trackCursor = len(p.tracks) - 1
				}
				if p.trackCursor < 0 {
					p.trackCursor = 0
				}
			}
		}
	}

	p.depWarning = player.CheckMediaDependencies()
}

// Items returns the current list of playlist items.
func (p *PlaylistsPanel) Items() []PlaylistItemFile {
	return p.items
}

// Len returns the number of playlists.
func (p *PlaylistsPanel) Len() int {
	return len(p.items)
}

// Cursor returns the currently focused playlist index.
func (p *PlaylistsPanel) Cursor() int {
	return p.cursor
}

// SelectByName sets the cursor to the playlist with the matching name.
func (p *PlaylistsPanel) SelectByName(name string) {
	for i, item := range p.items {
		if strings.EqualFold(item.Name, name) {
			p.cursor = i
			return
		}
	}
}

// MoveUp moves the selected playlist or track up one position.
func (p *PlaylistsPanel) MoveUp() {
	if p.viewMode == PlaylistViewTracks {
		p.MoveTrackUp()
		return
	}
	if p.cursor <= 0 || p.cursor >= len(p.items) {
		return
	}
	p.items[p.cursor], p.items[p.cursor-1] = p.items[p.cursor-1], p.items[p.cursor]
	p.cursor--
	p.saveOrder()
}

// MoveDown moves the selected playlist or track down one position.
func (p *PlaylistsPanel) MoveDown() {
	if p.viewMode == PlaylistViewTracks {
		p.MoveTrackDown()
		return
	}
	if p.cursor < 0 || p.cursor >= len(p.items)-1 {
		return
	}
	p.items[p.cursor], p.items[p.cursor+1] = p.items[p.cursor+1], p.items[p.cursor]
	p.cursor++
	p.saveOrder()
}

// MoveTrackUp moves the selected track up one position in the active playlist and saves to disk.
func (p *PlaylistsPanel) MoveTrackUp() {
	if p.viewMode != PlaylistViewTracks || p.trackCursor <= 0 || p.trackCursor >= len(p.tracks) {
		return
	}
	p.tracks[p.trackCursor], p.tracks[p.trackCursor-1] = p.tracks[p.trackCursor-1], p.tracks[p.trackCursor]
	p.trackCursor--
	_ = WriteM3U(p.activePlaylist.Path, p.tracks)
}

// MoveTrackDown moves the selected track down one position in the active playlist and saves to disk.
func (p *PlaylistsPanel) MoveTrackDown() {
	if p.viewMode != PlaylistViewTracks || p.trackCursor < 0 || p.trackCursor >= len(p.tracks)-1 {
		return
	}
	p.tracks[p.trackCursor], p.tracks[p.trackCursor+1] = p.tracks[p.trackCursor+1], p.tracks[p.trackCursor]
	p.trackCursor++
	_ = WriteM3U(p.activePlaylist.Path, p.tracks)
}

// RemoveSelectedTrack removes the selected track from the active playlist and saves to disk.
func (p *PlaylistsPanel) RemoveSelectedTrack() (M3UEntry, bool) {
	if p.viewMode != PlaylistViewTracks || p.trackCursor < 0 || p.trackCursor >= len(p.tracks) {
		return M3UEntry{}, false
	}
	removed := p.tracks[p.trackCursor]
	p.tracks = append(p.tracks[:p.trackCursor], p.tracks[p.trackCursor+1:]...)
	if p.trackCursor >= len(p.tracks) && p.trackCursor > 0 {
		p.trackCursor = len(p.tracks) - 1
	}
	_ = WriteM3U(p.activePlaylist.Path, p.tracks)
	return removed, true
}

// ViewMode returns whether the panel is in playlist list mode or inspecting tracks.
func (p *PlaylistsPanel) ViewMode() PlaylistViewMode {
	return p.viewMode
}

// ActivePlaylist returns the playlist currently being inspected.
func (p *PlaylistsPanel) ActivePlaylist() (PlaylistItemFile, bool) {
	if p.viewMode == PlaylistViewTracks {
		return p.activePlaylist, true
	}
	return PlaylistItemFile{}, false
}

// OpenTracks steps into the track inspector for the selected playlist.
func (p *PlaylistsPanel) OpenTracks() bool {
	if p.viewMode == PlaylistViewTracks {
		return false
	}
	if len(p.items) == 0 || p.cursor < 0 || p.cursor >= len(p.items) {
		return false
	}
	p.activePlaylist = p.items[p.cursor]
	tracks, _ := ReadM3U(p.activePlaylist.Path)
	p.tracks = tracks
	p.trackCursor = 0
	p.trackOffset = 0
	p.trackSearch = ""
	p.viewMode = PlaylistViewTracks
	return true
}

// CloseTracks returns from the track inspector to the playlist list.
func (p *PlaylistsPanel) CloseTracks() bool {
	if p.viewMode != PlaylistViewTracks {
		return false
	}
	p.viewMode = PlaylistViewList
	p.tracks = nil
	p.trackSearch = ""
	return true
}

// SelectedTrack returns the currently selected track within the inspected playlist.
func (p *PlaylistsPanel) SelectedTrack() (int, M3UEntry, bool) {
	if p.viewMode == PlaylistViewTracks && p.trackCursor >= 0 && p.trackCursor < len(p.tracks) {
		return p.trackCursor, p.tracks[p.trackCursor], true
	}
	return -1, M3UEntry{}, false
}

func (p *PlaylistsPanel) saveOrder() {
	names := make([]string, len(p.items))
	for i, item := range p.items {
		names[i] = item.Name
	}
	_ = SavePlaylistOrder(p.playlistsDir, names)
}

func (p *PlaylistsPanel) ID() PanelID {
	return PanelPlaylists
}

func (p *PlaylistsPanel) Title() string {
	if p.viewMode == PlaylistViewTracks {
		return fmt.Sprintf("1: Playlists > %s", p.activePlaylist.Name)
	}
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
	if p.viewMode == PlaylistViewTracks {
		if p.trackCursor > 0 {
			p.trackCursor--
		}
		return
	}
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *PlaylistsPanel) CursorDown() {
	if p.viewMode == PlaylistViewTracks {
		if p.trackCursor < len(p.tracks)-1 {
			p.trackCursor++
		}
		return
	}
	if p.cursor < len(p.items)-1 {
		p.cursor++
	}
}

func (p *PlaylistsPanel) PageUp() {
	if p.viewMode == PlaylistViewTracks {
		p.trackCursor = pageMove(p.trackCursor, len(p.tracks), p.height, -1)
		return
	}
	p.cursor = pageMove(p.cursor, len(p.items), p.height, -1)
}

func (p *PlaylistsPanel) PageDown() {
	if p.viewMode == PlaylistViewTracks {
		p.trackCursor = pageMove(p.trackCursor, len(p.tracks), p.height, 1)
		return
	}
	p.cursor = pageMove(p.cursor, len(p.items), p.height, 1)
}

// Search sets query as the active search and jumps cursor to nearest match.
func (p *PlaylistsPanel) Search(query string) bool {
	if p.viewMode == PlaylistViewTracks {
		p.trackSearch = query
		i, ok := p.findTrackMatch(p.trackCursor, 1, true)
		if ok {
			p.trackCursor = i
		}
		return ok
	}
	p.searchQuery = query
	i, ok := p.findMatch(p.cursor, 1, true)
	if ok {
		p.cursor = i
	}
	return ok
}

// ClearSearch discards the active search query.
func (p *PlaylistsPanel) ClearSearch() {
	if p.viewMode == PlaylistViewTracks {
		p.trackSearch = ""
		return
	}
	p.searchQuery = ""
}

// HasActiveSearch reports whether a search query is active.
func (p *PlaylistsPanel) HasActiveSearch() bool {
	if p.viewMode == PlaylistViewTracks {
		return p.trackSearch != ""
	}
	return p.searchQuery != ""
}

// NextMatch moves cursor to next match forward.
func (p *PlaylistsPanel) NextMatch() bool {
	if p.viewMode == PlaylistViewTracks {
		if p.trackSearch == "" {
			return false
		}
		i, ok := p.findTrackMatch(p.trackCursor, 1, false)
		if ok {
			p.trackCursor = i
		}
		return ok
	}
	if p.searchQuery == "" {
		return false
	}
	i, ok := p.findMatch(p.cursor, 1, false)
	if ok {
		p.cursor = i
	}
	return ok
}

// PrevMatch moves cursor to previous match backward.
func (p *PlaylistsPanel) PrevMatch() bool {
	if p.viewMode == PlaylistViewTracks {
		if p.trackSearch == "" {
			return false
		}
		i, ok := p.findTrackMatch(p.trackCursor, -1, false)
		if ok {
			p.trackCursor = i
		}
		return ok
	}
	if p.searchQuery == "" {
		return false
	}
	i, ok := p.findMatch(p.cursor, -1, false)
	if ok {
		p.cursor = i
	}
	return ok
}

func (p *PlaylistsPanel) findMatch(start, direction int, includeStart bool) (int, bool) {
	n := len(p.items)
	if n == 0 || p.searchQuery == "" {
		return 0, false
	}
	i := start
	if !includeStart {
		i += direction
	}
	i = ((i % n) + n) % n
	lowerQ := strings.ToLower(p.searchQuery)
	for step := 0; step < n; step++ {
		if strings.Contains(strings.ToLower(p.items[i].Name), lowerQ) {
			return i, true
		}
		i = ((i+direction)%n + n) % n
	}
	return 0, false
}

func (p *PlaylistsPanel) findTrackMatch(start, direction int, includeStart bool) (int, bool) {
	n := len(p.tracks)
	if n == 0 || p.trackSearch == "" {
		return 0, false
	}
	i := start
	if !includeStart {
		i += direction
	}
	i = ((i % n) + n) % n
	lowerQ := strings.ToLower(p.trackSearch)
	for step := 0; step < n; step++ {
		tr := p.tracks[i]
		if strings.Contains(strings.ToLower(tr.Title), lowerQ) ||
			strings.Contains(strings.ToLower(tr.Artist), lowerQ) ||
			strings.Contains(strings.ToLower(filepath.Base(tr.Path)), lowerQ) {
			return i, true
		}
		i = ((i+direction)%n + n) % n
	}
	return 0, false
}

// SelectedPlaylist returns the currently selected playlist file.
func (p *PlaylistsPanel) SelectedPlaylist() (PlaylistItemFile, bool) {
	if p.viewMode == PlaylistViewTracks {
		return p.activePlaylist, true
	}
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

	if p.viewMode == PlaylistViewTracks {
		return p.viewTracks(innerWidth, innerHeight)
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

			rowBaseStyle := lipgloss.NewStyle()
			if i == p.cursor && p.focused {
				rowBaseStyle = theme.CursorRowStyle
			} else if i == p.cursor && !p.focused {
				rowBaseStyle = lipgloss.NewStyle().Bold(true)
			}

			renderedName := theme.RenderWithMatches(truncated, p.searchQuery, rowBaseStyle, i == p.cursor)
			pad := (innerWidth - 1) - len([]rune(truncated))
			if pad < 0 {
				pad = 0
			}
			lineStr := fmt.Sprintf(" %s%s", renderedName, rowBaseStyle.Render(strings.Repeat(" ", pad)))
			lines = append(lines, lineStr)
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
	if p.searchQuery != "" {
		footer = "/" + p.searchQuery + " " + footer
	}

	warning := ""
	if p.depWarning != "" {
		p.depWarning = player.CheckMediaDependencies()
		if p.depWarning != "" {
			warning = lipgloss.NewStyle().Foreground(theme.PaletteRed).Bold(true).Render(p.depWarning)
		}
	}

	return theme.FrameWithTopRight("[1] Playlists", warning, body, innerWidth, innerHeight, p.focused, footer)
}

func (p *PlaylistsPanel) viewTracks(innerWidth, innerHeight int) string {
	var lines []string

	scrollMargin := 2
	if innerHeight > scrollMargin*2 {
		if p.trackCursor < p.trackOffset+scrollMargin {
			p.trackOffset = p.trackCursor - scrollMargin
		} else if p.trackCursor >= p.trackOffset+innerHeight-scrollMargin {
			p.trackOffset = p.trackCursor - innerHeight + 1 + scrollMargin
		}
	} else {
		if p.trackCursor < p.trackOffset {
			p.trackOffset = p.trackCursor
		} else if p.trackCursor >= p.trackOffset+innerHeight {
			p.trackOffset = p.trackCursor - innerHeight + 1
		}
	}

	if maxOffset := len(p.tracks) - innerHeight; maxOffset >= 0 && p.trackOffset > maxOffset {
		p.trackOffset = maxOffset
	}
	if p.trackOffset < 0 {
		p.trackOffset = 0
	}
	offset := p.trackOffset

	if len(p.tracks) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(" [Empty playlist - add tracks with 'p']"))
	} else {
		for i := offset; i < len(p.tracks) && len(lines) < innerHeight; i++ {
			item := p.tracks[i]
			title := item.Title
			if title == "" {
				title = filepath.Base(item.Path)
			}
			if item.Artist != "" {
				title = fmt.Sprintf("%s - %s", item.Artist, title)
			}

			prefix := fmt.Sprintf("%2d. ", i+1)
			availableWidth := innerWidth - len(prefix) - 1
			if availableWidth < 0 {
				availableWidth = 0
			}
			truncated := theme.TruncateLine(title, availableWidth)

			rowBaseStyle := lipgloss.NewStyle()
			if i == p.trackCursor && p.focused {
				rowBaseStyle = theme.CursorRowStyle
			} else if i == p.trackCursor && !p.focused {
				rowBaseStyle = lipgloss.NewStyle().Bold(true)
			}

			renderedName := theme.RenderWithMatches(truncated, p.trackSearch, rowBaseStyle, i == p.trackCursor)
			pad := availableWidth - len([]rune(truncated))
			if pad < 0 {
				pad = 0
			}
			lineStr := fmt.Sprintf(" %s%s%s", prefix, renderedName, rowBaseStyle.Render(strings.Repeat(" ", pad)))
			lines = append(lines, lineStr)
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}

	body := strings.Join(lines[:min(len(lines), innerHeight)], "\n")

	var footer string
	if len(p.tracks) > 0 {
		footer = fmt.Sprintf("%d/%d", p.trackCursor+1, len(p.tracks))
	}
	if p.trackSearch != "" {
		footer = "/" + p.trackSearch + " " + footer
	}

	title := fmt.Sprintf("[1] Playlists > %s", p.activePlaylist.Name)
	return theme.FrameWithTopRight(title, "", body, innerWidth, innerHeight, p.focused, footer)
}
