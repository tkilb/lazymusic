package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/config"
	"github.com/tylerkilburn/lazymusic/internal/ui/panels"
)

// Model represents the main Bubble Tea model for lazymusic.
type Model struct {
	cfg          *config.Config
	width        int
	height       int
	activePanel  panels.PanelID
	leftPanels   []panels.Panel
	queuePanel   panels.Panel
	statusPanel  *panels.StatusPanel
	lastLeftFocus panels.PanelID
}

// NewModel initializes the TUI model.
func NewModel(cfg *config.Config) *Model {
	m := &Model{
		cfg:          cfg,
		activePanel:  panels.PanelLibrary,
		lastLeftFocus: panels.PanelLibrary,
		leftPanels: []panels.Panel{
			panels.NewLibraryPanel(),
			panels.NewPlaylistsPanel(),
			panels.NewFileBrowserPanel(),
		},
		queuePanel:  panels.NewQueuePanel(),
		statusPanel: panels.NewStatusPanel(),
	}
	m.updateFocus()
	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateLayout()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

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
