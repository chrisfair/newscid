package chess

// GenerateLegalMoves returns all strictly legal moves for the current side to move.
func (b *Board) GenerateLegalMoves() []Move {
	pseudo := b.GeneratePseudoLegalMoves()
	legal := make([]Move, 0, len(pseudo))

	us := b.Turn
	for _, m := range pseudo {
		h := b.MakeMove(m)
		// If our king is in check after the move, it was illegal
		if !b.InCheck(us) {
			legal = append(legal, m)
		}
		b.UnmakeMove(h)
	}

	return legal
}

// GeneratePseudoLegalMoves generates all candidate moves without validating king safety.
func (b *Board) GeneratePseudoLegalMoves() []Move {
	moves := make([]Move, 0, 40)
	us := b.Turn
	them := us.Other()

	pieces := b.PieceList[us][:b.PieceCount[us]]

	for _, from := range pieces {
		p := b.Squares[from]
		if p == Empty || p.Color() != us {
			continue
		}
		pt := p.Type()

		switch pt {
		case Pawn:
			b.generatePawnMoves(from, us, &moves)
		case Knight:
			b.generateKnightMoves(from, them, &moves)
		case Bishop:
			b.generateSlidingMoves(from, them, [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}, &moves)
		case Rook:
			b.generateSlidingMoves(from, them, [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}, &moves)
		case Queen:
			b.generateSlidingMoves(from, them, [][2]int{
				{0, 1}, {0, -1}, {1, 0}, {-1, 0},
				{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
			}, &moves)
		case King:
			b.generateKingMoves(from, us, them, &moves)
		}
	}

	return moves
}

func (b *Board) generatePawnMoves(from Square, us Color, moves *[]Move) {
	f := from.File()
	r := from.Rank()

	forward := 1
	startRank := 1
	promoRank := 7
	if us == Black {
		forward = -1
		startRank = 6
		promoRank = 0
	}

	// 1 step forward
	oneStepRank := r + forward
	if oneStepRank >= 0 && oneStepRank <= 7 {
		toOne := MakeSquare(f, oneStepRank)
		if b.Squares[toOne] == Empty {
			if oneStepRank == promoRank {
				*moves = append(*moves,
					Move{From: from, To: toOne, Promo: Queen},
					Move{From: from, To: toOne, Promo: Rook},
					Move{From: from, To: toOne, Promo: Bishop},
					Move{From: from, To: toOne, Promo: Knight},
				)
			} else {
				*moves = append(*moves, Move{From: from, To: toOne})
				// 2 steps forward from starting rank
				if r == startRank {
					toTwo := MakeSquare(f, r+2*forward)
					if b.Squares[toTwo] == Empty {
						*moves = append(*moves, Move{From: from, To: toTwo})
					}
				}
			}
		}
	}

	// Captures (left and right)
	capFiles := []int{f - 1, f + 1}
	for _, cf := range capFiles {
		if cf >= 0 && cf <= 7 && oneStepRank >= 0 && oneStepRank <= 7 {
			targetSq := MakeSquare(cf, oneStepRank)
			targetPiece := b.Squares[targetSq]
			// Regular capture
			if targetPiece != Empty && targetPiece.Color() == us.Other() {
				if oneStepRank == promoRank {
					*moves = append(*moves,
						Move{From: from, To: targetSq, Promo: Queen, Capture: targetPiece.Type()},
						Move{From: from, To: targetSq, Promo: Rook, Capture: targetPiece.Type()},
						Move{From: from, To: targetSq, Promo: Bishop, Capture: targetPiece.Type()},
						Move{From: from, To: targetSq, Promo: Knight, Capture: targetPiece.Type()},
					)
				} else {
					*moves = append(*moves, Move{From: from, To: targetSq, Capture: targetPiece.Type()})
				}
			} else if targetSq == b.EPSquare && b.EPSquare != NoSquare {
				// En Passant capture
				*moves = append(*moves, Move{From: from, To: targetSq, Capture: Pawn})
			}
		}
	}
}

func (b *Board) generateKnightMoves(from Square, them Color, moves *[]Move) {
	f := from.File()
	r := from.Rank()
	knightOffsets := [][2]int{
		{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
		{1, -2}, {1, 2}, {2, -1}, {2, 1},
	}
	for _, off := range knightOffsets {
		nf := f + off[0]
		nr := r + off[1]
		if nf >= 0 && nf <= 7 && nr >= 0 && nr <= 7 {
			to := MakeSquare(nf, nr)
			targetPiece := b.Squares[to]
			if targetPiece == Empty {
				*moves = append(*moves, Move{From: from, To: to})
			} else if targetPiece.Color() == them {
				*moves = append(*moves, Move{From: from, To: to, Capture: targetPiece.Type()})
			}
		}
	}
}

func (b *Board) generateSlidingMoves(from Square, them Color, dirs [][2]int, moves *[]Move) {
	f := from.File()
	r := from.Rank()
	for _, d := range dirs {
		cf := f + d[0]
		cr := r + d[1]
		for cf >= 0 && cf <= 7 && cr >= 0 && cr <= 7 {
			to := MakeSquare(cf, cr)
			targetPiece := b.Squares[to]
			if targetPiece == Empty {
				*moves = append(*moves, Move{From: from, To: to})
			} else {
				if targetPiece.Color() == them {
					*moves = append(*moves, Move{From: from, To: to, Capture: targetPiece.Type()})
				}
				break
			}
			cf += d[0]
			cr += d[1]
		}
	}
}

func (b *Board) generateKingMoves(from Square, us, them Color, moves *[]Move) {
	f := from.File()
	r := from.Rank()

	for df := -1; df <= 1; df++ {
		for dr := -1; dr <= 1; dr++ {
			if df == 0 && dr == 0 {
				continue
			}
			nf := f + df
			nr := r + dr
			if nf >= 0 && nf <= 7 && nr >= 0 && nr <= 7 {
				to := MakeSquare(nf, nr)
				targetPiece := b.Squares[to]
				if targetPiece == Empty {
					*moves = append(*moves, Move{From: from, To: to})
				} else if targetPiece.Color() == them {
					*moves = append(*moves, Move{From: from, To: to, Capture: targetPiece.Type()})
				}
			}
		}
	}

	// Castling
	if us == White && from == E1 {
		// Kingside (E1 -> G1)
		if (b.Castling&CastlingWhiteKing) != 0 &&
			b.Squares[F1] == Empty && b.Squares[G1] == Empty &&
			!b.IsSquareAttacked(E1, them) &&
			!b.IsSquareAttacked(F1, them) &&
			!b.IsSquareAttacked(G1, them) {
			*moves = append(*moves, Move{From: E1, To: G1})
		}
		// Queenside (E1 -> C1)
		if (b.Castling&CastlingWhiteQueen) != 0 &&
			b.Squares[D1] == Empty && b.Squares[C1] == Empty && b.Squares[B1] == Empty &&
			!b.IsSquareAttacked(E1, them) &&
			!b.IsSquareAttacked(D1, them) &&
			!b.IsSquareAttacked(C1, them) {
			*moves = append(*moves, Move{From: E1, To: C1})
		}
	} else if us == Black && from == E8 {
		// Kingside (E8 -> G8)
		if (b.Castling&CastlingBlackKing) != 0 &&
			b.Squares[F8] == Empty && b.Squares[G8] == Empty &&
			!b.IsSquareAttacked(E8, them) &&
			!b.IsSquareAttacked(F8, them) &&
			!b.IsSquareAttacked(G8, them) {
			*moves = append(*moves, Move{From: E8, To: G8})
		}
		// Queenside (E8 -> C8)
		if (b.Castling&CastlingBlackQueen) != 0 &&
			b.Squares[D8] == Empty && b.Squares[C8] == Empty && b.Squares[B8] == Empty &&
			!b.IsSquareAttacked(E8, them) &&
			!b.IsSquareAttacked(D8, them) &&
			!b.IsSquareAttacked(C8, them) {
			*moves = append(*moves, Move{From: E8, To: C8})
		}
	}
}

// IsCheckmate returns true if side to move has no legal moves and is in check.
func (b *Board) IsCheckmate() bool {
	return b.InCheck(b.Turn) && len(b.GenerateLegalMoves()) == 0
}

// IsStalemate returns true if side to move has no legal moves and is NOT in check.
func (b *Board) IsStalemate() bool {
	return !b.InCheck(b.Turn) && len(b.GenerateLegalMoves()) == 0
}
