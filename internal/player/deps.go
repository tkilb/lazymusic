package player

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CheckMediaDependencies checks whether mpv and its mpris plugin are installed.
// Per lazymusic specs:
// 1. Checks for mpv first. If missing, returns "mpv not installed".
// 2. If mpv exists, checks for mpv-mpris. If missing on Linux, returns "mpv-mpris not installed".
// 3. If both exist (or on platforms where mpris does not apply), returns "".
func CheckMediaDependencies() string {
	if _, err := exec.LookPath("mpv"); err != nil {
		return "mpv not installed"
	}

	if runtime.GOOS == "linux" && !hasMpvMpris() {
		return "mpv-mpris not installed"
	}

	return ""
}

func hasMpvMpris() bool {
	paths := []string{
		"/etc/mpv/scripts/mpris.so",
		"/etc/mpv/scripts/mpv-mpris.so",
		"/usr/lib/mpv/mpris.so",
		"/usr/lib/mpv/mpv-mpris.so",
		"/usr/lib64/mpv/mpris.so",
		"/usr/lib64/mpv/mpv-mpris.so",
		"/usr/local/lib/mpv/mpris.so",
		"/usr/lib/mpv-mpris/mpris.so",
		"/usr/lib/x86_64-linux-gnu/mpv/mpris.so",
		"/usr/lib/aarch64-linux-gnu/mpv/mpris.so",
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "mpv", "scripts", "mpris.so"),
			filepath.Join(home, ".config", "mpv", "scripts", "mpv-mpris.so"),
		)
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths,
			filepath.Join(xdg, "mpv", "scripts", "mpris.so"),
			filepath.Join(xdg, "mpv", "scripts", "mpv-mpris.so"),
		)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
