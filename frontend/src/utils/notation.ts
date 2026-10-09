import { ref, computed } from 'vue'

export type NotationStyle = 'san' | 'fan' | 'lan' | 'descriptive' | 'uci' | 'iccf'

export interface NotationOption {
  id: NotationStyle
  name: string
  example: string
  description: string
}

export const NOTATION_OPTIONS: NotationOption[] = [
  { id: 'san', name: 'Standard Algebraic (SAN)', example: 'Nf3, e4, Bxf7+', description: 'Standard FIDE algebraic notation' },
  { id: 'fan', name: 'Figurine Algebraic (FAN)', example: '♘f3, e4, ♗xf7+', description: 'Symbol-based algebraic with Unicode piece glyphs' },
  { id: 'lan', name: 'Long Algebraic (LAN)', example: 'Ng1-f3, e2-e4, Bf1xf7+', description: 'Start and destination coordinates' },
  { id: 'descriptive', name: 'Classical Descriptive', example: 'N-KB3, P-K4, BxP+', description: 'Traditional English descriptive notation' },
  { id: 'uci', name: 'Computer Syntax (UCI)', example: 'g1f3, e2e4, f1f7', description: 'Engine coordinate syntax' },
  { id: 'iccf', name: 'ICCF Numeric', example: '7163, 5254, 6167', description: 'Telegraphic numerical correspondence notation' },
]

const storedStyle = (localStorage.getItem('newscid_notation_style') as NotationStyle) || 'san'
const currentStyle = ref<NotationStyle>(storedStyle)

export function useNotation() {
  function setNotationStyle(style: NotationStyle) {
    currentStyle.value = style
    localStorage.setItem('newscid_notation_style', style)
  }

  return {
    currentStyle: computed(() => currentStyle.value),
    setNotationStyle,
    options: NOTATION_OPTIONS,
  }
}

/**
 * Convert standard algebraic SAN string to Figurine Algebraic Notation (FAN).
 */
export function sanToFAN(san: string): string {
  if (!san) return ''
  if (san.startsWith('O-O') || san.startsWith('0-0')) return san

  let res = san
  const first = res[0]
  switch (first) {
    case 'K': res = '♔' + res.slice(1); break
    case 'Q': res = '♕' + res.slice(1); break
    case 'R': res = '♖' + res.slice(1); break
    case 'B': res = '♗' + res.slice(1); break
    case 'N': res = '♘' + res.slice(1); break
  }

  // Handle promotions
  res = res.replace(/=Q/g, '=♕')
           .replace(/=R/g, '=♖')
           .replace(/=B/g, '=♗')
           .replace(/=N/g, '=♘')

  return res
}

/**
 * Convert UCI move (e.g. "g1f3", "e2e4") and SAN to Long Algebraic Notation.
 */
export function moveToLAN(uci: string, san: string): string {
  if (!uci || uci.length < 4) return san || uci
  if (san.startsWith('O-O') || san.startsWith('0-0')) return san

  const from = uci.slice(0, 2)
  const to = uci.slice(2, 4)
  const isCapture = san.includes('x')
  const sep = isCapture ? 'x' : '-'

  let piece = ''
  if (san && /^[KQRBN]/.test(san)) {
    piece = san[0]
  }

  let suffix = ''
  if (san.endsWith('#')) suffix = '#'
  else if (san.endsWith('+')) suffix = '+'

  let promo = ''
  if (uci.length >= 5) {
    promo = '=' + uci[4].toUpperCase()
  }

  return `${piece}${from}${sep}${to}${promo}${suffix}`
}

/**
 * Convert UCI move to 4-digit (or 5-digit) ICCF Numeric notation.
 */
export function uciToICCF(uci: string): string {
  if (!uci || uci.length < 4) return uci
  const colFrom = uci.charCodeAt(0) - 96 // 'a' -> 1
  const rowFrom = uci[1]
  const colTo = uci.charCodeAt(2) - 96
  const rowTo = uci[3]

  let promo = ''
  if (uci.length >= 5) {
    const p = uci[4].toLowerCase()
    if (p === 'q') promo = '1'
    else if (p === 'r') promo = '2'
    else if (p === 'b') promo = '3'
    else if (p === 'n') promo = '4'
  }

  return `${colFrom}${rowFrom}${colTo}${rowTo}${promo}`
}

/**
 * Convert UCI + SAN + mover color into English Descriptive / Classical notation.
 */
export function moveToDescriptive(uci: string, san: string, isWhite: boolean): string {
  if (!uci || uci.length < 4) return san || uci
  if (san.startsWith('O-O-O') || san.startsWith('0-0-0')) return 'O-O-O'
  if (san.startsWith('O-O') || san.startsWith('0-0')) return 'O-O'

  const fileMap = ['QR', 'QN', 'QB', 'Q', 'K', 'KB', 'KN', 'KR']
  const toFileIdx = uci.charCodeAt(2) - 97
  const targetFileDesc = (toFileIdx >= 0 && toFileIdx < 8) ? fileMap[toFileIdx] : ''
  const algRank = parseInt(uci[3], 10)
  const targetRank = isWhite ? algRank : (9 - algRank)

  let checkSuffix = ''
  if (san.endsWith('#')) checkSuffix = '#'
  else if (san.endsWith('+')) checkSuffix = '+'

  const isCapture = san.includes('x')
  const pieceChar = /^[KQRBN]/.test(san) ? san[0] : 'P'

  if (isCapture) {
    if (pieceChar === 'P') {
      const fromFileIdx = uci.charCodeAt(0) - 97
      const fromFile = fileMap[fromFileIdx] || 'P'
      return `${fromFile[0]}PxP${checkSuffix}`
    }
    // Piece capture
    return `${pieceChar}xP${checkSuffix}`
  }

  if (pieceChar === 'P') {
    return `P-${targetFileDesc}${targetRank}${checkSuffix}`
  }

  return `${pieceChar}-${targetFileDesc}${targetRank}${checkSuffix}`
}

/**
 * Universal move formatter: formats any move according to the active or specified notation style.
 */
export function formatMoveNotation(
  san: string,
  uci: string,
  isWhite: boolean = true,
  style?: NotationStyle
): string {
  const targetStyle = style || currentStyle.value

  switch (targetStyle) {
    case 'fan':
      return sanToFAN(san)
    case 'lan':
      return moveToLAN(uci, san)
    case 'descriptive':
      return moveToDescriptive(uci, san, isWhite)
    case 'uci':
      return uci || san
    case 'iccf':
      return uciToICCF(uci)
    case 'san':
    default:
      return san
  }
}

/**
 * Universal sequence / PV formatter: transforms a formatted PV string (e.g. "1. e4 e5 2. Nf3")
 * into the target notation style.
 */
export function formatSequenceNotation(pvSAN: string, style?: NotationStyle): string {
  const targetStyle = style || currentStyle.value
  if (!pvSAN || targetStyle === 'san') return pvSAN

  if (targetStyle === 'fan') {
    return sanToFAN(pvSAN)
  }

  // Tokenize and transform SAN moves
  const tokens = pvSAN.split(/\s+/)
  return tokens.map(token => {
    // If it's a move number (e.g. "1." or "1..."), keep it
    if (/^\d+\.+$/.test(token)) return token
    return formatMoveNotation(token, '', true, targetStyle)
  }).join(' ')
}
