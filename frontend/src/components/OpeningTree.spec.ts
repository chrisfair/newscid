import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import OpeningTree from './OpeningTree.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetPolyglotBooks: vi.fn().mockResolvedValue(['Elo2400.bin', 'gm2600.bin']),
  QueryPolyglotBook: vi.fn().mockResolvedValue([
    { san: 'e4', uci: 'e2e4', weight: 1000, percentage: 65.5, learn: 1 },
    { san: 'd4', uci: 'd2d4', weight: 500, percentage: 34.5, learn: 0 },
  ]),
}))

describe('OpeningTree.vue', () => {
  const dummyTreeData = {
    fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    totalGames: 100,
    moves: [
      {
        move: 'e4',
        uci: 'e2e4',
        count: 65,
        whiteWins: 30,
        draws: 20,
        blackWins: 15,
        scorePct: 61.5,
      },
      {
        move: 'd4',
        uci: 'd2d4',
        count: 35,
        whiteWins: 15,
        draws: 10,
        blackWins: 10,
        scorePct: 57.1,
      },
    ],
  }

  it('renders DB tree table and emits playMove with UCI on row click', async () => {
    const wrapper = mount(OpeningTree, {
      props: {
        treeData: dummyTreeData,
        dbOpen: true,
      },
    })

    expect(wrapper.text()).toContain('DB Tree')
    expect(wrapper.text()).toContain('100')
    expect(wrapper.text()).toContain('e4')
    expect(wrapper.text()).toContain('d4')

    // Click the e4 row
    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(2)
    await rows[0].trigger('click')

    expect(wrapper.emitted('playMove')).toBeTruthy()
    expect(wrapper.emitted('playMove')![0]).toEqual(['e2e4'])
  })

  it('renders empty database state with openDb button when no database is open', async () => {
    const wrapper = mount(OpeningTree, {
      props: {
        treeData: { fen: 'start', totalGames: 0, moves: [] },
        dbOpen: false,
      },
    })

    expect(wrapper.text()).toContain('No database is currently open')
    const openBtn = wrapper.find('button.bg-emerald-600\\/20')
    expect(openBtn.exists()).toBe(true)

    await openBtn.trigger('click')
    expect(wrapper.emitted('openDb')).toBeTruthy()
  })

  it('switches to Polyglot opening book tab and plays book moves', async () => {
    const wrapper = mount(OpeningTree, {
      props: {
        treeData: dummyTreeData,
        fen: dummyTreeData.fen,
        dbOpen: true,
      },
    })

    // Click Opening Book tab
    const bookTabBtn = wrapper.findAll('.tree-header button')[1]
    await bookTabBtn.trigger('click')

    // Wait for book moves
    await new Promise((r) => setTimeout(r, 20))
    expect(wrapper.text()).toContain('Opening Book (.bin)')
  })

  it('emits refresh event when refresh button is clicked', async () => {
    const wrapper = mount(OpeningTree, {
      props: {
        treeData: dummyTreeData,
        dbOpen: true,
      },
    })

    const refreshBtn = wrapper.find('button[title="Refresh Opening Tree"]')
    expect(refreshBtn.exists()).toBe(true)
    await refreshBtn.trigger('click')
    expect(wrapper.emitted('refresh')).toBeTruthy()
  })
})
