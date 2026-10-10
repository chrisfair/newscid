import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import MoveNotation, { FlatMove } from './MoveNotation.vue'

describe('MoveNotation.vue', () => {
  const sampleMoves: FlatMove[] = [
    {
      index: 0,
      ply: 1,
      moveNumber: 1,
      isWhite: true,
      san: 'e4',
      uci: 'e2e4',
      fen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1',
      variationLevel: 0,
      nags: [1], // !
    },
    {
      index: 1,
      ply: 2,
      moveNumber: 1,
      isWhite: false,
      san: 'e5',
      uci: 'e7e5',
      fen: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2',
      variationLevel: 0,
    },
    {
      index: 2,
      ply: 3,
      moveNumber: 2,
      isWhite: true,
      san: 'Nf3',
      uci: 'g1f3',
      fen: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2',
      variationLevel: 0,
      comment: 'Standard development',
      variations: [
        {
          id: 1,
          moves: [
            {
              index: 0,
              ply: 3,
              moveNumber: 2,
              isWhite: true,
              san: 'f4',
              uci: 'f2f4',
              fen: '',
              variationLevel: 1,
            }
          ]
        }
      ]
    },
    {
      index: 3,
      ply: 4,
      moveNumber: 2,
      isWhite: false,
      san: 'Nc6',
      uci: 'b8c6',
      fen: 'r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3',
      variationLevel: 0,
    }
  ]

  it('renders move pairs with numbers and SAN notation', () => {
    const wrapper = mount(MoveNotation, {
      props: {
        moves: sampleMoves,
        currentPly: 1,
        result: '1-0',
      }
    })

    expect(wrapper.text()).toContain('1.')
    expect(wrapper.text()).toContain('e4')
    expect(wrapper.text()).toContain('e5')
    expect(wrapper.text()).toContain('2.')
    expect(wrapper.text()).toContain('Nf3')
    expect(wrapper.text()).toContain('Nc6')
    expect(wrapper.text()).toContain('1-0')
  })

  it('emits jump when clicking on a move button', async () => {
    const wrapper = mount(MoveNotation, {
      props: {
        moves: sampleMoves,
        currentPly: 1,
      }
    })

    const moveButtons = wrapper.findAll('.move-btn')
    expect(moveButtons.length).toBeGreaterThan(0)

    // Click on move 2 (e5, ply 2)
    await moveButtons[1].trigger('click')
    expect(wrapper.emitted('jump')).toBeTruthy()
    expect(wrapper.emitted('jump')![0]).toEqual([2])
  })

  it('emits navigation events when toolbar buttons are clicked', async () => {
    const wrapper = mount(MoveNotation, {
      props: {
        moves: sampleMoves,
        currentPly: 2,
        isPlaying: false,
      }
    })

    const navButtons = wrapper.findAll('.nav-btn')
    // buttons: first, prev, togglePlay, next, last
    await navButtons[0].trigger('click')
    expect(wrapper.emitted('first')).toBeTruthy()

    await navButtons[1].trigger('click')
    expect(wrapper.emitted('prev')).toBeTruthy()

    await navButtons[2].trigger('click')
    expect(wrapper.emitted('togglePlay')).toBeTruthy()

    await navButtons[3].trigger('click')
    expect(wrapper.emitted('next')).toBeTruthy()

    await navButtons[4].trigger('click')
    expect(wrapper.emitted('last')).toBeTruthy()
  })

  it('handles NAG annotations from the scorebook palette', async () => {
    const wrapper = mount(MoveNotation, {
      props: {
        moves: sampleMoves,
        currentPly: 3, // Nf3 has comment & variation
      }
    })

    // Find exclamation button '!' (NAG 1)
    const nagButtons = wrapper.findAll('.annotation-palette button')
    const goodMoveBtn = nagButtons.find(b => b.text() === '!')
    expect(goodMoveBtn).toBeDefined()

    await goodMoveBtn!.trigger('click')
    expect(wrapper.emitted('setNAG')).toBeTruthy()
    expect(wrapper.emitted('setNAG')![0]).toEqual([3, 1])
  })

  it('displays variations and emits promoteVariation and deleteVariation', async () => {
    const wrapper = mount(MoveNotation, {
      props: {
        moves: sampleMoves,
        currentPly: 3,
      }
    })

    const varLine = wrapper.find('.variation-line')
    expect(varLine.exists()).toBe(true)
    expect(varLine.text()).toContain('f4')

    // Find promote and delete buttons inside variation line
    const actionButtons = varLine.findAll('button')
    expect(actionButtons.length).toBe(2)

    await actionButtons[0].trigger('click')
    expect(wrapper.emitted('promoteVariation')).toBeTruthy()
    expect(wrapper.emitted('promoteVariation')![0]).toEqual([3, 0])

    await actionButtons[1].trigger('click')
    expect(wrapper.emitted('deleteVariation')).toBeTruthy()
    expect(wrapper.emitted('deleteVariation')![0]).toEqual([3, 0])
  })

  it('renders inline comments and opens comment editor', async () => {
    const wrapper = mount(MoveNotation, {
      props: {
        moves: sampleMoves,
        currentPly: 3,
      }
    })

    // Nf3 has comment "Standard development"
    expect(wrapper.text()).toContain('Standard development')

    // Click "Edit Comment" button
    const editCommentBtn = wrapper.findAll('.annotation-palette button').find(b => b.text().includes('Edit Comment'))
    expect(editCommentBtn).toBeDefined()
    await editCommentBtn!.trigger('click')

    // Check that comment modal is open
    const modal = wrapper.find('.comment-modal')
    expect(modal.exists()).toBe(true)

    const textarea = modal.find('textarea')
    expect(textarea.element.value).toBe('Standard development')

    await textarea.setValue('Updated brilliant development')
    const saveBtn = modal.findAll('button').find(b => b.text().includes('Save Comment'))
    await saveBtn!.trigger('click')

    expect(wrapper.emitted('setComment')).toBeTruthy()
    expect(wrapper.emitted('setComment')![0]).toEqual([3, 'Updated brilliant development'])
  })

  it('renders accuracy bar and triggers explain move', async () => {
    const wrapper = mount(MoveNotation, {
      props: {
        moves: sampleMoves,
        currentPly: 3,
        whiteAccuracy: 94.2,
        blackAccuracy: 88.5,
      }
    })

    const accuracyBar = wrapper.find('.accuracy-bar')
    expect(accuracyBar.exists()).toBe(true)
    expect(accuracyBar.text()).toContain('94.2%')
    expect(accuracyBar.text()).toContain('88.5%')

    // Find and click Explain Move button
    const explainBtn = wrapper.findAll('.annotation-palette button').find(b => b.text().includes('Explain Move'))
    expect(explainBtn).toBeDefined()
    await explainBtn!.trigger('click')

    expect(wrapper.emitted('explainMove')).toBeTruthy()
    expect(wrapper.emitted('explainMove')![0]).toEqual([3])
  })

  it('renders move classification badges and clock display', () => {
    const movesWithBadges: FlatMove[] = [
      {
        index: 0,
        ply: 1,
        moveNumber: 1,
        isWhite: true,
        san: 'e4',
        uci: 'e2e4',
        fen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1',
        variationLevel: 0,
        classification: 'brilliant',
        clock: '1:29:45',
      },
      {
        index: 1,
        ply: 2,
        moveNumber: 1,
        isWhite: false,
        san: 'e5',
        uci: 'e7e5',
        fen: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2',
        variationLevel: 0,
        classification: 'blunder',
        clock: '1:28:10',
      }
    ]

    const wrapper = mount(MoveNotation, {
      props: {
        moves: movesWithBadges,
        currentPly: 2,
      }
    })

    expect(wrapper.text()).toContain('!!')
    expect(wrapper.text()).toContain('??')
    expect(wrapper.text()).toContain('1:29:45')
    expect(wrapper.text()).toContain('1:28:10')
  })
})

