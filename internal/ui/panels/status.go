package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tylerkilburn/lazymusic/internal/ui/theme"
)

// StatusPanel implements the bottom Now Playing and status bar.
type StatusPanel struct {
	width     int
	height    int
	isPlaying bool
	title     string
	artist    string
	album     string
	year      string
	format    string
	currTime  string
	totalTime string
	percent   float64
	volume    int
}

// NewStatusPanel creates a new StatusPanel instance.
func NewStatusPanel() *StatusPanel {
	return &StatusPanel{
		isPlaying: true,
		title:     "Paranoid Android",
		artist:    "Radiohead",
		album:     "OK Computer",
		year:      "1997",
		format:    "FLAC 24-bit 96kHz",
		currTime:  "03:12",
		totalTime: "06:23",
		percent:   0.50,
		volume:    85,
	}
}

func (s *StatusPanel) SetSize(w, h int) {
	s.width = w
	s.height = h
}

func (s *StatusPanel) Update(msg tea.Msg) (*StatusPanel, tea.Cmd) {
	return s, nil
}

func (s *StatusPanel) View() string {
	innerWidth := s.width - 2
	innerHeight := s.height - 2
	if innerWidth < 0 {
		innerWidth = 0
	}
	if innerHeight < 0 {
		innerHeight = 0
	}

	stateIcon := "▶ Playing"
	if !s.isPlaying {
		stateIcon = "⏸ Paused"
	}

	// Line 1: [State] Title • Artist • Album (Year)
	trackInfo := fmt.Sprintf("[%s] %s • %s • %s (%s)",
		theme.StatusLabelStyle.Render(stateIcon),
		theme.CursorItemStyle.Render(s.title),
		s.artist,
		s.album,
		s.year,
	)

	// Line 2: Progress bar + format
	// 03:12 ━━━━━●━━━━━ 06:23   FLAC 24-bit 96kHz
	timePrefix := fmt.Sprintf("%s ", s.currTime)
	timeSuffix := fmt.Sprintf(" %s   %s", s.totalTime, theme.KeybindDescStyle.Render(s.format))
	barWidth := innerWidth - len(timePrefix) - len(timeSuffix)
	if barWidth < 10 {
		barWidth = 10
	}

	progressPos := int(s.percent * float64(barWidth))
	if progressPos >= barWidth {
		progressPos = barWidth - 1
	}
	if progressPos < 0 {
		progressPos = 0
	}

	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		if i == progressPos {
			bar.WriteString("●")
		} else {
			bar.WriteString("━")
		}
	}
	progressLine := fmt.Sprintf("%s%s%s", timePrefix, bar.String(), timeSuffix)

	// Line 3: Controls and keybind cheatsheet
	controls := fmt.Sprintf("Vol: %d%%  [Repeat: Off] [Shuffle: Off]  %s %s %s %s %s",
		s.volume,
		renderKeybind("Space", "Pause"),
		renderKeybind(">", "Next"),
		renderKeybind("<", "Prev"),
		renderKeybind("?", "Help"),
		renderKeybind("q", "Detach"),
	)

	lines := []string{trackInfo, progressLine, controls}
	content := strings.Join(lines[:min(len(lines), innerHeight)], "\n")

	return theme.StatusBarStyle.Width(innerWidth).Height(innerHeight).Render(content)
}

func renderKeybind(key, desc string) string {
	return fmt.Sprintf("[%s: %s]", theme.KeybindKeyStyle.Render(key), theme.KeybindDescStyle.Render(desc))
}
