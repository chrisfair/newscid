import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  workspacePanels,
  panelList,
  floatingPanels,
  rightDockedTabs,
  leftDockedPanels,
  bottomDockedPanels,
  setOnPanelsChangedHook,
  setPanelMode,
  setPanelWorkspace,
  setPanelDockZone,
  togglePanelFloater,
  togglePanelVisibility,
  bringFloaterToFront,
  updateFloaterRect,
  applyLayoutPreset,
  autoAdaptPanelsForScreenSize,
  hydrateWorkspacePanels,
  serializeWorkspacePanelsForPrefs,
  resetWorkspaceLayout,
} from './workspaceLayout'

describe('workspaceLayout.ts', () => {
  beforeEach(() => {
    localStorage.clear()
    setOnPanelsChangedHook(null)
    resetWorkspaceLayout()
  })

  it('invokes onPanelsChangedHook when panels change', () => {
    const hook = vi.fn()
    setOnPanelsChangedHook(hook)
    setPanelMode('engine', 'floater')
    expect(hook).toHaveBeenCalled()
    expect(workspacePanels.value.engine.mode).toBe('floater')
    expect(workspacePanels.value.engine.floaterRect.minimized).toBe(false)
  })

  it('manages dock zones, visibility, zIndex, and floater rects', () => {
    setPanelDockZone('engine', 'right')
    expect(workspacePanels.value.engine.dockZone).toBe('right')
    expect(workspacePanels.value.engine.mode).toBe('docked')

    togglePanelVisibility('games')
    expect(workspacePanels.value.games.mode).toBe('hidden')
    togglePanelVisibility('games')
    expect(workspacePanels.value.games.mode).toBe('docked')

    togglePanelFloater('tree')
    togglePanelFloater('engine')
    const prevZ = workspacePanels.value.tree.floaterRect.zIndex
    bringFloaterToFront('tree')
    expect(workspacePanels.value.tree.floaterRect.zIndex).toBeGreaterThan(prevZ)

    updateFloaterRect('tree', { x: 120, y: 150, width: 600, height: 420, minimized: true })
    expect(workspacePanels.value.tree.floaterRect.x).toBe(120)
    expect(workspacePanels.value.tree.floaterRect.minimized).toBe(true)

    expect(panelList.value.length).toBeGreaterThan(0)
    expect(floatingPanels.value.map(p => p.id)).toContain('tree')
    expect(rightDockedTabs.value.length).toBeGreaterThan(0)
    expect(leftDockedPanels.value.length).toBeGreaterThan(0)
    expect(bottomDockedPanels.value.length).toBeGreaterThan(0)
  })

  it('ignores unknown panel IDs safely', () => {
    setPanelMode('nonexistent' as any, 'floater')
    setPanelWorkspace('nonexistent' as any, '2')
    setPanelDockZone('nonexistent' as any, 'left')
    togglePanelFloater('nonexistent' as any)
    togglePanelVisibility('nonexistent' as any)
    bringFloaterToFront('nonexistent' as any)
    updateFloaterRect('nonexistent' as any, { x: 10 })
    expect(workspacePanels.value.board.mode).toBe('docked')
  })

  it('applies hyprland-multiscreen preset and adapts/skips based on screen size', () => {
    applyLayoutPreset('hyprland-multiscreen')
    expect(workspacePanels.value.games.mode).toBe('hidden')
    expect(workspacePanels.value.board.mode).toBe('docked')

    resetWorkspaceLayout()
    expect(autoAdaptPanelsForScreenSize(1920, 1080)).toBe(false)
    expect(autoAdaptPanelsForScreenSize(1024, 768)).toBe(true)
    expect(workspacePanels.value.imbalance.mode).toBe('hidden')
    expect(workspacePanels.value.engine.dockZone).toBe('right')
    expect(workspacePanels.value['db-roles'].mode).toBe('hidden')
    expect(workspacePanels.value.portal.mode).toBe('hidden')
    expect(workspacePanels.value.info.mode).toBe('hidden')
  })

  it('hydrates and ignores empty or invalid saved objects', () => {
    hydrateWorkspacePanels(undefined)
    hydrateWorkspacePanels({})
    const serialized = serializeWorkspacePanelsForPrefs()
    serialized.board.mode = 'floater'
    serialized.board.floaterRect = { x: 99, y: 88, width: 500, height: 500, minimized: false, zIndex: 50 }
    hydrateWorkspacePanels(serialized)
    expect(workspacePanels.value.board.mode).toBe('floater')
    expect(workspacePanels.value.board.floaterRect.x).toBe(99)
  })
})
