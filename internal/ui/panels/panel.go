package panels

import (
	tea "github.com/charmbracelet/bubbletea"
)

// PanelID represents the identifier for each panel in lazymusic.
type PanelID int

const (
	PanelQueue PanelID = iota // 0: Queue (right column)
	PanelLibrary              // 1: Library
	PanelPlaylists            // 2: Playlists
	PanelFileBrowser          // 3: File Browser
)

// Panel defines the interface that all TUI panels must implement.
type Panel interface {
	ID() PanelID
	Title() string
	SetSize(width, height int)
	Focus()
	Blur()
	IsFocused() bool
	NextTab()
	PrevTab()
	CursorUp()
	CursorDown()
	Update(msg tea.Msg) (Panel, tea.Cmd)
	View() string
}
