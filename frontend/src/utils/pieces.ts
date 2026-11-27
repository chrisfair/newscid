// Piece SVG loader using Vite eager asset glob (Vite 5+ query/import syntax)
const pieceSVGs = import.meta.glob('../assets/pieces/**/*.svg', {
	eager: true,
	query: '?url',
	import: 'default',
}) as Record<string, string>

export function getPieceSVG(setName: string, color: 'w' | 'b', pieceType: string): string {
	// Standardize pieceType: K, Q, R, B, N, P
	const p = pieceType.toUpperCase()
	const key = `../assets/pieces/${setName}/${color}${p}.svg`

	if (pieceSVGs[key]) {
		return pieceSVGs[key]
	}

	// Fallback to cburnett if requested set does not have piece
	const fallbackKey = `../assets/pieces/cburnett/${color}${p}.svg`
	if (pieceSVGs[fallbackKey]) {
		return pieceSVGs[fallbackKey]
	}

	// Fallback to merida
	const meridaKey = `../assets/pieces/merida/${color}${p}.svg`
	if (pieceSVGs[meridaKey]) {
		return pieceSVGs[meridaKey]
	}

	return ''
}
