# NewScid: Modern Chess Information Database

A fresh, database-compatible rewrite of **Scid vs. PC** featuring a modern, intuitive user interface built with **Wails v2** (Go backend) and **Vue 3** (TypeScript + Vite + Tailwind CSS).

---

## Highlights & Features

- **Full SCID Database Compatibility**:
  - Native binary support for **.si4** (Index), **.sn4** (NameBase), and **.sg4** (Game Data) databases.
  - Tested against real Scid databases (`tactics`, `matein1`, `matein2`, `endings`).
  - Native front-coding string decoder and custom Huffman/coordinate move-stream decompressor.
  - Native PGN parser and exporter with full support for variations, NAGs (`$1`, `!`, `?`), and comments.

- **Modern & Intuitive Vue 3 Interface**:
  - Interactive chessboard with drag-and-drop piece movement and click-to-move.
  - Legal move hints (subtle destination dots and capture rings).
  - Last-move indicator, check indicator, and automatic pawn promotion selector.
  - Formatted tournament scorebook notation with interactive variations and jump-to-move.
  - Autoplay / replay controls (First, Prev, Play/Pause, Next, Last).

- **Opening Tree Explorer ("Tree Window")**:
  - Dynamically calculates statistics for the current board position across the database.
  - Displays Move, Frequency, Score %, White wins, Draws, and Black wins.
  - Clicking any candidate move immediately plays it and drills deeper into the tree.

- **Real-Time UCI Engine Analysis**:
  - Live communication with UCI chess engines (`/usr/bin/fruit`, `/usr/bin/phalanx`, Stockfish, etc.).
  - Visual advantage evaluation bar with centipawn and mate depth indicators.
  - Live Principal Variation (PV) line display.

- **Piece Sets & Board Themes**:
  - **All Scid vs. PC Sets + Online Vector Sets**: Merida, Colin Burnett (cburnett), Alpha, Leipzig, Spatial, Fantasy, Staunty, Cardinal, Caliente, Dubrovny, Governor, Tatiana, Fresca, Riohacha, Kosal, Shapes, Letter.
  - **Board Themes**: Classic Wood, Tournament Green, Blue-White, Brown, Green-SA, M.Thomas, Grey, Rosy, SteelBlue, Walnut, Slate, Dark Wood.

- **Scid System Driver**:
  - Seamless bridge to `tcscid` on host systems for database verification and legacy batch commands.

---

## Architecture Overview

```
newscid/
├── app.go                      # Wails App Service & Go-Vue3 IPC Bridge
├── main.go                     # Desktop window lifecycle & configuration
├── wails.json                  # Wails configuration & tags
├── pkg/
│   ├── chess/                  # Core chess rules, FEN, SAN, Movegen, PGN
│   │   ├── board.go            # 64-square board, piece lists, castling, undo
│   │   ├── fen.go              # FEN parser & generator
│   │   ├── movegen.go          # Legal move generation & checkmate detection
│   │   ├── pgn.go              # PGN parser, scanner, & exporter
│   │   ├── san.go              # SAN parser & disambiguation formatter
│   │   └── types.go            # Pieces, colors, squares, moves, move tree nodes
│   ├── database/
│   │   ├── scid/               # Native SCID database (.si4, .sn4, .sg4)
│   │   │   ├── database.go     # High-level database manager
│   │   │   ├── gamedata.go     # .sg4 move compression & tag parser
│   │   │   ├── index.go        # .si4 47-byte index record parser/writer
│   │   │   └── names.go        # .sn4 front-coded namebase tree
│   │   └── sciddriver/         # Driver to execute tcscid scripts directly
│   │       └── driver.go
│   ├── eco/                    # Opening classifier (scid.eco compatible)
│   │   └── eco.go
│   ├── engine/                 # UCI chess engine runner & real-time parser
│   │   └── uci.go
│   ├── sets/                   # Piece sets and board themes catalog
│   │   └── sets.go
│   └── tree/                   # Opening Tree statistical aggregator
│       └── tree.go
└── frontend/                   # Vue 3 + TypeScript + Tailwind CSS Frontend
    ├── src/
    │   ├── components/
    │   │   ├── Chessboard.vue  # Interactive board with drag & drop
    │   │   ├── EnginePanel.vue # Live eval bar & PV line display
    │   │   ├── GameBrowser.vue # Paginated game table with search filters
    │   │   ├── HeaderBar.vue   # DB menu, sample bases, & settings modal
    │   │   ├── MoveNotation.vue# Tournament score-sheet & variations tree
    │   │   └── OpeningTree.vue # Opening stats table & clickable moves
    │   ├── utils/pieces.ts     # Piece SVG asset resolver
    │   ├── App.vue             # Main reactive orchestrator
    │   └── style.css           # Tailwind base styles
```

---

## Testing

All packages are fully tested using [`testify`](https://github.com/stretchr/testify):

```bash
# Run all unit tests
go test -v ./...

# Run without cache
go test -count=1 ./...
```

Tests cover:
- Chess moves, legal moves, castling through check, en passant, promotions, checkmate, stalemate.
- Roundtrip FEN and SAN disambiguation.
- SCID `.si4`, `.sn4`, `.sg4` parsing on real databases (`tactics`, `matein1`, `matein2`, `endings`).
- New database creation, appending games, deleting games, and index rewriting.
- `tcscid` direct script execution.
- Opening tree calculation and move frequency aggregation.
- UCI engine process communication with `/usr/bin/fruit`.
- Piece set and theme catalogs.

---

## Dependency Management & Toolchain

The project uses [**mise**](https://mise.jdx.dev) ([`mise.toml`](file:///home/christopher/Projects/newscid/mise.toml)) to manage Go, Node.js, and Wails CLI tool versions:

```bash
# Install toolchain and dependencies
mise install
```

---

## Makefile Automation

A fully documented [`Makefile`](file:///home/christopher/Projects/newscid/Makefile) is provided with an interactive help menu:

```bash
$ make
========================================================================
  NewScid - Modern Chess Information Database (Scid vs. PC Rewrite)
========================================================================
Usage: make <target>

Available Targets:
  help               Display this help menu with all available targets
  deps               Alias for setup (installs all toolchain and library dependencies)
  setup              Setup toolchain via mise, download Go modules, and install npm packages
  dev                Run live development server with hot-reload (Wails Dev)
  build              Build the standalone production binary (build/bin/newscid)
  build-frontend     Compile the Vue 3 + Vite frontend assets
  generate           Regenerate TypeScript bindings from Go App service
  test               Run all Go unit and integration tests with testify
  test-coverage      Run tests with code coverage analysis
  lint               Run static analysis and linting on Go and TypeScript code
  run                Run the built production application binary
  clean              Remove build artifacts, coverage outputs, and temporary files
```

### Common Commands

- **Install all dependencies**:
  ```bash
  make deps
  ```
- **Run test suite**:
  ```bash
  make test
  ```
- **Run tests with coverage profile**:
  ```bash
  make test-coverage
  ```
- **Start development server**:
  ```bash
  make dev
  ```
- **Build production desktop binary**:
  ```bash
  make build
  ```
- **Run production binary**:
  ```bash
  make run
  ```

