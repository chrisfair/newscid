import { describe, it, expect, beforeEach } from 'vitest'
import {
  sanToFAN,
  moveToLAN,
  uciToICCF,
  moveToDescriptive,
  formatMoveNotation,
  formatSequenceNotation,
  useNotation,
  NOTATION_OPTIONS
} from './notation'

describe('notation utils', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  describe('sanToFAN (Figurine Algebraic Notation)', () => {
    it('converts piece letters to unicode figurines', () => {
      expect(sanToFAN('Nf3')).toBe('♘f3')
      expect(sanToFAN('Bxf7+')).toBe('♗xf7+')
      expect(sanToFAN('Qd1')).toBe('♕d1')
      expect(sanToFAN('Rfe1')).toBe('♖fe1')
      expect(sanToFAN('Ke2')).toBe('♔e2')
    })

    it('preserves pawn moves and castling', () => {
      expect(sanToFAN('e4')).toBe('e4')
      expect(sanToFAN('exd5')).toBe('exd5')
      expect(sanToFAN('O-O')).toBe('O-O')
      expect(sanToFAN('O-O-O')).toBe('O-O-O')
      expect(sanToFAN('0-0')).toBe('0-0')
    })

    it('converts promotion figurine suffixes', () => {
      expect(sanToFAN('e8=Q#')).toBe('e8=♕#')
      expect(sanToFAN('a1=N')).toBe('a1=♘')
      expect(sanToFAN('b8=R')).toBe('b8=♖')
      expect(sanToFAN('c8=B')).toBe('c8=♗')
    })

    it('handles empty or blank input', () => {
      expect(sanToFAN('')).toBe('')
    })
  })

  describe('moveToLAN (Long Algebraic Notation)', () => {
    it('formats pawn quiet moves and captures', () => {
      expect(moveToLAN('e2e4', 'e4')).toBe('e2-e4')
      expect(moveToLAN('e4d5', 'exd5')).toBe('e4xd5')
    })

    it('formats piece moves with piece prefix', () => {
      expect(moveToLAN('g1f3', 'Nf3')).toBe('Ng1-f3')
      expect(moveToLAN('f1f7', 'Bxf7+')).toBe('Bf1xf7+')
      expect(moveToLAN('e1g1', 'O-O')).toBe('O-O')
      expect(moveToLAN('e1c1', 'O-O-O')).toBe('O-O-O')
    })

    it('formats promotion moves', () => {
      expect(moveToLAN('e7e8q', 'e8=Q#')).toBe('e7-e8=Q#')
    })

    it('falls back gracefully on missing or invalid UCI', () => {
      expect(moveToLAN('', 'Nf3')).toBe('Nf3')
      expect(moveToLAN('e4', '')).toBe('e4')
    })
  })

  describe('uciToICCF (ICCF Numeric Notation)', () => {
    it('converts coordinate moves to 4-digit numeric code', () => {
      // e2 (col 5, row 2) -> e4 (col 5, row 4) => 5254
      expect(uciToICCF('e2e4')).toBe('5254')
      // g1 (col 7, row 1) -> f3 (col 6, row 3) => 7163
      expect(uciToICCF('g1f3')).toBe('7163')
      // a1 -> h8 => 1188
      expect(uciToICCF('a1h8')).toBe('1188')
    })

    it('encodes promotion piece as 5th digit (1=Q, 2=R, 3=B, 4=N)', () => {
      expect(uciToICCF('e7e8q')).toBe('57581')
      expect(uciToICCF('e7e8r')).toBe('57582')
      expect(uciToICCF('e7e8b')).toBe('57583')
      expect(uciToICCF('e7e8n')).toBe('57584')
    })

    it('handles invalid or empty UCI', () => {
      expect(uciToICCF('')).toBe('')
      expect(uciToICCF('abc')).toBe('abc')
    })
  })

  describe('moveToDescriptive (Classical English Descriptive)', () => {
    it('formats 1. e4 as P-K4 and 1... e5 as P-K4 for Black', () => {
      expect(moveToDescriptive('e2e4', 'e4', true)).toBe('P-K4')
      expect(moveToDescriptive('e7e5', 'e5', false)).toBe('P-K4')
    })

    it('formats 1. Nf3 as N-KB3', () => {
      // f3 for White is KB3
      expect(moveToDescriptive('g1f3', 'Nf3', true)).toBe('N-KB3')
    })

    it('formats captures', () => {
      expect(moveToDescriptive('e4d5', 'exd5', true)).toBe('KPxP')
      expect(moveToDescriptive('f1f7', 'Bxf7+', true)).toBe('BxP+')
    })

    it('formats castling and checks', () => {
      expect(moveToDescriptive('e1g1', 'O-O', true)).toBe('O-O')
      expect(moveToDescriptive('e1c1', 'O-O-O', true)).toBe('O-O-O')
      expect(moveToDescriptive('d1h5', 'Qh5#', true)).toBe('Q-KR5#')
    })
  })

  describe('formatMoveNotation universal dispatcher', () => {
    it('dispatches to chosen style correctly', () => {
      expect(formatMoveNotation('Nf3', 'g1f3', true, 'san')).toBe('Nf3')
      expect(formatMoveNotation('Nf3', 'g1f3', true, 'fan')).toBe('♘f3')
      expect(formatMoveNotation('Nf3', 'g1f3', true, 'lan')).toBe('Ng1-f3')
      expect(formatMoveNotation('Nf3', 'g1f3', true, 'uci')).toBe('g1f3')
      expect(formatMoveNotation('Nf3', 'g1f3', true, 'iccf')).toBe('7163')
      expect(formatMoveNotation('Nf3', 'g1f3', true, 'descriptive')).toBe('N-KB3')
    })
  })

  describe('formatSequenceNotation', () => {
    it('formats PV sequences keeping move numbering intact', () => {
      const pv = '1. e4 e5 2. Nf3'
      expect(formatSequenceNotation(pv, 'san')).toBe('1. e4 e5 2. Nf3')
      expect(formatSequenceNotation(pv, 'fan')).toBe('1. e4 e5 2. ♘f3')
      expect(formatSequenceNotation('', 'fan')).toBe('')
    })
  })

  describe('useNotation composable', () => {
    it('provides options and updates localStorage when style changes', () => {
      const { currentStyle, setNotationStyle, options } = useNotation()
      expect(options).toEqual(NOTATION_OPTIONS)

      setNotationStyle('fan')
      expect(currentStyle.value).toBe('fan')
      expect(localStorage.getItem('newscid_notation_style')).toBe('fan')

      setNotationStyle('san')
      expect(currentStyle.value).toBe('san')
    })
  })
})
