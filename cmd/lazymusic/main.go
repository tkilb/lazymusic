package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tylerkilburn/lazymusic/internal/config"
	"github.com/tylerkilburn/lazymusic/internal/player"
	"github.com/tylerkilburn/lazymusic/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	pClient := player.NewClient(cfg.SocketPath)
	if err := pClient.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not connect to mpv daemon (%v). Starting in offline mode.\n", err)
	}

	m := ui.NewModel(cfg, pClient)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running lazymusic: %v\n", err)
		os.Exit(1)
	}
}
