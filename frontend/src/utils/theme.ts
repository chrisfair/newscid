import { ref } from 'vue'

export interface UITheme {
  id: string
  name: string
  description: string
  category: 'dark' | 'light'
  preview: {
    bg: string
    card: string
    border: string
    accent: string
    text: string
  }
}

export const UI_THEMES: UITheme[] = [
  {
    id: 'dark',
    name: 'Modern Dark',
    description: 'Sleek neutral dark workspace (Zinc / Slate)',
    category: 'dark',
    preview: { bg: '#09090b', card: '#18181b', border: '#27272a', accent: '#10b981', text: '#f4f4f5' },
  },
  {
    id: 'light',
    name: 'Modern Light',
    description: 'Clean, high-readability daylight workspace',
    category: 'light',
    preview: { bg: '#f8f9fb', card: '#ffffff', border: '#e2e8f0', accent: '#059669', text: '#0f172a' },
  },
  {
    id: 'oled',
    name: 'OLED Black',
    description: 'Pure pitch black for OLED displays & maximum contrast',
    category: 'dark',
    preview: { bg: '#000000', card: '#0f0f0f', border: '#222222', accent: '#3b82f6', text: '#ffffff' },
  },
  {
    id: 'nord',
    name: 'Nordic Frost',
    description: 'Calm arctic dark blue-slate palette',
    category: 'dark',
    preview: { bg: '#10141d', card: '#1b2230', border: '#2e384d', accent: '#38bdf8', text: '#e2e8f0' },
  },
  {
    id: 'dracula',
    name: 'Midnight Purple',
    description: 'Deep obsidian and vibrant violet accents',
    category: 'dark',
    preview: { bg: '#110e1e', card: '#1b172c', border: '#332b4f', accent: '#a855f7', text: '#f3e8ff' },
  },
  {
    id: 'retro',
    name: 'Classic Sepia',
    description: 'Warm parchment and vintage Scid study aesthetics',
    category: 'light',
    preview: { bg: '#ece5d3', card: '#faf7f0', border: '#d9cdb4', accent: '#d97706', text: '#1c160e' },
  },
]

const STORAGE_KEY = 'newscid-ui-theme'

export const currentUITheme = ref<string>('dark')

export function initUITheme(): string {
  const saved = typeof window !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null
  const initial = saved && UI_THEMES.some(t => t.id === saved) ? saved : 'dark'
  applyUITheme(initial)
  return initial
}

export function applyUITheme(themeId: string): void {
  const theme = UI_THEMES.find(t => t.id === themeId) || UI_THEMES[0]
  currentUITheme.value = theme.id
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('data-theme', theme.id)
    if (theme.category === 'light') {
      document.documentElement.classList.add('light-mode')
      document.documentElement.classList.remove('dark-mode')
    } else {
      document.documentElement.classList.add('dark-mode')
      document.documentElement.classList.remove('light-mode')
    }
    localStorage.setItem(STORAGE_KEY, theme.id)
  }
}

export function toggleDarkLight(): string {
  const isCurrentlyLight = currentUITheme.value === 'light' || currentUITheme.value === 'retro'
  const nextTheme = isCurrentlyLight ? 'dark' : 'light'
  applyUITheme(nextTheme)
  return nextTheme
}
