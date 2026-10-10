package config

import (
	"os"
	"path/filepath"
)

// Config holds application configuration settings and filepaths.
type Config struct {
	ConfigDir    string
	PlaylistsDir string
	SocketPath   string
	DBPath       string
	MusicDirs    []string
}

// Load loads the default configuration, creating config directories if necessary.
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".config", "lazymusic")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	playlistsDir := filepath.Join(configDir, "playlists")
	if err := os.MkdirAll(playlistsDir, 0755); err != nil {
		return nil, err
	}

	socketPath := filepath.Join(configDir, "mpv.sock")
	dbPath := filepath.Join(configDir, "library.db")

	defaultMusicDir := filepath.Join(homeDir, "Music")

	return &Config{
		ConfigDir:    configDir,
		PlaylistsDir: playlistsDir,
		SocketPath:   socketPath,
		DBPath:       dbPath,
		MusicDirs:    []string{defaultMusicDir},
	}, nil
}
