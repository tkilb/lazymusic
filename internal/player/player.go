package player

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// PlaylistItem represents a track entry in mpv's playlist.
type PlaylistItem struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	Title    string `json:"title"`
	Current  bool   `json:"current"`
	Playing  bool   `json:"playing"`
}

// PlaybackState represents current playback telemetry from mpv.
type PlaybackState struct {
	IsPlaying    bool
	Pause        bool
	MediaTitle   string
	Artist       string
	Album        string
	TimePos      float64
	Duration     float64
	Volume       float64
	AudioCodec   string
	AudioBitrate float64
	Samplerate   int
	IdleActive   bool
	Playlist     []PlaylistItem
	PlaylistPos  int
}

// Client manages the IPC connection to the mpv background daemon.
type Client struct {
	socketPath string
	conn       net.Conn
	reader     *bufio.Reader
	mu         sync.Mutex
	reqID      uint64
	callbacks  map[uint64]chan *mpvResponse
	events     chan PlaybackState
	state      PlaybackState
	stateMu    sync.RWMutex
	closed     bool
}

type mpvRequest struct {
	Command   []interface{} `json:"command"`
	RequestID uint64        `json:"request_id,omitempty"`
}

type mpvResponse struct {
	Error     string      `json:"error"`
	Data      interface{} `json:"data"`
	RequestID uint64      `json:"request_id"`
	Event     string      `json:"event"`
	Name      string      `json:"name"`
}

// NewClient initializes a client for the given socket path.
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		callbacks:  make(map[uint64]chan *mpvResponse),
		events:     make(chan PlaybackState, 32),
		state: PlaybackState{
			Volume: 100,
			Pause:  false,
		},
	}
}

// Connect attempts to connect to an existing mpv daemon, or starts one if not running.
func (c *Client) Connect() error {
	conn, err := net.Dial("unix", c.socketPath)
	if err == nil {
		c.conn = conn
		c.reader = bufio.NewReader(conn)
		go c.readLoop()
		c.observeProperties()
		c.syncInitialState()
		return nil
	}

	// Remove dead socket if present
	_ = os.Remove(c.socketPath)

	// Launch background daemon detached
	cmd := exec.Command("mpv",
		"--idle=yes",
		"--no-video",
		"--no-terminal",
		fmt.Sprintf("--input-ipc-server=%s", c.socketPath),
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mpv daemon: %w", err)
	}

	// Wait up to 3 seconds for socket to become available
	var connected bool
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		conn, err := net.Dial("unix", c.socketPath)
		if err == nil {
			c.conn = conn
			c.reader = bufio.NewReader(conn)
			connected = true
			break
		}
	}

	if !connected {
		return errors.New("timed out waiting for mpv IPC socket")
	}

	go c.readLoop()
	c.observeProperties()
	c.syncInitialState()
	return nil
}

// Events returns a receive-only channel of playback updates.
func (c *Client) Events() <-chan PlaybackState {
	return c.events
}

// GetState returns current cached playback state.
func (c *Client) GetState() PlaybackState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

func (c *Client) sendCommand(args ...interface{}) (*mpvResponse, error) {
	c.mu.Lock()
	if c.conn == nil || c.closed {
		c.mu.Unlock()
		return nil, errors.New("mpv client not connected")
	}

	id := atomic.AddUint64(&c.reqID, 1)
	req := mpvRequest{
		Command:   args,
		RequestID: id,
	}

	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}

	respChan := make(chan *mpvResponse, 1)
	c.callbacks[id] = respChan
	c.mu.Unlock()

	data = append(data, '\n')
	if _, err := c.conn.Write(data); err != nil {
		c.mu.Lock()
		delete(c.callbacks, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(2 * time.Second):
		c.mu.Lock()
		delete(c.callbacks, id)
		c.mu.Unlock()
		return nil, errors.New("command response timed out")
	}
}

func (c *Client) readLoop() {
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return
		}

		var resp mpvResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		if resp.RequestID != 0 {
			c.mu.Lock()
			ch, ok := c.callbacks[resp.RequestID]
			if ok {
				delete(c.callbacks, resp.RequestID)
				c.mu.Unlock()
				ch <- &resp
			} else {
				c.mu.Unlock()
			}
		} else if resp.Event == "property-change" {
			c.handlePropertyChange(resp.Name, resp.Data)
		}
	}
}

func (c *Client) handlePropertyChange(name string, val interface{}) {
	c.stateMu.Lock()
	changed := false

	switch name {
	case "pause":
		if b, ok := val.(bool); ok {
			c.state.Pause = b
			c.state.IsPlaying = !b && !c.state.IdleActive
			changed = true
		}
	case "idle-active":
		if b, ok := val.(bool); ok {
			c.state.IdleActive = b
			c.state.IsPlaying = !c.state.Pause && !b
			changed = true
		}
	case "time-pos":
		if f, ok := val.(float64); ok {
			c.state.TimePos = f
			changed = true
		}
	case "duration":
		if f, ok := val.(float64); ok {
			c.state.Duration = f
			changed = true
		}
	case "volume":
		if f, ok := val.(float64); ok {
			c.state.Volume = f
			changed = true
		}
	case "media-title":
		if s, ok := val.(string); ok {
			c.state.MediaTitle = s
			changed = true
		}
	case "metadata":
		if meta, ok := val.(map[string]interface{}); ok {
			if title, ok := meta["title"].(string); ok {
				c.state.MediaTitle = title
			}
			if artist, ok := meta["artist"].(string); ok {
				c.state.Artist = artist
			}
			if album, ok := meta["album"].(string); ok {
				c.state.Album = album
			}
			changed = true
		}
	case "audio-codec-name":
		if s, ok := val.(string); ok {
			c.state.AudioCodec = s
			changed = true
		}
	case "playlist-pos":
		if f, ok := val.(float64); ok {
			c.state.PlaylistPos = int(f)
			changed = true
		}
	case "playlist":
		if list, ok := val.([]interface{}); ok {
			var items []PlaylistItem
			for _, elem := range list {
				if itemMap, ok := elem.(map[string]interface{}); ok {
					var it PlaylistItem
					if id, ok := itemMap["id"].(float64); ok {
						it.ID = int(id)
					}
					if fn, ok := itemMap["filename"].(string); ok {
						it.Filename = fn
					}
					if title, ok := itemMap["title"].(string); ok {
						it.Title = title
					}
					if cur, ok := itemMap["current"].(bool); ok {
						it.Current = cur
					}
					if pl, ok := itemMap["playing"].(bool); ok {
						it.Playing = pl
					}
					items = append(items, it)
				}
			}
			c.state.Playlist = items
			changed = true
		}
	}

	snapshot := c.state
	c.stateMu.Unlock()

	if changed {
		select {
		case c.events <- snapshot:
		default:
		}
	}
}

func (c *Client) observeProperties() {
	props := []string{
		"pause",
		"idle-active",
		"time-pos",
		"duration",
		"volume",
		"media-title",
		"metadata",
		"audio-codec-name",
		"playlist",
		"playlist-pos",
	}
	for i, prop := range props {
		_, _ = c.sendCommand("observe_property", i+1, prop)
	}
}

func (c *Client) syncInitialState() {
	go func() {
		props := []string{
			"pause",
			"idle-active",
			"time-pos",
			"duration",
			"volume",
			"media-title",
			"metadata",
			"audio-codec-name",
			"playlist",
			"playlist-pos",
		}
		for _, prop := range props {
			resp, err := c.sendCommand("get_property", prop)
			if err == nil && resp.Error == "success" {
				c.handlePropertyChange(prop, resp.Data)
			}
		}
	}()
}

// TogglePause toggles playback pause state.
func (c *Client) TogglePause() error {
	_, err := c.sendCommand("cycle", "pause")
	return err
}

// Unpause ensures playback is unpaused.
func (c *Client) Unpause() error {
	_, err := c.sendCommand("set_property", "pause", false)
	return err
}

// Next plays the next track.
func (c *Client) Next() error {
	_, err := c.sendCommand("playlist-next")
	return err
}

// Prev plays the previous track.
func (c *Client) Prev() error {
	_, err := c.sendCommand("playlist-prev")
	return err
}

// SeekRelative seeks relative seconds forward or backward.
func (c *Client) SeekRelative(seconds float64) error {
	_, err := c.sendCommand("seek", seconds, "relative")
	return err
}

// VolumeUp increases volume by 5%.
func (c *Client) VolumeUp() error {
	_, err := c.sendCommand("add", "volume", 5)
	return err
}

// VolumeDown decreases volume by 5%.
func (c *Client) VolumeDown() error {
	_, err := c.sendCommand("add", "volume", -5)
	return err
}

// LoadFile loads an audio file (mode: "replace" or "append").
func (c *Client) LoadFile(path, mode string) error {
	_, err := c.sendCommand("loadfile", path, mode)
	return err
}

// PlayIndex switches playback to a specific index in mpv's playlist.
func (c *Client) PlayIndex(index int) error {
	_, err := c.sendCommand("set_property", "playlist-pos", index)
	return err
}

// RemoveFromPlaylist removes the track at index from mpv's playlist.
func (c *Client) RemoveFromPlaylist(index int) error {
	_, err := c.sendCommand("playlist-remove", index)
	return err
}

// MoveInPlaylist moves the track at oldIndex to newIndex in mpv's playlist.
func (c *Client) MoveInPlaylist(oldIndex, newIndex int) error {
	_, err := c.sendCommand("playlist-move", oldIndex, newIndex)
	return err
}

// ClearPlaylist clears all tracks from mpv's playlist.
func (c *Client) ClearPlaylist() error {
	_, err := c.sendCommand("playlist-clear")
	return err
}

// ShufflePlaylist shuffles mpv's playlist.
func (c *Client) ShufflePlaylist() error {
	_, err := c.sendCommand("playlist-shuffle")
	return err
}

// Detach gracefully closes the IPC connection leaving the daemon running.
func (c *Client) Detach() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// QuitDaemon sends the quit command to terminate mpv completely.
func (c *Client) QuitDaemon() error {
	_, _ = c.sendCommand("quit")
	return c.Detach()
}
