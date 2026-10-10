package library

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	id3v2 "github.com/bogem/id3v2/v2"
	"github.com/dhowden/tag"
	_ "github.com/mattn/go-sqlite3"
)

// Track represents an audio track metadata entry.
type Track struct {
	ID        int64
	Path      string
	Title     string
	Artist    string
	Album     string
	Genre     string
	Year      int
	TrackNum  int
	Duration  float64
	Format    string
	ModTime   int64
	CreatedAt time.Time
}

// Storage manages the SQLite library cache.
type Storage struct {
	db *sql.DB
	mu sync.RWMutex
}

// Open initializes the SQLite database and executes migrations.
func Open(dbPath string) (*Storage, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Storage) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		artist TEXT NOT NULL,
		album TEXT NOT NULL,
		genre TEXT,
		year INTEGER,
		track_num INTEGER,
		duration REAL DEFAULT 0,
		format TEXT,
		mod_time INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_tracks_artist ON tracks(artist);
	CREATE INDEX IF NOT EXISTS idx_tracks_album ON tracks(album);
	CREATE INDEX IF NOT EXISTS idx_tracks_path ON tracks(path);
	`
	_, err := s.db.Exec(query)
	return err
}

// UpsertTrack inserts or updates a track in the cache.
func (s *Storage) UpsertTrack(t *Track) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	INSERT INTO tracks (path, title, artist, album, genre, year, track_num, duration, format, mod_time)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(path) DO UPDATE SET
		title=excluded.title,
		artist=excluded.artist,
		album=excluded.album,
		genre=excluded.genre,
		year=excluded.year,
		track_num=excluded.track_num,
		duration=excluded.duration,
		format=excluded.format,
		mod_time=excluded.mod_time;
	`
	_, err := s.db.Exec(query, t.Path, t.Title, t.Artist, t.Album, t.Genre, t.Year, t.TrackNum, t.Duration, t.Format, t.ModTime)
	return err
}

// UpdateTrackMetadata writes ID3v2 tags directly into the audio file (if MP3) and updates SQLite.
func (s *Storage) UpdateTrackMetadata(path, title, artist, album string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".mp3" {
		tagFile, err := id3v2.Open(path, id3v2.Options{Parse: true})
		if err == nil {
			tagFile.SetTitle(title)
			tagFile.SetArtist(artist)
			tagFile.SetAlbum(album)
			_ = tagFile.Save()
			_ = tagFile.Close()
		}
	}

	info, _ := os.Stat(path)
	var modTime int64
	if info != nil {
		modTime = info.ModTime().Unix()
	} else {
		modTime = time.Now().Unix()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	query := `UPDATE tracks SET title = ?, artist = ?, album = ?, mod_time = ? WHERE path = ?`
	_, err := s.db.Exec(query, title, artist, album, modTime, path)
	return err
}

// DeleteTracksInDirectory removes cached tracks located under a directory.
func (s *Storage) DeleteTracksInDirectory(dir string) error {
	dir = filepath.Clean(dir)
	pattern := dir + "/%"
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM tracks WHERE path LIKE ? OR path = ?", pattern, dir)
	return err
}

// GetTrackModTime retrieves the cached modification timestamp for a path.
func (s *Storage) GetTrackModTime(path string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var modTime int64
	err := s.db.QueryRow("SELECT mod_time FROM tracks WHERE path = ?", path).Scan(&modTime)
	if err != nil {
		return 0, false
	}
	return modTime, true
}

// GetArtists retrieves distinct artist names sorted alphabetically.
func (s *Storage) GetArtists() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT DISTINCT artist FROM tracks WHERE artist != '' ORDER BY artist COLLATE NOCASE ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err == nil {
			artists = append(artists, a)
		}
	}
	return artists, nil
}

// GetAlbums retrieves distinct album names sorted alphabetically.
func (s *Storage) GetAlbums() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT DISTINCT album FROM tracks WHERE album != '' ORDER BY album COLLATE NOCASE ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err == nil {
			albums = append(albums, a)
		}
	}
	return albums, nil
}

// GetTracks retrieves tracks sorted by track number and title.
func (s *Storage) GetTracks() ([]Track, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT id, path, title, artist, album, genre, year, track_num, duration, format, mod_time FROM tracks ORDER BY artist COLLATE NOCASE, album COLLATE NOCASE, track_num, title COLLATE NOCASE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var t Track
		var genre, format sql.NullString
		err := rows.Scan(&t.ID, &t.Path, &t.Title, &t.Artist, &t.Album, &genre, &t.Year, &t.TrackNum, &t.Duration, &format, &t.ModTime)
		if err == nil {
			t.Genre = genre.String
			t.Format = format.String
			tracks = append(tracks, t)
		}
	}
	return tracks, nil
}

// GetTracksByArtist retrieves tracks by a specific artist.
func (s *Storage) GetTracksByArtist(artist string) ([]Track, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT id, path, title, artist, album, genre, year, track_num, duration, format, mod_time FROM tracks WHERE artist = ? ORDER BY album COLLATE NOCASE, track_num, title COLLATE NOCASE", artist)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var t Track
		var genre, format sql.NullString
		if err := rows.Scan(&t.ID, &t.Path, &t.Title, &t.Artist, &t.Album, &genre, &t.Year, &t.TrackNum, &t.Duration, &format, &t.ModTime); err == nil {
			t.Genre = genre.String
			t.Format = format.String
			tracks = append(tracks, t)
		}
	}
	return tracks, nil
}

// GetTracksByAlbum retrieves tracks for a specific album.
func (s *Storage) GetTracksByAlbum(album string) ([]Track, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT id, path, title, artist, album, genre, year, track_num, duration, format, mod_time FROM tracks WHERE album = ? ORDER BY track_num, title COLLATE NOCASE", album)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var t Track
		var genre, format sql.NullString
		if err := rows.Scan(&t.ID, &t.Path, &t.Title, &t.Artist, &t.Album, &genre, &t.Year, &t.TrackNum, &t.Duration, &format, &t.ModTime); err == nil {
			t.Genre = genre.String
			t.Format = format.String
			tracks = append(tracks, t)
		}
	}
	return tracks, nil
}

// SupportedExtensions lists recognized audio formats.
var SupportedExtensions = map[string]bool{
	".mp3":  true,
	".flac": true,
	".ogg":  true,
	".opus": true,
	".m4a":  true,
	".wav":  true,
}

// ScanDirectory walks a directory recursively, parsing tags and caching to SQLite.
func (s *Storage) ScanDirectory(rootDir string) (int, error) {
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return 0, nil
	}

	count := 0
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !SupportedExtensions[ext] {
			return nil
		}

		modTime := info.ModTime().Unix()
		cachedModTime, exists := s.GetTrackModTime(path)
		if exists && cachedModTime == modTime {
			count++
			return nil // Skip up-to-date tracks
		}

		// Read audio metadata tags
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		metadata, err := tag.ReadFrom(file)
		t := &Track{
			Path:    path,
			ModTime: modTime,
			Format:  strings.TrimPrefix(ext, "."),
		}

		if err != nil {
			// Fallback to filename if metadata cannot be parsed
			t.Title = strings.TrimSuffix(filepath.Base(path), ext)
			t.Artist = "Unknown Artist"
			t.Album = "Unknown Album"
		} else {
			t.Title = metadata.Title()
			if t.Title == "" {
				t.Title = strings.TrimSuffix(filepath.Base(path), ext)
			}
			t.Artist = metadata.Artist()
			if t.Artist == "" {
				t.Artist = "Unknown Artist"
			}
			t.Album = metadata.Album()
			if t.Album == "" {
				t.Album = "Unknown Album"
			}
			t.Genre = metadata.Genre()
			t.Year = metadata.Year()
			trackNum, _ := metadata.Track()
			t.TrackNum = trackNum
		}

		if err := s.UpsertTrack(t); err == nil {
			count++
		}
		return nil
	})

	return count, err
}
