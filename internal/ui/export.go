package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/player"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// ExportNamingMode defines how files are named when exporting.
type ExportNamingMode int

const (
	ExportFlatOriginal ExportNamingMode = iota // 1: Original filename (default), e.g. "Song.mp3"
	ExportNumbered                             // 2: Numbered prefix preserving queue order, e.g. "01 - Song.mp3"
)

// ExportProgressMsg conveys progress during a batch file export.
type ExportProgressMsg struct {
	Total      int
	Copied     int
	Current    string
	Err        error
	Done       bool
	TargetDir  string
}

// ExportField identifies which field in the export form currently has focus.
type ExportField int

const (
	ExportFieldPath ExportField = iota
	ExportFieldFormat
)

// ExportState holds state for the export popup modal.
type ExportState struct {
	focusedField   ExportField
	pathInput      string
	suggestions    []string
	selectedSugIdx int
	namingMode     ExportNamingMode

	// Export execution & progress
	running   bool
	done      bool
	total     int
	copied    int
	current   string
	lastErr   error
	targetDir string
}

func newExportState() ExportState {
	return ExportState{
		focusedField: ExportFieldPath,
		pathInput:   "",
		namingMode:  ExportFlatOriginal,
		suggestions: nil,
	}
}

// updateSuggestions refreshes directory autocomplete suggestions based on current input buffer.
func (e *ExportState) updateSuggestions() {
	raw := strings.TrimSpace(e.pathInput)
	if raw == "" {
		e.suggestions = nil
		e.selectedSugIdx = 0
		return
	}

	expanded := expandHome(raw)
	var parentDir, prefix string

	if strings.HasSuffix(raw, "/") {
		parentDir = expanded
		prefix = ""
	} else {
		parentDir = filepath.Dir(expanded)
		prefix = strings.ToLower(filepath.Base(expanded))
	}

	entries, err := os.ReadDir(parentDir)
	if err != nil {
		e.suggestions = nil
		e.selectedSugIdx = 0
		return
	}

	var matches []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}
		if prefix == "" || strings.Contains(strings.ToLower(name), prefix) {
			full := filepath.Join(parentDir, name)
			// Re-apply tilde representation if user typed ~
			if strings.HasPrefix(raw, "~") {
				if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(full, home) {
					full = "~" + strings.TrimPrefix(full, home)
				}
			}
			matches = append(matches, full)
		}
	}

	e.suggestions = matches
	if e.selectedSugIdx >= len(matches) {
		e.selectedSugIdx = 0
	}
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func (m *Model) startExportCmd(targetDir string, namingMode ExportNamingMode, items []player.PlaylistItem) tea.Cmd {
	return func() tea.Msg {
		expandedDir := expandHome(targetDir)
		if err := os.MkdirAll(expandedDir, 0755); err != nil {
			return ExportProgressMsg{
				Total:  len(items),
				Copied: 0,
				Err:    fmt.Errorf("failed to create directory: %w", err),
				Done:   true,
			}
		}

		usedNames := make(map[string]int)

		for i, item := range items {
			src := item.Filename
			if src == "" {
				continue
			}

			srcBase := filepath.Base(src)
			ext := filepath.Ext(srcBase)
			nameNoExt := strings.TrimSuffix(srcBase, ext)

			var destName string
			if namingMode == ExportNumbered {
				destName = fmt.Sprintf("%02d - %s%s", i+1, nameNoExt, ext)
			} else {
				count := usedNames[strings.ToLower(srcBase)]
				if count > 0 {
					destName = fmt.Sprintf("%s_%d%s", nameNoExt, count, ext)
				} else {
					destName = srcBase
				}
				usedNames[strings.ToLower(srcBase)]++
			}

			destPath := filepath.Join(expandedDir, destName)
			if err := copyFile(src, destPath); err != nil {
				return ExportProgressMsg{
					Total:     len(items),
					Copied:    i,
					Current:   srcBase,
					Err:       err,
					Done:      true,
					TargetDir: expandedDir,
				}
			}
		}

		return ExportProgressMsg{
			Total:     len(items),
			Copied:    len(items),
			Done:      true,
			TargetDir: expandedDir,
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func renderExportModal(m *Model) string {
	if m.exportState.running {
		var body strings.Builder
		body.WriteString(" Exporting queue to:\n")
		body.WriteString(fmt.Sprintf("   %s\n\n", lipgloss.NewStyle().Bold(true).Render(m.exportState.targetDir)))
		body.WriteString(fmt.Sprintf(" Copying files... please wait.\n"))
		return theme.PopupBox("Export Queue Files", "in progress", body.String(), m.width)
	}

	if m.exportState.done {
		var body strings.Builder
		if m.exportState.lastErr != nil {
			body.WriteString(fmt.Sprintf(" Error: %s\n\n", m.exportState.lastErr.Error()))
			return theme.PopupBox("Export Failed", "<esc> or <enter> to dismiss", body.String(), m.width, theme.PaletteRed)
		}
		body.WriteString(fmt.Sprintf(" Successfully exported %d tracks to:\n", m.exportState.copied))
		body.WriteString(fmt.Sprintf(" %s\n", lipgloss.NewStyle().Bold(true).Foreground(theme.PaletteGreen).Render(m.exportState.targetDir)))
		return theme.PopupBox("Export Complete", "<esc> or <enter> to dismiss", body.String(), m.width, theme.PaletteGreen)
	}

	// Field 1: Destination Directory
	cursor := ""
	if m.exportState.focusedField == ExportFieldPath {
		cursor = "█"
	}
	var dirContent strings.Builder
	dirContent.WriteString(" " + m.exportState.pathInput + cursor)

	if len(m.exportState.suggestions) > 0 {
		dirContent.WriteString("\n\n" + lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(" Matching directories (<tab> / <up/down> to select):") + "\n")
		visibleMax := 4
		startIdx := 0
		if m.exportState.selectedSugIdx >= visibleMax {
			startIdx = m.exportState.selectedSugIdx - visibleMax + 1
		}
		endIdx := min(len(m.exportState.suggestions), startIdx+visibleMax)

		for i := startIdx; i < endIdx; i++ {
			sug := m.exportState.suggestions[i]
			itemCursor := "   "
			style := lipgloss.NewStyle()
			if i == m.exportState.selectedSugIdx && m.exportState.focusedField == ExportFieldPath {
				itemCursor = " > "
				style = theme.CursorRowStyle
			}
			dirContent.WriteString(fmt.Sprintf("%s%s\n", itemCursor, style.Render(sug)))
		}
	} else if strings.TrimSpace(m.exportState.pathInput) != "" {
		dirContent.WriteString("\n" + lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(" (Directory will be created if it does not exist)"))
	}

	pathHint := ""
	if m.exportState.focusedField == ExportFieldPath {
		if len(m.exportState.suggestions) > 0 {
			pathHint = "<tab> complete"
		}
	}
	pathBox := theme.FieldBox("Destination Directory", pathHint, dirContent.String(), m.exportState.focusedField == ExportFieldPath, m.width)

	// Field 2: Naming Format
	mode1 := "(•) 1. Flat original: Song.mp3 (Default)"
	mode2 := "( ) 2. Numbered: 01 - Song.mp3"
	if m.exportState.namingMode == ExportNumbered {
		mode1 = "( ) 1. Flat original: Song.mp3 (Default)"
		mode2 = "(•) 2. Numbered: 01 - Song.mp3"
	}

	var formatContent strings.Builder
	if m.exportState.focusedField == ExportFieldFormat {
		if m.exportState.namingMode == ExportFlatOriginal {
			mode1 = theme.CursorRowStyle.Render(mode1)
		} else {
			mode2 = theme.CursorRowStyle.Render(mode2)
		}
	}
	formatContent.WriteString(" " + mode1 + "\n")
	formatContent.WriteString(" " + mode2)

	formatBox := theme.FieldBox("Naming Format", "<tab> next, <enter> export, <esc> cancel", formatContent.String(), m.exportState.focusedField == ExportFieldFormat, m.width)

	return strings.Join([]string{pathBox, formatBox}, "\n")
}
