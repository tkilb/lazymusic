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
	Name  string
	Path  string
	IsDir bool
}

// FileBrowserPanel implements Panel for Panel 3 (File Browser).
type FileBrowserPanel struct {
	focused    bool
	width      int
	height     int
	cursor     int
	currentDir string
	items      []FileItem
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
		cursor:     0,
		currentDir: initialDir,
	}
	p.ReadDir(initialDir)
	return p
}

func (p *FileBrowserPanel) ID() PanelID {
	return PanelFileBrowser
}

func (p *FileBrowserPanel) Title() string {
	return "3: File Browser"
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

// ReadDir loads directory contents, placing directories first, followed by audio files.
func (p *FileBrowserPanel) ReadDir(dir string) {
	p.currentDir = dir
	p.cursor = 0
	p.items = nil

	// Add parent directory navigation if not at root
	parent := filepath.Dir(dir)
	if parent != dir {
		p.items = append(p.items, FileItem{
			Name:  "../",
			Path:  parent,
			IsDir: true,
		})
	}

	entries, err := os.ReadDir(dir)
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

		fullPath := filepath.Join(dir, name)
		if entry.IsDir() {
			dirs = append(dirs, FileItem{
				Name:  name + "/",
				Path:  fullPath,
				IsDir: true,
			})
		} else {
			ext := strings.ToLower(filepath.Ext(name))
			switch ext {
			case ".mp3", ".flac", ".ogg", ".opus", ".m4a", ".wav":
				files = append(files, FileItem{
					Name:  name,
					Path:  fullPath,
					IsDir: false,
				})
			}
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

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

// Enter navigates into directory or returns false if file.
func (p *FileBrowserPanel) Enter() bool {
	item, ok := p.SelectedItem()
	if !ok {
		return false
	}
	if item.IsDir {
		p.ReadDir(item.Path)
		return true
	}
	return false
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

	offset := 0
	if p.cursor >= innerHeight {
		offset = p.cursor - innerHeight + 1
	}

	for i := offset; i < len(p.items) && len(lines) < innerHeight; i++ {
		item := p.items[i]
		name := item.Name
		lineStr := fmt.Sprintf(" %-*s", innerWidth-1, name)
		if len(lineStr) > innerWidth {
			lineStr = lineStr[:innerWidth]
		}

		if i == p.cursor && p.focused {
			lines = append(lines, theme.CursorRowStyle.Render(lineStr))
		} else if i == p.cursor && !p.focused {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render(lineStr))
		} else if item.IsDir {
			lines = append(lines, lipgloss.NewStyle().Bold(true).Render(lineStr))
		} else {
			lines = append(lines, lineStr)
		}
	}

	for len(lines) < innerHeight {
		lines = append(lines, strings.Repeat(" ", innerWidth))
	}

	body := strings.Join(lines[:min(len(lines), innerHeight)], "\n")

	// Shorten path for display
	home, _ := os.UserHomeDir()
	displayDir := p.currentDir
	if strings.HasPrefix(displayDir, home) {
		displayDir = "~" + strings.TrimPrefix(displayDir, home)
	}

	var footer string
	if len(p.items) > 0 {
		footer = fmt.Sprintf("%d/%d", p.cursor+1, len(p.items))
	}

	return theme.Frame("[3] Files: "+displayDir, body, innerWidth, innerHeight, p.focused, footer)
}
