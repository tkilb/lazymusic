# AGENTS.md - Instructions for AI Pair Programmers

Welcome to the **lazymusic** project! This document outlines the project guidelines, architecture, coding conventions, and developer workflows for all AI coding agents working on this repository.

---

## 1. Project Overview & Philosophy

**lazymusic** is a `lazygit`-inspired terminal user interface (TUI) for local, offline music management and playback.
- **Language**: Go (`>= 1.21`).
- **TUI Framework**: Charm ecosystem (`bubbletea`, `lipgloss`, `bubbles`).
- **Audio Engine**: Headless `mpv` background daemon communicating via JSON-RPC over a local Unix domain socket (`~/.config/lazymusic/mpv.sock`).
- **Library Storage**: Local SQLite database (`~/.config/lazymusic/library.db`) caching audio tags (ID3, Vorbis, FLAC).
- **Core Design Pillars**:
  - Offline-first: No cloud or streaming services.
  - Persistent playback: Quitting `lazymusic` leaves audio playing via `mpv` in the background; restarting reconnects seamlessly.
  - Ergonomics: Fast keyboard navigation using arrow keys (`↑`/`↓`), tabs (`[`/`]`), panel jump keys (`0`, `1`, `2`, `3`), and dynamic accordion pane expansion. `h/j/k/l` keys are strictly freed for custom operations.

---

## 2. Repository Layout

```
.
├── cmd/
│   └── lazymusic/         # Main entry point (main.go)
├── internal/
│   ├── config/            # User settings, paths, defaults
│   ├── ui/                # Bubble Tea models, views, and styles
│   │   ├── panels/        # Panel implementations (library, playlists, queue, status)
│   │   └── theme/         # Lip Gloss styles, colors, border definitions
│   ├── player/            # MPV IPC client, background daemon manager
│   └── library/           # Audio tag scanner, directory walker, SQLite caching
├── PLAN.md                # Comprehensive architecture & development roadmap
└── AGENTS.md              # Instructions & standards for AI coding agents
```

---

## 3. Coding Standards & Conventions

### Go Guidelines
- **Idiomatic Go**: Write clean, standard Go code with explicit error handling.
- **No Panics**: Never call `panic()` in library or background code; always return clean errors or Bubble Tea messages (`tea.Msg`).
- **Concurrency Safety**: Synchronize concurrent access to the queue, SQLite connection, and `mpv` socket using proper channels or `sync.Mutex`/`sync.RWMutex`.
- **Testing Philosophy**: Skip writing automated unit tests and mocks to preserve tokens and velocity; rely strictly on clean compilation checks (`go build`) and fast interactive human QA in the terminal.

### Shell & Commands
- Adhere strictly to the human-in-the-loop safety rules:
  - Do not run mutating git commands (`git commit`, `git push`) automatically; the user commits manually.
  - Do not invoke destructive external CLI tools without confirmation.
  - Prefer scoped agent tools (`view_file`, `replace_file_content`, `write_to_file`) over raw shell commands (`cat`, `sed`, `grep`).
  - When suggesting terminal utilities, prefer `fd` over `find`, `rg` over `grep`, and `fzf` for fuzzy filtering.

---

## 4. Architectural Rules

1. **Decoupled Audio Daemon**:
   - The UI must never directly control audio devices; it always speaks to `mpv` through the IPC socket.
   - If `lazymusic` crashes or exits, `mpv` must not crash or terminate abruptly unless explicitly commanded (`Q`).
2. **Dynamic UI Accordion**:
   - The left column panels (`1: Library`, `2: Playlists`, `3: File Browser`) must dynamically adjust height based on which panel is focused (~60% active, ~20% inactive).
   - Panel `0: Queue` occupies the right column.
   - Status & Now Playing bar is consolidated at the bottom.
3. **Keybinding Invariants**:
   - Navigation uses arrow keys (`↑`/`↓`), NOT `h/j/k/l`.
   - Sub-tab cycling uses `[` and `]`.
   - Panel cycling uses `<Tab>` / `<S-Tab>`.
