import { ref, computed } from 'vue'

export type PanelId =
  | 'board'
  | 'notation'
  | 'tree'
  | 'info'
  | 'engine'
  | 'imbalance'
  | 'games'
  | 'db-roles'
  | 'portal'

export type PanelMode = 'docked' | 'floater' | 'detached' | 'hidden'
export type DockZone = 'left' | 'right' | 'bottom'

export interface FloaterRect {
  x: number
  y: number
  width: number
  height: number
  minimized: boolean
  zIndex: number
}

export interface PanelConfig {
  id: PanelId
  title: string
  shortTitle: string
  description: string
  icon: string
  mode: PanelMode
  dockZone: DockZone
  defaultDockZone: DockZone
  targetWorkspace: string
  floaterRect: FloaterRect
}

export type LayoutPresetId = 'standard' | 'floaters' | 'hyprland-multiscreen' | 'minimal' | 'compact'

const STORAGE_KEY = 'newscid:workspacePanels:v2'

let onPanelsChangedHook: (() => void) | null = null

export function setOnPanelsChangedHook(fn: (() => void) | null) {
  onPanelsChangedHook = fn
}

function createDefaultPanels(): Record<PanelId, PanelConfig> {
  return {
    board: {
      id: 'board',
      title: 'Interactive Chessboard & Eval Bar',
      shortTitle: 'Chessboard',
      description: 'Primary interactive chessboard, coordinates, evaluation bar, and move arrows',
      icon: 'grid',
      mode: 'docked',
      dockZone: 'left',
      defaultDockZone: 'left',
      targetWorkspace: '',
      floaterRect: { x: 40, y: 70, width: 520, height: 540, minimized: false, zIndex: 30 },
    },
    notation: {
      id: 'notation',
      title: 'Move Notation & Eval Graph',
      shortTitle: 'Notation',
      description: 'Interactive move list, variations, NAGs, comments, and evaluation sparkline',
      icon: 'file-text',
      mode: 'docked',
      dockZone: 'right',
      defaultDockZone: 'right',
      targetWorkspace: '',
      floaterRect: { x: 620, y: 90, width: 460, height: 500, minimized: false, zIndex: 31 },
    },
    tree: {
      id: 'tree',
      title: 'Opening Tree & Theory Book',
      shortTitle: 'Opening Tree',
      description: 'Candidate moves, win/draw/loss percentages, Reference DB & Polyglot book stats',
      icon: 'book-open',
      mode: 'docked',
      dockZone: 'right',
      defaultDockZone: 'right',
      targetWorkspace: '',
      floaterRect: { x: 640, y: 110, width: 520, height: 480, minimized: false, zIndex: 32 },
    },
    'db-roles': {
      id: 'db-roles',
      title: 'Database Hub & Relationships',
      shortTitle: 'Databases',
      description: 'Visual map and switcher for Default DB, Active Working DB, Reference DB, and Clipbase',
      icon: 'database',
      mode: 'docked',
      dockZone: 'right',
      defaultDockZone: 'right',
      targetWorkspace: '',
      floaterRect: { x: 580, y: 120, width: 560, height: 480, minimized: false, zIndex: 33 },
    },
    portal: {
      id: 'portal',
      title: 'Live Portal (Lichess.org & Chess.com)',
      shortTitle: 'Lichess / Chess.com',
      description: 'Direct window into Lichess.org & Chess.com analysis, live TV, puzzles, and native screen launcher',
      icon: 'globe',
      mode: 'docked',
      dockZone: 'right',
      defaultDockZone: 'right',
      targetWorkspace: '',
      floaterRect: { x: 600, y: 100, width: 580, height: 520, minimized: false, zIndex: 34 },
    },
    info: {
      id: 'info',
      title: 'Game Metadata & Headers',
      shortTitle: 'Game Info',
      description: 'Players, Elo ratings, event, site, date, ECO code, and PGN tags',
      icon: 'info',
      mode: 'docked',
      dockZone: 'right',
      defaultDockZone: 'right',
      targetWorkspace: '',
      floaterRect: { x: 660, y: 140, width: 420, height: 380, minimized: false, zIndex: 35 },
    },
    engine: {
      id: 'engine',
      title: 'UCI Engine & Tablebases',
      shortTitle: 'Engine',
      description: 'Live Stockfish/UCI MultiPV evaluation lines, depth, nodes/sec, and 7-piece Syzygy tablebases',
      icon: 'cpu',
      mode: 'docked',
      dockZone: 'left',
      defaultDockZone: 'left',
      targetWorkspace: '',
      floaterRect: { x: 80, y: 520, width: 500, height: 250, minimized: false, zIndex: 36 },
    },
    imbalance: {
      id: 'imbalance',
      title: 'Material & Positional Imbalance',
      shortTitle: 'Imbalance',
      description: 'Material difference, bishop pair, pawn structure, king safety, and mobility breakdown',
      icon: 'bar-chart-3',
      mode: 'docked',
      dockZone: 'left',
      defaultDockZone: 'left',
      targetWorkspace: '',
      floaterRect: { x: 80, y: 420, width: 480, height: 190, minimized: false, zIndex: 37 },
    },
    games: {
      id: 'games',
      title: 'Database Games Browser',
      shortTitle: 'Games List',
      description: 'Paginated game list, quick search filter, and one-click game loading',
      icon: 'list',
      mode: 'docked',
      dockZone: 'bottom',
      defaultDockZone: 'bottom',
      targetWorkspace: '',
      floaterRect: { x: 140, y: 460, width: 820, height: 320, minimized: false, zIndex: 38 },
    },
  }
}

function loadPanelsFromStorage(): Record<PanelId, PanelConfig> {
  const defaults = createDefaultPanels()
  if (typeof window === 'undefined') return defaults
  try {
    const raw = localStorage.getItem(STORAGE_KEY) || localStorage.getItem('newscid:workspacePanels:v1')
    if (!raw) return defaults
    const parsed = JSON.parse(raw)
    for (const id of Object.keys(defaults) as PanelId[]) {
      if (parsed[id]) {
        defaults[id] = {
          ...defaults[id],
          mode: parsed[id].mode || defaults[id].mode,
          dockZone: parsed[id].dockZone || defaults[id].dockZone,
          targetWorkspace: parsed[id].targetWorkspace ?? defaults[id].targetWorkspace,
          floaterRect: {
            ...defaults[id].floaterRect,
            ...(parsed[id].floaterRect || {}),
          },
        }
      }
    }
  } catch {
    // Ignore storage parse errors
  }
  return defaults
}

export const workspacePanels = ref<Record<PanelId, PanelConfig>>(loadPanelsFromStorage())
let topZIndex = 40

export function saveWorkspacePanels(notifyHook = true) {
  if (typeof window !== 'undefined') {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(workspacePanels.value))
    } catch {
      // Ignore quota errors
    }
  }
  if (notifyHook && onPanelsChangedHook) {
    onPanelsChangedHook()
  }
}

export function serializeWorkspacePanelsForPrefs(): Record<string, any> {
  const out: Record<string, any> = {}
  for (const [id, p] of Object.entries(workspacePanels.value)) {
    out[id] = {
      mode: p.mode,
      dockZone: p.dockZone,
      targetWorkspace: p.targetWorkspace || '',
      floaterRect: { ...p.floaterRect },
    }
  }
  return out
}

export function hydrateWorkspacePanels(saved?: Record<string, any>) {
  if (!saved || typeof saved !== 'object') return
  const keys = Object.keys(saved)
  if (keys.length === 0) return
  for (const id of Object.keys(workspacePanels.value) as PanelId[]) {
    const cfg = saved[id]
    if (!cfg) continue
    const panel = workspacePanels.value[id]
    if (cfg.mode) panel.mode = cfg.mode as PanelMode
    if (cfg.dockZone) panel.dockZone = cfg.dockZone as DockZone
    if (typeof cfg.targetWorkspace === 'string') panel.targetWorkspace = cfg.targetWorkspace
    if (cfg.floaterRect && typeof cfg.floaterRect === 'object') {
      panel.floaterRect = {
        ...panel.floaterRect,
        ...cfg.floaterRect,
      }
    }
  }
  saveWorkspacePanels(false)
}

export const panelList = computed<PanelConfig[]>(() => Object.values(workspacePanels.value))

export const floatingPanels = computed<PanelConfig[]>(() =>
  Object.values(workspacePanels.value)
    .filter(p => p.mode === 'floater')
    .sort((a, b) => a.floaterRect.zIndex - b.floaterRect.zIndex)
)

export const rightDockedTabs = computed<PanelConfig[]>(() =>
  Object.values(workspacePanels.value).filter(p => p.mode === 'docked' && p.dockZone === 'right')
)

export const leftDockedPanels = computed<PanelConfig[]>(() =>
  Object.values(workspacePanels.value).filter(p => p.mode === 'docked' && p.dockZone === 'left')
)

export const bottomDockedPanels = computed<PanelConfig[]>(() =>
  Object.values(workspacePanels.value).filter(p => p.mode === 'docked' && p.dockZone === 'bottom')
)

export function setPanelMode(id: PanelId, mode: PanelMode) {
  const panel = workspacePanels.value[id]
  if (!panel) return
  panel.mode = mode
  if (mode === 'floater') {
    panel.floaterRect.minimized = false
    topZIndex++
    panel.floaterRect.zIndex = topZIndex
  }
  saveWorkspacePanels()
}

export function setPanelWorkspace(id: PanelId, workspace: string) {
  const panel = workspacePanels.value[id]
  if (!panel) return
  panel.targetWorkspace = workspace
  saveWorkspacePanels()
}

export function setPanelDockZone(id: PanelId, zone: DockZone) {
  const panel = workspacePanels.value[id]
  if (!panel) return
  panel.dockZone = zone
  panel.mode = 'docked'
  saveWorkspacePanels()
}

export function togglePanelFloater(id: PanelId) {
  const panel = workspacePanels.value[id]
  if (!panel) return
  if (panel.mode === 'floater') {
    panel.mode = 'docked'
  } else {
    panel.mode = 'floater'
    panel.floaterRect.minimized = false
    topZIndex++
    panel.floaterRect.zIndex = topZIndex
  }
  saveWorkspacePanels()
}

export function togglePanelVisibility(id: PanelId) {
  const panel = workspacePanels.value[id]
  if (!panel) return
  panel.mode = panel.mode === 'hidden' ? 'docked' : 'hidden'
  saveWorkspacePanels()
}

export function bringFloaterToFront(id: PanelId) {
  const panel = workspacePanels.value[id]
  if (!panel) return
  topZIndex++
  panel.floaterRect.zIndex = topZIndex
}

export function updateFloaterRect(id: PanelId, partial: Partial<FloaterRect>) {
  const panel = workspacePanels.value[id]
  if (!panel) return
  panel.floaterRect = {
    ...panel.floaterRect,
    ...partial,
  }
  saveWorkspacePanels()
}

export function applyLayoutPreset(preset: LayoutPresetId) {
  const fresh = createDefaultPanels()
  if (preset === 'standard') {
    workspacePanels.value = fresh
  } else if (preset === 'floaters') {
    fresh.tree.mode = 'floater'
    fresh.tree.floaterRect = { x: 560, y: 80, width: 520, height: 440, minimized: false, zIndex: 41 }
    fresh.engine.mode = 'floater'
    fresh.engine.floaterRect = { x: 60, y: 500, width: 490, height: 240, minimized: false, zIndex: 42 }
    fresh.portal.mode = 'floater'
    fresh.portal.floaterRect = { x: 620, y: 320, width: 540, height: 420, minimized: false, zIndex: 43 }
    workspacePanels.value = fresh
  } else if (preset === 'hyprland-multiscreen') {
    fresh.board.mode = 'docked'
    fresh.imbalance.mode = 'docked'
    fresh.engine.mode = 'docked'
    fresh.notation.mode = 'docked'
    fresh.games.mode = 'hidden'
    workspacePanels.value = fresh
  } else if (preset === 'compact') {
    fresh.board.mode = 'docked'
    fresh.imbalance.mode = 'hidden'
    fresh.engine.mode = 'docked'
    fresh.engine.dockZone = 'right'
    fresh.games.mode = 'docked'
    fresh.games.dockZone = 'right'
    fresh['db-roles'].mode = 'hidden'
    fresh.portal.mode = 'hidden'
    fresh.info.mode = 'hidden'
    workspacePanels.value = fresh
  } else if (preset === 'minimal') {
    fresh.board.mode = 'docked'
    fresh.imbalance.mode = 'hidden'
    fresh.engine.mode = 'hidden'
    fresh.games.mode = 'hidden'
    fresh.info.mode = 'hidden'
    workspacePanels.value = fresh
  }
  saveWorkspacePanels()
}

export function autoAdaptPanelsForScreenSize(screenW: number, screenH: number): boolean {
  const isSmallScreen = screenH < 860 || screenW < 1180
  if (isSmallScreen) {
    if (workspacePanels.value.imbalance.mode === 'docked' && workspacePanels.value.imbalance.dockZone === 'left') {
      workspacePanels.value.imbalance.mode = 'hidden'
    }
    if (screenH < 840 && workspacePanels.value.engine.mode === 'docked' && workspacePanels.value.engine.dockZone === 'left') {
      workspacePanels.value.engine.dockZone = 'right'
    }
    if (screenW < 1100) {
      if (workspacePanels.value['db-roles'].mode === 'docked') workspacePanels.value['db-roles'].mode = 'hidden'
      if (workspacePanels.value.portal.mode === 'docked') workspacePanels.value.portal.mode = 'hidden'
      if (workspacePanels.value.info.mode === 'docked') workspacePanels.value.info.mode = 'hidden'
    }
    return true
  }
  return false
}

export function resetWorkspaceLayout() {
  workspacePanels.value = createDefaultPanels()
  saveWorkspacePanels()
}
