package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tylerkilburn/lazymusic/internal/player"
	"github.com/tylerkilburn/lazymusic/internal/ui/statusbar"
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
	currTime     string
	totalTime    string
	percent      float64
	volume       int
	loopFile     bool
	loopPlaylist bool
	bindings     []statusbar.Binding
	searchPrompt string
}

// NewStatusPanel creates a new StatusPanel instance.
func NewStatusPanel() *StatusPanel {
	return &StatusPanel{
		isPlaying: false,
		title:     "No track loaded",
		artist:    "-",
		album:     "-",
		year:      "-",
		format:    "-",
		currTime:  "00:00",
		totalTime: "00:00",
		percent:   0.0,
		volume:    100,
	}
}

func (s *StatusPanel) SetKeybindings(b []statusbar.Binding) {
	s.bindings = b
}

func (s *StatusPanel) SetSearchPrompt(prompt string) {
	s.searchPrompt = prompt
}

func (s *StatusPanel) SetPlaybackState(state player.PlaybackState) {
	s.isPlaying = state.IsPlaying
	if state.MediaTitle != "" {
		s.title = state.MediaTitle
	} else if state.IdleActive {
		s.title = "Idle (Queue empty)"
	}
	if state.Artist != "" {
		s.artist = state.Artist
	}
	if state.Album != "" {
		s.album = state.Album
	}
	if state.AudioCodec != "" {
		s.format = strings.ToUpper(state.AudioCodec)
	}
	s.currTime = formatDuration(state.TimePos)
	s.totalTime = formatDuration(state.Duration)
	if state.Duration > 0 {
		s.percent = state.TimePos / state.Duration
	} else {
		s.percent = 0.0
	}
	s.volume = int(state.Volume)
	s.loopFile = state.LoopFile
	s.loopPlaylist = state.LoopPlaylist
}

func formatDuration(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	m := int(seconds) / 60
	sec := int(seconds) % 60
	return fmt.Sprintf("%02d:%02d", m, sec)
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

	loopTag := ""
	if s.loopFile {
		loopTag = " " + lipgloss.NewStyle().Bold(true).Foreground(theme.PaletteCyan).Render("[Loop: Song]")
	} else if s.loopPlaylist {
		loopTag = " " + lipgloss.NewStyle().Bold(true).Foreground(theme.PaletteCyan).Render("[Loop: Queue]")
	}

	// Line 1: [State] [Loop] Title • Artist • Album (Year)
	trackInfo := fmt.Sprintf(" [%s]%s %s • %s • %s (%s)",
		lipgloss.NewStyle().Bold(true).Foreground(theme.PaletteGreen).Render(stateIcon),
		loopTag,
		lipgloss.NewStyle().Bold(true).Render(s.title),
		s.artist,
		s.album,
		s.year,
	)

	// Line 2: Progress bar + format
	timePrefix := fmt.Sprintf(" %s ", s.currTime)
	timeSuffix := fmt.Sprintf(" %s   %s", s.totalTime, lipgloss.NewStyle().Foreground(theme.PaletteDim).Render(s.format))
	barWidth := innerWidth - len(timePrefix) - len(timeSuffix) - 1
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

	// Line 3: lazygit-style context-aware keybindings or active search prompt
	controls := " " + statusbar.Render(s.bindings)
	if s.searchPrompt != "" {
		controls = " " + lipgloss.NewStyle().Bold(true).Foreground(theme.PaletteGreen).Render(s.searchPrompt)
	}

	lines := []string{trackInfo, progressLine, controls}
	body := strings.Join(lines[:min(len(lines), innerHeight)], "\n")

	return theme.Frame("Status & Controls", body, innerWidth, innerHeight, false)
}
