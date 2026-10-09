import { describe, it, expect } from 'vitest'
import { getPieceSVG } from './pieces'

describe('pieces.ts', () => {
  it('loads valid piece SVG URL for standard sets', () => {
    const whiteKnight = getPieceSVG('cburnett', 'w', 'N')
    expect(whiteKnight).toBeTruthy()
    expect(typeof whiteKnight).toBe('string')

    const blackQueen = getPieceSVG('merida', 'b', 'Q')
    expect(blackQueen).toBeTruthy()
    expect(typeof blackQueen).toBe('string')
  })

  it('falls back to cburnett or merida when an unknown piece set is requested', () => {
    const fallbackPiece = getPieceSVG('nonexistent_set_123', 'w', 'K')
    expect(fallbackPiece).toBeTruthy()
    expect(typeof fallbackPiece).toBe('string')
  })

  it('handles lowercase piece type input gracefully', () => {
    const pawn = getPieceSVG('cburnett', 'b', 'p')
    expect(pawn).toBeTruthy()
  })
})
