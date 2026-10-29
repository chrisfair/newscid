# NewScid

NewScid is a Linux chess database and analysis program written in Go (Wails v2) and Vue 3. It reads and writes native Scid v4 binary databases (`.si4`, `.sg4`, `.sn4`) directly without requiring Tcl/Tk, alongside PGN and Zstandard-compressed PGN archives.

## Overview

Scid vs. PC is fast and battle-tested, but its Tcl/Tk interface struggles on modern Wayland compositors, HiDPI displays, and tiling window managers. NewScid implements the Scid v4 binary format in Go (47-byte index records in `.si4`, Huffman-encoded move trees in `.sg4`, and front-coded namebase tries in `.sn4`) and pairs it with a dockable, multi-window UI and standalone command-line tools.

### Database & Search

- Read/write support for `.si4`/`.sg4`/`.sn4` databases, `.pgn`, and `.pgn.zst` streams.
- Position search (exact board match), material/pattern search, header filters, ECO range filters, and move-sequence filtering.
- Per-database write policies (`Editable`, `Sync-Only`, `Locked`) to avoid accidentally writing test variations or casual games into reference databases.
- Sub-database extraction across multiple source databases with 64-bit game-signature deduplication and physical `.si4`/`.sg4` compaction.
- Player alias consolidation ("Personas") to group FIDE OTB names, Lichess handles, and Chess.com accounts into a single profile and `.si4` database.
- Incremental online sync from Lichess and Chess.com (fetches games newer than the latest recorded timestamp first, then backfills older archives while skipping existing game signatures), plus TWIC and Lichess monthly archive imports.

### Analysis, Preparation & Training

- Multi-PV UCI engine analysis, full-game annotation, evaluation graphs, and 7-piece Syzygy/Lichess endgame tablebase probing.
- Opponent scouting reports and opening tree vulnerability scanning with Polyglot (`.bin`) opening book export.
- Longitudinal statistical consistency screening across Bullet, Blitz, Rapid, and Classical time controls (ACPL, move-rank distribution, blunder rate, and move-time correlation).
- Sparring against historical player repertoires from the open database, transitioning to a rating-adjusted UCI engine once out of book.
- Spaced-repetition (SM-2) opening repertoire trainer, repertoire line pruning by frequency, and personal blunder review.
- Tactical and board-vision drills: cyclical puzzle repetition extracted from your own games with Lichess motif classification, solitaire chess (guess the move), progressive blindfold tactics, coordinate speed drills, undefended piece scanning, and `[%clk]` time-management analysis.
- Six live notation formats (`SAN`, `FAN`, `LAN`, `Descriptive`, `UCI`, `ICCF`), 16 FIDE NAGs, and game export to PGN, HTML, LaTeX, and PDF.

### Workspaces & Window Management

All seven panels (Board, Scorebook, Engine, Opening Tree, Game Browser, Eval Graph, and Study/Filter) can be docked to the left, right, or bottom zones, floated as sub-windows, or launched as separate OS windows (`--window-panel <id>`) for multi-monitor and tiling WM setups such as Hyprland or Sway. Layouts and settings are stored in `~/.config/newscid/preferences.json`. Built-in documentation is accessible offline via `F1` or `man 1 newscid`.

---

## Companion CLI Tools

Three command-line utilities are built alongside the desktop binary for terminal scripting and headless batch jobs:

```bash
# Import a PGN archive into a Scid database, deduplicate, and compact deleted records
newscid-import -db ~/.local/share/newscid/databases/master.si4 -pgn twic.pgn -dedup -compact

# Incrementally sync new Lichess and Chess.com games into a personal database
newscid-import -db ~/.local/share/newscid/databases/my_games.si4 -lichess my_user -chesscom my_user -max 250

# Run multi-game time-control and move-quality screening for a player
newscid-fairplay -db ~/.local/share/newscid/databases/master.si4 -player "PlayerName" -max 300

# Generate an opponent scouting report and export a Polyglot (.bin) refutation book
newscid-scout -db ~/.local/share/newscid/databases/master.si4 -player "OpponentName" -exploit-color white -book-out prep.bin

# Print a tournament crosstable with Sonneborn-Berger, Buchholz, and TPR
newscid-scout -db ~/.local/share/newscid/databases/master.si4 -crosstable "Tata Steel Masters"
```

---

## Installation

### Arch Linux (AUR)

```bash
yay -S newscid
```

### Building from Source

Requires Go 1.24+, Node.js 20+, GTK3, and WebKit2GTK (`webkit2gtk-4.1` on Arch/Fedora, `libwebkit2gtk-4.1-dev` on Debian/Ubuntu).

```bash
cd frontend && npm install && npm run build && cd ..
go build -tags webkit2_41 -o build/bin/newscid .
go build -o build/bin/newscid-import ./cmd/newscid-import
go build -o build/bin/newscid-fairplay ./cmd/newscid-fairplay
go build -o build/bin/newscid-scout ./cmd/newscid-scout
```

To build a local Arch package:

```bash
./packaging/scripts/build-pacman.sh
sudo pacman -U build/dist/newscid-*-x86_64.pkg.tar.*
```

Packaging scripts for `.deb`, `.rpm`, AppImage, Flatpak, and Snap are in `packaging/scripts/`.

---

## Testing

```bash
go test -v -race ./pkg/...
cd frontend && npm test -- --run
```

---

## Keyboard Shortcuts

| Key | Action |
| :--- | :--- |
| `F1` | Open local manual & contextual help |
| `F2` | Toggle UCI engine analysis |
| `F3` | Opening Tree & Polyglot book explorer |
| `F6` | ECO opening browser |
| `F7` | Tournament crosstables & standings |
| `Ctrl+1` .. `Ctrl+4` | Switch workspace (Database, Analysis, Training, Prep) |
| `Ctrl+K` | Command palette |
| `Ctrl+,` | Preferences & multi-screen panel setup |
| `Ctrl+O` / `Ctrl+N` / `Ctrl+S` | Open database / New game / Save game |
| `Ctrl+H` | Header, position, material & pattern search |
| `Ctrl+M` | Database maintenance, compaction & sorting |
| `Ctrl+B` | Board setup / FEN editor |
| `Ctrl+Space` | Toggle trial variation mode |
| `F` / `Space` / `Z` | Flip board / Toggle autoplay / Zen mode |

---

## License

GPL-3.0-or-later. See [LICENSE](LICENSE).
