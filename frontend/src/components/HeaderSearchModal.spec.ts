import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import HeaderSearchModal from './HeaderSearchModal.vue'
import * as AppApi from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  CountSearchMatches: vi.fn().mockResolvedValue(42),
}))

describe('HeaderSearchModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal with database info and all search categories', () => {
    const wrapper = mount(HeaderSearchModal, {
      props: {
        dbName: 'MasterBase',
        totalGames: 12000,
      }
    })

    expect(wrapper.text()).toContain('Header / General Database Search')
    expect(wrapper.text()).toContain('MasterBase')
    expect(wrapper.text()).toContain('Players & Rating Limits')
    expect(wrapper.text()).toContain('Event, Location & Date Range')
    expect(wrapper.text()).toContain('Material & Pawn Structure Search')
    expect(wrapper.text()).toContain('Annotation & Commentary Search')
  })

  it('populates initial criteria when provided', async () => {
    const wrapper = mount(HeaderSearchModal, {
      props: {
        dbName: 'TestDB',
        totalGames: 500,
        initialCriteria: {
          player: 'Kasparov',
          pawnStructure: 'carlsbad',
          commentText: 'novelty',
          annotator: 'Shipov',
        }
      }
    })
    await flushPromises()

    const playerInput = wrapper.find('input[placeholder*="Kasparov"]')
    expect((playerInput.element as HTMLInputElement).value).toBe('Kasparov')

    const annotatorInput = wrapper.find('input[placeholder="e.g. Kasparov, G. or Shipov"]')
    expect((annotatorInput.element as HTMLInputElement).value).toBe('Shipov')

    const commentInput = wrapper.find('input[placeholder="e.g. novelty, tactical blunder, king attack"]')
    expect((commentInput.element as HTMLInputElement).value).toBe('novelty')
  })

  it('triggers CountSearchMatches when Check Matches button is clicked', async () => {
    const wrapper = mount(HeaderSearchModal, {
      props: {
        dbName: 'TestDB',
        totalGames: 500,
      }
    })

    const checkBtn = wrapper.findAll('button').find(b => b.text().includes('Check Matches'))
    expect(checkBtn).toBeDefined()
    await checkBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.CountSearchMatches).toHaveBeenCalled()
    expect(wrapper.text()).toContain('Found 42 matching games')
  })

  it('emits applySearch with configured material, pawn structure, opening moves, positionFen, and annotation filters', async () => {
    const wrapper = mount(HeaderSearchModal, {
      props: {
        dbName: 'TestDB',
        totalGames: 500,
        currentFen: 'rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2',
      }
    })

    // Click Famous Player chip
    const fischerBtn = wrapper.findAll('button').find(b => b.text() === 'Bobby Fischer')
    expect(fischerBtn).toBeDefined()
    await fischerBtn!.trigger('click')

    // Set opening move sequence
    const movesInput = wrapper.find('input[placeholder*="1.e4 c5"]')
    await movesInput.setValue('1.e4 c5 2.Nf3')

    const pawnSelect = wrapper.find('select.focus\\:border-purple-500')
    await pawnSelect.setValue('french_chain')

    const commentInput = wrapper.find('input[placeholder="e.g. novelty, tactical blunder, king attack"]')
    await commentInput.setValue('sacrifice')

    const searchBtn = wrapper.findAll('button').find(b => b.text().includes('Search Database'))
    expect(searchBtn).toBeDefined()
    await searchBtn!.trigger('click')

    const emitted = wrapper.emitted('applySearch')
    expect(emitted).toBeTruthy()
    expect(emitted![0][0]).toMatchObject({
      player: 'Bobby Fischer',
      openingMoves: ['e4', 'c5', 'Nf3'],
      pawnStructure: 'french_chain',
      commentText: 'sacrifice',
    })
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('resets all filters when Reset All is clicked', async () => {
    const wrapper = mount(HeaderSearchModal, {
      props: {
        dbName: 'TestDB',
        totalGames: 500,
        initialCriteria: {
          player: 'Fischer',
          pawnStructure: 'stonewall',
          annotator: 'Huebner',
        }
      }
    })

    const resetBtn = wrapper.findAll('button').find(b => b.text().includes('Reset All'))
    expect(resetBtn).toBeDefined()
    await resetBtn!.trigger('click')

    const playerInput = wrapper.find('input[placeholder*="Kasparov"]')
    expect((playerInput.element as HTMLInputElement).value).toBe('')

    const annotatorInput = wrapper.find('input[placeholder="e.g. Kasparov, G. or Shipov"]')
    expect((annotatorInput.element as HTMLInputElement).value).toBe('')
  })
})
