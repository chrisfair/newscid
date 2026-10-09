<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  HelpCircle,
  X,
  ExternalLink,
  Search,
  BookOpen,
  Database,
  Users,
  Globe,
  Bot,
  Cpu,
  Edit3,
  Award,
  Keyboard,
  ChevronRight,
  Sparkles,
  CheckCircle2
} from '@lucide/vue'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'openShortcuts'): void
}>()

const searchFilter = ref('')
const activeSectionId = ref('getting-started')

interface HelpTopic {
  id: string
  title: string
  icon: any
  summary: string
  tags: string[]
  content: {
    heading: string
    subheading?: string
    paragraphs: string[]
    bullets?: string[]
    tip?: string
  }[]
}

const HELP_TOPICS: HelpTopic[] = [
  {
    id: 'getting-started',
    title: 'Getting Started',
    icon: BookOpen,
    summary: 'Essential navigation, board controls, and core concepts.',
    tags: ['start', 'basics', 'board', 'move', 'trial', 'flip'],
    content: [
      {
        heading: 'Welcome to NewScid',
        subheading: 'Modern, high-performance chess information database and analysis workbench.',
        paragraphs: [
          'NewScid is engineered in pure Go and Vue 3 to provide modern, blisteringly fast database management, tournament analysis, and interactive training.',
          'The application interface consists of the Menu Bar, Interactive Board, Scorebook & Move Tree on the left, and the Database Game Browser on the bottom panel.'
        ],
        bullets: [
          'Play Moves: Click or drag pieces to make moves on the board.',
          'Flip Board: Press F or use Options → Flip Board to switch player perspectives.',
          'Trial Mode: Press Ctrl+Space to enter sandbox mode where you can experiment with variations without altering the active game.',
          'Board Setup / FEN Editor: Press Ctrl+B to design custom board positions, set castling rights, or paste FEN strings.'
        ],
        tip: 'Press Space at any time to start or pause autoplay move replay.'
      }
    ]
  },
  {
    id: 'databases',
    title: 'Databases & PGN',
    icon: Database,
    summary: 'Native .si4 binary databases, PGN imports, compacting, and sorting.',
    tags: ['database', 'si4', 'pgn', 'import', 'export', 'sort', 'compact', 'maintenance'],
    content: [
      {
        heading: 'SCID Binary Format (.si4, .sg4, .sn4)',
        subheading: 'High-speed storage format capable of handling millions of games.',
        paragraphs: [
          'NewScid natively reads and writes the industry-standard SCID format:',
          '• .si4 (Index file): Contains 47-byte records storing metadata, player IDs, ECO, dates, ratings, and file offsets.',
          '• .sg4 (Game file): Stores game moves using compact Huffman compression.',
          '• .sn4 (Namebase file): Efficient front-coded trie storing player, event, site, and annotator names.'
        ]
      },
      {
        heading: 'PGN Import & Export',
        paragraphs: [
          'Import single games or multi-game PGN batches via File → Import PGN Game (Ctrl+I).',
          'Export games to PGN via File → Export PGN Game (Ctrl+E) or copy FEN to clipboard with Ctrl+C.'
        ]
      },
      {
        heading: 'Compacting & Maintenance',
        paragraphs: [
          'Deleted games remain marked until physically compacted.',
          'Open File → Database Maintenance (Ctrl+M) to run physical compaction, sort databases by date/rating/ECO, or run signature-based duplicate detection.'
        ],
        tip: 'Compacting creates automatic safety backups before rewriting database files.'
      }
    ]
  },
  {
    id: 'personas',
    title: 'Player Personas & Aliases',
    icon: Users,
    summary: 'Consolidating multiple usernames into unified profiles with dedicated database sync.',
    tags: ['persona', 'alias', 'merge', 'profile', 'designator', 'stats', 'archive'],
    content: [
      {
        heading: 'Multi-Alias Consolidation',
        subheading: 'Unify all your chess identities into a single master Persona.',
        paragraphs: [
          'Players often appear under different names across platforms (e.g. FIDE OTB format "Fair, Christopher", Chess.com "ChristopherFair", and Lichess "QuenyenHighElf").',
          'Personas allow you to link all aliases under one unified identity to aggregate games, opening repertoires, and statistics.'
        ]
      },
      {
        heading: 'Dedicated Persona Databases (.si4)',
        paragraphs: [
          'Every Persona can have its own dedicated autonomous SCID database (.si4).',
          'Click "Sync from Active Base" in Tools → Player Personas to scan any active database and extract all games matching any of the persona\'s aliases with automated deduplication.'
        ],
        bullets: [
          'Candidate Name Suggestions: NewScid searches the active database and suggests spelling matches.',
          'Consolidated Statistics: View career win/draw/loss rates, peak Elo, and per-alias breakdowns.',
          '1-Click Sparring: Practice against an AI bot simulating the persona\'s unified repertoire.'
        ],
        tip: 'Persona profiles are persistently stored in ~/.config/newscid/personas.json.'
      }
    ]
  },
  {
    id: 'online-sync',
    title: 'Online Account Sync',
    icon: Globe,
    summary: 'Direct integration with Chess.com and Lichess.org public APIs.',
    tags: ['online', 'sync', 'chesscom', 'lichess', 'download', 'profile', 'api'],
    content: [
      {
        heading: 'Direct Platform Integration',
        subheading: 'Import your recent online games without manual PGN downloads.',
        paragraphs: [
          'Access Tools → Online Account Sync to fetch games directly from Chess.com and Lichess.org.',
          'You can configure multiple account profiles and choose where games are routed:'
        ],
        bullets: [
          'Active Database: Merge downloaded games directly into whichever database is currently open.',
          'Dedicated Database: Store games in an isolated, standalone .si4 repository.',
          'Blended Strategy: Maintain both an isolated online archive and an integrated master database.'
        ],
        tip: 'Signature-based deduplication guarantees that games already imported will never create duplicates.'
      }
    ]
  },
  {
    id: 'player-sparring',
    title: 'AI Player Simulation Bot',
    icon: Bot,
    summary: 'Spar against opponents simulating their real opening repertoire and rating.',
    tags: ['spar', 'bot', 'simulation', 'ai', 'grandmaster', 'train', 'opponent'],
    content: [
      {
        heading: 'Historical Repertoire & Strength Modeling',
        subheading: 'Train against grandmasters, club rivals, or yourself.',
        paragraphs: [
          'The Player Simulation module (Play → Simulate Database Player) analyzes all games played by a chosen player across your databases.',
          'It extracts opening book trees for White and Black and plays historical opening moves probabilistically according to their real-world frequency.'
        ]
      },
      {
        heading: 'Calibrated Midgame AI',
        paragraphs: [
          'Once out of their opening book, the bot transitions to a calibrated UCI engine (Stockfish) with Elo and Skill Level dialed in to match the opponent\'s historical rating.'
        ],
        bullets: [
          'Unified Personas Sparring: Switch to the "Unified Personas" tab to spar against consolidated multi-alias profiles.',
          'Time Controls: Play untimed casual games or timed blitz/rapid games (3m, 5m, 10m).',
          'Post-Match Review: Completed sparring matches can be annotated or saved directly to your database.'
        ]
      }
    ]
  },
  {
    id: 'engine-analysis',
    title: 'Engine & Multi-PV Analysis',
    icon: Cpu,
    summary: 'Live evaluation, multi-candidate lines, eval graphs, and automated blunder annotations.',
    tags: ['engine', 'uci', 'stockfish', 'analysis', 'multipv', 'graph', 'eval'],
    content: [
      {
        heading: 'UCI Engine Support',
        subheading: 'Real-time analysis powered by Stockfish and modern UCI engines.',
        paragraphs: [
          'Press F2 or click Tools → Start Live Analysis to begin engine evaluation.',
          'NewScid auto-discovers engines from system paths (/usr/bin, /usr/games) and legacy Scid configurations (~/.scidvspc/engines.options).'
        ]
      },
      {
        heading: 'Multi-PV Candidate Lines',
        paragraphs: [
          'Analyze 1 to 5 candidate moves simultaneously.',
          'Candidate moves are displayed with rank badges, centipawn evaluations, and can be hovered to preview move arrows on the board.'
        ]
      },
      {
        heading: 'Full Game Blunder Annotation',
        paragraphs: [
          'Select Tools → Full Game Analysis to scan an entire game with the engine.',
          'The engine evaluates every move, flags blunders (??), mistakes (?), and inaccuracies (?!), and calculates player accuracy percentages.'
        ]
      }
    ]
  },
  {
    id: 'scorebook-notation',
    title: 'Scorebook & Notation Styles',
    icon: Edit3,
    summary: 'Universal notation switcher, 16-NAG scorebook palette, and variations.',
    tags: ['scorebook', 'notation', 'nag', 'fan', 'descriptive', 'variation', 'comment'],
    content: [
      {
        heading: 'Universal Notation Styles',
        subheading: 'Switch move notation universally across the entire interface on the fly.',
        paragraphs: [
          'Use the Style dropdown in the Move Notation header or Options → Notation Style to select:',
          '• Standard Algebraic (SAN): Standard FIDE notation (Nf3, e4, Bxf7+).',
          '• Figurine Algebraic (FAN): Symbol-based notation using Unicode glyphs (♘f3, ♗xf7+).',
          '• Long Algebraic (LAN): Start and destination coordinates (Ng1-f3, Bf1xf7+).',
          '• Classical Descriptive: Traditional English descriptive notation (N-KB3, P-K4, BxP+).',
          '• Computer Syntax (UCI): Engine coordinate format (g1f3, e2e4, f1f7).',
          '• ICCF Numeric: Telegraphic numeric correspondence notation (7163, 5254).'
        ]
      },
      {
        heading: 'Scorebook Annotation Palette & NAGs',
        paragraphs: [
          'The collapsible annotation toolbar provides 1-click access to 16 FIDE Numeric Annotation Glyphs (NAGs):',
          '• Move Quality: ! (Good), ? (Mistake), !! (Brilliant), ?? (Blunder), !? (Interesting), ?! (Dubious)',
          '• Position Evaluation: = (Equal), ⩲ (White slightly better), ⩱ (Black slightly better), ± (White better), ∓ (Black better), +- (White winning), -+ (Black winning), ∞ (Unclear), ⯹ (With compensation)'
        ]
      },
      {
        heading: 'Variations & Move Commentary',
        paragraphs: [
          'Click the comment button or double-click any move to add strategic notes.',
          'Sub-variations can be promoted to the main line or deleted with one click.'
        ]
      }
    ]
  },
  {
    id: 'training',
    title: 'Training Modules',
    icon: Award,
    summary: 'Tactics trainers, spaced-repetition repertoire builders, and puzzle mining.',
    tags: ['tactics', 'repertoire', 'miner', 'puzzle', 'practice', 'training'],
    content: [
      {
        heading: 'Tactics Puzzle Trainer',
        paragraphs: [
          'Solve curated tactical puzzles from system bases (tactics, matein1, endings) with instant move validation and hint capabilities.'
        ]
      },
      {
        heading: 'Opening Repertoire Builder',
        paragraphs: [
          'Construct White and Black opening trees with spaced-repetition flashcard drills to memorize theoretical lines.'
        ]
      },
      {
        heading: 'Database Puzzle Miner',
        paragraphs: [
          'Scan your own database games with Stockfish to automatically detect major blunder turnarounds and generate personalized tactical puzzles from your own games.'
        ]
      }
    ]
  }
]

const filteredTopics = computed(() => {
  const q = searchFilter.value.toLowerCase().trim()
  if (!q) return HELP_TOPICS
  return HELP_TOPICS.filter(t =>
    t.title.toLowerCase().includes(q) ||
    t.summary.toLowerCase().includes(q) ||
    t.tags.some(tag => tag.includes(q)) ||
    t.content.some(c =>
      c.heading.toLowerCase().includes(q) ||
      c.paragraphs.some(p => p.toLowerCase().includes(q))
    )
  )
})

const activeTopic = computed(() => {
  return HELP_TOPICS.find(t => t.id === activeSectionId.value) || HELP_TOPICS[0]
})

function selectTopic(id: string) {
  activeSectionId.value = id
}

function openOnlineRepo() {
  try {
    BrowserOpenURL('https://github.com/christopherpatrickfair/newscid#readme')
  } catch (err) {
    window.open('https://github.com/christopherpatrickfair/newscid#readme', '_blank')
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="modal-card bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-4xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
      <!-- Modal Header -->
      <div class="flex items-center justify-between p-4 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-xl bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
            <HelpCircle class="w-4 h-4" />
          </div>
          <div>
            <h2 class="text-base font-bold text-zinc-100 flex items-center gap-2">
              <span>NewScid User Guide & Online Help</span>
              <span class="px-2 py-0.5 rounded-full text-[10px] font-mono bg-emerald-950/80 border border-emerald-800/60 text-emerald-300">
                Documentation
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Interactive manual, guides, feature walkthroughs, and keyboard shortcuts.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button
            class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer border border-zinc-700"
            title="Open online GitHub documentation in your web browser"
            @click="openOnlineRepo"
          >
            <ExternalLink class="w-3.5 h-3.5 text-emerald-400" />
            <span>Web Documentation</span>
          </button>
          <button
            class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
            @click="emit('close')"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Main Layout: Sidebar & Content -->
      <div class="flex-1 flex overflow-hidden">
        <!-- Sidebar Navigation -->
        <div class="w-72 bg-zinc-950/60 border-r border-zinc-800 flex flex-col shrink-0">
          <!-- Search Input -->
          <div class="p-3 border-b border-zinc-800/80">
            <div class="relative">
              <Search class="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-zinc-500" />
              <input
                v-model="searchFilter"
                type="text"
                placeholder="Search help topics..."
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
          <div class="space-y-1 border-b border-zinc-800 pb-3">
            <h1 class="text-lg font-bold text-zinc-100 flex items-center gap-2">
              <component :is="activeTopic.icon" class="w-5 h-5 text-emerald-400" />
              <span>{{ activeTopic.title }}</span>
            </h1>
            <p class="text-xs text-zinc-400">{{ activeTopic.summary }}</p>
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

              <div class="space-y-2 text-xs text-zinc-300 leading-relaxed font-sans">
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
    </div>
  </div>
</template>
