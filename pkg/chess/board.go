package chess

// Board represents a chessboard state with full SCID piece list compatibility.
type Board struct {
	Squares     [64]Piece
	Turn        Color
	Castling    uint8
	EPSquare    Square
	HalfMove    int
	FullMove    int
	PieceList   [2][16]Square
	PieceCount  [2]int
	PiecePos    [64]int
}

// StateHistory tracks previous board states for UndoMove.
type StateHistory struct {
	Move       Move
	Captured   Piece
	CapturedSq Square
	Castling   uint8
	EPSquare   Square
	HalfMove   int
	PieceList  [2][16]Square
	PieceCount [2]int
	PiecePos   [64]int
}

func NewBoard() *Board {
	b := &Board{
		Turn:     White,
		Castling: CastlingWhiteKing | CastlingWhiteQueen | CastlingBlackKing | CastlingBlackQueen,
		EPSquare: NoSquare,
		HalfMove: 0,
		FullMove: 1,
	}
	b.SetupInitial()
	return b
}

func (b *Board) Clear() {
	for i := range b.Squares {
		b.Squares[i] = Empty
		b.PiecePos[i] = -1
	}
	b.PieceCount[0] = 0
	b.PieceCount[1] = 0
	b.Turn = White
	b.Castling = 0
	b.EPSquare = NoSquare
	b.HalfMove = 0
	b.FullMove = 1
}

func (b *Board) SetupInitial() {
	b.Clear()
	// White back rank
	b.SetPiece(A1, WR)
	b.SetPiece(B1, WN)
	b.SetPiece(C1, WB)
	b.SetPiece(D1, WQ)
	b.SetPiece(E1, WK)
	b.SetPiece(F1, WB)
	b.SetPiece(G1, WN)
	b.SetPiece(H1, WR)
	// White pawns
	for sq := A2; sq <= H2; sq++ {
		b.SetPiece(sq, WP)
	}

	// Black back rank
	b.SetPiece(A8, BR)
	b.SetPiece(B8, BN)
	b.SetPiece(C8, BB)
	b.SetPiece(D8, BQ)
	b.SetPiece(E8, BK)
	b.SetPiece(F8, BB)
	b.SetPiece(G8, BN)
	b.SetPiece(H8, BR)
	// Black pawns
	for sq := A7; sq <= H7; sq++ {
		b.SetPiece(sq, BP)
	}

	// Setup initial SCID piece list order
	b.PieceList[White][0] = E1; b.PiecePos[E1] = 0
	b.PieceList[White][1] = A1; b.PiecePos[A1] = 1
	b.PieceList[White][2] = B1; b.PiecePos[B1] = 2
	b.PieceList[White][3] = C1; b.PiecePos[C1] = 3
	b.PieceList[White][4] = D1; b.PiecePos[D1] = 4
	b.PieceList[White][5] = F1; b.PiecePos[F1] = 5
	b.PieceList[White][6] = G1; b.PiecePos[G1] = 6
	b.PieceList[White][7] = H1; b.PiecePos[H1] = 7
	for i := 0; i < 8; i++ {
		sq := Square(int(A2) + i)
		b.PieceList[White][8+i] = sq
		b.PiecePos[sq] = 8 + i
	}
	b.PieceCount[White] = 16

	b.PieceList[Black][0] = E8; b.PiecePos[E8] = 0
	b.PieceList[Black][1] = A8; b.PiecePos[A8] = 1
	b.PieceList[Black][2] = B8; b.PiecePos[B8] = 2
	b.PieceList[Black][3] = C8; b.PiecePos[C8] = 3
	b.PieceList[Black][4] = D8; b.PiecePos[D8] = 4
	b.PieceList[Black][5] = F8; b.PiecePos[F8] = 5
	b.PieceList[Black][6] = G8; b.PiecePos[G8] = 6
	b.PieceList[Black][7] = H8; b.PiecePos[H8] = 7
	for i := 0; i < 8; i++ {
		sq := Square(int(A7) + i)
		b.PieceList[Black][8+i] = sq
		b.PiecePos[sq] = 8 + i
	}
	b.PieceCount[Black] = 16

	b.Turn = White
	b.Castling = CastlingWhiteKing | CastlingWhiteQueen | CastlingBlackKing | CastlingBlackQueen
	b.EPSquare = NoSquare
	b.HalfMove = 0
	b.FullMove = 1
}

func (b *Board) SetPiece(sq Square, p Piece) {
	if sq >= 64 {
		return
	}
	b.Squares[sq] = p
}

func (b *Board) PieceAt(sq Square) Piece {
	if sq >= 64 {
		return Empty
	}
	return b.Squares[sq]
}

func (b *Board) KingSquare(c Color) Square {
	if b.PieceCount[c] > 0 {
		return b.PieceList[c][0]
	}
	return NoSquare
}

func (b *Board) Clone() *Board {
	nb := *b
	return &nb
}

// IsSquareAttacked returns true if sq is attacked by side 'by'.
func (b *Board) IsSquareAttacked(sq Square, by Color) bool {
	if sq >= 64 {
		return false
	}
	f := sq.File()
	r := sq.Rank()

	// 1. Pawn attacks
	if by == White {
		if r > 0 {
			if f > 0 && b.Squares[MakeSquare(f-1, r-1)] == WP {
				return true
			}
			if f < 7 && b.Squares[MakeSquare(f+1, r-1)] == WP {
				return true
			}
		}
	} else {
		if r < 7 {
			if f > 0 && b.Squares[MakeSquare(f-1, r+1)] == BP {
				return true
			}
			if f < 7 && b.Squares[MakeSquare(f+1, r+1)] == BP {
				return true
			}
		}
	}

	// 2. Knight attacks
	knightOffsets := [][2]int{
		{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
		{1, -2}, {1, 2}, {2, -1}, {2, 1},
	}
	enemyKnight := WN
	if by == Black {
		enemyKnight = BN
	}
	for _, off := range knightOffsets {
		nf := f + off[0]
		nr := r + off[1]
		if nf >= 0 && nf <= 7 && nr >= 0 && nr <= 7 {
			if b.Squares[MakeSquare(nf, nr)] == enemyKnight {
				return true
			}
		}
	}

	// 3. King attacks
	enemyKing := WK
	if by == Black {
		enemyKing = BK
	}
	for df := -1; df <= 1; df++ {
		for dr := -1; dr <= 1; dr++ {
			if df == 0 && dr == 0 {
				continue
			}
			nf := f + df
			nr := r + dr
			if nf >= 0 && nf <= 7 && nr >= 0 && nr <= 7 {
				if b.Squares[MakeSquare(nf, nr)] == enemyKing {
					return true
				}
			}
		}
	}

	// 4. Sliding rays (Rook/Queen)
	enemyRook := WR
	enemyQueen := WQ
	if by == Black {
		enemyRook = BR
		enemyQueen = BQ
	}
	orthogonalDirs := [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	for _, d := range orthogonalDirs {
		cf := f + d[0]
		cr := r + d[1]
		for cf >= 0 && cf <= 7 && cr >= 0 && cr <= 7 {
			targetSq := MakeSquare(cf, cr)
			piece := b.Squares[targetSq]
			if piece != Empty {
				if piece == enemyRook || piece == enemyQueen {
					return true
				}
				break
			}
			cf += d[0]
			cr += d[1]
		}
	}

	// 5. Sliding rays (Bishop/Queen)
	enemyBishop := WB
	if by == Black {
		enemyBishop = BB
	}
	diagonalDirs := [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for _, d := range diagonalDirs {
		cf := f + d[0]
		cr := r + d[1]
		for cf >= 0 && cf <= 7 && cr >= 0 && cr <= 7 {
			targetSq := MakeSquare(cf, cr)
			piece := b.Squares[targetSq]
			if piece != Empty {
				if piece == enemyBishop || piece == enemyQueen {
					return true
				}
				break
			}
			cf += d[0]
			cr += d[1]
		}
	}

	return false
}

// InCheck returns true if the king of side 'c' is in check.
func (b *Board) InCheck(c Color) bool {
	ksq := b.KingSquare(c)
	if ksq == NoSquare {
		return false
	}
	return b.IsSquareAttacked(ksq, c.Other())
}

// MakeMove updates the board with m and returns StateHistory.
func (b *Board) MakeMove(m Move) StateHistory {
	fromPiece := b.Squares[m.From]
	toPiece := b.Squares[m.To]

	history := StateHistory{
		Move:       m,
		Captured:   toPiece,
		CapturedSq: m.To,
		Castling:   b.Castling,
		EPSquare:   b.EPSquare,
		HalfMove:   b.HalfMove,
		PieceList:  b.PieceList,
		PieceCount: b.PieceCount,
		PiecePos:   b.PiecePos,
	}

	b.HalfMove++
	if b.Turn == Black {
		b.FullMove++
	}

	// Reset en passant
	b.EPSquare = NoSquare

	// Handle En Passant capture
	if fromPiece.Type() == Pawn && m.To == history.EPSquare && history.EPSquare != NoSquare {
		var capturedPawnSq Square
		if b.Turn == White {
			capturedPawnSq = MakeSquare(m.To.File(), m.To.Rank()-1)
		} else {
			capturedPawnSq = MakeSquare(m.To.File(), m.To.Rank()+1)
		}
		history.Captured = b.Squares[capturedPawnSq]
		history.CapturedSq = capturedPawnSq
		b.Squares[capturedPawnSq] = Empty
	}

	// Handle captures in piece list
	if history.Captured != Empty {
		b.HalfMove = 0
		enemy := b.Turn.Other()
		capNum := b.PiecePos[history.CapturedSq]
		if capNum >= 0 && capNum < b.PieceCount[enemy] {
			b.PieceCount[enemy]--
			lastSq := b.PieceList[enemy][b.PieceCount[enemy]]
			b.PieceList[enemy][capNum] = lastSq
			b.PiecePos[lastSq] = capNum
			b.PiecePos[history.CapturedSq] = -1
		}
	}

	// Handle 2-square pawn move (sets EP square)
	if fromPiece.Type() == Pawn {
		b.HalfMove = 0
		diff := int(m.To.Rank()) - int(m.From.Rank())
		if diff == 2 {
			b.EPSquare = MakeSquare(m.From.File(), m.From.Rank()+1)
		} else if diff == -2 {
			b.EPSquare = MakeSquare(m.From.File(), m.From.Rank()-1)
		}
	}

	// Move piece on board
	b.Squares[m.From] = Empty
	if m.Promo != None {
		b.Squares[m.To] = MakePiece(b.Turn, m.Promo)
	} else {
		b.Squares[m.To] = fromPiece
	}

	// Update mover in PieceList
	pNum := b.PiecePos[m.From]
	if pNum >= 0 && pNum < b.PieceCount[b.Turn] {
		b.PieceList[b.Turn][pNum] = m.To
		b.PiecePos[m.To] = pNum
		b.PiecePos[m.From] = -1
	}

	// Handle Castling rook move
	if fromPiece.Type() == King {
		if m.From == E1 && m.To == G1 {
			b.Squares[H1] = Empty; b.Squares[F1] = WR
			rNum := b.PiecePos[H1]
			b.PieceList[White][rNum] = F1; b.PiecePos[F1] = rNum; b.PiecePos[H1] = -1
		} else if m.From == E1 && m.To == C1 {
			b.Squares[A1] = Empty; b.Squares[D1] = WR
			rNum := b.PiecePos[A1]
			b.PieceList[White][rNum] = D1; b.PiecePos[D1] = rNum; b.PiecePos[A1] = -1
		} else if m.From == E8 && m.To == G8 {
			b.Squares[H8] = Empty; b.Squares[F8] = BR
			rNum := b.PiecePos[H8]
			b.PieceList[Black][rNum] = F8; b.PiecePos[F8] = rNum; b.PiecePos[H8] = -1
		} else if m.From == E8 && m.To == C8 {
			b.Squares[A8] = Empty; b.Squares[D8] = BR
			rNum := b.PiecePos[A8]
			b.PieceList[Black][rNum] = D8; b.PiecePos[D8] = rNum; b.PiecePos[A8] = -1
		}
		// King moved: revoke castling
		if b.Turn == White {
			b.Castling &= ^(CastlingWhiteKing | CastlingWhiteQueen)
		} else {
			b.Castling &= ^(CastlingBlackKing | CastlingBlackQueen)
		}
	}

	// Rook moved or captured: update castling
	if m.From == A1 || m.To == A1 {
		b.Castling &= ^CastlingWhiteQueen
	}
	if m.From == H1 || m.To == H1 {
		b.Castling &= ^CastlingWhiteKing
	}
	if m.From == A8 || m.To == A8 {
		b.Castling &= ^CastlingBlackQueen
	}
	if m.From == H8 || m.To == H8 {
		b.Castling &= ^CastlingBlackKing
	}

	b.Turn = b.Turn.Other()
	return history
}

// UnmakeMove reverts the board state.
func (b *Board) UnmakeMove(h StateHistory) {
	b.Squares[h.Move.From] = b.Squares[h.Move.To]
	if h.Move.Promo != None {
		b.Squares[h.Move.From] = MakePiece(b.Turn.Other(), Pawn)
	}
	b.Squares[h.Move.To] = Empty
	if h.Captured != Empty {
		b.Squares[h.CapturedSq] = h.Captured
	}

	// Revert Castling rook movement
	movedPiece := b.Squares[h.Move.From]
	if movedPiece.Type() == King {
		if h.Move.From == E1 && h.Move.To == G1 {
			b.Squares[F1] = Empty; b.Squares[H1] = WR
		} else if h.Move.From == E1 && h.Move.To == C1 {
			b.Squares[D1] = Empty; b.Squares[A1] = WR
		} else if h.Move.From == E8 && h.Move.To == G8 {
			b.Squares[F8] = Empty; b.Squares[H8] = BR
		} else if h.Move.From == E8 && h.Move.To == C8 {
			b.Squares[D8] = Empty; b.Squares[A8] = BR
		}
	}

	b.Turn = b.Turn.Other()
	if b.Turn == Black {
		b.FullMove--
	}
	b.Castling = h.Castling
	b.EPSquare = h.EPSquare
	b.HalfMove = h.HalfMove
	b.PieceList = h.PieceList
	b.PieceCount = h.PieceCount
	b.PiecePos = h.PiecePos
}
