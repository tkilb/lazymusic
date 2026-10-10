package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Config holds application configuration settings and filepaths.
type Config struct {
	ConfigDir    string   `json:"-"`
	ConfigFile   string   `json:"-"`
	PlaylistsDir string   `json:"-"`
	SocketPath   string   `json:"-"`
	DBPath       string   `json:"-"`
	MusicDirs    []string `json:"music_dirs"`
}

// Load loads the configuration from ~/.config/lazymusic/config.json, creating defaults if not found.
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

	configFile := filepath.Join(configDir, "config.json")
	socketPath := filepath.Join(configDir, "mpv.sock")
	dbPath := filepath.Join(configDir, "library.db")

	defaultMusicDir := filepath.Join(homeDir, "Music")

	cfg := &Config{
		ConfigDir:    configDir,
		ConfigFile:   configFile,
		PlaylistsDir: playlistsDir,
		SocketPath:   socketPath,
		DBPath:       dbPath,
		MusicDirs:    []string{defaultMusicDir},
	}

	if data, err := os.ReadFile(configFile); err == nil {
		_ = json.Unmarshal(data, cfg)
	} else {
		_ = cfg.Save()
	}

	return cfg, nil
}

// Save writes the current configuration to config.json.
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.ConfigFile, data, 0644)
}

// AddMusicDir adds a directory if not already present, normalizing paths.
func (c *Config) AddMusicDir(dir string) bool {
	dir = ExpandHome(dir)
	dir = filepath.Clean(dir)

	for _, existing := range c.MusicDirs {
		if filepath.Clean(existing) == dir {
			return false
		}
	}
	c.MusicDirs = append(c.MusicDirs, dir)
	_ = c.Save()
	return true
}

// RemoveMusicDir removes a directory from the configuration.
func (c *Config) RemoveMusicDir(dir string) bool {
	dir = ExpandHome(dir)
	dir = filepath.Clean(dir)

	var updated []string
	found := false
	for _, existing := range c.MusicDirs {
		if filepath.Clean(existing) == dir {
			found = true
			continue
		}
		updated = append(updated, existing)
	}
	if found {
		c.MusicDirs = updated
		_ = c.Save()
	}
	return found
}

// ExpandHome resolves "~" to user's home directory.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}
