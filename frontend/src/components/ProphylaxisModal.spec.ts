import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ProphylaxisModal from './ProphylaxisModal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetProphylaxisDrillBatch: vi.fn().mockResolvedValue([
    {
      id: 'pro-01-karpov-kh1',
      title: "Karpov's Prophylactic King Step",
      fen: 'r1b2rk1/pp1nq1pp/4p3/2ppPp2/3P4/2PB1N2/PP1Q1PPP/R4RK1 w - - 0 13',
      turn: 'white',
      category: 'king_safety',
      categoryLabel: 'King Safety Unpin',
      prompt: 'Black is ready to launch counterplay. Step out of potential diagonal pins.',
      winningMoveUCI: 'g1h1',
      winningMoveSAN: 'Kh1',
      alternativeUCIs: [],
      opponentThreat: 'Black intends ...c4 followed by tactical bishop checks.',
      threatArrowFrom: 'c5',
      threatArrowTo: 'c4',
      explanation: 'Kh1! A signature Karpovian calm move.',
      source: 'Karpov vs. Spassky 1974',
      difficulty: 'advanced',
    },
  ]),
  GenerateProphylaxisDrillsFromDatabase: vi.fn().mockResolvedValue([]),
  SubmitProphylaxisAnswer: vi.fn().mockImplementation((drillId, uci) => {
    const isCorrect = (uci === 'g1h1' || uci === 'kh1')
    return Promise.resolve({
      isCorrect,
      pointsAwarded: isCorrect ? 150 : 0,
      bestMoveUCI: 'g1h1',
      bestMoveSAN: 'Kh1',
      explanation: 'Kh1! A signature Karpovian calm move.',
      opponentThreat: 'Black intends ...c4 followed by tactical bishop checks.',
    })
  }),
}))

describe('ProphylaxisModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal header, stats ribbon, and loads drills', async () => {
    const wrapper = mount(ProphylaxisModal)

    expect(wrapper.text()).toContain('Quiet Moves & Prophylaxis Drill')
    expect(wrapper.text()).toContain('Non-Forcing Tactics Trainer')

    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    expect(wrapper.text()).toContain("Karpov's Prophylactic King Step")
    expect(wrapper.text()).toContain('King Safety Unpin')
    expect(wrapper.text()).toContain('Step out of potential diagonal pins.')
  })

  it('toggles Threat Radar on and off', async () => {
    const wrapper = mount(ProphylaxisModal)
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    const radarBtn = wrapper.findAll('button').find(b => b.text().includes('Threat Radar'))
    expect(radarBtn).toBeDefined()
    expect(radarBtn!.text()).toContain('OFF')

    await radarBtn!.trigger('click')
    expect(radarBtn!.text()).toContain('ON')
    expect(wrapper.text()).toContain('Detected Enemy Threat:')
    expect(wrapper.text()).toContain('Black intends ...c4 followed by tactical bishop checks.')
  })

  it('submits move via text input and displays correct feedback', async () => {
    const wrapper = mount(ProphylaxisModal)
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    const input = wrapper.find('input[type="text"]')
    expect(input.exists()).toBe(true)
    await input.setValue('Kh1')

    const submitBtn = wrapper.findAll('button').find(b => b.text() === 'Submit')
    expect(submitBtn).toBeDefined()
    await submitBtn!.trigger('click')

    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    expect(wrapper.text()).toContain('MASTERFUL PROPHYLAXIS')
    expect(wrapper.text()).toContain('Solution: Kh1')
    expect(wrapper.text()).toContain('Kh1! A signature Karpovian calm move.')
  })

  it('reveals solution when Show Solution button is clicked', async () => {
    const wrapper = mount(ProphylaxisModal)
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    const showSolBtn = wrapper.findAll('button').find(b => b.text().includes('Show Solution'))
    expect(showSolBtn).toBeDefined()
    await showSolBtn!.trigger('click')

    expect(wrapper.text()).toContain('Solution: Kh1')
  })

  it('emits close on clicking close button', async () => {
    const wrapper = mount(ProphylaxisModal)
    const closeBtn = wrapper.find('button[title="Close"]')
    expect(closeBtn.exists()).toBe(true)
    await closeBtn.trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
