import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import Chessboard from './Chessboard.vue'
import MaterialSearchModal from './MaterialSearchModal.vue'
import EndgameTrainerModal from './EndgameTrainerModal.vue'
import LichessStudyModal from './LichessStudyModal.vue'

describe('NewScid v2.1 Frontend Features', () => {
  beforeEach(() => {
    ;(window as any).go = {
      main: {
        App: {
          SearchMaterialAndStructure: vi.fn().mockResolvedValue([
            {
              gameId: 1,
              white: 'Tal',
              black: 'Club',
              event: 'Test',
              date: '2026.01.01',
              result: '1-0',
              eco: 'C50',
              matchPly: 10,
              matchFen: 'startpos',
              summary: 'White sacrificed 3.0 pawns and won',
            },
          ]),
          GetEndgameDrills: vi.fn().mockResolvedValue([
            {
              id: 'eg-lucena',
              title: 'The Lucena Position',
              category: 'rook',
              difficulty: 'intermediate',
              objective: 'win',
              sideToPlay: 'white',
              fen: '1K1k4/1P6/8/8/8/8/r7/2R5 w - - 0 1',
              keyConcept: 'Build a bridge',
              hint: 'Rd1+',
              firstMoveUci: 'c1d1',
              firstMoveSan: 'Rd1+',
            },
          ]),
          ParseFEN: vi.fn().mockResolvedValue({ legalMoves: ['c1d1'] }),
          ImportLichessStudy: vi.fn().mockResolvedValue({
            studyId: 'AbCd1234',
            studyUrl: 'https://lichess.org/study/AbCd1234',
            totalChapters: 1,
            importedCount: 1,
            chapters: [
              { chapterIndex: 1, title: 'Chapter 1', white: 'A', black: 'B', fen: '', plyCount: 10, pgn: '1. e4 e5 *', gameId: 1 },
            ],
          }),
        },
      },
    }
  })

  it('supports progressive blindfold levels and peek button on Chessboard', async () => {
    const wrapper = mount(Chessboard, {
      props: {
        fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
        blindfoldLevel: 'pawns',
      },
    })
    // 16 pawns hidden as dots, 16 pieces visible as img
    expect(wrapper.findAll('.blindfold-dot').length).toBe(16)
    expect(wrapper.findAll('.piece-img').length).toBe(16)
    const peekBtn = wrapper.find('button')
    expect(peekBtn.text()).toContain('Peek (pawns)')
    await peekBtn.trigger('click')
    expect(wrapper.findAll('.piece-img').length).toBe(32)
  })

  it('renders MaterialSearchModal and runs search', async () => {
    const wrapper = mount(MaterialSearchModal, {
      props: { show: true, dbName: 'MasterDB' },
    })
    const scanBtn = wrapper.findAll('button').find(b => b.text().includes('Scan Database'))
    expect(scanBtn).toBeTruthy()
    await scanBtn!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Tal vs Club')
  })

  it('renders EndgameTrainerModal and loads drills', async () => {
    const wrapper = mount(EndgameTrainerModal, {
      props: { show: false },
    })
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(wrapper.text()).toContain('The Lucena Position')
  })

  it('renders LichessStudyModal and fetches chapters', async () => {
    const wrapper = mount(LichessStudyModal, {
      props: { show: true, dbName: 'MasterDB' },
    })
    await wrapper.find('input[type="text"]').setValue('https://lichess.org/study/AbCd1234')
    const fetchBtn = wrapper.findAll('button').find(b => b.text().includes('Fetch Study'))
    await fetchBtn!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Chapter 1')
  })
})
