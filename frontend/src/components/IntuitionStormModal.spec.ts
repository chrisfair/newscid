import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import IntuitionStormModal from './IntuitionStormModal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetIntuitionDrillBatch: vi.fn().mockResolvedValue([
    {
      id: 'int-1',
      fen: 'r1bqk2r/pp2bppp/2n1pn2/2pp4/2PP4/2N1PN2/PP2BPPP/R1BQK2R w KQkq - 4 7',
      whitePlayer: 'Karpov, Anatoly',
      blackPlayer: 'Kasparov, Garry',
      event: 'World Championship 1985',
      evaluationCp: 15,
      category: 'equal',
      keyFeature: "Symmetrical Queen's Gambit: equal central presence and harmonious development.",
      materialDelta: 0,
      timeLimitSec: 5,
    },
    {
      id: 'int-3',
      fen: 'r1b2rk1/pp3ppp/2n1p3/3q4/3P4/5N2/PP1QBPPP/R4RK1 b - - 0 12',
      whitePlayer: 'Capablanca, Jose',
      blackPlayer: 'Tartakower, Savielly',
      event: 'New York 1924',
      evaluationCp: 380,
      category: 'white_winning',
      keyFeature: 'Decisive positional control: White dominates the open d-file with superior active pieces.',
      materialDelta: 0,
      timeLimitSec: 5,
    }
  ]),
  GenerateIntuitionDrillsFromDatabase: vi.fn().mockResolvedValue([]),
  SubmitIntuitionAnswer: vi.fn().mockImplementation((challengeId, userChoice) => {
    const isCorrect = (userChoice === 'equal')
    return Promise.resolve({
      isCorrect,
      userChoice,
      actualCategory: 'equal',
      evaluationCp: 15,
      scoreFormatted: '+0.15',
      pointsAwarded: isCorrect ? 220 : 0,
      speedBonus: isCorrect ? 120 : 0,
      explanation: isCorrect
        ? '🎯 Spot on! Evaluated in 1000ms (+0.15).'
        : '❌ Misjudgment. The correct assessment is Equal (=) (+0.15).',
      keyFeature: "Symmetrical Queen's Gambit: equal central presence and harmonious development."
    })
  })
}))

describe('IntuitionStormModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal header, stats ribbon, and loads challenges', async () => {
    const wrapper = mount(IntuitionStormModal)

    expect(wrapper.text()).toContain('Intuition Storm')
    expect(wrapper.text()).toContain('Speed Reflex Trainer')
    expect(wrapper.text()).toContain('Streak:')
    expect(wrapper.text()).toContain('Instinct Rating:')

    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    expect(wrapper.text()).toContain('Karpov, Anatoly vs Kasparov, Garry')
    expect(wrapper.text()).toContain('White is Winning')
    expect(wrapper.text()).toContain('Balanced / Equal')
    expect(wrapper.text()).toContain('Black is Winning')
  })

  it('submits correct answer and displays positive feedback with score', async () => {
    const wrapper = mount(IntuitionStormModal)
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    // Find and click the Equal button [2]
    const buttons = wrapper.findAll('button')
    const equalBtn = buttons.find(b => b.text().includes('Balanced / Equal'))
    expect(equalBtn).toBeDefined()
    await equalBtn!.trigger('click')

    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    expect(wrapper.text()).toContain('CORRECT JUDGMENT')
    expect(wrapper.text()).toContain('+220 pts')
  })

  it('handles keyboard hotkeys [1], [2], [3]', async () => {
    const wrapper = mount(IntuitionStormModal)
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    // Trigger keydown '2'
    window.dispatchEvent(new KeyboardEvent('keydown', { key: '2' }))
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    expect(wrapper.text()).toContain('CORRECT JUDGMENT')
  })

  it('emits close on Escape key', async () => {
    const wrapper = mount(IntuitionStormModal)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
