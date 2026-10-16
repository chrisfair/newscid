import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RepertoireShortenerModal from './RepertoireShortenerModal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  ListRepertoires: vi.fn().mockResolvedValue([
    {
      id: 'white-openings-italian',
      name: 'White: Italian Game & Open Repertoire',
      color: 'w',
      description: 'Classical 1.e4 opening lines',
      numMoves: 45,
    },
    {
      id: 'black-openings-sicilian',
      name: 'Black: Sicilian Najdorf',
      color: 'b',
      description: 'Sicilian lines',
      numMoves: 38,
    },
  ]),
  ShortenRepertoire: vi.fn().mockResolvedValue({
    originalId: 'white-openings-italian',
    prunedRepertoireId: 'white-openings-italian-pareto-80',
    originalMoveCount: 45,
    prunedMoveCount: 18,
    movesRemoved: 27,
    reductionPct: 60.0,
    coverageRetainedPct: 82.5,
    prunedBranches: [
      {
        sanLine: 'e4 e5 Nf3 Nc6 Bc4 h6',
        depth: 4,
        frequencyPct: 1.2,
        totalGames: 4,
        reason: 'Sideline below 80/20 cutoff (occurs in only 1.2% of games)',
      },
    ],
    highYieldLines: [
      {
        sanLine: 'e4 e5 Nf3 Nc6 Bc4 Bc5',
        coveragePct: 85.0,
        movesCount: 18,
        keyIdea: 'Italian Game: Giuoco Piano',
      },
    ],
    prunedRepertoire: {
      id: 'white-openings-italian-pareto-80',
      name: 'White: Italian Game (Pareto 80% Core)',
      color: 'w',
      description: 'Optimized',
      positions: {},
    },
  }),
}))

describe('RepertoireShortenerModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal header, sliders, and loads repertoires', async () => {
    const wrapper = mount(RepertoireShortenerModal)

    expect(wrapper.text()).toContain('Repertoire Shortener')
    expect(wrapper.text()).toContain('Pareto 80/20 Pruner')
    expect(wrapper.text()).toContain('Target Coverage:')
    expect(wrapper.text()).toContain('80%')

    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    expect(wrapper.text()).toContain('White: Italian Game & Open Repertoire')
  })

  it('runs pruning and displays reduction scorecard and pruned branches', async () => {
    const wrapper = mount(RepertoireShortenerModal)
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    const pruneBtn = wrapper.findAll('button').find(b => b.text().includes('Prune Repertoire'))
    expect(pruneBtn).toBeDefined()
    await pruneBtn!.trigger('click')

    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    expect(wrapper.text()).toContain('Study Workload Saved')
    expect(wrapper.text()).toContain('60%')
    expect(wrapper.text()).toContain('27 moves pruned')
    expect(wrapper.text()).toContain('83%') // 82.5 rounded
    expect(wrapper.text()).toContain('e4 e5 Nf3 Nc6 Bc4 h6')
  })

  it('switches between pruned sidelines and core mainlines', async () => {
    const wrapper = mount(RepertoireShortenerModal)
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    const pruneBtn = wrapper.findAll('button').find(b => b.text().includes('Prune Repertoire'))
    await pruneBtn!.trigger('click')
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    const mainlinesBtn = wrapper.findAll('button').find(b => b.text().includes('Core High-Yield Lines'))
    expect(mainlinesBtn).toBeDefined()
    await mainlinesBtn!.trigger('click')

    expect(wrapper.text()).toContain('Italian Game: Giuoco Piano')
  })

  it('emits close on clicking close button', async () => {
    const wrapper = mount(RepertoireShortenerModal)
    const closeBtn = wrapper.findAll('button').find(b => b.find('svg').exists() && !b.text())
    expect(closeBtn).toBeDefined()
    await closeBtn!.trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
