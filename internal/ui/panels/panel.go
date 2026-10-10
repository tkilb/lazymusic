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
	PageUp()
	PageDown()
	Update(msg tea.Msg) (Panel, tea.Cmd)
	View() string
}

// pageMove returns the cursor after moving one page (direction +1/-1) within
// a list of n items, given the panel's outer height. A page is the number of
// visible rows minus one so a line of context stays on screen.
func pageMove(cursor, n, height, direction int) int {
	if n <= 0 {
		return 0
	}
	step := height - 2 - 1
	if step < 1 {
		step = 1
	}
	cursor += direction * step
	if cursor < 0 {
		cursor = 0
	}
	if cursor > n-1 {
		cursor = n - 1
	}
	return cursor
}
