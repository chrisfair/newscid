<script setup lang="ts">
import { ref } from 'vue'
import {
  GripHorizontal,
  Minimize2,
  Maximize2,
  ExternalLink,
  PanelRight,
  X,
  Pin
} from '@lucide/vue'
import type { PanelConfig, DockZone } from '../utils/workspaceLayout'

const props = defineProps<{
  panel: PanelConfig
}>()

const emit = defineEmits<{
  (e: 'focus', id: PanelConfig['id']): void
  (e: 'updateRect', id: PanelConfig['id'], partial: Partial<PanelConfig['floaterRect']>): void
  (e: 'dock', id: PanelConfig['id'], zone?: DockZone): void
  (e: 'detachScreen', id: PanelConfig['id'], workspace?: string): void
  (e: 'hide', id: PanelConfig['id']): void
}>()

const isDragging = ref(false)
const isResizing = ref(false)

function startDrag(e: MouseEvent) {
  if ((e.target as HTMLElement)?.closest('button, select')) return
  e.preventDefault()
  emit('focus', props.panel.id)
  isDragging.value = true

  const startX = e.clientX
  const startY = e.clientY
  const initialX = props.panel.floaterRect.x
  const initialY = props.panel.floaterRect.y

  document.body.style.userSelect = 'none'

  function onMouseMove(ev: MouseEvent) {
    const dx = ev.clientX - startX
    const dy = ev.clientY - startY
    const maxX = Math.max(40, window.innerWidth - 120)
    const maxY = Math.max(40, window.innerHeight - 60)
    const nextX = Math.max(8, Math.min(maxX, initialX + dx))
    const nextY = Math.max(36, Math.min(maxY, initialY + dy))
    emit('updateRect', props.panel.id, { x: nextX, y: nextY })
  }

  function onMouseUp() {
    isDragging.value = false
    document.body.style.userSelect = ''
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
  }

  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function startResize(e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation()
  emit('focus', props.panel.id)
  isResizing.value = true

  const startX = e.clientX
  const startY = e.clientY
  const initialW = props.panel.floaterRect.width
  const initialH = props.panel.floaterRect.height

  document.body.style.userSelect = 'none'
  document.body.style.cursor = 'nwse-resize'

  function onMouseMove(ev: MouseEvent) {
    const dw = ev.clientX - startX
    const dh = ev.clientY - startY
    const nextW = Math.max(280, Math.min(window.innerWidth - 24, initialW + dw))
    const nextH = Math.max(160, Math.min(window.innerHeight - 60, initialH + dh))
    emit('updateRect', props.panel.id, { width: nextW, height: nextH })
  }

  function onMouseUp() {
    isResizing.value = false
    document.body.style.userSelect = ''
    document.body.style.cursor = ''
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
  }

  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function toggleMinimize() {
  emit('updateRect', props.panel.id, {
    minimized: !props.panel.floaterRect.minimized,
  })
}
</script>

<template>
  <div
    class="floater-window fixed bg-zinc-900/95 backdrop-blur-md border border-emerald-500/40 rounded-xl shadow-2xl flex flex-col overflow-hidden transition-shadow"
    :class="{ 'ring-2 ring-emerald-400/50': isDragging || isResizing }"
    :style="{
      left: `min(${panel.floaterRect.x}px, max(8px, calc(100vw - ${Math.min(panel.floaterRect.width, 320)}px)))`,
      top: `min(${panel.floaterRect.y}px, max(36px, calc(100vh - 56px)))`,
      width: `${panel.floaterRect.width}px`,
      maxWidth: 'calc(100vw - 16px)',
      height: panel.floaterRect.minimized ? '38px' : `${panel.floaterRect.height}px`,
      maxHeight: 'calc(100vh - 48px)',
      zIndex: panel.floaterRect.zIndex || 35,
    }"
    @mousedown="emit('focus', panel.id)"
  >
    <!-- Draggable Floater Title Bar -->
    <div
      class="floater-header h-9 px-3 bg-zinc-950/90 border-b border-zinc-800 flex items-center justify-between cursor-move select-none shrink-0"
      @mousedown="startDrag"
      @dblclick="toggleMinimize"
    >
      <div class="flex items-center gap-2 min-w-0">
        <GripHorizontal class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
        <span class="px-1.5 py-0.5 rounded bg-emerald-500/15 border border-emerald-500/30 text-[9px] font-mono uppercase text-emerald-300 shrink-0">
          Floater
        </span>
        <span class="text-xs font-bold text-zinc-100 truncate">{{ panel.title }}</span>
      </div>

      <div class="flex items-center gap-1 shrink-0">
        <button
          class="px-1.5 py-0.5 rounded hover:bg-zinc-800 text-zinc-400 hover:text-emerald-300 text-[10px] flex items-center gap-1 transition-colors cursor-pointer"
          title="Re-dock panel into workspace"
          @click.stop="emit('dock', panel.id)"
        >
          <PanelRight class="w-3 h-3" />
          <span>Dock</span>
        </button>

        <button
          class="px-1.5 py-0.5 rounded hover:bg-zinc-800 text-zinc-400 hover:text-cyan-300 text-[10px] flex items-center gap-1 transition-colors cursor-pointer"
          title="Detach to separate OS / Hyprland screen window"
          @click.stop="emit('detachScreen', panel.id, panel.targetWorkspace)"
        >
          <ExternalLink class="w-3 h-3" />
          <span>Screen</span>
        </button>

        <select
          :value="panel.targetWorkspace || ''"
          class="bg-zinc-900 border border-zinc-700/80 rounded px-1 py-0.5 text-[9px] text-cyan-300 font-mono cursor-pointer focus:outline-none"
          title="Detach & relocate panel directly to Hyprland Workspace 1..10"
          @change.stop="emit('detachScreen', panel.id, ($event.target as HTMLSelectElement).value)"
        >
          <option value="">WS ▾</option>
          <option v-for="ws in 10" :key="ws" :value="String(ws)">WS {{ ws }}</option>
        </select>

        <button
          class="p-1 rounded hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 transition-colors cursor-pointer"
          :title="panel.floaterRect.minimized ? 'Expand floater' : 'Shade / minimize floater'"
          @click.stop="toggleMinimize"
        >
          <Maximize2 v-if="panel.floaterRect.minimized" class="w-3.5 h-3.5" />
          <Minimize2 v-else class="w-3.5 h-3.5" />
        </button>

        <button
          class="p-1 rounded hover:bg-rose-950/60 text-zinc-400 hover:text-rose-400 transition-colors cursor-pointer"
          title="Hide / remove panel"
          @click.stop="emit('hide', panel.id)"
        >
          <X class="w-3.5 h-3.5" />
        </button>
      </div>
    </div>

    <!-- Floater Body Content -->
    <div v-show="!panel.floaterRect.minimized" class="flex-1 overflow-auto p-2 relative min-h-0">
      <slot />
    </div>

    <!-- Bottom-Right Corner Resize Handle -->
    <div
      v-if="!panel.floaterRect.minimized"
      class="floater-resizer absolute bottom-0 right-0 w-4 h-4 cursor-nwse-resize flex items-end justify-end p-0.5 text-zinc-500 hover:text-emerald-400 z-20 select-none"
      title="Drag to resize floater"
      @mousedown="startResize"
    >
      <svg class="w-3 h-3" viewBox="0 0 12 12" fill="none">
        <path d="M11 1L1 11M11 5L5 11M11 9L9 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
      </svg>
    </div>
  </div>
</template>
