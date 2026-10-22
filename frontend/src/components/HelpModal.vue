<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  HelpCircle,
  X,
  Search,
  BookOpen,
  Database,
  Users,
  Globe,
  Bot,
  Cpu,
  Edit3,
  Award,
  Trophy,
  Keyboard,
  Sparkles,
  LayoutGrid,
  ShieldAlert,
  Crosshair,
  Eye,
  Clock,
  Terminal,
  FileText
} from '@lucide/vue'

const props = withDefaults(defineProps<{
  initialTopic?: string
  initialTab?: 'manual' | 'manpage'
}>(), {
  initialTopic: 'getting-started',
  initialTab: 'manual'
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'openShortcuts'): void
}>()

const searchFilter = ref('')
const activeTab = ref<'manual' | 'manpage'>(props.initialTab || 'manual')
const activeSectionId = ref(props.initialTopic || 'getting-started')

watch(() => props.initialTopic, (newTopic) => {
  if (newTopic) {
    activeSectionId.value = newTopic
    activeTab.value = 'manual'
  }
})

watch(() => props.initialTab, (newTab) => {
  if (newTab) {
    activeTab.value = newTab
  }
})

interface HelpTopic {
  id: string
  title: string
  icon: any
  summary: string
  tags: string[]
  content: {
    heading: string
    subheading?: string
    paragraphs?: string[]
    bullets?: string[]
    tip?: string
  }[]
}

const HELP_TOPICS: HelpTopic[] = [
  {
    id: 'getting-started',
    title: 'Getting Started',
    icon: BookOpen,
    summary: 'Essential navigation, board controls, contextual help, and core workstation concepts.',
    tags: ['start', 'basics', 'board', 'move', 'trial', 'flip', 'help', 'f1'],
    content: [
      {
        heading: 'Welcome to the NewScid Chess Workstation',
        subheading: '100% local, ultra-fast chess information database, analysis workbench, and training suite.',
        paragraphs: [
          'NewScid is engineered in pure Go and Vue 3 to provide blisteringly fast native SCID (.si4) database management, grandmaster preparation, longitudinal fair-play inspection, and interactive training.',
          'All documentation, manuals, man pages, and contextual help are stored 100% locally on your machine—no internet connection is ever required to consult the manual.'
        ],
        bullets: [
          'Play Moves: Click or drag pieces on the board, or use the Left/Right arrow keys to step through moves.',
          'Contextual Help (F1): Press F1 at any time to open this local manual directly to the topic matching your currently active dialog or workspace.',
          'Flip Board: Press F or use Options → Flip Board to switch White/Black board orientation.',
          'Trial Mode: Press Ctrl+Space to enter sandbox mode where you can test variations without altering the scorebook.',
          'Board Setup / FEN Editor: Press Ctrl+B to construct custom positions, configure castling rights, or paste FEN strings.'
        ],
        tip: 'Press Ctrl+K anywhere in the workstation to open the Command Palette and launch any tool by name.'
      }
    ]
  },
  {
    id: 'workspaces-panels',
    title: 'Workspaces & Detachable Panels',
    icon: LayoutGrid,
    summary: 'Relocate, dock, float, or hide any panel—including the Chessboard—per workspace.',
    tags: ['workspace', 'panel', 'detach', 'float', 'dock', 'layout', 'preferences', 'config', 'theme'],
    content: [
      {
        heading: 'Customizable Workspaces (Ctrl+1 – Ctrl+4)',
        subheading: 'Switch instantly between Database, Analysis, Training, and Opening Prep layouts.',
        paragraphs: [
          'Every workspace maintains its own independent panel arrangement. You can customize which panels are visible, which dock zone they inhabit, or whether they float as detached windows.'
        ],
        bullets: [
          'Every Panel is Relocatable: All 7 workstation panels—Chessboard, Scorebook & Move Notation, Engine & Coach, Opening Tree & Tablebase, Database Game List, Eval Graph, and Study/Filter—can be moved freely.',
          'Dock Zones (L / R / B): Use the Panels popover in the top header bar to move any panel (including the Chessboard) between the Left, Right, or Bottom dock zones and reorder them with ↑ / ↓.',
          'Detached Floating Windows: Click the Float (↗) button on any panel header or in the Panels manager to pop that panel into a draggable, resizable floating window over the workstation.',
          'Quick Re-Dock: Click "Dock" on any floating window header to snap it right back into its workspace dock slot.'
        ],
        tip: 'All workspace layouts, default startup database, board/piece themes, notation style, and UI preferences are saved automatically to ~/.config/newscid/preferences.json.'
      }
    ]
  },
  {
    id: 'databases',
    title: 'Databases, PGN & Mega Updates',
    icon: Database,
    summary: 'Intuitive Position/Player/Opening search, Famous Players selector, Write Policies, Sub-DB Builder, Managed Storage & Factory Reset.',
    tags: ['database', 'si4', 'pgn', 'search', 'position', 'opening', 'famous', 'fischer', 'carlsen', 'lock', 'editable', 'sync_only', 'locked', 'storage', 'factory', 'reset', 'sub-database', 'mega', 'twic'],
    content: [
      {
        heading: 'Intuitive Search: Board Position, Famous Players & Openings',
        subheading: 'Filter millions of games in milliseconds directly from the Database Game Browser or Search Modal (Ctrl+H).',
        paragraphs: [
          'The Database Game Browser toolbar provides 3 instant search dimensions that can be combined freely:'
        ],
        bullets: [
          'Match Board Position (1-Click): Click "Match Board" in the Game Browser toolbar (or in Search → Position) to filter the database for every game that reached the exact position currently on your chessboard.',
          'Famous & Custom Players Dropdown: Open the "Famous Players" dropdown to immediately filter by 24+ world champions and legends (Bobby Fischer, Magnus Carlsen, Garry Kasparov, Mikhail Tal, Capablanca, Judit Polgar, etc.) across all historical FIDE name aliases, with live game counts and peak Elo in the open database. Click the Customize (+) icon to add any player you find interesting or hide/restore defaults.',
          'Opening ECO & Move Sequence Search: Filter by ECO code/range (e.g. "B90" or "B20-B99"), click 1-click opening family chips (Sicilian, Ruy Lopez, Queen\'s Gambit, King\'s Indian, French, Caro-Kann), or type an opening move sequence (e.g. "1.e4 c5 2.Nf3 d6").'
        ],
        tip: 'Click "Extract Sub-DB" in the Game Browser toolbar to turn your current search filter into a dedicated, focused .si4 database!'
      },
      {
        heading: 'Database Write Policies & Contamination Locks',
        subheading: 'Protect reference archives while keeping personal and persona databases editable.',
        paragraphs: [
          'Every database has a clickable Write Policy badge in the Game Browser, Header Bar, and Database Hub:'
        ],
        bullets: [
          'Editable (Personal / Persona Default): Permits direct game saving, PGN import, header editing, deletion, sorting, and compaction.',
          'Sync-Only (Imported Large / Reference Default): Blocks manual edits or accidental game saves/deletions to prevent contaminating master databases, while still permitting Online Account Sync, TWIC updates, and incremental local database merges.',
          'Locked (Strict Read-Only): Blocks all modifications and sync operations.'
        ]
      },
      {
        heading: 'Multi-Database Sub-Database Builder, Managed Storage & Factory Reset',
        subheading: 'File → Build Focused Sub-Database / Managed Storage Locations / Factory Reset.',
        paragraphs: [
          'Open the Workstation Manager (via the File menu, Database Hub, or Command Palette) to manage three core workstation workflows:'
        ],
        bullets: [
          'Build Focused Sub-Database: Select multiple source .si4 or .pgn databases and extract matching games by Players, Color, ECO range, Opening Move Sequence ("Use Moves on Board"), Minimum Elo, and Year range—with automatic cross-database deduplication.',
          'Managed Storage & Asset Importer: NewScid stores assets in ~/.local/share/newscid/{databases,personas,engines,books/opening,books/endgame,repertoires,backups} and ~/.config/newscid. Copy external .si4 databases, UCI engine binaries, Polyglot (.bin) opening books, or Syzygy (.rtbw/.rtbz) endgame tablebases directly into NewScid\'s managed directories with 1 click.',
          'Factory Reset (Wipe to Empty State with Opt-In Backup): Completely reset all databases, personas, repertoires, training vaults, and saved settings back to FACTORY SETTINGS (empty/nonexistent) so you can start over from scratch. You are warned before wiping and can optionally check the opt-in backup box to save a compressed .tar.gz archive first.'
        ],
        tip: 'Use File → Database Maintenance (Ctrl+M) to physically compact deleted records, sort by Date/Elo/ECO, or set your current database as the automatic startup default.'
      }
    ]
  },
  {
    id: 'personas',
    title: 'Player Personas & Aliases',
    icon: Users,
    summary: 'Unify OTB and online aliases into consolidated profiles with incremental sync.',
    tags: ['persona', 'alias', 'merge', 'profile', 'designator', 'stats', 'archive'],
    content: [
      {
        heading: 'Multi-Alias Consolidation',
        subheading: 'Combine FIDE OTB names, Lichess handles, and Chess.com accounts into one identity.',
        paragraphs: [
          'Players frequently appear under different names across platforms (e.g. OTB "Smith, John", Chess.com "JohnSmith", and Lichess "JohnSmithChess").',
          'Personas link all aliases under one profile to aggregate career statistics, opening repertoires, sparring bots, and dedicated .si4 databases.'
        ],
        bullets: [
          'Incremental + Backfill Online Sync: Syncing a Persona fetches the newest games first since your last sync and automatically backfills older historical archives until the requested count of new, unimported games is reached.',
          'Sync from Active Base: Extract all games matching any alias from a multi-million-game master database into the persona\'s dedicated .si4 base with zero duplicates.',
          '1-Click Sparring & Exploiter: Launch a calibrated sparring bot or run the Anti-Persona Opening Exploiter directly from any persona card.'
        ],
        tip: 'Persona profiles are stored locally in ~/.config/newscid/personas.json.'
      }
    ]
  },
  {
    id: 'online-sync',
    title: 'Online Account Sync',
    icon: Globe,
    summary: 'Incremental newest-first + historical backfill sync from Lichess and Chess.com.',
    tags: ['online', 'sync', 'chesscom', 'lichess', 'download', 'profile', 'api', 'update'],
    content: [
      {
        heading: 'Incremental & Backfill Game Synchronization',
        subheading: 'Update any player or persona without recounting games already in your database.',
        paragraphs: [
          'Access Tools → Online Account Sync to import games from Lichess.org and Chess.com public APIs.',
          'Whenever you sync a player—whether for yourself, a Persona, or an opponent—NewScid inspects the target database first to find your newest and oldest existing game timestamps and builds a signature set of all existing games.'
        ],
        bullets: [
          'Phase 1 (Newest Games): Fetches all recent games played after your latest recorded game timestamp.',
          'Phase 2 (Historical Backfill): If additional games are requested, automatically queries older archives prior to your earliest recorded game without wasting quota on games you already have.',
          'Exact Signature Deduplication: Games already present in the database are skipped and never counted toward your requested new-game limit.'
        ],
        tip: 'You can also sync from the command line using newscid-import -db <base.si4> -lichess <user> -chesscom <user>.'
      }
    ]
  },
  {
    id: 'fairplay',
    title: 'Fair-Play & Integrity Inspector',
    icon: ShieldAlert,
    summary: 'Longitudinal statistical anomaly detection across Bullet, Blitz, Rapid, and Classical.',
    tags: ['fairplay', 'cheat', 'inspector', 'acpl', 'anomaly', 'zscore', 'integrity', 'consistency'],
    content: [
      {
        heading: 'Longitudinal Statistical & Time-Control Consistency Analyzer',
        subheading: 'Multi-game mathematical screening with calibrated human baselines.',
        paragraphs: [
          'Open Tools → Fair-Play & Integrity Inspector to evaluate all games of any player in the active database (or fetch & analyze any Lichess/Chess.com player on the fly).',
          'Single-game accuracy spikes happen naturally in short tactical games. Genuine fair-play analysis requires examining longitudinal metrics across dozens or hundreds of games:'
        ],
        bullets: [
          'Empirical Move-Quality & ACPL: Evaluates every non-book ply (from ply 13 onward) using engine annotations when present or NewScid\'s fast static tactical/positional evaluator on unannotated games.',
          'Time-Control Consistency (Bullet / Blitz / Rapid / Classical): Compares accuracy and blunder rates across time controls. Humans naturally play less accurately in Bullet/Blitz than Rapid/Classical; inverted gaps or near-zero blunder rates in fast time controls are flagged.',
          'Complexity & Elo Correlation: Measures whether move quality remains suspiciously flat in high-tension positions or against much stronger opponents.',
          'Calibrated Human Baselines: Normal club and titled human play scores 0–34 (Normal Human Variance), ensuring honest players are never falsely flagged.'
        ],
        tip: 'Export a complete Markdown dossier for tournament arbiters with one click, or run newscid-fairplay from the terminal.'
      }
    ]
  },
  {
    id: 'scouting-exploiter',
    title: 'Opponent Scouting & Opening Exploiter',
    icon: Crosshair,
    summary: 'Pre-game scouting dossiers, weak opening node detection, and Polyglot .bin export.',
    tags: ['scout', 'opponent', 'exploit', 'anti-persona', 'polyglot', 'book', 'prep', 'vulnerability'],
    content: [
      {
        heading: 'Opponent Scouting Dossier',
        subheading: 'Comprehensive breakdown of an opponent\'s habits, ECO distribution, and phase stamina.',
        paragraphs: [
          'Select Tools → Opponent Scouting Report to scan the active database for all games played by your upcoming opponent.',
          'The report summarizes their White and Black win/draw/loss rates, top ECO openings, game-length distribution (short tactical vs. long endgame grinds), rating performance trend, and tailored preparation recommendations.'
        ]
      },
      {
        heading: 'Anti-Persona Opening Exploiter & Polyglot Book Generator',
        paragraphs: [
          'Select Tools → Anti-Persona Opening Exploiter to build a move-by-move opening tree of a target player or Persona and automatically pinpoint statistical vulnerabilities—positions where the opponent frequently loses or scores poorly.',
          'Click "Export Polyglot (.bin)" to compile those refutation lines into a standard binary Polyglot opening book that you can load into the Opening Tree or use for targeted sparring.'
        ],
        tip: 'Both scouting dossiers and Polyglot exploit books can also be generated headlessly via the newscid-scout CLI binary.'
      }
    ]
  },
  {
    id: 'player-sparring',
    title: 'AI Player Simulation & Sparring',
    icon: Bot,
    summary: 'Spar against calibrated UCI engines or historical database player personas.',
    tags: ['spar', 'bot', 'simulation', 'ai', 'grandmaster', 'train', 'opponent', 'engine'],
    content: [
      {
        heading: 'Simulate Database Player or Unified Persona',
        subheading: 'Play against an opponent\'s actual opening book and rating-calibrated midgame.',
        paragraphs: [
          'Open Play → Simulate Database Player to spar against any player in your database or any configured Unified Persona.',
          'In the opening, the bot selects moves weighted by the player\'s real-world historical frequencies. Once out of book, Stockfish plays at a calibrated Elo and Skill Level matching the opponent, with a tactical safety guard so it avoids unnatural random piece giveaways.'
        ],
        bullets: [
          'Consistent Advantage Indicators: Both the vertical evaluation bar beside the board and the Positional & Material Imbalance bar use an absolute White-perspective sign convention (+ = White advantage, - = Black advantage).',
          'Live Sparring Telemetry: After each bot move, the status banner reports whether the move came from the player\'s historical repertoire or calibrated engine evaluation.'
        ]
      }
    ]
  },
  {
    id: 'engine-analysis',
    title: 'Engine, Coach & Blunder Radar',
    icon: Cpu,
    summary: 'Multi-PV UCI analysis, Natural-Language Position Coach, Imbalance Bar, and Tablebases.',
    tags: ['engine', 'uci', 'stockfish', 'analysis', 'multipv', 'coach', 'imbalance', 'tablebase', 'radar'],
    content: [
      {
        heading: 'UCI Multi-PV Analysis & Full-Game Annotation',
        subheading: 'Real-time evaluation powered by Stockfish and UCI-compatible chess engines.',
        paragraphs: [
          'Press F2 to toggle live engine analysis. Configure 1–5 Multi-PV candidate lines, or open Tools → Full Game Analysis to automatically annotate an entire game with blunders (??), mistakes (?), inaccuracies (?!), and player accuracy percentages.'
        ]
      },
      {
        heading: 'Natural-Language Coach, Imbalance Engine & Blunder Radar',
        bullets: [
          'Natural-Language Position Coach: Explains the best move\'s strategic and tactical motifs (forks, pins, outposts, king safety, passed pawns) in plain English.',
          'Positional & Material Imbalance Bar: Breaks down static evaluation into 6 classical pillars: Material, Pawn Structure, Piece Mobility, King Safety, Center Control, and Space.',
          'Real-Time Blunder Radar: Highlights hanging pieces, undefended targets, and tactical danger squares directly on the board.',
          'Endgame Tablebase Explorer: Queries 7-piece Syzygy/Lichess tablebases for exact WDL (Win/Draw/Loss) and DTZ (Distance to Zeroing) in positions with ≤ 7 pieces.'
        ]
      }
    ]
  },
  {
    id: 'scorebook-notation',
    title: 'Scorebook, Notation & Publishing',
    icon: Edit3,
    summary: '6 live notation formats, 16-NAG palette, variations, and LaTeX/HTML/PDF publishing.',
    tags: ['scorebook', 'notation', 'nag', 'fan', 'descriptive', 'variation', 'comment', 'export', 'latex', 'pdf'],
    content: [
      {
        heading: 'Universal Notation Styles (6 Formats)',
        subheading: 'Switch move notation across the entire workstation on the fly.',
        paragraphs: [
          'Use Options → Notation Style or the Scorebook dropdown to switch between Standard Algebraic (SAN: Nf3), Figurine Algebraic (FAN: ♘f3), Long Algebraic (LAN: Ng1-f3), Classical Descriptive (N-KB3), Computer UCI (g1f3), and ICCF Numeric (7163).'
        ]
      },
      {
        heading: '16-NAG Annotation Palette & Print-Ready Publishing',
        paragraphs: [
          'Annotate moves with 16 standard FIDE Numeric Annotation Glyphs (!, ?, !!, ??, !?, ?!, =, ±, +-, etc.), add text commentary, and promote or delete sub-variations.',
          'Use File → Publish Annotated Game (PDF / LaTeX / HTML) to export publication-grade double-column magazine layouts with embedded vector chessboard diagrams at key moments.'
        ]
      }
    ]
  },
  {
    id: 'training',
    title: 'Tactics, Solitaire Chess & Repertoire',
    icon: Award,
    summary: 'Tactics puzzles, Blindfold ladders, Guess-the-Move, and Spaced-Repetition openings.',
    tags: ['tactics', 'repertoire', 'miner', 'puzzle', 'practice', 'training', 'solitaire', 'blindfold', 'shortener'],
    content: [
      {
        heading: 'Interactive Chess Training Suite',
        subheading: 'Deliberate practice tools built directly into your database workstation.',
        paragraphs: [
          'Access the Play menu to launch structured training modes powered by your databases and UCI engine:'
        ],
        bullets: [
          'Tactics Puzzle Trainer & Progressive Blindfold Tactics: Solve tactical puzzles or train visualization as pieces progressively fade into invisible coordinates across 4 blindfold tiers.',
          'Solitaire Chess (Guess the Move): Step through master games concealing the winner\'s moves; earn points for finding the master move or engine top choices.',
          'Spaced-Repetition Opening Repertoire & Repertoire Shortener: Build White/Black repertoire trees with SM-2 flashcard scheduling, or use the Practical Repertoire Shortener to prune low-probability sidelines while preserving >90% practical coverage.',
          'Database Puzzle Miner: Automatically extract tactical turnaround puzzles from your own database games.'
        ]
      }
    ]
  },
  {
    id: 'cyclical-drill',
    title: 'Cyclical Tactical Drill & Theme Radar',
    icon: Award,
    summary: 'Easy-to-Hard cyclical puzzle loops from your games + Lichess 9-axis theme radar.',
    tags: ['cyclical', 'woodpecker', 'cycle', 'radar', 'themes', 'lichess', 'easy', 'hard', 'improvement', 'tactics'],
    content: [
      {
        heading: 'High-Frequency Cyclical Pattern Repetition (Easy → Hard)',
        subheading: 'Build automatic tactical reflexes from your own game history.',
        paragraphs: [
          'Open Play → Cyclical Tactical Drill & Theme Radar to build a fixed set of tactical puzzles extracted directly from your own database games, Unified Personas, and Blunder Vault.',
          'This module implements cyclical spaced puzzle-set repetition—similar to the training method popularized by GM Axel Smith and IM Hans Tikkanen\'s book "The Woodpecker Method"—where you solve the exact same set of puzzles across up to 7 repeated cycles, aiming to halve your completion time in each cycle (Independent open-source implementation; not affiliated with or endorsed by Quality Chess or Chessable).'
        ],
        bullets: [
          'Easy → Medium → Hard Progression: Every cycle set is automatically sorted by difficulty tier and estimated rating (1000–2500) so you warm up on 1-move patterns and hanging pieces before tackling deep multi-move combinations, sacrifices, and quiet moves.',
          'Multi-Cycle Improvement History: Tracks Total Time, Halved Target Time, Speedup Factor vs. Cycle 1 (e.g., 2.3x faster), First-Try Accuracy %, Avg Seconds/Puzzle, and per-tier (Easy/Medium/Hard) accuracy across every completed cycle.',
          'Lichess 30+ Theme Tagger & 9-Axis Radar Chart: Classifies every puzzle using Lichess\'s open-source tactical motif taxonomy (fork, pin, skewer, discoveredAttack, sacrifice, deflection, intermezzo, quietMove, defensiveMove, backRankMate, smotheredMate, endgame, etc.) and computes per-theme Performance Ratings (RatingAvg - 500 + 1000 × FirstWins / Attempts).',
          'Targeted Theme Loops: Filter any new cycle set by specific Lichess themes or click "Select My Weakest Themes" to drill your lowest-rated tactical motifs.'
        ],
        tip: 'After completing Cycle 1, click "Next Faster Cycle (Halved Time)" to start Cycle 2 with a 50% faster target pace.'
      }
    ]
  },
  {
    id: 'cognitive-drills',
    title: 'Board Vision & Cognitive Drills',
    icon: Eye,
    summary: 'Coordinate speed trainer, Piece Safety scanner, Intuition Storm, and Prophylaxis drills.',
    tags: ['coordinate', 'vision', 'safety', 'undefended', 'intuition', 'prophylaxis', 'threat', 'drill'],
    content: [
      {
        heading: '4 High-Speed Board Vision & Calculation Drills',
        subheading: 'Eliminate one-move blunders and sharpen pattern recognition.',
        paragraphs: [
          'All four cognitive drills are available under the Play menu:'
        ],
        bullets: [
          'Coordinate Speed Trainer: 60-second timed square & move-coordinate blitz with an interactive 8×8 per-square accuracy heatmap.',
          'Piece Safety & Undefended Piece Scanner: Tap every loose (undefended) or hanging piece on complex middlegame boards under time pressure.',
          'Intuition Evaluation Storm: View rich middlegame positions for a few seconds and classify who stands better (White Winning, Slight Edge, Equal, Black Winning) from pure positional judgment.',
          'Prophylaxis ("What Is Threatening?") Drill: Identify the opponent\'s hidden tactical threat if they were given a free move, then play the accurate prophylactic defense.'
        ]
      }
    ]
  },
  {
    id: 'self-improvement',
    title: 'Blunder Vault & Time Management',
    icon: Clock,
    summary: 'Personal SRS Blunder Vault and %clk Time-Trouble & Clock-Management Dossier.',
    tags: ['blunder', 'vault', 'srs', 'clock', 'time', 'trouble', 'management', 'telemetry'],
    content: [
      {
        heading: 'Personal Blunder Vault (Spaced-Repetition)',
        paragraphs: [
          'Open Play → Personal Blunder Vault (SRS) to review mistakes extracted from your own games. Positions are scheduled using the SuperMemo-2 (SM-2) algorithm so recurring tactical blind spots are drilled until mastered.'
        ]
      },
      {
        heading: 'Time-Trouble & Clock-Management Dossier',
        paragraphs: [
          'Open Tools → Time Management Dossier to parse [%clk] annotations across your database games. The analyzer computes your move-time profiles by phase, detects "Panic Mode" blunder spikes under 30 seconds, and flags "Premature Instinct" errors where critical positions were played too quickly.'
        ]
      }
    ]
  },
  {
    id: 'tournaments',
    title: 'Tournaments & Engine Shootouts',
    icon: Trophy,
    summary: 'F7 Round-Robin/Swiss Crosstables, FIDE tiebreaks, and automated Engine Tournaments.',
    tags: ['tournament', 'crosstable', 'round-robin', 'swiss', 'standings', 'sonneborn-berger', 'buchholz', 'tpr'],
    content: [
      {
        heading: 'Tournament Crosstables & Standings (F7)',
        subheading: 'Instant standings, NxN matrices, and FIDE tiebreaks for any database event.',
        paragraphs: [
          'Press F7 or select Tools → Tournament Crosstables & Standings to generate Round-Robin or Swiss crosstables with Sonneborn-Berger (SB), Buchholz, Buchholz Cut-1, and Tournament Performance Rating (TPR). Click any cell to load that game onto the board.'
        ]
      },
      {
        heading: 'Automated UCI Engine Tournament Manager',
        paragraphs: [
          'Select Tools → Engine Tournament Manager to pit multiple UCI engines or configurations against each other in automated Round-Robin or Gauntlet matches with live PGN recording.'
        ]
      }
    ]
  },
  {
    id: 'cli-workstation',
    title: 'Companion Workstation CLI Tools',
    icon: Terminal,
    summary: 'Standalone high-speed binaries: newscid-import, newscid-fairplay, and newscid-scout.',
    tags: ['cli', 'terminal', 'command', 'newscid-import', 'newscid-fairplay', 'newscid-scout', 'headless', 'batch'],
    content: [
      {
        heading: 'Unix Philosophy Chess Workstation Utilities',
        subheading: 'Run heavy batch imports, fair-play audits, and opponent scouting directly from the terminal.',
        paragraphs: [
          'Alongside the main newscid GUI binary, NewScid installs three dedicated, zero-GUI companion command-line programs for ultra-fast batch scripting and headless operation:'
        ],
        bullets: [
          'newscid-import: High-speed PGN and Online (Lichess/Chess.com) incremental importer, signature deduplicator, and physical database compactor. Example: newscid-import -db master.si4 -pgn twic.pgn -dedup -compact',
          'newscid-fairplay: Standalone longitudinal Fair-Play & Time-Control Consistency analyzer outputting Markdown or JSON dossiers. Example: newscid-fairplay -db games.si4 -player "AlphaPlayer"',
          'newscid-scout: Standalone Opponent Scouting Dossier, Anti-Persona Opening Exploiter (with Polyglot .bin export), and Tournament Crosstable generator. Example: newscid-scout -db games.si4 -player "Opponent" -exploit-color white -book-out prep.bin'
        ],
        tip: 'Switch to the "System Man Page (man 1)" tab at the top of this window to read the complete command-line manual.'
      }
    ]
  }
]

const LOCAL_MAN_PAGE = `NEWSCID(1)                   NewScid Chess Manual                   NEWSCID(1)

NAME
       newscid - Modern high-performance chess workstation and .si4 database suite
       newscid-import - Standalone PGN & online incremental database importer
       newscid-fairplay - Standalone longitudinal fair-play & consistency analyzer
       newscid-scout - Standalone opponent scouting, exploit book & crosstable tool

SYNOPSIS
       newscid [options] [file.si4 | file.pgn]
       newscid-import -db <base.si4> [-pgn <file.pgn>] [-lichess <u1>] [-chesscom <u2>] [-dedup] [-compact]
       newscid-fairplay -db <base.si4> -player <Name> [-max 300] [-json]
       newscid-scout -db <base.si4> [-player <Name>] [-exploit-color white|black] [-book-out <out.bin>] [-crosstable <Event>] [-json]

DESCRIPTION
       NewScid is a complete, 100% local Linux chess workstation engineered in
       Go and Vue 3. It reads and writes native SCID v4 binary databases (.si4,
       .sg4, .sn4) capable of searching millions of games in milliseconds, while
       integrating UCI multi-PV analysis, natural-language coaching, longitudinal
       fair-play inspection, opponent preparation, and cognitive training drills.

       All documentation is stored locally via the interactive manual (F1),
       contextual help buttons (?), and this system manual page (man newscid).

WORKSPACES & DETACHABLE PANELS
       Every workspace (Database Ctrl+1, Analysis Ctrl+2, Training Ctrl+3,
       Opening Prep Ctrl+4) supports full panel customization:
       • All 7 panels (including the Chessboard) can be docked Left, Right, or
         Bottom, reordered, hidden, or detached into floating windows.
       • All workspace layouts, startup database, board/piece themes, and
         notation styles persist automatically in ~/.config/newscid/preferences.json.

COMPANION WORKSTATION PROGRAMS
       newscid-import
              Imports PGN archives or incrementally syncs Lichess/Chess.com
              accounts into a target .si4 database using 64-bit game-signature
              hashing so existing games are never duplicated or recounted.

       newscid-fairplay
              Executes longitudinal statistical fair-play screening across
              Bullet, Blitz, Rapid, and Classical time controls, outputting a
              Markdown or JSON integrity dossier.

       newscid-scout
              Generates Opponent Scouting Dossiers, Anti-Persona Opening Exploit
              trees (with binary Polyglot .bin book export), and FIDE Tournament
              Crosstables from the command line.

OPTIONS (newscid)
       --headless
              Run NewScid as a local HTTP/WebSocket server without launching a
              desktop window.

       --addr <host:port>
              Listen address when running in --headless mode (default 127.0.0.1:8088).

       --version
              Display version information and exit.

KEYBOARD SHORTCUTS
       F1            Open local contextual help for the active tool or workspace
       F2            Start / Stop live UCI engine analysis
       F6            Open Classic Book Window
       F7            Open Tournament Crosstables & Standings
       Ctrl+1..4     Switch between Database, Analysis, Training, and Prep workspaces
       Ctrl+K        Open Command Palette
       Ctrl+O / N    Open / Create SCID (.si4) database
       Ctrl+I / E    Import / Export PGN games
       Ctrl+F        Advanced Header, Material & Pattern Search
       Ctrl+M        Database Maintenance, Compaction & Sorting
       Ctrl+B        Board Setup & FEN Editor
       Ctrl+Space    Toggle Trial (Sandbox) Variation Mode
       Space         Toggle Autoplay Move Replay
       F             Flip Board Perspective

FILES
       ~/.config/newscid/preferences.json
              Persistent workstation layout, floating panel geometry, default
              startup database, database write policies, custom famous players,
              and visual theme settings.

       ~/.config/newscid/personas.json
              Multi-alias player persona definitions and sync metadata.

       ~/.local/share/newscid/databases/
              Managed directory for local SCID (.si4/.sg4/.sn4) databases.

       ~/.local/share/newscid/engines/
              Managed directory for copied UCI chess engine binaries.

       ~/.local/share/newscid/books/opening/ & ~/.local/share/newscid/books/endgame/
              Managed directories for Polyglot (.bin) opening books and Syzygy
              (.rtbw/.rtbz) endgame tablebases.

       ~/.local/share/newscid/backups/
              Opt-in compressed (.tar.gz) archives created prior to Factory Reset.

AUTHOR
       Written by Christopher Patrick Fair.
`

const filteredTopics = computed(() => {
  const q = searchFilter.value.toLowerCase().trim()
  if (!q) return HELP_TOPICS
  return HELP_TOPICS.filter(t =>
    t.title.toLowerCase().includes(q) ||
    t.summary.toLowerCase().includes(q) ||
    t.tags.some(tag => tag.includes(q)) ||
    t.content.some(c =>
      c.heading.toLowerCase().includes(q) ||
      (c.paragraphs && c.paragraphs.some(p => p.toLowerCase().includes(q))) ||
      (c.bullets && c.bullets.some(b => b.toLowerCase().includes(q)))
    )
  )
})

const activeTopic = computed(() => {
  return HELP_TOPICS.find(t => t.id === activeSectionId.value) || HELP_TOPICS[0]
})

function selectTopic(id: string) {
  activeSectionId.value = id
  activeTab.value = 'manual'
}
</script>

<template>
  <div class="fixed inset-0 z-[70] bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="modal-card bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-5xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
      <!-- Modal Header -->
      <div class="flex items-center justify-between p-4 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-xl bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
            <HelpCircle class="w-4 h-4" />
          </div>
          <div>
            <h2 class="text-base font-bold text-zinc-100 flex items-center gap-2">
              <span>NewScid Local Workstation Manual & Contextual Help</span>
              <span class="px-2 py-0.5 rounded-full text-[10px] font-mono bg-emerald-950/80 border border-emerald-800/60 text-emerald-300">
                100% Offline Local Docs
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Complete local manual, contextual feature guides, companion CLI reference, and built-in man(1) page.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <div class="flex items-center bg-zinc-900 border border-zinc-800 rounded-lg p-0.5">
            <button
              class="px-3 py-1 rounded-md text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
              :class="activeTab === 'manual' ? 'bg-emerald-600 text-white' : 'text-zinc-400 hover:text-zinc-200'"
              @click="activeTab = 'manual'"
            >
              <BookOpen class="w-3.5 h-3.5" />
              <span>Interactive Manual</span>
            </button>
            <button
              class="px-3 py-1 rounded-md text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
              :class="activeTab === 'manpage' ? 'bg-emerald-600 text-white' : 'text-zinc-400 hover:text-zinc-200'"
              @click="activeTab = 'manpage'"
            >
              <FileText class="w-3.5 h-3.5" />
              <span>System Man Page (man 1)</span>
            </button>
          </div>
          <button
            class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
            @click="emit('close')"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Main Layout: Sidebar & Content -->
      <div v-if="activeTab === 'manual'" class="flex-1 flex overflow-hidden">
        <!-- Sidebar Navigation -->
        <div class="w-72 bg-zinc-950/60 border-r border-zinc-800 flex flex-col shrink-0">
          <!-- Search Input -->
          <div class="p-3 border-b border-zinc-800/80">
            <div class="relative">
              <Search class="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-zinc-500" />
              <input
                v-model="searchFilter"
                type="text"
                placeholder="Search local manual..."
                class="w-full bg-zinc-900 border border-zinc-700/80 rounded-lg pl-8 pr-2.5 py-1.5 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500"
              />
            </div>
          </div>

          <!-- Topics List -->
          <div class="flex-1 overflow-y-auto p-2 space-y-1">
            <button
              v-for="topic in filteredTopics"
              :key="topic.id"
              class="w-full p-2.5 rounded-xl border text-left flex items-start gap-2.5 transition-all cursor-pointer"
              :class="activeSectionId === topic.id
                ? 'bg-emerald-950/50 border-emerald-500/50 shadow-xs'
                : 'bg-zinc-900/60 border-zinc-800/80 hover:bg-zinc-800/60 hover:border-zinc-700'"
              @click="selectTopic(topic.id)"
            >
              <component
                :is="topic.icon"
                class="w-4 h-4 mt-0.5 shrink-0"
                :class="activeSectionId === topic.id ? 'text-emerald-400' : 'text-zinc-400'"
              />
              <div class="min-w-0 flex-1">
                <div class="font-bold text-xs" :class="activeSectionId === topic.id ? 'text-emerald-200' : 'text-zinc-200'">
                  {{ topic.title }}
                </div>
                <div class="text-[10px] text-zinc-400 line-clamp-1 mt-0.5">
                  {{ topic.summary }}
                </div>
              </div>
            </button>

            <div v-if="filteredTopics.length === 0" class="p-6 text-center text-xs text-zinc-500">
              No matching help topics found.
            </div>
          </div>

          <!-- Quick Shortcuts Banner -->
          <div class="p-3 bg-zinc-950 border-t border-zinc-800/80 flex items-center justify-between text-xs">
            <span class="text-zinc-400 text-[11px] flex items-center gap-1.5">
              <Keyboard class="w-3.5 h-3.5 text-zinc-500" />
              <span>Hotkeys List</span>
            </span>
            <button
              class="text-emerald-400 hover:text-emerald-300 font-semibold text-[11px] cursor-pointer"
              @click="emit('openShortcuts')"
            >
              View (Ctrl+?)
            </button>
          </div>
        </div>

        <!-- Topic Details Body -->
        <div class="flex-1 overflow-y-auto p-6 space-y-6 bg-zinc-900/40">
          <div class="space-y-1 border-b border-zinc-800 pb-3 flex items-center justify-between">
            <div>
              <h1 class="text-lg font-bold text-zinc-100 flex items-center gap-2">
                <component :is="activeTopic.icon" class="w-5 h-5 text-emerald-400" />
                <span>{{ activeTopic.title }}</span>
              </h1>
              <p class="text-xs text-zinc-400">{{ activeTopic.summary }}</p>
            </div>
            <span class="px-2.5 py-1 rounded-md bg-zinc-800/90 border border-zinc-700 text-[11px] font-mono text-zinc-300">
              Topic ID: {{ activeTopic.id }}
            </span>
          </div>

          <!-- Content Sections -->
          <div class="space-y-6">
            <div
              v-for="(section, idx) in activeTopic.content"
              :key="idx"
              class="bg-zinc-950/70 border border-zinc-800/90 rounded-xl p-4 space-y-3"
            >
              <div>
                <h3 class="text-sm font-bold text-zinc-200">{{ section.heading }}</h3>
                <p v-if="section.subheading" class="text-xs text-emerald-400/90 mt-0.5">
                  {{ section.subheading }}
                </p>
              </div>

              <div v-if="section.paragraphs && section.paragraphs.length > 0" class="space-y-2 text-xs text-zinc-300 leading-relaxed font-sans">
                <p v-for="(p, pIdx) in section.paragraphs" :key="pIdx">
                  {{ p }}
                </p>
              </div>

              <!-- Bullets -->
              <ul v-if="section.bullets && section.bullets.length > 0" class="space-y-1.5 text-xs text-zinc-300 font-sans pl-2">
                <li v-for="(bullet, bIdx) in section.bullets" :key="bIdx" class="flex items-start gap-2">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 mt-1.5 shrink-0" />
                  <span>{{ bullet }}</span>
                </li>
              </ul>

              <!-- Pro Tip Box -->
              <div
                v-if="section.tip"
                class="mt-3 p-3 rounded-lg bg-emerald-950/40 border border-emerald-800/40 text-xs text-emerald-300 flex items-start gap-2"
              >
                <Sparkles class="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                <span class="leading-relaxed"><strong>ProTip:</strong> {{ section.tip }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Built-in Local System Man Page Viewer -->
      <div v-else class="flex-1 overflow-y-auto p-6 bg-zinc-950">
        <div class="max-w-3xl mx-auto space-y-3">
          <div class="flex items-center justify-between border-b border-zinc-800 pb-3">
            <div class="flex items-center gap-2 text-xs text-emerald-400 font-mono">
              <Terminal class="w-4 h-4" />
              <span>Local System Manual Page — man 1 newscid</span>
            </div>
            <span class="text-[11px] text-zinc-400 font-mono">
              Installed at /usr/share/man/man1/newscid.1.gz
            </span>
          </div>
          <pre class="text-xs font-mono text-zinc-200 leading-relaxed whitespace-pre-wrap bg-zinc-900/90 border border-zinc-800 rounded-xl p-5 select-text">{{ LOCAL_MAN_PAGE }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>
