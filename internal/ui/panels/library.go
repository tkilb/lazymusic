package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tylerkilburn/lazymusic/internal/library"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// LibraryPanel implements Panel for Panel 1 (Library).
type LibraryPanel struct {
	focused      bool
	width        int
	height       int
	tabs         []string
	curTab       int
	cursor       int
	storage      *library.Storage
	items        map[int][]string
	tracks       []library.Track
	filterQuery  string
	filterActive bool
}

// NewLibraryPanel creates a new LibraryPanel instance connected to SQLite storage.
func NewLibraryPanel(storage *library.Storage) *LibraryPanel {
	p := &LibraryPanel{
		tabs:    []string{"Dirs", "Artists", "Albums", "Tracks"},
		curTab:  0,
		cursor:  0,
		storage: storage,
		items:   make(map[int][]string),
	}
	p.Refresh()
	return p
}

// Refresh reloads artists, albums, and tracks from SQLite.
func (p *LibraryPanel) Refresh() {
	if p.storage == nil {
		return
	}

	artists, _ := p.storage.GetArtists()
	albums, _ := p.storage.GetAlbums()
	tracks, _ := p.storage.GetTracks()

	p.tracks = tracks

	var trackTitles []string
	for _, t := range tracks {
		trackTitles = append(trackTitles, fmt.Sprintf("%s - %s", t.Artist, t.Title))
	}

	p.items[0] = []string{"~/Music"}
	p.items[1] = artists
	p.items[2] = albums
	p.items[3] = trackTitles
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
	p.filterQuery = ""
	p.filterActive = false
}

func (p *LibraryPanel) PrevTab() {
	p.curTab = (p.curTab - 1 + len(p.tabs)) % len(p.tabs)
	p.cursor = 0
	p.filterQuery = ""
	p.filterActive = false
}

func (p *LibraryPanel) filteredItems() []string {
	all := p.items[p.curTab]
	if p.filterQuery == "" {
		return all
	}
	query := strings.ToLower(p.filterQuery)
	var filtered []string
	for _, item := range all {
		if strings.Contains(strings.ToLower(item), query) {
			filtered = append(filtered, item)
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
	list := p.filteredItems()
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

// SelectedTrack returns the currently highlighted track, if in the Tracks tab.
func (p *LibraryPanel) SelectedTrack() (library.Track, bool) {
	filtered := p.filteredItems()
	if p.curTab == 3 && p.cursor >= 0 && p.cursor < len(filtered) {
		selectedTitle := filtered[p.cursor]
		for _, t := range p.tracks {
			if fmt.Sprintf("%s - %s", t.Artist, t.Title) == selectedTitle {
				return t, true
			}
		}
	}
	return library.Track{}, false
}

func (p *LibraryPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *LibraryPanel) View() string {
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
	if p.filterQuery != "" {
		header += fmt.Sprintf(" %s", theme.KeybindDescStyle.Render("/"+p.filterQuery))
	}

	var lines []string
	lines = append(lines, header)

	currentItems := p.filteredItems()
	visibleSlots := innerHeight - 1
	if visibleSlots < 0 {
		visibleSlots = 0
	}

	// Handle scroll offset
	offset := 0
	if p.cursor >= visibleSlots {
		offset = p.cursor - visibleSlots + 1
	}

	for i := offset; i < len(currentItems) && len(lines) < innerHeight; i++ {
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
