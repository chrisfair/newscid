package academy

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"newscid/pkg/chess"
)

// CandidateMove represents a UCI engine Multi-PV line evaluated for personality selection.
type CandidateMove struct {
	UCI     string `json:"uci"`
	ScoreCp int    `json:"scoreCp"`
	Mate    int    `json:"mate"`
}

// PersonalityMoveResult is the move chosen by the Personality Middleware.
type PersonalityMoveResult struct {
	UCI        string `json:"uci"`
	SAN        string `json:"san"`
	FromBook   bool   `json:"fromBook"`
	Commentary string `json:"commentary"`
}

// SquareControlInfo describes attack/defense balance on a single square for the Heatmap.
type SquareControlInfo struct {
	Square         string `json:"square"`
	WhiteAttackers int    `json:"whiteAttackers"`
	BlackAttackers int    `json:"blackAttackers"`
	NetControl     int    `json:"netControl"` // WhiteAttackers - BlackAttackers
	Piece          string `json:"piece,omitempty"`
	IsHanging      bool   `json:"isHanging"`
}

// SpyThreatInfo describes the opponent's sharpest immediate threat (PyChess Spy Mode).
type SpyThreatInfo struct {
	HasThreat bool   `json:"hasThreat"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	MoveUCI   string `json:"moveUci,omitempty"`
	MoveSAN   string `json:"moveSan,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Severity  string `json:"severity,omitempty"` // "mate", "high", "medium"
}

// CoachReport is an offline natural-language post-game summary linking mistakes to University lessons.
type CoachReport struct {
	Headline           string   `json:"headline"`
	OpeningNote        string   `json:"openingNote"`
	MiddlegameNote     string   `json:"middlegameNote"`
	EndgameNote        string   `json:"endgameNote"`
	KeyMistakes        []string `json:"keyMistakes"`
	Strengths          []string `json:"strengths"`
	RecommendedLessons []string `json:"recommendedLessons"`
}

var pieceVal = [7]int{
	chess.None:   0,
	chess.King:   0,
	chess.Queen:  9,
	chess.Rook:   5,
	chess.Bishop: 3,
	chess.Knight: 3,
	chess.Pawn:   1,
}

func sqName(sq chess.Square) string {
	return string([]byte{byte('a' + sq%8), byte('1' + sq/8)})
}

// SelectPersonalityMove chooses a move matching the personality's opening book and sliders.
// Works with Multi-PV candidates from any UCI engine, or falls back to legal move heuristics if empty.
func SelectPersonalityMove(fen string, p Personality, candidates []CandidateMove, rng *rand.Rand) (PersonalityMoveResult, error) {
	board, err := chess.ParseFEN(fen)
	if err != nil {
		return PersonalityMoveResult{}, fmt.Errorf("invalid FEN: %w", err)
	}
	legal := board.GenerateLegalMoves()
	if len(legal) == 0 {
		return PersonalityMoveResult{}, fmt.Errorf("no legal moves available")
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(42))
	}

	// 1. Check personality opening book first.
	boardKey := strings.Split(strings.TrimSpace(fen), " ")[0]
	if bookMoves, ok := p.OpeningBook[boardKey]; ok && len(bookMoves) > 0 {
		validBook := make([]chess.Move, 0, len(bookMoves))
		for _, uci := range bookMoves {
			if m, err := board.ParseUCI(uci); err == nil {
				for _, lm := range legal {
					if lm == m {
						validBook = append(validBook, m)
						break
					}
				}
			}
		}
		if len(validBook) > 0 {
			chosen := validBook[rng.Intn(len(validBook))]
			return PersonalityMoveResult{
				UCI:        chosen.String(),
				SAN:        board.MoveToSAN(chosen),
				FromBook:   true,
				Commentary: fmt.Sprintf("%s plays a favorite book move (%s).", p.Name, board.MoveToSAN(chosen)),
			}, nil
		}
	}

	// 2. Build scored candidate list from UCI Multi-PV or fallback legal moves.
	type scoredMove struct {
		move  chess.Move
		score float64
		san   string
	}
	var pool []scoredMove

	if len(candidates) > 0 {
		for _, c := range candidates {
			m, err := board.ParseUCI(c.UCI)
			if err != nil {
				continue
			}
			base := float64(c.ScoreCp)
			if c.Mate > 0 {
				base = 10000.0 - float64(c.Mate*10)
			} else if c.Mate < 0 {
				base = -10000.0 - float64(c.Mate*10)
			}
			bonus := styleBonus(board, m, p.Sliders)
			pool = append(pool, scoredMove{move: m, score: base + bonus, san: board.MoveToSAN(m)})
		}
	}

	if len(pool) == 0 {
		for _, m := range legal {
			base := staticMoveHeuristic(board, m)
			bonus := styleBonus(board, m, p.Sliders)
			pool = append(pool, scoredMove{move: m, score: base + bonus, san: board.MoveToSAN(m)})
		}
	}

	sort.Slice(pool, func(i, j int) bool {
		return pool[i].score > pool[j].score
	})

	// Apply Elo-calibrated human inaccuracy sampling based on BlunderRate.
	pickIdx := 0
	if len(pool) > 1 && p.Sliders.BlunderRate > 0 {
		roll := rng.Intn(100)
		if roll < p.Sliders.BlunderRate/2 {
			maxPick := min(len(pool), 2+p.Sliders.BlunderRate/25)
			pickIdx = rng.Intn(maxPick)
		}
	}

	chosen := pool[pickIdx]
	return PersonalityMoveResult{
		UCI:        chosen.move.String(),
		SAN:        chosen.san,
		FromBook:   false,
		Commentary: fmt.Sprintf("%s (%s) chooses %s.", p.Name, p.StyleTag, chosen.san),
	}, nil
}

func styleBonus(b *chess.Board, m chess.Move, s PersonalitySliders) float64 {
	clone := b.Clone()
	captured := b.Squares[m.To]
	moving := b.Squares[m.From]
	clone.MakeMove(m)

	var bonus float64
	if clone.IsCheckmate() {
		return 5000.0
	}
	if clone.InCheck(clone.Turn) {
		bonus += float64(s.Aggression-50) * 0.9
	}
	if captured.Type() != chess.None {
		if captured.Type() == moving.Type() {
			bonus += float64(s.TradeTendency-50) * 1.1
		} else {
			bonus += float64(s.Aggression-50) * 0.6
		}
	}
	// Reward castling for solid players
	if moving.Type() == chess.King && (m.From-m.To == 2 || m.To-m.From == 2) {
		bonus += float64(s.PositionalSolidity) * 0.5
	}
	return bonus
}

func staticMoveHeuristic(b *chess.Board, m chess.Move) float64 {
	score := 0.0
	captured := b.Squares[m.To]
	moving := b.Squares[m.From]
	if captured.Type() != chess.None {
		score += float64(pieceVal[captured.Type()]*100 - pieceVal[moving.Type()]*15)
	}
	if m.Promo != chess.None {
		score += float64(pieceVal[m.Promo] * 90)
	}
	file := int(m.To % 8)
	rank := int(m.To / 8)
	if file >= 2 && file <= 5 && rank >= 2 && rank <= 5 {
		score += 15.0
	}
	clone := b.Clone()
	clone.MakeMove(m)
	if clone.IsCheckmate() {
		score += 10000.0
	} else if clone.InCheck(clone.Turn) {
		score += 35.0
	}
	return score
}

// ComputeSquareControl calculates attacker/defender counts for all 64 squares
// and flags hanging pieces for the Square Control Heatmap.
func ComputeSquareControl(fen string) ([]SquareControlInfo, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, fmt.Errorf("invalid FEN: %w", err)
	}

	whiteAtk := countAttacks(b, chess.White)
	blackAtk := countAttacks(b, chess.Black)

	out := make([]SquareControlInfo, 0, 64)
	for sq := chess.Square(0); sq < 64; sq++ {
		p := b.Squares[sq]
		wA := whiteAtk[sq]
		bA := blackAtk[sq]
		hanging := false
		pieceStr := ""
		if p != chess.Empty {
			colorPrefix := "w"
			if p.Color() == chess.Black {
				colorPrefix = "b"
			}
			pieceStr = colorPrefix + pieceChar(p.Type())
			if p.Type() != chess.King {
				if p.Color() == chess.White && bA > 0 && wA == 0 {
					hanging = true
				} else if p.Color() == chess.Black && wA > 0 && bA == 0 {
					hanging = true
				}
			}
		}
		out = append(out, SquareControlInfo{
			Square:         sqName(sq),
			WhiteAttackers: wA,
			BlackAttackers: bA,
			NetControl:     wA - bA,
			Piece:          pieceStr,
			IsHanging:      hanging,
		})
	}
	return out, nil
}

// ComputeSpyThreat runs a fast null-move threat scan to show what the opponent
// would play immediately if it were their turn (PyChess Spy Mode).
func ComputeSpyThreat(fen string) (SpyThreatInfo, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return SpyThreatInfo{}, fmt.Errorf("invalid FEN: %w", err)
	}
	if b.InCheck(b.Turn) {
		return SpyThreatInfo{
			HasThreat: false,
			Reason:    "You are already in check—resolve the check first.",
		}, nil
	}

	// Flip turn (null-move) on a clone to see opponent's immediate threat.
	clone := b.Clone()
	clone.Turn = b.Turn.Other()
	clone.EPSquare = chess.NoSquare
	oppMoves := clone.GenerateLegalMoves()
	if len(oppMoves) == 0 {
		return SpyThreatInfo{HasThreat: false}, nil
	}

	bestScore := -1.0
	var bestMove chess.Move
	bestReason := ""
	bestSeverity := "medium"

	for _, m := range oppMoves {
		after := clone.Clone()
		after.MakeMove(m)
		captured := clone.Squares[m.To]
		moving := clone.Squares[m.From]

		score := 0.0
		reason := ""
		severity := "medium"

		if after.IsCheckmate() {
			score = 10000.0
			reason = fmt.Sprintf("Immediate checkmate threat with %s!", clone.MoveToSAN(m))
			severity = "mate"
		} else if captured.Type() != chess.None {
			val := pieceVal[captured.Type()]
			// Check if target is undefended or a winning capture
			defenders := countAttacks(clone, b.Turn)[m.To]
			if defenders == 0 {
				score = float64(val * 120)
				reason = fmt.Sprintf("Threatens to win your undefended piece on %s with %s.", sqName(m.To), clone.MoveToSAN(m))
				severity = "high"
			} else if val > pieceVal[moving.Type()] {
				score = float64((val - pieceVal[moving.Type()]) * 110)
				reason = fmt.Sprintf("Threatens a favorable capture (%s) on %s.", clone.MoveToSAN(m), sqName(m.To))
				severity = "high"
			}
		} else if after.InCheck(after.Turn) {
			score = 95.0
			reason = fmt.Sprintf("Threatens a disruptive check with %s.", clone.MoveToSAN(m))
			severity = "medium"
		}

		if score > bestScore {
			bestScore = score
			bestMove = m
			bestReason = reason
			bestSeverity = severity
		}
	}

	if bestScore < 90.0 {
		return SpyThreatInfo{
			HasThreat: false,
			Reason:    "No immediate forcing threat detected.",
		}, nil
	}

	return SpyThreatInfo{
		HasThreat: true,
		From:      sqName(bestMove.From),
		To:        sqName(bestMove.To),
		MoveUCI:   bestMove.String(),
		MoveSAN:   clone.MoveToSAN(bestMove),
		Reason:    bestReason,
		Severity:  bestSeverity,
	}, nil
}

// DiagnoseGameForCoach produces an offline natural-language coach summary from a PGN
// and links detected weaknesses directly to Chess University lesson IDs.
func DiagnoseGameForCoach(pgnText, playerColor string) (CoachReport, error) {
	game, err := chess.ParsePGN(pgnText)
	if err != nil {
		return CoachReport{}, fmt.Errorf("invalid PGN: %w", err)
	}

	wantColor := chess.White
	if strings.EqualFold(strings.TrimSpace(playerColor), "black") {
		wantColor = chess.Black
	}

	b := chess.NewBoard()
	if fen, ok := game.Tags["FEN"]; ok && strings.TrimSpace(fen) != "" {
		if parsed, err := chess.ParseFEN(fen); err == nil {
			b = parsed
		}
	}

	ply := 0
	castled := false
	earlyQueenMoves := 0
	hangingEvents := 0
	checksGiven := 0
	var mistakes []string
	lessonSet := map[string]bool{}

	for node := game.RootMove; node != nil; node = node.Next {
		ply++
		mover := b.Turn
		m := node.Move
		movingPiece := b.Squares[m.From]
		san := b.MoveToSAN(m)

		if mover == wantColor {
			if movingPiece.Type() == chess.King && (m.From-m.To == 2 || m.To-m.From == 2) {
				castled = true
			}
			if ply <= 10 && movingPiece.Type() == chess.Queen {
				earlyQueenMoves++
			}
		}

		b.MakeMove(m)

		if mover == wantColor {
			if b.InCheck(b.Turn) {
				checksGiven++
			}
			// Did we leave the moved piece hanging?
			oppAtk := countAttacks(b, mover.Other())[m.To]
			ourDef := countAttacks(b, mover)[m.To]
			if oppAtk > 0 && ourDef == 0 && movingPiece.Type() != chess.Pawn && movingPiece.Type() != chess.King {
				hangingEvents++
				if len(mistakes) < 3 {
					mistakes = append(mistakes, fmt.Sprintf("Move %d (%s): left the %s undefended on %s.", (ply+1)/2, san, strings.ToLower(pieceCharName(movingPiece.Type())), sqName(m.To)))
				}
				lessonSet["t1_piece_safety"] = true
			}
		}
	}

	var strengths []string
	if castled {
		strengths = append(strengths, "Good king safety: you castled your king out of the center.")
	} else if ply >= 16 {
		mistakes = append(mistakes, "You kept your king uncastled into the middlegame.")
		lessonSet["t1_opening_principles"] = true
	}
	if earlyQueenMoves >= 2 {
		mistakes = append(mistakes, "Moving the queen multiple times in the opening slowed your minor-piece development.")
		lessonSet["t1_opening_principles"] = true
	}
	if checksGiven >= 2 {
		strengths = append(strengths, fmt.Sprintf("Active initiative: you generated %d forcing checks.", checksGiven))
	}
	if hangingEvents == 0 && ply >= 12 {
		strengths = append(strengths, "Clean board vision: you avoided leaving minor/major pieces completely undefended.")
	}

	if len(lessonSet) == 0 {
		if ply >= 40 {
			lessonSet["t4_lucena_philidor"] = true
		} else {
			lessonSet["t2_forks_pins"] = true
		}
	}

	var recLessons []string
	for id := range lessonSet {
		recLessons = append(recLessons, id)
	}
	sort.Strings(recLessons)

	headline := "Solid game! Keep sharpening your tactical calculation and endgame technique."
	if hangingEvents > 0 {
		headline = fmt.Sprintf("Focus on piece safety: %d undefended piece oversight(s) shaped this game.", hangingEvents)
	} else if !castled && ply >= 16 {
		headline = "Prioritize early castling and development before launching flank attacks."
	}

	openingNote := "Opening development looked clean."
	if earlyQueenMoves >= 2 || (!castled && ply >= 16) {
		openingNote = "Focus on developing knights/bishops and castling before moving the queen repeatedly."
	}
	middlegameNote := "Watch for forks, pins, and undefended pieces during tactical exchanges."
	if hangingEvents == 0 {
		middlegameNote = "You kept your pieces well-coordinated through the middlegame."
	}
	endgameNote := "Game ended before a deep endgame."
	if ply >= 40 {
		endgameNote = "Reached a real endgame—review King Opposition and Lucena/Philidor technique."
	}

	return CoachReport{
		Headline:           headline,
		OpeningNote:        openingNote,
		MiddlegameNote:     middlegameNote,
		EndgameNote:        endgameNote,
		KeyMistakes:        mistakes,
		Strengths:          strengths,
		RecommendedLessons: recLessons,
	}, nil
}

func countAttacks(b *chess.Board, by chess.Color) [64]int {
	var counts [64]int
	for sq := chess.Square(0); sq < 64; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty || p.Color() != by {
			continue
		}
		f := int(sq % 8)
		r := int(sq / 8)
		switch p.Type() {
		case chess.Pawn:
			dir := 1
			if by == chess.Black {
				dir = -1
			}
			nr := r + dir
			if nr >= 0 && nr < 8 {
				if f-1 >= 0 {
					counts[nr*8+(f-1)]++
				}
				if f+1 < 8 {
					counts[nr*8+(f+1)]++
				}
			}
		case chess.Knight:
			deltas := [8][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
			for _, d := range deltas {
				nf, nr := f+d[0], r+d[1]
				if nf >= 0 && nf < 8 && nr >= 0 && nr < 8 {
					counts[nr*8+nf]++
				}
			}
		case chess.King:
			for df := -1; df <= 1; df++ {
				for dr := -1; dr <= 1; dr++ {
					if df == 0 && dr == 0 {
						continue
					}
					nf, nr := f+df, r+dr
					if nf >= 0 && nf < 8 && nr >= 0 && nr < 8 {
						counts[nr*8+nf]++
					}
				}
			}
		case chess.Bishop, chess.Rook, chess.Queen:
			var dirs [][2]int
			if p.Type() == chess.Bishop || p.Type() == chess.Queen {
				dirs = append(dirs, [2]int{-1, -1}, [2]int{-1, 1}, [2]int{1, -1}, [2]int{1, 1})
			}
			if p.Type() == chess.Rook || p.Type() == chess.Queen {
				dirs = append(dirs, [2]int{-1, 0}, [2]int{1, 0}, [2]int{0, -1}, [2]int{0, 1})
			}
			for _, d := range dirs {
				nf, nr := f+d[0], r+d[1]
				for nf >= 0 && nf < 8 && nr >= 0 && nr < 8 {
					target := chess.Square(nr*8 + nf)
					counts[target]++
					if b.Squares[target] != chess.Empty {
						break
					}
					nf += d[0]
					nr += d[1]
				}
			}
		}
	}
	return counts
}

var pieceCharTable = [7]string{
	chess.Pawn:   "P",
	chess.Knight: "N",
	chess.Bishop: "B",
	chess.Rook:   "R",
	chess.Queen:  "Q",
	chess.King:   "K",
}

var pieceNameTable = [7]string{
	chess.None:   "Piece",
	chess.Pawn:   "Pawn",
	chess.Knight: "Knight",
	chess.Bishop: "Bishop",
	chess.Rook:   "Rook",
	chess.Queen:  "Queen",
	chess.King:   "King",
}

func pieceChar(pt chess.PieceType) string {
	if int(pt) < len(pieceCharTable) {
		return pieceCharTable[pt]
	}
	return ""
}

func pieceCharName(pt chess.PieceType) string {
	if int(pt) < len(pieceNameTable) {
		return pieceNameTable[pt]
	}
	return "Piece"
}
