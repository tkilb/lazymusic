package panels

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// FileItem represents a directory or audio file in the file browser.
type FileItem struct {
	Name    string
	Path    string // Real physical path (for reading/playback)
	IsDir   bool
	IsLink  bool
	Target  string
}

// FileBrowserPanel implements Panel for Panel 3 (File Browser).
type FileBrowserPanel struct {
	focused     bool
	width       int
	height      int
	cursor      int
	offset      int
	currentPath string   // Logical current directory path (preserving symlink structure)
	history     []string // Breadcrumb stack of previous directories for clean backtracking
	items       []FileItem
	searchQuery string
}

// NewFileBrowserPanel creates a new FileBrowserPanel initialized to the user's music or home directory.
func NewFileBrowserPanel(initialDir string) *FileBrowserPanel {
	if initialDir == "" {
		home, _ := os.UserHomeDir()
		initialDir = filepath.Join(home, "Music")
		if _, err := os.Stat(initialDir); os.IsNotExist(err) {
			initialDir = home
		}
	}

	p := &FileBrowserPanel{
		cursor:      0,
		currentPath: filepath.Clean(initialDir),
	}
	p.ReadDir(initialDir)
	return p
}

func (p *FileBrowserPanel) ID() PanelID {
	return PanelFileBrowser
}

func (p *FileBrowserPanel) Title() string {
	return "2: Files"
}

// CurrentPath returns the current logical directory path.
func (p *FileBrowserPanel) CurrentPath() string {
	return p.currentPath
}

func (p *FileBrowserPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *FileBrowserPanel) Focus() {
	p.focused = true
}

func (p *FileBrowserPanel) Blur() {
	p.focused = false
}

func (p *FileBrowserPanel) IsFocused() bool {
	return p.focused
}

func (p *FileBrowserPanel) NextTab() {}

func (p *FileBrowserPanel) PrevTab() {}

func (p *FileBrowserPanel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *FileBrowserPanel) CursorDown() {
	if p.cursor < len(p.items)-1 {
		p.cursor++
	}
}

func (p *FileBrowserPanel) PageUp()   { p.cursor = pageMove(p.cursor, len(p.items), p.height, -1) }
func (p *FileBrowserPanel) PageDown() { p.cursor = pageMove(p.cursor, len(p.items), p.height, 1) }

// Search sets query as the active search and jumps cursor to nearest match.
func (p *FileBrowserPanel) Search(query string) bool {
	p.searchQuery = query
	i, ok := p.findMatch(p.cursor, 1, true)
	if ok {
		p.cursor = i
	}
	return ok
}

// ClearSearch discards the active search query.
func (p *FileBrowserPanel) ClearSearch() {
	p.searchQuery = ""
}

// HasActiveSearch reports whether a search query is active.
func (p *FileBrowserPanel) HasActiveSearch() bool {
	return p.searchQuery != ""
}

// NextMatch moves cursor to next match forward.
func (p *FileBrowserPanel) NextMatch() bool {
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
func (p *FileBrowserPanel) PrevMatch() bool {
	if p.searchQuery == "" {
		return false
	}
	i, ok := p.findMatch(p.cursor, -1, false)
	if ok {
		p.cursor = i
	}
	return ok
}

func (p *FileBrowserPanel) findMatch(start, direction int, includeStart bool) (int, bool) {
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

// ReadDir loads directory contents, placing directories first, followed by audio files.
func (p *FileBrowserPanel) ReadDir(logicalDir string) {
	logicalDir = filepath.Clean(logicalDir)
	p.currentPath = logicalDir
	p.cursor = 0
	p.offset = 0
	p.items = nil

	entries, err := os.ReadDir(logicalDir)
	if err != nil {
		return
	}

	var dirs []FileItem
	var files []FileItem

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue // Skip hidden files
		}

		logicalItemPath := filepath.Join(logicalDir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		isDir := entry.IsDir()
		isLink := (info.Mode() & os.ModeSymlink) != 0
		targetPath := logicalItemPath

		if isLink {
			// Resolve symlink target
			resolved, err := filepath.EvalSymlinks(logicalItemPath)
			if err == nil {
				targetPath = resolved
				stat, err := os.Stat(resolved)
				if err == nil && stat.IsDir() {
					isDir = true
				}
			}
		}

		if isDir {
			displayName := name + "/"
			if isLink {
				targetDisplay := targetPath
				home, _ := os.UserHomeDir()
				if strings.HasPrefix(targetDisplay, home) {
					targetDisplay = "~" + strings.TrimPrefix(targetDisplay, home)
				}
				displayName = name + "/ -> " + targetDisplay
			}
			dirs = append(dirs, FileItem{
				Name:   displayName,
				Path:   logicalItemPath,
				IsDir:  true,
				IsLink: isLink,
				Target: targetPath,
			})
		} else {
			ext := strings.ToLower(filepath.Ext(name))
			switch ext {
			case ".mp3", ".flac", ".ogg", ".opus", ".m4a", ".wav":
				displayName := name
				if isLink {
					targetDisplay := targetPath
					home, _ := os.UserHomeDir()
					if strings.HasPrefix(targetDisplay, home) {
						targetDisplay = "~" + strings.TrimPrefix(targetDisplay, home)
					}
					displayName = name + " -> " + targetDisplay
				}
				files = append(files, FileItem{
					Name:   displayName,
					Path:   logicalItemPath,
					IsDir:  false,
					IsLink: isLink,
					Target: targetPath,
				})
			}
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name) })
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name) })

	p.items = append(p.items, dirs...)
	p.items = append(p.items, files...)
}

// SelectedItem returns the file or directory currently under the cursor.
func (p *FileBrowserPanel) SelectedItem() (FileItem, bool) {
	if p.cursor >= 0 && p.cursor < len(p.items) {
		return p.items[p.cursor], true
	}
	return FileItem{}, false
}

// Enter navigates into directory or returns false if file (Yazi 'right' / Enter).
func (p *FileBrowserPanel) Enter() bool {
	item, ok := p.SelectedItem()
	if !ok {
		return false
	}
	if item.IsDir {
		// Push current path to history stack
		p.history = append(p.history, p.currentPath)
		p.ReadDir(item.Path)
		return true
	}
	return false
}

// GoParent navigates to the previous directory (Yazi 'left' arrow).
// If a history stack exists, returns to the previous directory where you came from.
// Otherwise, navigates to the logical parent directory (filepath.Dir).
func (p *FileBrowserPanel) GoParent() bool {
	oldPath := p.currentPath
	var targetDir string

	if len(p.history) > 0 {
		// Pop the last visited directory from the history stack
		targetDir = p.history[len(p.history)-1]
		p.history = p.history[:len(p.history)-1]
	} else {
		parent := filepath.Dir(p.currentPath)
		if parent == p.currentPath {
			return false // Already at root
		}
		targetDir = parent
	}

	p.ReadDir(targetDir)

	// Restore cursor onto the directory we just navigated out of
	for i, item := range p.items {
		if item.Path == oldPath || filepath.Clean(item.Target) == filepath.Clean(oldPath) {
			p.cursor = i
			break
		}
	}
	return true
}

func (p *FileBrowserPanel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p *FileBrowserPanel) View() string {
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

	for i := offset; i < len(p.items) && len(lines) < innerHeight; i++ {
		item := p.items[i]
		name := item.Name
		truncated := theme.TruncateLine(name, innerWidth-1)

		rowBaseStyle := lipgloss.NewStyle()
		if i == p.cursor && p.focused {
			rowBaseStyle = theme.CursorRowStyle
		} else if i == p.cursor && !p.focused {
			rowBaseStyle = lipgloss.NewStyle().Bold(true)
		} else if item.IsDir {
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

	for len(lines) < innerHeight {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}

	body := strings.Join(lines[:min(len(lines), innerHeight)], "\n")

	// Shorten path for display
	home, _ := os.UserHomeDir()
	displayDir := p.currentPath
	if strings.HasPrefix(displayDir, home) {
		displayDir = "~" + strings.TrimPrefix(displayDir, home)
	}

	var footer string
	if len(p.items) > 0 {
		footer = fmt.Sprintf("%d/%d", p.cursor+1, len(p.items))
	}
	if p.searchQuery != "" {
		footer = "/" + p.searchQuery + " " + footer
	}

	return theme.Frame("[2] Files: "+displayDir, body, innerWidth, innerHeight, p.focused, footer)
}
