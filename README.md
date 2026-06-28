# Tusic

A lightning-fast, Vim-driven Terminal User Interface (TUI) for streaming Music. 

Tusic strips away the bloat of modern electron apps, giving you a clean, keyboard-centric grid layout.

![Project Screenshot](./ss/demo.png)

> **Disclaimer:** Tusic is a personal educational project built to explore concurrent systems programming, terminal state management, and API bridging in Go. It is not intended for commercial use. Please respect the Terms of Service of your media providers. Do not upload active `cookies.txt` files to public repositories.

## Features

* **Vim-Native Navigation:** Keep your hands on the home row. Navigate panes, scroll lists, and search entirely via standard Vim keybindings (`h`, `j`, `k`, `l`, `/`).
* **Smart "Made For You":** Analyzes your local listening history, calculates your top artists, and automatically curates a YouTube Music radio mix on startup.
* **Local Database:** Fully private, local SQLite database stores your listening history and custom saved playlists.
* **Gapless MPV Playback:** Audio is handled asynchronously via a background `mpv` IPC socket for zero-latency playback.

## Requirements

* **Go 1.21+** (To build the binary)
* **mpv** (Required: Core audio engine)

### System Dependencies
You need the `mpv` media player and 'yt-dlp' installed on your system for the audio engine to work.

## Installation & Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/impossibleclone/tusic-go.git
   cd tusic-go
   ```

2. **Build the program:**
   ```bash
   go mod tidy
   make build
   ```

4. **Run the program:**
   ```bash
   // If built from source
   ./tusic
   // or 
   make run
   ```

## Keybindings

| Key | Action | Context |
| :--- | :--- | :--- |
| `h` | Focus Left (Library Pane) | Normal Mode |
| `l` | Focus Right (Songs Table) | Normal Mode |
| `H` | Focus Left (Search Results) | Up Next Table |
| `L` | Focus Right (Up Next Table) | Search Results |
| `j` | Move Cursor Down | Normal Mode |
| `k` | Move Cursor Up | Normal Mode |
| `o` | Loop Current Song | Global |
| `/` | Focus Search Bar | Normal Mode |
| `Enter` | Play Selected Track / Submit Search | Normal/Search Mode |
| `p` | Play / Pause Audio | Global |
| `s` | Save Song to Local Playlist | Focused on Songs Table |
| `d` | Delete Song from Local Playlist | Focused on Songs Table |
| `r` | Refresh Recommendations | Normal Mode |
| `?` | Toggle Help Menu | Global |
| `Esc` | Unfocus Search / Close Help | Search/Help Mode |
| `<C-c>` | Quit Tusic | Global |


## Workdone
- [x] Grid Layout
- [x] Local SQLite History & Playlists
- [x] Dynamic "Made For You" Curation
- [x] Timestamps can be seen
- [x] Windows for search results and song recommendations to play next in "Up Next".
- [x] Auto-play functionality (play next song from "Up Next" when current finishes)
- [x] Loop Songs
- [ ] going back and forward in song timeline.
