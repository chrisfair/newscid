/**
 * Audio Synthesizer for Chess Sound Effects
 * Uses Web Audio API - 100% self-contained, offline-compatible, zero external audio assets needed.
 */

import { ref } from 'vue'

const STORAGE_KEY = 'newscid:sound_enabled'
export const soundEnabled = ref<boolean>(localStorage.getItem(STORAGE_KEY) !== 'false')

export function setSoundEnabled(enabled: boolean) {
  soundEnabled.value = enabled
  localStorage.setItem(STORAGE_KEY, enabled ? 'true' : 'false')
}

export function toggleSound(): boolean {
  setSoundEnabled(!soundEnabled.value)
  if (soundEnabled.value) {
    playMoveSound()
  }
  return soundEnabled.value
}

let audioCtx: AudioContext | null = null

function getAudioContext(): AudioContext | null {
  if (typeof window === 'undefined') return null
  if (!audioCtx) {
    const AudioContextClass = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
    if (AudioContextClass) {
      audioCtx = new AudioContextClass()
    }
  }
  if (audioCtx && audioCtx.state === 'suspended') {
    audioCtx.resume().catch(() => {})
  }
  return audioCtx
}

/**
 * Play authentic wooden move tap sound
 */
export function playMoveSound() {
  if (!soundEnabled.value) return
  const ctx = getAudioContext()
  if (!ctx) return

  const osc = ctx.createOscillator()
  const gain = ctx.createGain()

  const now = ctx.currentTime
  osc.type = 'triangle'
  osc.frequency.setValueAtTime(340, now)
  osc.frequency.exponentialRampToValueAtTime(140, now + 0.06)

  gain.gain.setValueAtTime(0.35, now)
  gain.gain.exponentialRampToValueAtTime(0.001, now + 0.07)

  osc.connect(gain)
  gain.connect(ctx.destination)

  osc.start(now)
  osc.stop(now + 0.075)
}

/**
 * Play heavier wooden capture strike sound
 */
export function playCaptureSound() {
  if (!soundEnabled.value) return
  const ctx = getAudioContext()
  if (!ctx) return

  const now = ctx.currentTime

  // Primary impulse
  const osc1 = ctx.createOscillator()
  const gain1 = ctx.createGain()
  osc1.type = 'sine'
  osc1.frequency.setValueAtTime(420, now)
  osc1.frequency.exponentialRampToValueAtTime(90, now + 0.09)

  gain1.gain.setValueAtTime(0.45, now)
  gain1.gain.exponentialRampToValueAtTime(0.001, now + 0.1)

  osc1.connect(gain1)
  gain1.connect(ctx.destination)
  osc1.start(now)
  osc1.stop(now + 0.11)

  // Secondary rattle
  const osc2 = ctx.createOscillator()
  const gain2 = ctx.createGain()
  osc2.type = 'triangle'
  osc2.frequency.setValueAtTime(260, now + 0.02)
  osc2.frequency.exponentialRampToValueAtTime(70, now + 0.09)

  gain2.gain.setValueAtTime(0.2, now + 0.02)
  gain2.gain.exponentialRampToValueAtTime(0.001, now + 0.1)

  osc2.connect(gain2)
  gain2.connect(ctx.destination)
  osc2.start(now + 0.02)
  osc2.stop(now + 0.11)
}

/**
 * Play check chime sound
 */
export function playCheckSound() {
  if (!soundEnabled.value) return
  const ctx = getAudioContext()
  if (!ctx) return

  const now = ctx.currentTime
  const osc = ctx.createOscillator()
  const gain = ctx.createGain()

  osc.type = 'sine'
  osc.frequency.setValueAtTime(780, now)
  osc.frequency.exponentialRampToValueAtTime(880, now + 0.04)

  gain.gain.setValueAtTime(0.3, now)
  gain.gain.exponentialRampToValueAtTime(0.001, now + 0.28)

  osc.connect(gain)
  gain.connect(ctx.destination)

  osc.start(now)
  osc.stop(now + 0.3)
}

/**
 * Play castling slide sound
 */
export function playCastleSound() {
  if (!soundEnabled.value) return
  playMoveSound()
  setTimeout(() => {
    playMoveSound()
  }, 110)
}

/**
 * Play victory chime
 */
export function playVictorySound() {
  if (!soundEnabled.value) return
  const ctx = getAudioContext()
  if (!ctx) return

  const notes = [523.25, 659.25, 783.99, 1046.50] // C5, E5, G5, C6
  notes.forEach((freq, idx) => {
    setTimeout(() => {
      const now = ctx.currentTime
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.type = 'triangle'
      osc.frequency.setValueAtTime(freq, now)
      gain.gain.setValueAtTime(0.25, now)
      gain.gain.exponentialRampToValueAtTime(0.001, now + 0.22)
      osc.connect(gain)
      gain.connect(ctx.destination)
      osc.start(now)
      osc.stop(now + 0.23)
    }, idx * 90)
  })
}
