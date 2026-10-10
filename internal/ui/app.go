package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/config"
	"github.com/tylerkilburn/lazymusic/internal/library"
	"github.com/tylerkilburn/lazymusic/internal/player"
	"github.com/tylerkilburn/lazymusic/internal/ui/helppopup"
	"github.com/tylerkilburn/lazymusic/internal/ui/panels"
	"github.com/tylerkilburn/lazymusic/internal/ui/statusbar"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

type ModalMode int

const (
	ModalNone ModalMode = iota
	ModalHelp
	ModalSearch
	ModalTagEdit
)

// Model represents the main Bubble Tea model for lazymusic.
type Model struct {
	cfg           *config.Config
	player        *player.Client
	storage       *library.Storage
	width         int
	height        int
	activePanel   panels.PanelID
	leftPanels    []panels.Panel
	queuePanel    panels.Panel
	statusPanel   *panels.StatusPanel
	lastLeftFocus panels.PanelID

	// Modal states
	modalMode    ModalMode
	helpScroll   int
	searchBuffer string
	tagTrackPath string
	tagTitle     string
	tagArtist    string
	tagAlbum     string
	tagFieldIdx  int // 0: Title, 1: Artist, 2: Album
}

// PlaybackMsg wraps player state updates for Bubble Tea.
type PlaybackMsg player.PlaybackState

// LibraryScanMsg signals that a library background scan finished.
type LibraryScanMsg struct {
	TrackCount int
	Err        error
}

// NewModel initializes the TUI model.
func NewModel(cfg *config.Config, pClient *player.Client, storage *library.Storage) *Model {
	m := &Model{
		cfg:           cfg,
		player:        pClient,
		storage:       storage,
		activePanel:   panels.PanelLibrary,
		lastLeftFocus: panels.PanelLibrary,
		leftPanels: []panels.Panel{
			panels.NewLibraryPanel(storage),
			panels.NewPlaylistsPanel(cfg.PlaylistsDir),
			panels.NewFileBrowserPanel(""),
		},
		queuePanel:  panels.NewQueuePanel(),
		statusPanel: panels.NewStatusPanel(),
		modalMode:   ModalNone,
	}
	m.updateFocus()
	m.updateStatusBar()
	return m
}

var globalStatusBindings = []statusbar.Binding{
	{Key: "Space", Label: "pause"},
	{Key: ">/<", Label: "next/prev"},
	{Key: "+/-", Label: "vol"},
	{Key: "q", Label: "detach"},
	{Key: "?", Label: "help"},
}

var globalHelpBindings = []statusbar.Binding{
	{Key: "Space", Label: "toggle play / pause"},
	{Key: ">", Label: "next track"},
	{Key: "<", Label: "previous track"},
	{Key: "+ / =", Label: "volume up"},
	{Key: "- / _", Label: "volume down"},
	{Key: "Shift+←/→", Label: "seek backward / forward 5s"},
	{Key: "?", Label: "keybindings reference"},
	{Key: "q", Label: "detach (keep mpv playing in background)"},
	{Key: "Q", Label: "stop mpv daemon and quit"},
}

func (m *Model) statusBindings() []statusbar.Binding {
	var local []statusbar.Binding
	switch m.activePanel {
	case panels.PanelLibrary:
		local = []statusbar.Binding{
			{Key: "Enter", Label: "play"},
			{Key: "a", Label: "queue"},
			{Key: "/", Label: "filter"},
			{Key: "e", Label: "tag edit"},
			{Key: "[/]", Label: "tabs"},
		}
	case panels.PanelPlaylists:
		local = []statusbar.Binding{
			{Key: "Enter", Label: "load"},
			{Key: "a", Label: "queue"},
		}
	case panels.PanelFileBrowser:
		local = []statusbar.Binding{
			{Key: "→/Enter", Label: "open"},
			{Key: "←", Label: "back"},
			{Key: "a", Label: "queue"},
		}
	case panels.PanelQueue:
		local = []statusbar.Binding{
			{Key: "Space/Enter", Label: "play"},
			{Key: "d", Label: "remove"},
			{Key: "D", Label: "clear"},
			{Key: "s", Label: "shuffle"},
			{Key: "Shift+↑/↓", Label: "reorder"},
			{Key: "/", Label: "filter"},
		}
	}

	res := make([]statusbar.Binding, 0, len(local)+len(globalStatusBindings))
	res = append(res, local...)
	res = append(res, globalStatusBindings...)
	return res
}

func (m *Model) helpLocalBindings() []statusbar.Binding {
	switch m.activePanel {
	case panels.PanelLibrary:
		return []statusbar.Binding{
			{Key: "Enter", Label: "play selected track immediately"},
			{Key: "a", Label: "append selected track to queue"},
			{Key: "/", Label: "open live fuzzy filter search"},
			{Key: "e", Label: "edit ID3v2 metadata tags for selected track"},
			{Key: "[ / ]", Label: "switch sub-tab (Dirs / Artists / Albums / Tracks)"},
		}
	case panels.PanelPlaylists:
		return []statusbar.Binding{
			{Key: "Enter", Label: "load playlist (replaces queue)"},
			{Key: "a", Label: "append all playlist tracks to queue"},
		}
	case panels.PanelFileBrowser:
		return []statusbar.Binding{
			{Key: "→ / Enter", Label: "enter directory or play selected audio file"},
			{Key: "←", Label: "go back to previous directory (Yazi-style)"},
			{Key: "a", Label: "append selected audio file to queue"},
		}
	case panels.PanelQueue:
		return []statusbar.Binding{
			{Key: "Space / Enter", Label: "play selected track (or pause if currently playing)"},
			{Key: "d", Label: "remove selected track from queue"},
			{Key: "D", Label: "clear entire queue"},
			{Key: "Shift+↑", Label: "move selected track up in queue"},
			{Key: "Shift+↓", Label: "move selected track down in queue"},
			{Key: "s", Label: "shuffle queue"},
			{Key: "/", Label: "filter tracks in queue"},
		}
	}
	return nil
}

func (m *Model) helpNavigationBindings() []statusbar.Binding {
	return []statusbar.Binding{
		{Key: "0-3", Label: "jump directly to panel (0: Queue, 1: Lib, 2: Playlists, 3: Files)"},
		{Key: "Tab / Shift+Tab", Label: "cycle panels clockwise / counter-clockwise"},
		{Key: "↑ / ↓", Label: "navigate items in current panel"},
	}
}

func (m *Model) helpSections() []helppopup.Section {
	sections := make([]helppopup.Section, 0, 3)
	if local := m.helpLocalBindings(); len(local) > 0 {
		panelName := "Library"
		switch m.activePanel {
		case panels.PanelPlaylists:
			panelName = "Playlists"
		case panels.PanelFileBrowser:
			panelName = "File Browser"
		case panels.PanelQueue:
			panelName = "Queue"
		}
		sections = append(sections, helppopup.Section{Title: "Local (" + panelName + ")", Bindings: local})
	}
	sections = append(sections, helppopup.Section{Title: "Global", Bindings: globalHelpBindings})
	sections = append(sections, helppopup.Section{Title: "Navigation", Bindings: m.helpNavigationBindings()})
	return sections
}

func (m *Model) updateStatusBar() {
	if m.statusPanel != nil {
		m.statusPanel.SetKeybindings(m.statusBindings())
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.listenPlayerEvents(),
		m.scanLibraryCmd(),
	)
}

func (m *Model) scanLibraryCmd() tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil || m.cfg == nil {
			return nil
		}
		total := 0
		for _, dir := range m.cfg.MusicDirs {
			count, err := m.storage.ScanDirectory(dir)
			if err != nil {
				return LibraryScanMsg{TrackCount: total, Err: err}
			}
			total += count
		}
		return LibraryScanMsg{TrackCount: total, Err: nil}
	}
}

func (m *Model) listenPlayerEvents() tea.Cmd {
	return func() tea.Msg {
		if m.player == nil {
			return nil
		}
		state := <-m.player.Events()
		return PlaybackMsg(state)
	}
}

func (m *Model) updateFocus() {
	for _, p := range m.leftPanels {
		if p.ID() == m.activePanel {
			p.Focus()
		} else {
			p.Blur()
		}
	}
	if m.queuePanel.ID() == m.activePanel {
		m.queuePanel.Focus()
	} else {
		m.queuePanel.Blur()
	}
}

func (m *Model) currentPanel() panels.Panel {
	if m.activePanel == panels.PanelQueue {
		return m.queuePanel
	}
	for _, p := range m.leftPanels {
		if p.ID() == m.activePanel {
			return p
		}
	}
	return m.leftPanels[0]
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case LibraryScanMsg:
		if lib, ok := m.leftPanels[0].(*panels.LibraryPanel); ok {
			lib.Refresh()
		}
		return m, nil

	case PlaybackMsg:
		state := player.PlaybackState(msg)
		m.statusPanel.SetPlaybackState(state)
		if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
			q.SetPlaylist(state.Playlist, state.PlaylistPos)
		}
		return m, m.listenPlayerEvents()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateLayout()
		return m, nil

	case tea.KeyMsg:
		// Handle modals first
		if m.modalMode != ModalNone {
			return m.handleModalKey(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			if m.player != nil {
				_ = m.player.Detach()
			}
			return m, tea.Quit

		// Detach and quit (mpv continues playing in background)
		case "q":
			if m.player != nil {
				_ = m.player.Detach()
			}
			return m, tea.Quit

		// Stop daemon and quit completely
		case "Q":
			if m.player != nil {
				_ = m.player.QuitDaemon()
			}
			return m, tea.Quit

		// Modals
		case "?":
			m.modalMode = ModalHelp
			return m, nil

		case "/":
			if m.activePanel == panels.PanelLibrary || m.activePanel == panels.PanelQueue {
				m.modalMode = ModalSearch
				m.searchBuffer = ""
			}
			return m, nil

		case "e":
			// Open tag editor if on a track in Library or Queue
			if m.activePanel == panels.PanelLibrary {
				if lib, ok := m.leftPanels[0].(*panels.LibraryPanel); ok {
					if track, ok := lib.SelectedTrack(); ok {
						m.modalMode = ModalTagEdit
						m.tagTrackPath = track.Path
						m.tagTitle = track.Title
						m.tagArtist = track.Artist
						m.tagAlbum = track.Album
						m.tagFieldIdx = 0
					}
				}
			}
			return m, nil

		// Enter: Play track immediately, open folder, or load playlist
		case "enter":
			if m.activePanel == panels.PanelLibrary {
				if lib, ok := m.leftPanels[0].(*panels.LibraryPanel); ok {
					if track, found := lib.SelectedTrack(); found && m.player != nil {
						_ = m.player.LoadFile(track.Path, "replace")
					}
				}
			} else if m.activePanel == panels.PanelPlaylists {
				if pl, ok := m.leftPanels[1].(*panels.PlaylistsPanel); ok {
					if item, found := pl.SelectedPlaylist(); found && m.player != nil {
						_ = m.player.LoadFile(item.Path, "replace")
					}
				}
			} else if m.activePanel == panels.PanelFileBrowser {
				if fb, ok := m.leftPanels[2].(*panels.FileBrowserPanel); ok {
					if !fb.Enter() {
						if item, ok := fb.SelectedItem(); ok && !item.IsDir && m.player != nil {
							_ = m.player.LoadFile(item.Path, "replace")
						}
					}
				}
			} else if m.activePanel == panels.PanelQueue {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok && m.player != nil {
					if idx, _, found := q.SelectedTrack(); found {
						_ = m.player.PlayIndex(idx)
					}
				}
			}
			return m, nil

		// a: Append to queue
		case "a":
			if m.activePanel == panels.PanelLibrary {
				if lib, ok := m.leftPanels[0].(*panels.LibraryPanel); ok {
					if track, found := lib.SelectedTrack(); found && m.player != nil {
						_ = m.player.LoadFile(track.Path, "append")
					}
				}
			} else if m.activePanel == panels.PanelPlaylists {
				if pl, ok := m.leftPanels[1].(*panels.PlaylistsPanel); ok {
					if item, found := pl.SelectedPlaylist(); found && m.player != nil {
						_ = m.player.LoadFile(item.Path, "append")
					}
				}
			} else if m.activePanel == panels.PanelFileBrowser {
				if fb, ok := m.leftPanels[2].(*panels.FileBrowserPanel); ok {
					if item, ok := fb.SelectedItem(); ok && !item.IsDir && m.player != nil {
						_ = m.player.LoadFile(item.Path, "append")
					}
				}
			}
			return m, nil

		// d: Remove selected track from queue
		case "d":
			if m.activePanel == panels.PanelQueue && m.player != nil {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
					if idx, _, found := q.SelectedTrack(); found {
						_ = m.player.RemoveFromPlaylist(idx)
					}
				}
			}
			return m, nil

		// D: Clear queue
		case "D":
			if m.activePanel == panels.PanelQueue && m.player != nil {
				_ = m.player.ClearPlaylist()
			}
			return m, nil

		// Shift+Up: Move track up in queue
		case "shift+up":
			if m.activePanel == panels.PanelQueue && m.player != nil {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
					idx := q.Cursor()
					if idx > 0 {
						_ = m.player.MoveInPlaylist(idx, idx-1)
						q.CursorUp()
					}
				}
			}
			return m, nil

		// Shift+Down: Move track down in queue
		case "shift+down":
			if m.activePanel == panels.PanelQueue && m.player != nil {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
					idx := q.Cursor()
					_ = m.player.MoveInPlaylist(idx, idx+1)
					q.CursorDown()
				}
			}
			return m, nil

		// s: Shuffle playlist
		case "s":
			if m.player != nil {
				_ = m.player.ShufflePlaylist()
			}
			return m, nil

		// Playback controls / Space: in Queue panel play selected track; globally toggle pause
		case " ":
			if m.activePanel == panels.PanelQueue && m.player != nil {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
					if idx, track, found := q.SelectedTrack(); found {
						// If already on the currently playing track, toggle pause
						if track.Current || track.Playing {
							_ = m.player.TogglePause()
						} else {
							_ = m.player.PlayIndex(idx)
						}
						return m, nil
					}
				}
			}
			if m.player != nil {
				_ = m.player.TogglePause()
			}
			return m, nil
		case ">":
			if m.player != nil {
				_ = m.player.Next()
			}
			return m, nil
		case "<":
			if m.player != nil {
				_ = m.player.Prev()
			}
			return m, nil
		case "+", "=":
			if m.player != nil {
				_ = m.player.VolumeUp()
			}
			return m, nil
		case "-", "_":
			if m.player != nil {
				_ = m.player.VolumeDown()
			}
			return m, nil
		case "shift+right", "alt+right":
			if m.player != nil {
				_ = m.player.SeekRelative(5)
			}
			return m, nil
		case "shift+left", "alt+left":
			if m.player != nil {
				_ = m.player.SeekRelative(-5)
			}
			return m, nil

		// Direct panel navigation
		case "0":
			m.activePanel = panels.PanelQueue
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil
		case "1":
			m.activePanel = panels.PanelLibrary
			m.lastLeftFocus = panels.PanelLibrary
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil
		case "2":
			m.activePanel = panels.PanelPlaylists
			m.lastLeftFocus = panels.PanelPlaylists
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil
		case "3":
			m.activePanel = panels.PanelFileBrowser
			m.lastLeftFocus = panels.PanelFileBrowser
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil

		// Panel cycling: 1 -> 2 -> 3 -> 0 -> 1
		case "tab":
			switch m.activePanel {
			case panels.PanelLibrary:
				m.activePanel = panels.PanelPlaylists
				m.lastLeftFocus = panels.PanelPlaylists
			case panels.PanelPlaylists:
				m.activePanel = panels.PanelFileBrowser
				m.lastLeftFocus = panels.PanelFileBrowser
			case panels.PanelFileBrowser:
				m.activePanel = panels.PanelQueue
			case panels.PanelQueue:
				m.activePanel = panels.PanelLibrary
				m.lastLeftFocus = panels.PanelLibrary
			}
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil

		case "shift+tab":
			switch m.activePanel {
			case panels.PanelLibrary:
				m.activePanel = panels.PanelQueue
			case panels.PanelPlaylists:
				m.activePanel = panels.PanelLibrary
				m.lastLeftFocus = panels.PanelLibrary
			case panels.PanelFileBrowser:
				m.activePanel = panels.PanelPlaylists
				m.lastLeftFocus = panels.PanelPlaylists
			case panels.PanelQueue:
				m.activePanel = panels.PanelFileBrowser
				m.lastLeftFocus = panels.PanelFileBrowser
			}
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil

		// Sub-tab cycling
		case "[":
			m.currentPanel().PrevTab()
			return m, nil
		case "]":
			m.currentPanel().NextTab()
			return m, nil

		// Cursor navigation strictly with arrow keys
		case "up":
			m.currentPanel().CursorUp()
			return m, nil
		case "down":
			m.currentPanel().CursorDown()
			return m, nil

		// Yazi-like navigation for File Browser (left: parent directory, right: enter directory / play file)
		case "left":
			if m.activePanel == panels.PanelFileBrowser {
				if fb, ok := m.leftPanels[2].(*panels.FileBrowserPanel); ok {
					fb.GoParent()
				}
			}
			return m, nil

		case "right":
			if m.activePanel == panels.PanelFileBrowser {
				if fb, ok := m.leftPanels[2].(*panels.FileBrowserPanel); ok {
					if !fb.Enter() {
						if item, ok := fb.SelectedItem(); ok && !item.IsDir && m.player != nil {
							_ = m.player.LoadFile(item.Path, "replace")
						}
					}
				}
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.modalMode {
	case ModalHelp:
		switch key {
		case "esc", "?", "enter", "q":
			m.modalMode = ModalNone
			m.helpScroll = 0
		case "up", "k":
			if m.helpScroll > 0 {
				m.helpScroll--
			}
		case "down", "j":
			max := helppopup.MaxScroll(m.helpSections(), m.height)
			if m.helpScroll < max {
				m.helpScroll++
			}
		}
		return m, nil

	case ModalSearch:
		switch key {
		case "esc":
			m.modalMode = ModalNone
			m.searchBuffer = ""
			m.applySearchFilter()
		case "enter":
			m.modalMode = ModalNone
		case "backspace":
			if len(m.searchBuffer) > 0 {
				m.searchBuffer = m.searchBuffer[:len(m.searchBuffer)-1]
				m.applySearchFilter()
			}
		default:
			if len(key) == 1 {
				m.searchBuffer += key
				m.applySearchFilter()
			}
		}
		return m, nil

	case ModalTagEdit:
		switch key {
		case "esc":
			m.modalMode = ModalNone
		case "tab", "down":
			m.tagFieldIdx = (m.tagFieldIdx + 1) % 3
		case "shift+tab", "up":
			m.tagFieldIdx = (m.tagFieldIdx - 1 + 3) % 3
		case "backspace":
			switch m.tagFieldIdx {
			case 0:
				if len(m.tagTitle) > 0 {
					m.tagTitle = m.tagTitle[:len(m.tagTitle)-1]
				}
			case 1:
				if len(m.tagArtist) > 0 {
					m.tagArtist = m.tagArtist[:len(m.tagArtist)-1]
				}
			case 2:
				if len(m.tagAlbum) > 0 {
					m.tagAlbum = m.tagAlbum[:len(m.tagAlbum)-1]
				}
			}
		case "enter":
			// Save metadata to file & SQLite
			if m.storage != nil {
				_ = m.storage.UpdateTrackMetadata(m.tagTrackPath, m.tagTitle, m.tagArtist, m.tagAlbum)
				if lib, ok := m.leftPanels[0].(*panels.LibraryPanel); ok {
					lib.Refresh()
				}
			}
			m.modalMode = ModalNone
		default:
			if len(key) == 1 {
				switch m.tagFieldIdx {
				case 0:
					m.tagTitle += key
				case 1:
					m.tagArtist += key
				case 2:
					m.tagAlbum += key
				}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) applySearchFilter() {
	if m.activePanel == panels.PanelLibrary {
		if lib, ok := m.leftPanels[0].(*panels.LibraryPanel); ok {
			lib.SetFilterQuery(m.searchBuffer)
		}
	} else if m.activePanel == panels.PanelQueue {
		if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
			q.SetFilterQuery(m.searchBuffer)
		}
	}
}

func (m *Model) recalculateLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	statusBarHeight := 5
	mainHeight := m.height - statusBarHeight
	if mainHeight < 10 {
		mainHeight = 10
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	// Queue panel (Panel 0)
	m.queuePanel.SetSize(rightWidth, mainHeight)

	// Accordion height split for left column (Panels 1, 2, 3)
	activeLeft := m.activePanel
	if activeLeft == panels.PanelQueue {
		activeLeft = m.lastLeftFocus
	}

	expandedHeight := int(float64(mainHeight) * 0.60)
	collapsedHeight := (mainHeight - expandedHeight) / 2
	remainingHeight := mainHeight - expandedHeight - collapsedHeight

	for _, p := range m.leftPanels {
		var h int
		if p.ID() == activeLeft {
			h = expandedHeight
		} else if p.ID() == panels.PanelPlaylists && activeLeft == panels.PanelLibrary {
			h = collapsedHeight
		} else {
			h = remainingHeight
		}
		p.SetSize(leftWidth, h)
	}

	// Status panel
	m.statusPanel.SetSize(m.width, statusBarHeight)
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing lazymusic..."
	}

	// Render left column
	var leftViews []string
	for _, p := range m.leftPanels {
		leftViews = append(leftViews, p.View())
	}
	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftViews...)

	// Render right column
	rightCol := m.queuePanel.View()

	// Join main columns horizontally
	mainGrid := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	// Join main grid and status bar vertically
	baseView := lipgloss.JoinVertical(lipgloss.Left, mainGrid, m.statusPanel.View())

	// If modal is active, overlay it
	if m.modalMode != ModalNone {
		return m.renderModalOverlay(baseView)
	}

	return baseView
}

func (m *Model) renderModalOverlay(base string) string {
	if m.modalMode == ModalHelp {
		helpBox := helppopup.Box(m.helpSections(), m.width, m.height, m.helpScroll)
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			helpBox,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)
	}

	var modalContent string
	var title string

	switch m.modalMode {
	case ModalSearch:
		title = "Fuzzy Filter"
		modalContent = fmt.Sprintf("Type to filter:\n\n  /%s█\n\n[Esc: Clear/Close | Enter: Confirm]", m.searchBuffer)

	case ModalTagEdit:
		title = "Edit Audio Metadata (ID3v2)"
		cursor0, cursor1, cursor2 := "  ", "  ", "  "
		switch m.tagFieldIdx {
		case 0:
			cursor0 = "> "
		case 1:
			cursor1 = "> "
		case 2:
			cursor2 = "> "
		}

		modalContent = strings.Join([]string{
			fmt.Sprintf("File: %s", lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(m.tagTrackPath)),
			"",
			fmt.Sprintf("%sTitle:  %s", cursor0, m.tagTitle),
			fmt.Sprintf("%sArtist: %s", cursor1, m.tagArtist),
			fmt.Sprintf("%sAlbum:  %s", cursor2, m.tagAlbum),
			"",
			"[Tab/Arrows: Next field | Enter: Save | Esc: Cancel]",
		}, "\n")
	}

	box := theme.ModalBoxStyle.Width(min(60, m.width-4)).Render(
		fmt.Sprintf("%s\n\n%s", lipgloss.NewStyle().Bold(true).Foreground(theme.PaletteGreen).Render(title), modalContent),
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}
