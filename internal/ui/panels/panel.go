package panels

import (
	tea "github.com/charmbracelet/bubbletea"
)

// PanelID represents the identifier for each panel in lazymusic.
type PanelID int

const (
	PanelQueue       PanelID = iota // 0: Queue (right column)
	PanelPlaylists                  // 1: Playlists
	PanelFileBrowser                // 2: Files (File Browser)
	PanelLibrary                    // 3: Dirs (Music Directories)
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
