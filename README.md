# NewScid: Complete Linux Chess Workstation & `.si4` Database Suite

**NewScid** is an ultra-fast, native **Linux Chess Workstation** and complete rewrite of **Scid vs. PC**, engineered in **Go** (Wails v2 backend + standalone companion CLI utilities) and **Vue 3** (TypeScript + Vite + Tailwind CSS).

It reads and writes industry-standard **SCID v4 binary databases** (`.si4` index, `.sg4` Huffman-compressed move streams, `.sn4` front-coded namebase tries) capable of searching millions of games in milliseconds, while providing 18+ integrated grandmaster preparation, longitudinal fair-play screening, and cognitive training tools—with **100% local documentation, contextual help (`F1`), and system `man(1)` pages**.

---

## Key Workstation Capabilities

### 1. Customizable Workspaces, Detachable Panels & Persistent Config
- **Every Panel is Relocatable & Detachable**: All workstation panels—including the **Chessboard**, **Scorebook & Move Tree**, **Live UCI Engine & Coach**, **Opening Tree & Tablebase**, **Database Game Browser**, **Evaluation Graph**, and **Database Hub / Live Portal**—can be docked to the Left, Right, or Bottom zones, reordered, hidden, floated as draggable windows, or launched as dedicated multi-screen/Hyprland workspace windows (`--window-panel <id> --workspace <n>`).
- **Automatic Persistent Preferences**: Every workspace layout change, default startup database, reference opening database, board theme, piece set, UI theme, and notation style is automatically saved to `~/.config/newscid/preferences.json`.
- **Small-Screen & Responsive Auto-Fit**: Automatically scales the chessboard so 100% of ranks `1–8` and files `a–h` remain visible with zero clipping on compact laptop screens.

### 2. Native `.si4` Databases, Intuitive Search, Write Locking, Sub-DB Builder & Factory Reset
- **Full SCID Binary Compatibility**: Reads and writes 47-byte `.si4` index records, `.sg4` Huffman move streams, and `.sn4` front-coded namebases with physical compaction, multi-key sorting, and 17-pattern pawn-structure search (IQP, Carlsbad, Maroczy, Hedgehog, Stonewall, Dragon, etc.).
- **Intuitive Position, Famous Players & Opening Search**:
  - **1-Click Board Position Search (`Match Board`)**: Filter any database in milliseconds for games reaching the exact position currently on the chessboard.
  - **Famous & Custom Players Selector**: Instant dropdown in the Game Browser and Search Modal discovering 24+ legends (Bobby Fischer, Magnus Carlsen, Garry Kasparov, Mikhail Tal, Capablanca, Judit Polgar, etc.) across all historical FIDE aliases with live game counts and peak Elo, plus full customization to add your own players of interest.
  - **Opening ECO & Move-Sequence Search**: Filter by ECO code/range (`B90`, `B20-B99`), 1-click opening family chips, or exact initial move sequences (`1.e4 c5 2.Nf3 d6`).
- **Database Write Policies & Contamination Locks**: Every database has a clickable lock policy (`Editable` for personal/persona databases, `Sync-Only` for imported reference databases, or `Locked` read-only) so large master archives cannot be accidentally contaminated by manual edits while still supporting online/TWIC sync.
- **Multi-Database Focused Sub-Database Builder**: Extract matching games across multiple source `.si4` or `.pgn` databases into a focused `.si4` study database by opening line, players, Elo, or year with cross-database deduplication.
- **Managed Workstation Storage & Factory Reset**: Copies databases, UCI engines, Polyglot opening books, and Syzygy endgame tablebases into `~/.local/share/newscid/{databases,personas,engines,books/opening,books/endgame,repertoires,backups}` with 1-click path discovery, and includes a **Factory Reset** tool (with opt-in `.tar.gz` backup) to wipe all content back to empty factory settings.
- **Incremental Mega Database & Online Sync (Lichess & Chess.com)**: Download/update master archives (**LumbrasGigaBase**, **TWIC**) or incrementally sync Lichess & Chess.com accounts (newest-first + historical backfill) without duplicating existing games.

### 3. Longitudinal Fair-Play & Time-Control Consistency Inspector
- **Calibrated Multi-Game Integrity Screening**: Evaluates all games of any database or online player across **Bullet, Blitz, Rapid, and Classical** time controls.
- **Empirical Move-Quality & ACPL**: Computes per-ply centipawn loss, top-move rank distribution, blunder rates, high-tension complexity correlation, and opponent Elo correlation from engine annotations or NewScid's fast static tactical/positional evaluator on unannotated games, calibrated against realistic human baselines.

### 4. Opponent Preparation, Personas & Calibrated AI Sparring
- **Multi-Alias Player Personas**: Unify FIDE OTB names, Lichess handles, and Chess.com accounts into consolidated profiles with dedicated `.si4` databases.
- **Opponent Scouting Dossier & Anti-Persona Opening Exploiter**: Scan an opponent's historical games to uncover high-vulnerability opening nodes and export custom weaponized **Polyglot (`.bin`)** refutation books.
- **AI Player Simulation Bot**: Spar against any database player or Persona using their real historical opening frequencies followed by a rating-calibrated UCI engine with tactical blunder protection. Both the board evaluation bar and the 6-pillar **Positional & Material Imbalance Bar** use a unified absolute White-perspective sign convention (`+` = White advantage, `-` = Black advantage).

### 5. Cognitive Drills, Blunder Vault & Self-Improvement Telemetry
- **4 Board Vision & Calculation Drills**:
  - **Coordinate Speed Trainer**: 60-second square & move-coordinate blitz with an interactive 8×8 per-square accuracy heatmap.
  - **Piece Safety & Undefended Piece Scanner**: Rapidly identify loose and hanging pieces in complex middlegames.
  - **Intuition Evaluation Storm**: Rapid positional judgment under countdown pressure.
  - **Prophylaxis ("What Is Threatening?") Drill**: Detect the opponent's hidden threat and find the calm prophylactic defense.
- **Cyclical Tactical Drill & Lichess Theme Radar**: Builds fixed puzzle sets from your own game history, Personas, and Blunder Vault, sorted strictly from **Easy → Medium → Hard**, and tracks your **History of Improvement** across up to 7 repeated cycles (halving the target completion time each cycle—similar to the cyclical pattern training popularized by *The Woodpecker Method*). Includes a 30+ motif Lichess-compatible tactical classifier, a **9-Axis SVG Theme Radar Chart** (`PerformanceRating = RatingAvg - 500 + 1000 × FirstWins / Attempts`), and 1-click weak-theme cycle generation.
- **Personal Blunder Vault (SM-2 Spaced Repetition)** & **Database Puzzle Miner**: Mine tactical turnarounds from your own games and drill them via SuperMemo-2 scheduling.
- **Time-Trouble & Clock-Management Dossier**: Parses `[%clk]` telemetry to detect "Panic Mode" blunder spikes under 30 seconds and "Premature Instinct" errors.
- **Tactics, Blindfold Ladders, Solitaire Chess (Guess the Move) & Practical Repertoire Shortener (80/20 Pareto Pruner)**.

### 6. 100% Local Manual, Built-In `man(1)` Viewer & Contextual Help
- **Zero External Web Dependencies for Help**: Pressing **`F1`** (or selecting **Help → Local Manual & Contextual Help**) opens the built-in offline workstation manual directly to the topic corresponding to your active dialog or workspace.
- **Built-In System Man Page Viewer**: View the complete `NEWSCID(1)` manual page inside the Help modal (**System Man Page (man 1)** tab) or from any terminal via `man newscid`.

---

## Companion Workstation CLI Programs (`cmd/`)

Following the Unix philosophy and classic Scid's companion tools, NewScid installs four fast binaries:

| Binary | Source | Purpose |
| :--- | :--- | :--- |
| **`newscid`** | [`main.go`](file:///home/christopher/Projects/newscid/main.go) | Full graphical Wails v2 / Vue 3 Chess Workstation & multi-screen panel host |
| **`newscid-import`** | [`cmd/newscid-import/main.go`](file:///home/christopher/Projects/newscid/cmd/newscid-import/main.go) | Standalone high-speed PGN & Online (Lichess/Chess.com) incremental importer, signature deduplicator, and `.si4` compactor |
| **`newscid-fairplay`** | [`cmd/newscid-fairplay/main.go`](file:///home/christopher/Projects/newscid/cmd/newscid-fairplay/main.go) | Standalone longitudinal Fair-Play & Time-Control Consistency analyzer (Markdown & JSON output) |
| **`newscid-scout`** | [`cmd/newscid-scout/main.go`](file:///home/christopher/Projects/newscid/cmd/newscid-scout/main.go) | Standalone Opponent Scouting Dossier, Anti-Persona Opening Exploiter (with Polyglot `.bin` export), and Tournament Crosstable generator |

### CLI Usage Examples

```bash
# 1. Import a PGN archive into a SCID database, skipping existing games, and compact:
newscid-import -db ~/.local/share/newscid/databases/master.si4 -pgn twic1600.pgn -dedup -compact

# 2. Incrementally sync new Lichess & Chess.com games into a database from the terminal:
newscid-import -db ~/.local/share/newscid/databases/mygames.si4 -lichess QuenyenHighElf -chesscom ChristopherFair -max 500

# 3. Run a longitudinal Fair-Play & Time-Control Consistency audit on a player:
newscid-fairplay -db ~/.local/share/newscid/databases/mygames.si4 -player "ChristopherFair"

# 4. Generate an Opponent Scouting Dossier or export a Polyglot (.bin) exploit book:
newscid-scout -db ~/.local/share/newscid/databases/master.si4 -player "Carlsen, Magnus"
newscid-scout -db ~/.local/share/newscid/databases/master.si4 -player "Carlsen, Magnus" -exploit-color white -book-out prep_white.bin

# 5. Generate a FIDE Tournament Crosstable (Round-Robin / Swiss with SB, Buchholz & TPR):
newscid-scout -db ~/.local/share/newscid/databases/master.si4 -crosstable "Tata Steel Masters"
```

---

## Architecture & Performance Design

Hot-path loops across the Go backend use compact integer constants (`uint8` / `int` `iota` enums such as `chess.Color`, `chess.PieceType`, `scid.PawnStructureType`, `fairplay.TimeControlID`, `fairplay.ClassificationID`, `epd.epdOpcode`, and fixed-size array tables) rather than string allocations or string-keyed maps during board generation, tree aggregation, sorting, and statistical analysis.

```
newscid/
├── app.go                        # Wails App Service & Go-Vue3 IPC Bridge
├── main.go                       # Desktop workstation lifecycle & multi-window CLI flags
├── cmd/
│   ├── newscid-import/main.go    # Standalone PGN & online incremental importer CLI
│   ├── newscid-fairplay/main.go  # Standalone longitudinal fair-play analyzer CLI
│   └── newscid-scout/main.go     # Standalone scouting, exploit book & crosstable CLI
├── pkg/
│   ├── blundervault/             # SuperMemo-2 (SM-2) personal blunder vault
│   ├── chess/                    # Core chess rules, 64-square board, FEN, SAN, 6 notation formats, PGN
│   ├── database/
│   │   ├── scid/                 # Native SCID v4 binary engine (.si4, .sg4, .sn4), search, sort, compact
│   │   └── sciddriver/           # Host tcscid batch script bridge
│   ├── eco/                      # ECO opening classifier (A00–E99)
│   ├── engine/                   # UCI engine runner, Multi-PV parser & full-game blunder annotator
│   ├── epd/                      # Extended Position Description (EPD) parser & formatter
│   ├── explainer/                # Natural-Language Position Coach & tactical motif detector
│   ├── fairplay/                 # Longitudinal Fair-Play & Time-Control Consistency Inspector
│   ├── gameexport/               # Print-ready LaTeX, HTML & SVG diagram scorebook publisher
│   ├── imbalance/                # 6-pillar Positional & Material Imbalance evaluator
│   ├── online/                   # Lichess, Chess.com, TWIC & LumbrasGigaBase incremental sync
│   ├── persona/                  # Multi-alias Player Personas, preferences & Anti-Persona Exploiter
│   ├── playerbot/                # Historical repertoire + calibrated UCI opponent simulator
│   ├── polyglot/                 # Binary Polyglot (.bin) Zobrist opening book reader & writer
│   ├── puzzleminer/              # Automated tactical turnaround puzzle extractor
│   ├── repertoire/               # Spaced-repetition opening trainer & 80/20 Pareto shortener
│   ├── scout/                    # Opponent Scouting Dossier generator
│   ├── server/                   # Headless HTTP/WebSocket workstation server
│   ├── sets/                     # Piece sets (17 SVG sets) and board themes (12 themes)
│   ├── stats/                    # Database demographics, Elo tracker & %clk Time Management Dossier
│   ├── tablebase/                # 7-piece Syzygy / Lichess endgame tablebase client
│   ├── tournament/               # FIDE Crosstables (SB/Buchholz/TPR) & UCI Engine Tournament runner
│   ├── training/                 # Solitaire Chess, Blindfold Tactics & 4 Cognitive Board-Vision Drills
│   ├── tree/                     # Opening Tree statistical aggregator
│   ├── version/                  # Semantic version & build metadata
│   └── workstation/              # Companion CLI workstation runner (import, fairplay, scout)
└── frontend/                     # Vue 3 + TypeScript + Tailwind CSS Workstation UI
```

---

## Testing & Coverage (>= 95% Statement Coverage)

Every backend package in `pkg/` maintains **>= 95% statement coverage**, complemented by Vitest component specs for the Vue 3 frontend:

```bash
# Run all Go backend unit tests with coverage verification (>= 95% per package)
go test -count=1 -cover ./pkg/...

# Run all Vue 3 / TypeScript frontend component tests
cd frontend && npm test -- --run
```

---

## Building, Installing & Packaging

```bash
# Build main workstation + all 3 companion CLI binaries (build/bin/newscid*)
make build

# Install to user environment (~/.local/bin and ~/.local/share/man/man1/newscid.1)
make install-user

# Install system-wide (/usr/local or /usr via DESTDIR/PREFIX)
sudo make install PREFIX=/usr

# View the local manual page in the terminal
man newscid
```

### Arch Linux (AUR) & Cross-Platform Packages
NewScid is published on the **Arch User Repository (AUR)** as `newscid` and supports automated packaging via `make pacman`, `make deb`, `make rpm`, `make appimage`, `make flatpak`, and `make snap`.
