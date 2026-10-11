package panels

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tylerkilburn/lazymusic/internal/library"
)

// M3UEntry represents a track item within an Extended M3U playlist file.
type M3UEntry struct {
	Path     string
	Duration int
	Artist   string
	Title    string
}

// ReadM3U parses an M3U or M3U8 file into structured M3UEntry records.
// Handles standard #EXTM3U and #EXTINF lines, and plain path lists.
func ReadM3U(filePath string) ([]M3UEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []M3UEntry
	scanner := bufio.NewScanner(file)
	dir := filepath.Dir(filePath)

	currentDuration := 0
	currentArtist := ""
	currentTitle := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF:") {
			info := strings.TrimPrefix(line, "#EXTINF:")
			parts := strings.SplitN(info, ",", 2)
			if len(parts) > 0 {
				if d, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
					currentDuration = d
				}
			}
			if len(parts) == 2 {
				artistTitle := strings.TrimSpace(parts[1])
				if at := strings.SplitN(artistTitle, " - ", 2); len(at) == 2 {
					currentArtist = strings.TrimSpace(at[0])
					currentTitle = strings.TrimSpace(at[1])
				} else {
					currentTitle = artistTitle
				}
			}
			continue
		}

		if strings.HasPrefix(line, "#") {
			// Skip other directives or comments
			continue
		}

		trackPath := line
		if !filepath.IsAbs(trackPath) {
			trackPath = filepath.Join(dir, trackPath)
		}

		entries = append(entries, M3UEntry{
			Path:     trackPath,
			Duration: currentDuration,
			Artist:   currentArtist,
			Title:    currentTitle,
		})

		currentDuration = 0
		currentArtist = ""
		currentTitle = ""
	}

	return entries, scanner.Err()
}

// WriteM3U writes a list of M3UEntry records to disk in standard Extended M3U format.
func WriteM3U(filePath string, entries []M3UEntry) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, _ = writer.WriteString("#EXTM3U\n")

	for _, entry := range entries {
		label := entry.Title
		if entry.Artist != "" && entry.Title != "" {
			label = fmt.Sprintf("%s - %s", entry.Artist, entry.Title)
		} else if label == "" {
			label = strings.TrimSuffix(filepath.Base(entry.Path), filepath.Ext(entry.Path))
		}

		_, _ = writer.WriteString(fmt.Sprintf("#EXTINF:%d,%s\n", entry.Duration, label))
		_, _ = writer.WriteString(entry.Path + "\n")
	}

	return writer.Flush()
}

// AppendToM3U appends a single track entry to an existing or new M3U file.
func AppendToM3U(filePath string, entry M3UEntry) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	var needsHeader bool
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		needsHeader = true
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	if needsHeader {
		_, _ = writer.WriteString("#EXTM3U\n")
	}

	label := entry.Title
	if entry.Artist != "" && entry.Title != "" {
		label = fmt.Sprintf("%s - %s", entry.Artist, entry.Title)
	} else if label == "" {
		label = strings.TrimSuffix(filepath.Base(entry.Path), filepath.Ext(entry.Path))
	}

	_, _ = writer.WriteString(fmt.Sprintf("#EXTINF:%d,%s\n", entry.Duration, label))
	_, _ = writer.WriteString(entry.Path + "\n")
	return writer.Flush()
}

// HealAndResolveM3U parses an M3U playlist file, checks every track path,
// and automatically heals missing paths against musicDirs and SQLite library cache.
// If any paths were healed, it rewrites the .m3u file with the corrected paths.
// Returns the resolved, valid file paths ready for mpv.
func HealAndResolveM3U(filePath string, musicDirs []string, storage *library.Storage) ([]string, error) {
	entries, err := ReadM3U(filePath)
	if err != nil {
		return nil, err
	}

	var validPaths []string
	healedCount := 0

	for i := range entries {
		entry := &entries[i]

		// Stage 1: Direct file check
		if _, err := os.Stat(entry.Path); err == nil {
			validPaths = append(validPaths, entry.Path)
			continue
		}

		// Stage 2: Suffix check across configured musicDirs
		healed := false
		base := filepath.Base(entry.Path)
		for _, musicDir := range musicDirs {
			candidate := filepath.Join(musicDir, base)
			if _, err := os.Stat(candidate); err == nil {
				entry.Path = candidate
				validPaths = append(validPaths, candidate)
				healed = true
				healedCount++
				break
			}
		}

		if healed {
			continue
		}

		// Stage 3: SQLite metadata healing
		if storage != nil {
			track, err := storage.FindTrackMatch(entry.Title, entry.Artist, entry.Path)
			if err == nil && track != nil && track.Path != "" {
				entry.Path = track.Path
				if entry.Duration == 0 {
					entry.Duration = int(track.Duration)
				}
				if entry.Artist == "" {
					entry.Artist = track.Artist
				}
				if entry.Title == "" {
					entry.Title = track.Title
				}
				validPaths = append(validPaths, track.Path)
				healedCount++
				continue
			}
		}

		// If track genuinely cannot be found on disk, skip from active playback
		// but keep the entry in the M3U so it isn't erased.
	}

	// Stage 4: Rewrite M3U if any tracks were healed to their new locations
	if healedCount > 0 {
		_ = WriteM3U(filePath, entries)
	}

	return validPaths, nil
}

// ReadPlaylistOrder reads ~/.config/lazymusic/playlists/.order manifest.
func ReadPlaylistOrder(playlistsDir string) []string {
	orderFile := filepath.Join(playlistsDir, ".order")
	file, err := os.Open(orderFile)
	if err != nil {
		return nil
	}
	defer file.Close()

	var order []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			order = append(order, line)
		}
	}
	return order
}

// SavePlaylistOrder writes the playlist ordering manifest ~/.config/lazymusic/playlists/.order.
func SavePlaylistOrder(playlistsDir string, order []string) error {
	if err := os.MkdirAll(playlistsDir, 0755); err != nil {
		return err
	}

	orderFile := filepath.Join(playlistsDir, ".order")
	file, err := os.Create(orderFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, name := range order {
		_, _ = writer.WriteString(name + "\n")
	}
	return writer.Flush()
}
