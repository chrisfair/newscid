import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import GameBrowser from './GameBrowser.vue'
import * as AppApi from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetNotablePlayersDropdown: vi.fn().mockResolvedValue([
    {
      name: 'Bobby Fischer',
      title: 'World Champion (1972-1975)',
      era: '1955-1992',
      aliases: ['Fischer, Robert James', 'Fischer, Robert J'],
      matchedNames: ['Fischer, Robert James'],
      gameCount: 824,
      peakElo: 2785,
      yearStart: 1955,
      yearEnd: 1992,
      isCustom: false,
    },
    {
      name: 'Magnus Carlsen',
      title: 'World Champion (2013-2023)',
      era: '2001-present',
      aliases: ['Carlsen, Magnus'],
      matchedNames: ['Carlsen, Magnus'],
      gameCount: 1420,
      peakElo: 2882,
      yearStart: 2001,
      yearEnd: 2025,
      isCustom: false,
    }
  ]),
  AddCustomNotablePlayer: vi.fn().mockResolvedValue([]),
  RemoveNotablePlayer: vi.fn().mockResolvedValue([]),
  ResetNotablePlayersToDefault: vi.fn().mockResolvedValue([]),
}))

describe('GameBrowser.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders famous players dropdown, write policy badge, and intuitive search controls', async () => {
    const wrapper = mount(GameBrowser, {
      props: {
        games: [
          {
            id: 0,
            white: 'Fischer, Robert James',
            black: 'Spassky, Boris V',
            result: '1-0',
            date: '1972.07.23',
            event: 'World Championship',
            site: 'Reykjavik',
            round: '6',
            whiteElo: 2785,
            blackElo: 2660,
            eco: 'D59',
            moves: 41,
            deleted: false,
          }
        ],
        totalGames: 1,
        currentPage: 1,
        pageSize: 50,
        dbName: 'Mega2026',
        writePolicy: 'sync_only',
        currentFen: 'rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2',
      }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Mega2026')
    expect(wrapper.text()).toContain('Sync-Only')
    expect(AppApi.GetNotablePlayersDropdown).toHaveBeenCalled()
    expect(wrapper.text()).toContain('★ Bobby Fischer (824)')
    expect(wrapper.text()).toContain('★ Magnus Carlsen (1420)')
  })

  it('filters by Famous Player when selected from dropdown', async () => {
    const wrapper = mount(GameBrowser, {
      props: {
        games: [],
        totalGames: 0,
        currentPage: 1,
        pageSize: 50,
        dbName: 'TestDB',
      }
    })
    await flushPromises()

    const select = wrapper.find('select[aria-label="Famous Players Quick Lookup"]')
    await select.setValue('Bobby Fischer')

    const emitted = wrapper.emitted('search')
    expect(emitted).toBeTruthy()
    expect(emitted![0][0]).toMatchObject({
      player: 'Bobby Fischer',
      players: ['Fischer, Robert James'],
    })
  })

  it('parses both ECO ranges and natural opening move sequences in the opening input', async () => {
    const wrapper = mount(GameBrowser, {
      props: {
        games: [],
        totalGames: 0,
        currentPage: 1,
        pageSize: 50,
        dbName: 'TestDB',
      }
    })
    await flushPromises()

    const openingInput = wrapper.findAll('input').find(i => i.attributes('placeholder')?.includes('ECO'))
    expect(openingInput).toBeDefined()
    await openingInput!.setValue('1.e4 c5 2.Nf3')

    const emitted = wrapper.emitted('search')
    expect(emitted).toBeTruthy()
    expect(emitted![0][0]).toMatchObject({
      openingMoves: ['e4', 'c5', 'Nf3'],
    })

    await openingInput!.setValue('B20-B99')
    expect(emitted![1][0]).toMatchObject({
      ecoFrom: 'B20',
      ecoTo: 'B99',
    })
  })

  it('toggles 1-click Match Board Position and emits cycleWritePolicy and openSubDbBuilder', async () => {
    const fen = 'rnbqkbnr/pp1ppppp/8/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2'
    const wrapper = mount(GameBrowser, {
      props: {
        games: [],
        totalGames: 0,
        currentPage: 1,
        pageSize: 50,
        dbName: 'TestDB',
        writePolicy: 'editable',
        currentFen: fen,
      }
    })
    await flushPromises()

    const posBtn = wrapper.findAll('button').find(b => b.text().includes('Match Board Position'))
    expect(posBtn).toBeDefined()
    await posBtn!.trigger('click')

    const emittedSearch = wrapper.emitted('search')
    expect(emittedSearch).toBeTruthy()
    expect(emittedSearch![0][0]).toMatchObject({
      positionFen: fen,
    })

    const lockBtn = wrapper.findAll('button').find(b => b.text() === 'Editable')
    await lockBtn!.trigger('click')
    expect(wrapper.emitted('cycleWritePolicy')).toBeTruthy()

    const subDbBtn = wrapper.findAll('button').find(b => b.text().includes('Extract Sub-DB'))
    await subDbBtn!.trigger('click')
    expect(wrapper.emitted('openSubDbBuilder')).toBeTruthy()
  })
})
