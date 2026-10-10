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
	ModalTagEdit
	ModalAddDir
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
	playlistsPanel *panels.PlaylistsPanel
	libraryPanel   *panels.LibraryPanel
	fileBrowser    *panels.FileBrowserPanel
	queuePanel    panels.Panel
	statusPanel   *panels.StatusPanel
	lastLeftFocus panels.PanelID

	// Search state (lazytask/lazygit inline search)
	searching    bool
	searchBuffer string

	// Modal states
	modalMode    ModalMode
	helpScroll   int
	addDirBuffer string
	tagTrackPath string
	tagTitle     string
	tagArtist    string
	tagAlbum     string
	tagFieldIdx  int // 0: Title, 1: Artist, 2: Album

	// Preview state
	previewing       bool
	previewStarted   bool
	previewEntryID   int
	previewPos       int
	previewSavedPos  int
	previewSavedTime float64
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
	plPanel := panels.NewPlaylistsPanel(cfg.PlaylistsDir)
	libPanel := panels.NewLibraryPanel(cfg, storage)
	fbPanel := panels.NewFileBrowserPanel("")
	m := &Model{
		cfg:            cfg,
		player:         pClient,
		storage:        storage,
		activePanel:    panels.PanelPlaylists,
		lastLeftFocus:  panels.PanelPlaylists,
		playlistsPanel: plPanel,
		libraryPanel:   libPanel,
		fileBrowser:    fbPanel,
		leftPanels: []panels.Panel{
			plPanel,
			fbPanel,
			libPanel,
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
			{Key: "Enter", Label: "open in files"},
			{Key: "a", Label: "add dir"},
			{Key: "d", Label: "remove dir"},
			{Key: "r", Label: "rescan"},
			{Key: "/", Label: "search"},
		}
	case panels.PanelPlaylists:
		local = []statusbar.Binding{
			{Key: "Enter", Label: "load"},
			{Key: "a", Label: "queue"},
			{Key: "/", Label: "search"},
		}
	case panels.PanelFileBrowser:
		local = []statusbar.Binding{
			{Key: "→/Enter", Label: "open/preview"},
			{Key: "←", Label: "back"},
			{Key: "A", Label: "add to lib"},
			{Key: "a", Label: "queue"},
			{Key: "/", Label: "search"},
		}
	case panels.PanelQueue:
		local = []statusbar.Binding{
			{Key: "Space/Enter", Label: "play"},
			{Key: "d", Label: "remove"},
			{Key: "D", Label: "clear"},
			{Key: "s", Label: "shuffle"},
			{Key: "Shift+↑/↓", Label: "reorder"},
			{Key: "/", Label: "search"},
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
			{Key: "Enter", Label: "open selected directory in File Browser"},
			{Key: "a", Label: "add new music directory to library"},
			{Key: "d", Label: "remove selected directory from library"},
			{Key: "r", Label: "rescan audio files in library"},
			{Key: "/", Label: "search music directories"},
		}
	case panels.PanelPlaylists:
		return []statusbar.Binding{
			{Key: "Enter", Label: "load playlist (replaces queue)"},
			{Key: "a", Label: "append all playlist tracks to queue"},
			{Key: "/", Label: "search playlists"},
		}
	case panels.PanelFileBrowser:
		return []statusbar.Binding{
			{Key: "→ / Enter", Label: "enter directory or preview audio file (leaves queue intact)"},
			{Key: "←", Label: "go back to previous directory (Yazi-style)"},
			{Key: "A", Label: "add current directory to Library Music Dirs"},
			{Key: "a", Label: "append selected audio file to queue"},
			{Key: "/", Label: "search files and directories"},
		}
	case panels.PanelQueue:
		return []statusbar.Binding{
			{Key: "Space / Enter", Label: "play selected track (or pause if currently playing)"},
			{Key: "d", Label: "remove selected track from queue"},
			{Key: "D", Label: "clear entire queue"},
			{Key: "Shift+↑", Label: "move selected track up in queue"},
			{Key: "Shift+↓", Label: "move selected track down in queue"},
			{Key: "s", Label: "shuffle queue"},
			{Key: "/", Label: "search tracks in queue"},
		}
	}
	return nil
}

func (m *Model) helpNavigationBindings() []statusbar.Binding {
	return []statusbar.Binding{
		{Key: "0-3", Label: "jump directly to panel (0: Queue, 1: Playlists, 2: Files, 3: Dirs)"},
		{Key: "Tab / Shift+Tab", Label: "cycle panels clockwise / counter-clockwise"},
		{Key: "↑ / ↓", Label: "navigate items in current panel"},
		{Key: ". / ,", Label: "page down / page up in current panel"},
		{Key: "/", Label: "search items in focused panel (inline prompt)"},
		{Key: "n / N", Label: "jump to next / previous match"},
		{Key: "Esc", Label: "clear active search query / cancel prompt"},
	}
}

func (m *Model) helpSections() []helppopup.Section {
	sections := make([]helppopup.Section, 0, 3)
	if local := m.helpLocalBindings(); len(local) > 0 {
		panelName := "Playlists"
		switch m.activePanel {
		case panels.PanelFileBrowser:
			panelName = "Files"
		case panels.PanelLibrary:
			panelName = "Dirs"
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
		if m.libraryPanel != nil {
			m.libraryPanel.Refresh()
		}
		return m, nil

	case PlaybackMsg:
		state := player.PlaybackState(msg)
		m.statusPanel.SetPlaybackState(state)

		if m.previewing {
			// Decide everything from the playlist snapshot's own "current"
			// flags. PlaylistPos / IdleActive are cached separately and can
			// lag the playlist, so mixing them caused false "preview ended"
			// detections when previewing files in quick succession.
			previewCurrent := false
			otherCurrent := false
			for idx, item := range state.Playlist {
				if item.ID == m.previewEntryID {
					m.previewPos = idx
					previewCurrent = item.Current
				} else if item.Current {
					otherCurrent = true
				}
			}

			if previewCurrent {
				m.previewStarted = true
			}

			if m.previewStarted && !previewCurrent {
				// Preview finished: drop it. Resume the saved queue position only
				// if the preview ended on its own (nothing current); if the user
				// jumped to another queue track, keep that.
				finishedID := m.previewEntryID
				userJumped := otherCurrent
				m.previewing = false
				m.previewStarted = false
				m.previewEntryID = 0
				if m.player != nil {
					if idx, err := m.player.PlaylistIndexOfEntry(finishedID); err == nil && idx >= 0 {
						_ = m.player.RemoveFromPlaylist(idx)
					}
					if !userJumped && m.previewSavedPos >= 0 {
						_ = m.player.PlayIndex(m.previewSavedPos)
						if m.previewSavedTime > 0 {
							_ = m.player.SeekAbsolute(m.previewSavedTime)
						}
						_ = m.player.Unpause()
					}
				}
			}
		}

		if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
			if m.previewing && m.previewEntryID > 0 {
				// Hide preview item from Queue display so the user's queue is completely unaffected
				filtered := make([]player.PlaylistItem, 0, len(state.Playlist))
				for _, item := range state.Playlist {
					if item.ID != m.previewEntryID {
						filtered = append(filtered, item)
					}
				}
				q.SetPlaylist(filtered, m.previewSavedPos)
			} else {
				q.SetPlaylist(state.Playlist, state.PlaylistPos)
			}
		}
		return m, m.listenPlayerEvents()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateLayout()
		return m, nil

	case tea.KeyMsg:
		// Handle inline search first if actively typing query
		if m.searching {
			return m.handleSearchKey(msg)
		}

		// Handle modals next
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
			m.searching = true
			m.searchBuffer = ""
			if m.statusPanel != nil {
				m.statusPanel.SetSearchPrompt("/█")
			}
			return m, nil

		case "n":
			m.nextSearchMatch()
			return m, nil

		case "N":
			m.prevSearchMatch()
			return m, nil

		case "esc":
			if m.hasActiveSearch() {
				m.clearActiveSearch()
				return m, nil
			}
			return m, nil

		case "e":
			// Open tag editor if on a track in Queue
			if m.activePanel == panels.PanelQueue {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
					if _, item, ok := q.SelectedTrack(); ok {
						m.modalMode = ModalTagEdit
						m.tagTrackPath = item.Filename
						m.tagTitle = item.Title
						m.tagArtist = ""
						m.tagAlbum = ""
						m.tagFieldIdx = 0
					}
				}
			}
			return m, nil

		// Enter: Open directory in File Browser, load playlist, or play track
		case "enter":
			if m.activePanel == panels.PanelLibrary && m.libraryPanel != nil {
				if dir, found := m.libraryPanel.SelectedDir(); found {
					// Jump to File Browser and open this directory
					if m.fileBrowser != nil {
						m.fileBrowser.ReadDir(dir)
					}
					m.activePanel = panels.PanelFileBrowser
					m.lastLeftFocus = panels.PanelFileBrowser
					m.updateFocus()
					m.updateStatusBar()
					m.recalculateLayout()
				}
			} else if m.activePanel == panels.PanelPlaylists && m.playlistsPanel != nil {
				if item, found := m.playlistsPanel.SelectedPlaylist(); found && m.player != nil {
					_ = m.player.LoadFile(item.Path, "replace")
					_ = m.player.Unpause()
				}
			} else if m.activePanel == panels.PanelFileBrowser && m.fileBrowser != nil {
				if !m.fileBrowser.Enter() {
					if item, ok := m.fileBrowser.SelectedItem(); ok && !item.IsDir && m.player != nil {
						m.playPreview(item.Path)
					}
				}
			} else if m.activePanel == panels.PanelQueue {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok && m.player != nil {
					if idx, _, found := q.SelectedTrack(); found {
						_ = m.player.PlayIndex(idx)
						_ = m.player.Unpause()
					}
				}
			}
			return m, nil

		// a: Add directory (in Library) or append to queue (in Playlists / File Browser)
		case "a":
			if m.activePanel == panels.PanelLibrary {
				m.modalMode = ModalAddDir
				m.addDirBuffer = ""
				return m, nil
			} else if m.activePanel == panels.PanelPlaylists && m.playlistsPanel != nil {
				if item, found := m.playlistsPanel.SelectedPlaylist(); found && m.player != nil {
					_ = m.player.LoadFile(item.Path, "append")
				}
			} else if m.activePanel == panels.PanelFileBrowser && m.fileBrowser != nil {
				if item, ok := m.fileBrowser.SelectedItem(); ok && !item.IsDir && m.player != nil {
					_ = m.player.LoadFile(item.Path, "append")
				}
			}
			return m, nil

		// A: Add directory in File Browser to Library Music Dirs
		case "A":
			if m.activePanel == panels.PanelFileBrowser && m.cfg != nil && m.fileBrowser != nil {
				dirToAdd := m.fileBrowser.CurrentPath()
				if item, ok := m.fileBrowser.SelectedItem(); ok && item.IsDir {
					dirToAdd = item.Path
				}
				if m.cfg.AddMusicDir(dirToAdd) {
					if m.libraryPanel != nil {
						m.libraryPanel.Refresh()
					}
					// Scan the newly added directory
					return m, m.scanLibraryCmd()
				}
			}
			return m, nil

		// d: Remove directory (in Library) or remove track from queue (in Queue)
		case "d":
			if m.activePanel == panels.PanelLibrary && m.cfg != nil && m.libraryPanel != nil {
				if dir, found := m.libraryPanel.SelectedDir(); found {
					if m.cfg.RemoveMusicDir(dir) {
						if m.storage != nil {
							_ = m.storage.DeleteTracksInDirectory(dir)
						}
						m.libraryPanel.Refresh()
					}
				}
			} else if m.activePanel == panels.PanelQueue && m.player != nil {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
					if idx, _, found := q.SelectedTrack(); found {
						_ = m.player.RemoveFromPlaylist(idx)
					}
				}
			}
			return m, nil

		// r: Rescan library directories
		case "r":
			if m.activePanel == panels.PanelLibrary {
				return m, m.scanLibraryCmd()
			}
			return m, nil

		// D: Clear queue
		case "D":
			if m.activePanel == panels.PanelQueue && m.player != nil {
				_ = m.player.ClearPlaylist()
			}
			return m, nil

		// Shift+Up: Move track up in queue
		case "shift+up", "ctrl+shift+up":
			if m.activePanel == panels.PanelQueue && m.player != nil {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
					idx := q.Cursor()
					if idx > 0 && idx < q.Len() {
						_ = m.player.MoveInPlaylist(idx, idx-1)
						q.Move(idx, idx-1)
						q.CursorUp()
					}
				}
			}
			return m, nil

		// Shift+Down: Move track down in queue
		case "shift+down", "ctrl+shift+down":
			if m.activePanel == panels.PanelQueue && m.player != nil {
				if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
					idx := q.Cursor()
					if idx >= 0 && idx < q.Len()-1 {
						_ = m.player.MoveInPlaylist(idx, idx+1)
						q.Move(idx, idx+1)
						q.CursorDown()
					}
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
							_ = m.player.Unpause()
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
			m.activePanel = panels.PanelPlaylists
			m.lastLeftFocus = panels.PanelPlaylists
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil
		case "2":
			m.activePanel = panels.PanelFileBrowser
			m.lastLeftFocus = panels.PanelFileBrowser
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil
		case "3":
			m.activePanel = panels.PanelLibrary
			m.lastLeftFocus = panels.PanelLibrary
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil

		// Panel cycling: 1 -> 2 -> 3 -> 0 -> 1
		case "tab":
			switch m.activePanel {
			case panels.PanelPlaylists:
				m.activePanel = panels.PanelFileBrowser
				m.lastLeftFocus = panels.PanelFileBrowser
			case panels.PanelFileBrowser:
				m.activePanel = panels.PanelLibrary
				m.lastLeftFocus = panels.PanelLibrary
			case panels.PanelLibrary:
				m.activePanel = panels.PanelQueue
			case panels.PanelQueue:
				m.activePanel = panels.PanelPlaylists
				m.lastLeftFocus = panels.PanelPlaylists
			}
			m.updateFocus()
			m.updateStatusBar()
			m.recalculateLayout()
			return m, nil

		case "shift+tab":
			switch m.activePanel {
			case panels.PanelPlaylists:
				m.activePanel = panels.PanelQueue
			case panels.PanelFileBrowser:
				m.activePanel = panels.PanelPlaylists
				m.lastLeftFocus = panels.PanelPlaylists
			case panels.PanelLibrary:
				m.activePanel = panels.PanelFileBrowser
				m.lastLeftFocus = panels.PanelFileBrowser
			case panels.PanelQueue:
				m.activePanel = panels.PanelLibrary
				m.lastLeftFocus = panels.PanelLibrary
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

		// Paging
		case ".":
			m.currentPanel().PageDown()
			return m, nil
		case ",":
			m.currentPanel().PageUp()
			return m, nil

		// Yazi-like navigation for File Browser (left: parent directory, right: enter directory / play file)
		case "left":
			if m.activePanel == panels.PanelFileBrowser && m.fileBrowser != nil {
				m.fileBrowser.GoParent()
			}
			return m, nil

		case "right":
			if m.activePanel == panels.PanelFileBrowser && m.fileBrowser != nil {
				if !m.fileBrowser.Enter() {
					if item, ok := m.fileBrowser.SelectedItem(); ok && !item.IsDir && m.player != nil {
						m.playPreview(item.Path)
					}
				}
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.searching = false
		m.searchBuffer = ""
		if m.statusPanel != nil {
			m.statusPanel.SetSearchPrompt("")
		}
		m.clearActiveSearch()
		return m, nil

	case "enter":
		m.searching = false
		query := strings.TrimSpace(m.searchBuffer)
		m.searchBuffer = ""
		if m.statusPanel != nil {
			m.statusPanel.SetSearchPrompt("")
		}
		if query != "" {
			m.applySearch(query)
		}
		return m, nil

	case "backspace":
		if len(m.searchBuffer) > 0 {
			m.searchBuffer = m.searchBuffer[:len(m.searchBuffer)-1]
		}
		if m.statusPanel != nil {
			m.statusPanel.SetSearchPrompt("/" + m.searchBuffer + "█")
		}
		return m, nil

	default:
		if len(key) == 1 {
			m.searchBuffer += key
			if m.statusPanel != nil {
				m.statusPanel.SetSearchPrompt("/" + m.searchBuffer + "█")
			}
		}
		return m, nil
	}
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
				if m.libraryPanel != nil {
					m.libraryPanel.Refresh()
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

	case ModalAddDir:
		switch key {
		case "esc":
			m.modalMode = ModalNone
			m.addDirBuffer = ""
		case "backspace":
			if len(m.addDirBuffer) > 0 {
				m.addDirBuffer = m.addDirBuffer[:len(m.addDirBuffer)-1]
			}
		case "enter":
			dir := strings.TrimSpace(m.addDirBuffer)
			if dir != "" && m.cfg != nil {
				if m.cfg.AddMusicDir(dir) {
					if m.libraryPanel != nil {
						m.libraryPanel.Refresh()
					}
					m.modalMode = ModalNone
					m.addDirBuffer = ""
					return m, m.scanLibraryCmd()
				}
			}
			m.modalMode = ModalNone
			m.addDirBuffer = ""
		default:
			if len(key) == 1 {
				m.addDirBuffer += key
			}
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) applySearch(query string) {
	switch m.activePanel {
	case panels.PanelPlaylists:
		if m.playlistsPanel != nil {
			m.playlistsPanel.Search(query)
		}
	case panels.PanelFileBrowser:
		if m.fileBrowser != nil {
			m.fileBrowser.Search(query)
		}
	case panels.PanelLibrary:
		if m.libraryPanel != nil {
			m.libraryPanel.Search(query)
		}
	case panels.PanelQueue:
		if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
			q.Search(query)
		}
	}
}

func (m *Model) clearActiveSearch() {
	switch m.activePanel {
	case panels.PanelPlaylists:
		if m.playlistsPanel != nil {
			m.playlistsPanel.ClearSearch()
		}
	case panels.PanelFileBrowser:
		if m.fileBrowser != nil {
			m.fileBrowser.ClearSearch()
		}
	case panels.PanelLibrary:
		if m.libraryPanel != nil {
			m.libraryPanel.ClearSearch()
		}
	case panels.PanelQueue:
		if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
			q.ClearSearch()
		}
	}
}

func (m *Model) hasActiveSearch() bool {
	switch m.activePanel {
	case panels.PanelPlaylists:
		if m.playlistsPanel != nil {
			return m.playlistsPanel.HasActiveSearch()
		}
	case panels.PanelFileBrowser:
		if m.fileBrowser != nil {
			return m.fileBrowser.HasActiveSearch()
		}
	case panels.PanelLibrary:
		if m.libraryPanel != nil {
			return m.libraryPanel.HasActiveSearch()
		}
	case panels.PanelQueue:
		if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
			return q.HasActiveSearch()
		}
	}
	return false
}

func (m *Model) nextSearchMatch() {
	switch m.activePanel {
	case panels.PanelPlaylists:
		if m.playlistsPanel != nil {
			m.playlistsPanel.NextMatch()
		}
	case panels.PanelFileBrowser:
		if m.fileBrowser != nil {
			m.fileBrowser.NextMatch()
		}
	case panels.PanelLibrary:
		if m.libraryPanel != nil {
			m.libraryPanel.NextMatch()
		}
	case panels.PanelQueue:
		if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
			q.NextMatch()
		}
	}
}

func (m *Model) prevSearchMatch() {
	switch m.activePanel {
	case panels.PanelPlaylists:
		if m.playlistsPanel != nil {
			m.playlistsPanel.PrevMatch()
		}
	case panels.PanelFileBrowser:
		if m.fileBrowser != nil {
			m.fileBrowser.PrevMatch()
		}
	case panels.PanelLibrary:
		if m.libraryPanel != nil {
			m.libraryPanel.PrevMatch()
		}
	case panels.PanelQueue:
		if q, ok := m.queuePanel.(*panels.QueuePanel); ok {
			q.PrevMatch()
		}
	}
}

func (m *Model) playPreview(filePath string) {
	if m.player == nil {
		return
	}

	state := m.player.GetState()

	// If already in a preview, remove the previous preview entry first
	// (queried directly from mpv; the cached playlist can be stale).
	if m.previewing {
		if idx, err := m.player.PlaylistIndexOfEntry(m.previewEntryID); err == nil && idx >= 0 {
			_ = m.player.RemoveFromPlaylist(idx)
		}
		m.previewing = false
		m.previewEntryID = 0
	} else {
		// Save current queue playback state to resume later
		m.previewSavedPos = state.PlaylistPos
		m.previewSavedTime = state.TimePos
	}

	entryID, err := m.player.AppendFile(filePath)
	if err != nil || entryID == 0 {
		return
	}
	previewIdx, err := m.player.PlaylistIndexOfEntry(entryID)
	if err != nil || previewIdx < 0 {
		return
	}

	m.previewing = true
	m.previewStarted = false
	m.previewEntryID = entryID
	m.previewPos = previewIdx

	_ = m.player.PlayIndex(previewIdx)
	_ = m.player.Unpause()
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
	inactiveCount := len(m.leftPanels) - 1
	if inactiveCount <= 0 {
		inactiveCount = 1
	}
	baseInactiveHeight := (mainHeight - expandedHeight) / inactiveCount
	extraHeight := (mainHeight - expandedHeight) % inactiveCount

	inactiveIdx := 0
	for _, p := range m.leftPanels {
		if p.ID() == activeLeft {
			p.SetSize(leftWidth, expandedHeight)
		} else {
			h := baseInactiveHeight
			if inactiveIdx < extraHeight {
				h++
			}
			p.SetSize(leftWidth, h)
			inactiveIdx++
		}
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
	case ModalAddDir:
		title = "Add Music Directory"
		modalContent = fmt.Sprintf("Enter directory path:\n\n  %s█\n\n[Esc: Cancel | Enter: Add & Scan]", m.addDirBuffer)

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
