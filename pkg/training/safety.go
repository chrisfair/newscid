package training

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// Piece values in centipawns for Static Exchange Evaluation (SEE), indexed by chess.PieceType.
var pieceValues = [7]int{
	chess.None:   0,
	chess.King:   20000,
	chess.Queen:  900,
	chess.Rook:   500,
	chess.Bishop: 330,
	chess.Knight: 320,
	chess.Pawn:   100,
}

var colorLower = [2]string{
	chess.White: "white",
	chess.Black: "black",
}

var colorTitle = [2]string{
	chess.White: "White",
	chess.Black: "Black",
}

type safetyStatusID uint8

const (
	statusIDDefended safetyStatusID = iota
	statusIDLoose
	statusIDUnderdefended
	statusIDHanging
)

// PieceSafetyStatus classifies the vulnerability of a piece.
type PieceSafetyStatus string

const (
	StatusHanging       PieceSafetyStatus = "hanging"       // Attacked with 0 defenders or net positive capture (SEE > 0)
	StatusUnderdefended PieceSafetyStatus = "underdefended" // Attacked with more attackers than defenders
	StatusLoose         PieceSafetyStatus = "loose"         // 0 defenders, not currently attacked (LPDO risk)
	StatusDefended      PieceSafetyStatus = "defended"      // Adequately protected
)

var safetyStatusLabels = [4]PieceSafetyStatus{
	statusIDDefended:      StatusDefended,
	statusIDLoose:         StatusLoose,
	statusIDUnderdefended: StatusUnderdefended,
	statusIDHanging:       StatusHanging,
}

// PieceSafety represents the safety audit of a single piece on the board.
type PieceSafety struct {
	Square      string            `json:"square"`      // e.g. "e4"
	Piece       string            `json:"piece"`       // e.g. "N", "B", "R", "Q", "P", "K"
	Color       string            `json:"color"`       // "white" or "black"
	PieceValue  int               `json:"pieceValue"`  // Centipawns (e.g. 320)
	Attackers   []string          `json:"attackers"`   // Squares of attacking enemy pieces
	Defenders   []string          `json:"defenders"`   // Squares of defending friendly pieces
	Status      PieceSafetyStatus `json:"status"`      // "hanging", "underdefended", "loose", "defended"
	NetLoss     int               `json:"netLoss"`     // Expected centipawn loss if captured
	Explanation string            `json:"explanation"` // Tactical explanation
}

// BoardSafetyReport summarizes safety and hanging pieces for an entire board position.
type BoardSafetyReport struct {
	FEN               string        `json:"fen"`
	Turn              string        `json:"turn"` // "white" or "black"
	Pieces            []PieceSafety `json:"pieces"`
	HangingPieces     []PieceSafety `json:"hangingPieces"`       // Pieces with Status == "hanging"
	Underdefended     []PieceSafety `json:"underdefendedPieces"` // Pieces with Status == "underdefended"
	LoosePieces       []PieceSafety `json:"loosePieces"`         // Pieces with Status == "loose"
	HangingSquares    []string      `json:"hangingSquares"`      // Direct list of squares with hanging pieces
	TotalHangingWhite int           `json:"totalHangingWhite"`
	TotalHangingBlack int           `json:"totalHangingBlack"`
	CriticalThreat    *PieceSafety  `json:"criticalThreat,omitempty"` // Highest value hanging piece
	BlunderCount      int           `json:"blunderCount"`
}

// SafetyDrillMode defines the style of exercise.
type SafetyDrillMode string

const (
	ModeFindAllHanging   SafetyDrillMode = "find_all_hanging"   // Click all hanging pieces on board
	ModeDefendYourPieces SafetyDrillMode = "defend_your_pieces" // Spot your own hanging pieces
	ModePunishBlunder    SafetyDrillMode = "punish_blunder"    // Spot and click opponent's free pieces
	ModeCaptureHanging   SafetyDrillMode = "capture_hanging"   // Make the legal move capturing the hanging piece
)

// SafetyDrill represents an interactive challenge presented to the user.
type SafetyDrill struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	FEN            string          `json:"fen"`
	Turn           string          `json:"turn"` // "white" or "black"
	Mode           SafetyDrillMode `json:"mode"`
	Prompt         string          `json:"prompt"`
	TargetColor    string          `json:"targetColor"` // "both", "white", "black"
	CorrectSquares []string        `json:"correctSquares"`
	WinningMoveUCI string          `json:"winningMoveUCI,omitempty"`
	WinningMoveSAN string          `json:"winningMoveSAN,omitempty"`
	Explanation    string          `json:"explanation"`
	Difficulty     string          `json:"difficulty"` // "beginner", "intermediate", "advanced"
	Source         string          `json:"source"`     // "Curated", "Master Game", "Database Mine"
}

// SafetyDrillFeedback contains the evaluation result of the user's attempt.
type SafetyDrillFeedback struct {
	IsCorrect      bool     `json:"isCorrect"`
	Score          int      `json:"score"`
	CorrectSquares []string `json:"correctSquares"`
	MissedSquares  []string `json:"missedSquares"`
	WrongSquares   []string `json:"wrongSquares"`
	Explanation    string   `json:"explanation"`
	BestMoveUCI    string   `json:"bestMoveUCI,omitempty"`
	BestMoveSAN    string   `json:"bestMoveSAN,omitempty"`
}

// FindAttackers finds all squares containing pieces of color 'by' attacking 'sq'.
func FindAttackers(b *chess.Board, sq chess.Square, by chess.Color) []chess.Square {
	if sq >= 64 {
		return nil
	}
	var attackers []chess.Square
	f := sq.File()
	r := sq.Rank()

	// Pawn attacks
	if by == chess.White {
		if r > 0 {
			if f > 0 && b.Squares[chess.MakeSquare(f-1, r-1)] == chess.WP {
				attackers = append(attackers, chess.MakeSquare(f-1, r-1))
			}
			if f < 7 && b.Squares[chess.MakeSquare(f+1, r-1)] == chess.WP {
				attackers = append(attackers, chess.MakeSquare(f+1, r-1))
			}
		}
	} else {
		if r < 7 {
			if f > 0 && b.Squares[chess.MakeSquare(f-1, r+1)] == chess.BP {
				attackers = append(attackers, chess.MakeSquare(f-1, r+1))
			}
			if f < 7 && b.Squares[chess.MakeSquare(f+1, r+1)] == chess.BP {
				attackers = append(attackers, chess.MakeSquare(f+1, r+1))
			}
		}
	}

	// Knight attacks
	knightOffsets := [][2]int{
		{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
		{1, -2}, {1, 2}, {2, -1}, {2, 1},
	}
	enemyKnight := chess.WN
	if by == chess.Black {
		enemyKnight = chess.BN
	}
	for _, off := range knightOffsets {
		nf := f + off[0]
		nr := r + off[1]
		if nf >= 0 && nf <= 7 && nr >= 0 && nr <= 7 {
			nsq := chess.MakeSquare(nf, nr)
			if b.Squares[nsq] == enemyKnight {
				attackers = append(attackers, nsq)
			}
		}
	}

	// King attacks
	kingOffsets := [][2]int{
		{-1, -1}, {-1, 0}, {-1, 1},
		{0, -1}, {0, 1},
		{1, -1}, {1, 0}, {1, 1},
	}
	enemyKing := chess.WK
	if by == chess.Black {
		enemyKing = chess.BK
	}
	for _, off := range kingOffsets {
		nf := f + off[0]
		nr := r + off[1]
		if nf >= 0 && nf <= 7 && nr >= 0 && nr <= 7 {
			ksq := chess.MakeSquare(nf, nr)
			if b.Squares[ksq] == enemyKing {
				attackers = append(attackers, ksq)
			}
		}
	}

	// Diagonal sliders (Bishops and Queens)
	diagDirs := [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	enemyB := chess.WB
	enemyQ := chess.WQ
	if by == chess.Black {
		enemyB = chess.BB
		enemyQ = chess.BQ
	}
	for _, dir := range diagDirs {
		for step := 1; step < 8; step++ {
			nf := f + dir[0]*step
			nr := r + dir[1]*step
			if nf < 0 || nf > 7 || nr < 0 || nr > 7 {
				break
			}
			tsq := chess.MakeSquare(nf, nr)
			p := b.Squares[tsq]
			if p != chess.Empty {
				if p == enemyB || p == enemyQ {
					attackers = append(attackers, tsq)
				}
				break
			}
		}
	}

	// Orthogonal sliders (Rooks and Queens)
	orthoDirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	enemyR := chess.WR
	if by == chess.Black {
		enemyR = chess.BR
	}
	for _, dir := range orthoDirs {
		for step := 1; step < 8; step++ {
			nf := f + dir[0]*step
			nr := r + dir[1]*step
			if nf < 0 || nf > 7 || nr < 0 || nr > 7 {
				break
			}
			tsq := chess.MakeSquare(nf, nr)
			p := b.Squares[tsq]
			if p != chess.Empty {
				if p == enemyR || p == enemyQ {
					attackers = append(attackers, tsq)
				}
				break
			}
		}
	}

	return attackers
}

// FindDefenders finds all friendly pieces of color 'by' defending 'sq'.
func FindDefenders(b *chess.Board, sq chess.Square, by chess.Color) []chess.Square {
	if sq >= 64 {
		return nil
	}
	targetPiece := b.Squares[sq]
	// Temporarily remove or set dummy enemy piece on sq so friendly sliding pieces can aim through/at sq
	dummyEnemy := chess.BP
	if by == chess.Black {
		dummyEnemy = chess.WP
	}
	b.Squares[sq] = dummyEnemy

	// Find all pieces of color 'by' attacking that square
	defenders := FindAttackers(b, sq, by)

	// Restore original piece
	b.Squares[sq] = targetPiece
	return defenders
}

// CalculateSEE performs Static Exchange Evaluation on 'toSq' starting with the capture from 'fromSq'.
func CalculateSEE(b *chess.Board, fromSq, toSq chess.Square) int {
	capturedPiece := b.Squares[toSq]
	if capturedPiece == chess.Empty {
		return 0
	}
	attackerPiece := b.Squares[fromSq]
	gain := pieceValues[capturedPiece.Type()]

	// If attacker piece value is lower or equal, initial gain is solid
	loss := pieceValues[attackerPiece.Type()]
	enemyColor := attackerPiece.Color().Other()

	// Defenders of toSq (from enemy perspective)
	defenders := FindDefenders(b, toSq, enemyColor)
	if len(defenders) == 0 {
		return gain
	}

	// If there are defenders, and our piece is worth more than captured, it might be a net loss
	if loss > gain {
		return gain - loss
	}

	return gain
}

// AnalyzeBoardSafety inspects all pieces on the board and produces a complete Safety Report.
func AnalyzeBoardSafety(fen string) (*BoardSafetyReport, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, fmt.Errorf("invalid FEN: %w", err)
	}

	report := &BoardSafetyReport{
		FEN:            fen,
		Turn:           colorLower[b.Turn&1],
		Pieces:         make([]PieceSafety, 0, 32),
		HangingPieces:  make([]PieceSafety, 0, 8),
		Underdefended:  make([]PieceSafety, 0, 8),
		LoosePieces:    make([]PieceSafety, 0, 8),
		HangingSquares: make([]string, 0, 8),
	}

	var maxThreatLoss = -1

	for sq := chess.A1; sq <= chess.H8; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty {
			continue
		}

		pieceType := p.Type()
		// Kings cannot be "hung" or captured, they can only be checked/mated
		if pieceType == chess.King {
			continue
		}

		color := p.Color()
		cIdx := color & 1
		colorStr := colorLower[cIdx]
		cTitle := colorTitle[cIdx]
		val := pieceValues[pieceType]
		sqStr := sq.String()
		pieceStr := pieceType.String()

		attackers := FindAttackers(b, sq, color.Other())
		defenders := FindDefenders(b, sq, color)

		var attackerSqs []string
		for _, asq := range attackers {
			attackerSqs = append(attackerSqs, asq.String())
		}
		var defenderSqs []string
		for _, dsq := range defenders {
			defenderSqs = append(defenderSqs, dsq.String())
		}

		statusID := statusIDDefended
		netLoss := 0
		explanation := fmt.Sprintf("%s %s on %s is safely defended (%d attackers, %d defenders).",
			cTitle, pieceStr, sqStr, len(attackers), len(defenders))

		if len(attackers) == 0 {
			if len(defenders) == 0 && pieceType != chess.Pawn {
				statusID = statusIDLoose
				explanation = fmt.Sprintf("%s %s on %s is loose (0 defenders) — Loose Piece Drops Off (LPDO).",
					cTitle, pieceStr, sqStr)
			}
		} else {
			// Find lowest-value attacker
			minAttackerVal := 999999
			var bestAttackerSq chess.Square
			for _, asq := range attackers {
				ap := b.Squares[asq]
				aval := pieceValues[ap.Type()]
				if aval < minAttackerVal {
					minAttackerVal = aval
					bestAttackerSq = asq
				}
			}

			// Condition 1: 0 defenders (completely free piece)
			if len(defenders) == 0 {
				statusID = statusIDHanging
				netLoss = val
				explanation = fmt.Sprintf("%s %s on %s is completely hanging with 0 defenders! Attacked by %s.",
					cTitle, pieceStr, sqStr, bestAttackerSq.String())
			} else if minAttackerVal < val {
				// Condition 2: Attacked by a piece of strictly lower value (e.g. Queen by Knight/Bishop, Rook by Minor)
				statusID = statusIDHanging
				netLoss = val - minAttackerVal
				explanation = fmt.Sprintf("%s %s on %s is attacked by lower-value %s on %s; capture wins material despite %d defender(s).",
					cTitle, pieceStr, sqStr,
					b.Squares[bestAttackerSq].Type().String(), bestAttackerSq.String(), len(defenders))
			} else if len(attackers) > len(defenders) {
				// Condition 3: Deficit in defenders vs attackers
				see := CalculateSEE(b, bestAttackerSq, sq)
				if see > 0 {
					statusID = statusIDHanging
					netLoss = see
					explanation = fmt.Sprintf("%s %s on %s has more attackers (%d) than defenders (%d); tactical liquidation wins material.",
						cTitle, pieceStr, sqStr, len(attackers), len(defenders))
				} else {
					statusID = statusIDUnderdefended
					explanation = fmt.Sprintf("%s %s on %s is underdefended (%d attackers vs %d defenders).",
						cTitle, pieceStr, sqStr, len(attackers), len(defenders))
				}
			}
		}

		ps := PieceSafety{
			Square:      sqStr,
			Piece:       pieceStr,
			Color:       colorStr,
			PieceValue:  val,
			Attackers:   attackerSqs,
			Defenders:   defenderSqs,
			Status:      safetyStatusLabels[statusID],
			NetLoss:     netLoss,
			Explanation: explanation,
		}

		report.Pieces = append(report.Pieces, ps)

		switch statusID {
		case statusIDHanging:
			report.HangingPieces = append(report.HangingPieces, ps)
			report.HangingSquares = append(report.HangingSquares, sqStr)
			if color == chess.White {
				report.TotalHangingWhite++
			} else {
				report.TotalHangingBlack++
			}
			if netLoss > maxThreatLoss {
				maxThreatLoss = netLoss
				psCopy := ps
				report.CriticalThreat = &psCopy
			}
		case statusIDUnderdefended:
			report.Underdefended = append(report.Underdefended, ps)
		case statusIDLoose:
			report.LoosePieces = append(report.LoosePieces, ps)
		}
	}

	report.BlunderCount = len(report.HangingPieces)
	return report, nil
}

// Built-in curated safety challenges from master and club games.
var builtinChallenges = []SafetyDrill{
	{
		ID:             "hang_01_queen_scandi",
		Title:          "The Overextended Queen",
		FEN:            "rnb1kbnr/ppp1pppp/8/q7/3P4/2N5/PPP2PPP/R1BQKBNR b KQkq - 1 4",
		Turn:           "black",
		Mode:           ModeDefendYourPieces,
		TargetColor:    "black",
		Prompt:         "Spot the loose Black piece that has no defenders and is exposed to discovery!",
		CorrectSquares: []string{"a5"},
		WinningMoveUCI: "c1d2",
		WinningMoveSAN: "Bd2",
		Explanation:    "The Black Queen on a5 is loose and undefended. White plays Bd2 setting up a devastating discovered attack.",
		Difficulty:     "beginner",
		Source:         "Scandinavian Defense",
	},
	{
		ID:             "hang_02_italian_hanging_knight",
		Title:          "Free Knight on e4",
		FEN:            "r1bqk2r/pppp1ppp/2n5/2b1p3/2B1n3/3P1N2/PPP2PPP/RNBQ1RK1 b kq - 0 6",
		Turn:           "white",
		Mode:           ModePunishBlunder,
		TargetColor:    "black",
		Prompt:         "Black left a minor piece en prise with 0 defenders. Click the hanging piece!",
		CorrectSquares: []string{"e4"},
		WinningMoveUCI: "d3e4",
		WinningMoveSAN: "dxe4",
		Explanation:    "Black's Knight on e4 is attacked by White's d3 pawn and defended by nothing! White wins a full piece with dxe4.",
		Difficulty:     "beginner",
		Source:         "Italian Game Blunder",
	},
	{
		ID:             "hang_03_double_hanging",
		Title:          "Dual Loose Pieces",
		FEN:            "r2q1rk1/ppp2ppp/2n1b3/3np3/2B5/3P1N2/PPP2PPP/RN1Q1RK1 w - - 0 9",
		Turn:           "white",
		Mode:           ModeFindAllHanging,
		TargetColor:    "both",
		Prompt:         "Identify every hanging or underdefended piece on the board!",
		CorrectSquares: []string{"c4", "e6"},
		Explanation:    "White's Bishop on c4 is defended only once and attacked, while Black's Bishop on e6 is under pressure.",
		Difficulty:     "intermediate",
		Source:         "Open Game Tactics",
	},
	{
		ID:             "hang_04_fork_hanging_rook",
		Title:          "Hanging Rook in the Corner",
		FEN:            "r1b1k2r/pppp1ppp/5q2/4n3/3QP3/2N5/PPP2PPP/R3KB1R b KQkq - 1 8",
		Turn:           "white",
		Mode:           ModePunishBlunder,
		TargetColor:    "black",
		Prompt:         "Find Black's hanging minor piece!",
		CorrectSquares: []string{"e5"},
		WinningMoveUCI: "d4e5",
		WinningMoveSAN: "Qxe5",
		Explanation:    "Black's Knight on e5 is attacked by the Queen on d4 and has no defenders.",
		Difficulty:     "beginner",
		Source:         "Scotch Opening",
	},
	{
		ID:             "hang_05_pinned_defender",
		Title:          "The Pinned False Defender",
		FEN:            "r1bqk2r/pp2bppp/2n1p3/3p4/3P4/2PB1N2/PP1N1PPP/R2QK2R w KQkq - 0 10",
		Turn:           "white",
		Mode:           ModeFindAllHanging,
		TargetColor:    "both",
		Prompt:         "Identify loose and vulnerable pieces across both camps!",
		CorrectSquares: []string{"c6", "e7"},
		Explanation:    "Both Black's Knight on c6 and Bishop on e7 are undefended targets.",
		Difficulty:     "intermediate",
		Source:         "French Defense Formation",
	},
	{
		ID:             "hang_06_rook_en_prise",
		Title:          "Back-Rank Blunder",
		FEN:            "5rk1/5ppp/8/8/8/8/5PPP/R3R1K1 w - - 0 1",
		Turn:           "white",
		Mode:           ModePunishBlunder,
		TargetColor:    "black",
		Prompt:         "Is Black hanging any major pieces?",
		CorrectSquares: []string{"f8"},
		WinningMoveUCI: "a1a8",
		WinningMoveSAN: "Ra8",
		Explanation:    "Black's Rook on f8 is defended only by the King and vulnerable to a back-rank skewer.",
		Difficulty:     "beginner",
		Source:         "Endgame Basics",
	},
	{
		ID:             "hang_07_hanging_bishop",
		Title:          "Unguarded Bishop",
		FEN:            "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
		Turn:           "black",
		Mode:           ModeDefendYourPieces,
		TargetColor:    "white",
		Prompt:         "White left a piece loose on an active square. Spot it!",
		CorrectSquares: []string{"c4"},
		Explanation:    "The Bishop on c4 is undefended. If Black plays Nxe4 followed by d5, White faces an immediate center fork.",
		Difficulty:     "intermediate",
		Source:         "Fork Trick Drill",
	},
	{
		ID:             "hang_08_free_queen",
		Title:          "Blundered Queen",
		FEN:            "r1b1k2r/pppp1ppp/2n5/4p3/2B1P1q1/3P1N2/PPP2PPP/RNBQK2R w KQkq - 1 7",
		Turn:           "white",
		Mode:           ModePunishBlunder,
		TargetColor:    "black",
		Prompt:         "Black's Queen ventured into enemy lines without protection. Click it!",
		CorrectSquares: []string{"g4"},
		WinningMoveUCI: "c4f7",
		WinningMoveSAN: "Bxf7+",
		Explanation:    "The Queen on g4 is loose. White can exploit this with tactical deflections or direct attacks.",
		Difficulty:     "beginner",
		Source:         "Amateur Blunder Vault",
	},
}

// GetBuiltinSafetyChallenges returns all predefined curated safety drills.
func GetBuiltinSafetyChallenges() []SafetyDrill {
	return builtinChallenges
}

// GetSafetyDrillBatch returns a randomized or filtered batch of safety challenges.
func GetSafetyDrillBatch(count int, difficulty string) []SafetyDrill {
	var pool []SafetyDrill
	for _, c := range builtinChallenges {
		if difficulty == "" || difficulty == "all" || c.Difficulty == difficulty {
			pool = append(pool, c)
		}
	}
	if len(pool) == 0 {
		pool = builtinChallenges
	}

	rand.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})

	if count > 0 && count < len(pool) {
		return pool[:count]
	}
	return pool
}

// ExtractSafetyDrillsFromDatabase scans games in the database for hanging piece positions.
func ExtractSafetyDrillsFromDatabase(db *scid.Database, maxDrills int) []SafetyDrill {
	if db == nil || !db.IsOpen || maxDrills <= 0 {
		return GetSafetyDrillBatch(maxDrills, "all")
	}

	numGames := db.NumGames()
	if numGames == 0 {
		return GetSafetyDrillBatch(maxDrills, "all")
	}

	var drills []SafetyDrill
	step := 1
	if numGames > 100 {
		step = numGames / 50
	}

	for gID := 1; gID <= numGames && len(drills) < maxDrills; gID += step {
		g, err := db.GetGame(gID)
		if err != nil || g == nil || g.RootMove == nil {
			continue
		}

		b := chess.NewBoard()
		if g.StartFEN != "" && g.StartFEN != chess.StartingFEN {
			if parsed, err := chess.ParseFEN(g.StartFEN); err == nil {
				b = parsed
			}
		}
		ply := 0
		for curr := g.RootMove; curr != nil && ply < 40; curr = curr.Next {
			b.MakeMove(curr.Move)
			ply++
			if ply < 8 {
				continue
			}

			fen := b.FEN()
			report, err := AnalyzeBoardSafety(fen)
			if err != nil || report == nil {
				continue
			}

			if len(report.HangingPieces) >= 1 && report.CriticalThreat != nil {
				threat := report.CriticalThreat
				drill := SafetyDrill{
					ID:             fmt.Sprintf("db_mine_g%d_p%d", gID, ply),
					Title:          fmt.Sprintf("Game #%d: Blunder at Move %d", gID, (ply/2)+1),
					FEN:            fen,
					Turn:           report.Turn,
					Mode:           ModeFindAllHanging,
					TargetColor:    "both",
					Prompt:         fmt.Sprintf("Identify the hanging piece (%s on %s) blundered in this position!", threat.Piece, threat.Square),
					CorrectSquares: report.HangingSquares,
					Explanation:    threat.Explanation,
					Difficulty:     "intermediate",
					Source:         fmt.Sprintf("%s vs %s", g.Tag("White"), g.Tag("Black")),
				}
				drills = append(drills, drill)
				if len(drills) >= maxDrills {
					break
				}
			}
		}
	}

	if len(drills) < maxDrills {
		remaining := maxDrills - len(drills)
		drills = append(drills, GetSafetyDrillBatch(remaining, "all")...)
	}

	return drills
}

// EvaluateSafetySubmission grades the user's submitted squares or move against the drill.
func EvaluateSafetySubmission(drill *SafetyDrill, selectedSquares []string, moveUCI string) *SafetyDrillFeedback {
	if drill == nil {
		return &SafetyDrillFeedback{IsCorrect: false, Explanation: "Drill not found"}
	}

	// Move capture mode
	if drill.WinningMoveUCI != "" && moveUCI != "" {
		isMatch := strings.EqualFold(strings.TrimSpace(drill.WinningMoveUCI), strings.TrimSpace(moveUCI))
		score := 0
		if isMatch {
			score = 100
		}
		return &SafetyDrillFeedback{
			IsCorrect:      isMatch,
			Score:          score,
			CorrectSquares: drill.CorrectSquares,
			BestMoveUCI:    drill.WinningMoveUCI,
			BestMoveSAN:    drill.WinningMoveSAN,
			Explanation:    drill.Explanation,
		}
	}

	// Square selection mode (Click all hanging pieces)
	correctSet := make(map[string]bool)
	for _, sq := range drill.CorrectSquares {
		correctSet[strings.ToLower(strings.TrimSpace(sq))] = true
	}

	selectedSet := make(map[string]bool)
	for _, sq := range selectedSquares {
		selectedSet[strings.ToLower(strings.TrimSpace(sq))] = true
	}

	var missed []string
	var wrong []string

	for sq := range correctSet {
		if !selectedSet[sq] {
			missed = append(missed, sq)
		}
	}

	for sq := range selectedSet {
		if !correctSet[sq] {
			wrong = append(wrong, sq)
		}
	}

	sort.Strings(missed)
	sort.Strings(wrong)

	isPerfect := len(missed) == 0 && len(wrong) == 0
	score := 0
	if isPerfect {
		score = 100
	} else if len(correctSet) > 0 {
		hits := len(correctSet) - len(missed)
		score = (hits * 100) / len(correctSet)
		score -= len(wrong) * 20
		if score < 0 {
			score = 0
		}
	}

	return &SafetyDrillFeedback{
		IsCorrect:      isPerfect,
		Score:          score,
		CorrectSquares: drill.CorrectSquares,
		MissedSquares:  missed,
		WrongSquares:   wrong,
		Explanation:    drill.Explanation,
		BestMoveUCI:    drill.WinningMoveUCI,
		BestMoveSAN:    drill.WinningMoveSAN,
	}
}
