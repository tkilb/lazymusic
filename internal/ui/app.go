package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/config"
	"github.com/tylerkilburn/lazymusic/internal/library"
	"github.com/tylerkilburn/lazymusic/internal/player"
	"github.com/tylerkilburn/lazymusic/internal/ui/panels"
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
			panels.NewPlaylistsPanel(),
			panels.NewFileBrowserPanel(""),
		},
		queuePanel:  panels.NewQueuePanel(),
		statusPanel: panels.NewStatusPanel(),
	}
	m.updateFocus()
	return m
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
		// Refresh library panel with freshly scanned data
		if lib, ok := m.leftPanels[0].(*panels.LibraryPanel); ok {
			lib.Refresh()
		}
		return m, nil

	case PlaybackMsg:
		m.statusPanel.SetPlaybackState(player.PlaybackState(msg))
		return m, m.listenPlayerEvents()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateLayout()
		return m, nil

	case tea.KeyMsg:
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

		// Enter: Play track immediately or open folder
		case "enter":
			if m.activePanel == panels.PanelLibrary {
				if lib, ok := m.leftPanels[0].(*panels.LibraryPanel); ok {
					if track, found := lib.SelectedTrack(); found && m.player != nil {
						_ = m.player.LoadFile(track.Path, "replace")
					}
				}
			} else if m.activePanel == panels.PanelFileBrowser {
				if fb, ok := m.leftPanels[2].(*panels.FileBrowserPanel); ok {
					if !fb.Enter() {
						// It's a file, play it!
						if item, ok := fb.SelectedItem(); ok && !item.IsDir && m.player != nil {
							_ = m.player.LoadFile(item.Path, "replace")
						}
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
			} else if m.activePanel == panels.PanelFileBrowser {
				if fb, ok := m.leftPanels[2].(*panels.FileBrowserPanel); ok {
					if item, ok := fb.SelectedItem(); ok && !item.IsDir && m.player != nil {
						_ = m.player.LoadFile(item.Path, "append")
					}
				}
			}
			return m, nil

		// Playback controls
		case " ":
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
		case "alt+right":
			if m.player != nil {
				_ = m.player.SeekRelative(5)
			}
			return m, nil
		case "alt+left":
			if m.player != nil {
				_ = m.player.SeekRelative(-5)
			}
			return m, nil

		// Direct panel navigation
		case "0":
			m.activePanel = panels.PanelQueue
			m.updateFocus()
			m.recalculateLayout()
			return m, nil
		case "1":
			m.activePanel = panels.PanelLibrary
			m.lastLeftFocus = panels.PanelLibrary
			m.updateFocus()
			m.recalculateLayout()
			return m, nil
		case "2":
			m.activePanel = panels.PanelPlaylists
			m.lastLeftFocus = panels.PanelPlaylists
			m.updateFocus()
			m.recalculateLayout()
			return m, nil
		case "3":
			m.activePanel = panels.PanelFileBrowser
			m.lastLeftFocus = panels.PanelFileBrowser
			m.updateFocus()
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
			m.recalculateLayout()
			return m, nil

		// Sub-tab cycling
		case "[":
			m.currentPanel().PrevTab()
			return m, nil
		case "]":
			m.currentPanel().NextTab()
			return m, nil

		// Cursor navigation strictly with arrow keys (h/j/k/l are freed)
		case "up":
			m.currentPanel().CursorUp()
			return m, nil
		case "down":
			m.currentPanel().CursorDown()
			return m, nil
		}
	}

	return m, nil
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
	// Active left panel takes ~60%, inactive take ~20% each.
	// If Queue is focused, use m.lastLeftFocus to determine heights.
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
	return lipgloss.JoinVertical(lipgloss.Left, mainGrid, m.statusPanel.View())
}
