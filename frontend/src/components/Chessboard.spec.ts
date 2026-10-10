import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Chessboard from './Chessboard.vue'

describe('Chessboard.vue', () => {
  const startFEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

  it('renders standard 64 squares', () => {
    const wrapper = mount(Chessboard, {
      props: {
        fen: startFEN,
        legalMoves: ['e2e4', 'e2e3'],
      },
    })

    const squares = wrapper.findAll('.square')
    expect(squares).toHaveLength(64)
  })

  it('renders piece images in normal mode', () => {
    const wrapper = mount(Chessboard, {
      props: {
        fen: startFEN,
        blindfold: false,
      },
    })

    const pieceImages = wrapper.findAll('.piece-img')
    expect(pieceImages.length).toBeGreaterThan(0)
    expect(wrapper.findAll('.blindfold-dot')).toHaveLength(0)
  })

  it('hides piece images and shows blindfold dots in blindfold mode', () => {
    const wrapper = mount(Chessboard, {
      props: {
        fen: startFEN,
        blindfold: true,
      },
    })

    expect(wrapper.findAll('.piece-img')).toHaveLength(0)
    expect(wrapper.findAll('.blindfold-dot').length).toBe(32)
  })

  it('renders vertical evaluation gauge when showEvalBar is enabled', () => {
    const wrapper = mount(Chessboard, {
      props: {
        fen: startFEN,
        showEvalBar: true,
        evalScore: 180, // +1.8
      },
    })

    const evalBar = wrapper.find('.eval-bar')
    expect(evalBar.exists()).toBe(true)
    expect(evalBar.text()).toContain('+1.8')
  })

  it('renders SVG move arrows when arrows prop is passed', () => {
    const wrapper = mount(Chessboard, {
      props: {
        fen: startFEN,
        arrows: [
          { from: 'e2', to: 'e4', color: '#10b981' },
          { from: 'd7', to: 'd5', color: '#ef4444' },
        ],
      },
    })

    const lines = wrapper.findAll('svg.board-arrows-overlay line')
    expect(lines).toHaveLength(2)
  })

  it('highlights threat squares when threatSquares prop is passed', () => {
    const wrapper = mount(Chessboard, {
      props: {
        fen: startFEN,
        threatSquares: ['e4', 'd5'],
      },
    })

    const threatSquares = wrapper.findAll('.threat-square')
    expect(threatSquares).toHaveLength(2)
  })
})
