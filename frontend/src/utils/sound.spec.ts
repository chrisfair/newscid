import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  soundEnabled,
  setSoundEnabled,
  toggleSound,
  playMoveSound,
  playCaptureSound,
  playCheckSound,
  playCastleSound,
  playVictorySound,
} from './sound'

function createMockAudioContext() {
  const oscMock = {
    type: 'sine',
    frequency: {
      setValueAtTime: vi.fn(),
      exponentialRampToValueAtTime: vi.fn(),
    },
    connect: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
  }
  const gainMock = {
    gain: {
      setValueAtTime: vi.fn(),
      exponentialRampToValueAtTime: vi.fn(),
    },
    connect: vi.fn(),
  }
  const ctxMock = {
    state: 'suspended',
    currentTime: 0.5,
    destination: {},
    resume: vi.fn().mockResolvedValue(undefined),
    createOscillator: vi.fn(() => oscMock),
    createGain: vi.fn(() => gainMock),
  }
  return { ctxMock, oscMock, gainMock }
}

describe('sound utility', () => {
  const origAudioContext = (window as any).AudioContext

  beforeEach(() => {
    localStorage.clear()
    setSoundEnabled(true)
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    ;(window as any).AudioContext = origAudioContext
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

  it('does not play sounds when disabled', () => {
    const { ctxMock } = createMockAudioContext()
    ;(window as any).AudioContext = vi.fn(() => ctxMock)

    setSoundEnabled(false)
    playMoveSound()
    playCaptureSound()
    playCheckSound()
    playCastleSound()
    playVictorySound()
    vi.runAllTimers()

    expect(ctxMock.createOscillator).not.toHaveBeenCalled()
  })

  it('synthesizes move, capture, check, castle, and victory sounds with Web Audio API', () => {
    const { ctxMock, oscMock, gainMock } = createMockAudioContext()
    ;(window as any).AudioContext = vi.fn(() => ctxMock)

    playMoveSound()
    expect(ctxMock.resume).toHaveBeenCalled()
    expect(ctxMock.createOscillator).toHaveBeenCalled()
    expect(oscMock.start).toHaveBeenCalled()
    expect(gainMock.connect).toHaveBeenCalledWith(ctxMock.destination)

    const oscCallsBeforeCapture = ctxMock.createOscillator.mock.calls.length
    playCaptureSound()
    expect(ctxMock.createOscillator.mock.calls.length).toBe(oscCallsBeforeCapture + 2)

    const oscCallsBeforeCheck = ctxMock.createOscillator.mock.calls.length
    playCheckSound()
    expect(ctxMock.createOscillator.mock.calls.length).toBe(oscCallsBeforeCheck + 1)

    const oscCallsBeforeCastle = ctxMock.createOscillator.mock.calls.length
    playCastleSound()
    vi.advanceTimersByTime(150)
    expect(ctxMock.createOscillator.mock.calls.length).toBe(oscCallsBeforeCastle + 2)

    const oscCallsBeforeVictory = ctxMock.createOscillator.mock.calls.length
    playVictorySound()
    vi.advanceTimersByTime(500)
    expect(ctxMock.createOscillator.mock.calls.length).toBe(oscCallsBeforeVictory + 4)
  })
})
