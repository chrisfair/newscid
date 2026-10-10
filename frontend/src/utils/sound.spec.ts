import { describe, it, expect, beforeEach, vi } from 'vitest'
import { soundEnabled, setSoundEnabled, toggleSound, playMoveSound, playCaptureSound, playCheckSound, playCastleSound } from './sound'

describe('sound utility', () => {
  beforeEach(() => {
    localStorage.clear()
    setSoundEnabled(true)
  })

  it('initializes with sound enabled', () => {
    expect(soundEnabled.value).toBe(true)
  })

  it('toggles sound on and off', () => {
    const state1 = toggleSound()
    expect(state1).toBe(false)
    expect(soundEnabled.value).toBe(false)

    const state2 = toggleSound()
    expect(state2).toBe(true)
    expect(soundEnabled.value).toBe(true)
  })

  it('persists preference to localStorage', () => {
    setSoundEnabled(false)
    expect(localStorage.getItem('newscid:sound_enabled')).toBe('false')

    setSoundEnabled(true)
    expect(localStorage.getItem('newscid:sound_enabled')).toBe('true')
  })

  it('sound triggers execute without throwing in node/happy-dom environment', () => {
    expect(() => playMoveSound()).not.toThrow()
    expect(() => playCaptureSound()).not.toThrow()
    expect(() => playCheckSound()).not.toThrow()
    expect(() => playCastleSound()).not.toThrow()
  })
})
