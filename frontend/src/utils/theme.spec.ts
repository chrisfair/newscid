import { describe, it, expect, beforeEach } from 'vitest'
import { UI_THEMES, currentUITheme, initUITheme, applyUITheme, toggleDarkLight } from './theme'

describe('UI Theme System', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
    document.documentElement.classList.remove('dark-mode', 'light-mode')
  })

  it('provides a curated collection of themes with both dark and light modes', () => {
    expect(UI_THEMES.length).toBeGreaterThanOrEqual(6)
    
    const darkThemes = UI_THEMES.filter(t => t.category === 'dark')
    const lightThemes = UI_THEMES.filter(t => t.category === 'light')

    expect(darkThemes.length).toBeGreaterThanOrEqual(1)
    expect(lightThemes.length).toBeGreaterThanOrEqual(1)

    expect(UI_THEMES.some(t => t.id === 'dark')).toBe(true)
    expect(UI_THEMES.some(t => t.id === 'light')).toBe(true)
    expect(UI_THEMES.some(t => t.id === 'oled')).toBe(true)
    expect(UI_THEMES.some(t => t.id === 'nord')).toBe(true)
    expect(UI_THEMES.some(t => t.id === 'dracula')).toBe(true)
    expect(UI_THEMES.some(t => t.id === 'retro')).toBe(true)
  })

  it('initializes to dark theme by default when no preference is saved', () => {
    const initial = initUITheme()
    expect(initial).toBe('dark')
    expect(currentUITheme.value).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark-mode')).toBe(true)
    expect(document.documentElement.classList.contains('light-mode')).toBe(false)
  })

  it('restores saved theme preference from localStorage on init', () => {
    localStorage.setItem('newscid-ui-theme', 'nord')
    const initial = initUITheme()
    expect(initial).toBe('nord')
    expect(currentUITheme.value).toBe('nord')
    expect(document.documentElement.getAttribute('data-theme')).toBe('nord')
    expect(document.documentElement.classList.contains('dark-mode')).toBe(true)
  })

  it('falls back to default if stored theme in localStorage is invalid', () => {
    localStorage.setItem('newscid-ui-theme', 'non-existent-theme-xyz')
    const initial = initUITheme()
    expect(initial).toBe('dark')
    expect(currentUITheme.value).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
  })

  it('applies light themes with appropriate classes and persistence', () => {
    applyUITheme('light')
    expect(currentUITheme.value).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    expect(document.documentElement.classList.contains('light-mode')).toBe(true)
    expect(document.documentElement.classList.contains('dark-mode')).toBe(false)
    expect(localStorage.getItem('newscid-ui-theme')).toBe('light')
  })

  it('applies retro sepia theme with light-mode class', () => {
    applyUITheme('retro')
    expect(currentUITheme.value).toBe('retro')
    expect(document.documentElement.getAttribute('data-theme')).toBe('retro')
    expect(document.documentElement.classList.contains('light-mode')).toBe(true)
    expect(document.documentElement.classList.contains('dark-mode')).toBe(false)
  })

  it('toggles between dark and light modes reliably', () => {
    applyUITheme('dark')
    expect(currentUITheme.value).toBe('dark')

    const toggledToLight = toggleDarkLight()
    expect(toggledToLight).toBe('light')
    expect(currentUITheme.value).toBe('light')
    expect(document.documentElement.classList.contains('light-mode')).toBe(true)

    const toggledToDark = toggleDarkLight()
    expect(toggledToDark).toBe('dark')
    expect(currentUITheme.value).toBe('dark')
    expect(document.documentElement.classList.contains('dark-mode')).toBe(true)
  })
})
