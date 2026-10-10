package training

import (
	"fmt"
	"strings"

	"newscid/pkg/chess"
)

// MoveRecord holds a move and board context in a solitaire game.
type MoveRecord struct {
	Ply     int        `json:"ply"`
	Turn    string     `json:"turn"` // "white" or "black"
	SAN     string     `json:"san"`
	UCI     string     `json:"uci"`
	FEN     string     `json:"fen"`
	Comment string     `json:"comment,omitempty"`
	Move    chess.Move `json:"-"`
}

// PerformanceSummary encapsulates the training session results.
type PerformanceSummary struct {
	Score          int     `json:"score"`
	MaxScore       int     `json:"maxScore"`
	Percentage     float64 `json:"percentage"`
	AccuracyPct    float64 `json:"accuracyPct"`
	CorrectGuesses int     `json:"correctGuesses"`
	TotalTurns     int     `json:"totalTurns"`
	HintsUsed      int     `json:"hintsUsed"`
	RatingTitle    string  `json:"ratingTitle"`
	Feedback       string  `json:"feedback"`
}

// GuessFeedback represents the result of a submitted move guess.
type GuessFeedback struct {
	IsMatch          bool                `json:"isMatch"`
	ActualMoveSAN    string              `json:"actualMoveSAN"`
	ActualMoveUCI    string              `json:"actualMoveUCI"`
	PointsEarned     int                 `json:"pointsEarned"`
	TotalScore       int                 `json:"totalScore"`
	CurrentFEN       string              `json:"currentFEN"`
	OpponentReplySAN string              `json:"opponentReplySAN,omitempty"`
	OpponentReplyUCI string              `json:"opponentReplyUCI,omitempty"`
	IsFinished       bool                `json:"isFinished"`
	Summary          *PerformanceSummary `json:"summary,omitempty"`
}

// SolitaireSession tracks a Guess-the-Move training game.
type SolitaireSession struct {
	White          string       `json:"white"`
	Black          string       `json:"black"`
	Event          string       `json:"event"`
	Date           string       `json:"date"`
	Result         string       `json:"result"`
	UserSide       chess.Color  `json:"userSide"`
	CurrentPlyIdx  int          `json:"currentPlyIdx"`
	Score          int          `json:"score"`
	CorrectGuesses int          `json:"correctGuesses"`
	TotalTurns     int          `json:"totalTurns"`
	HintsUsed      int          `json:"hintsUsed"`
	CurrentHints   int          `json:"currentHints"`
	Moves          []MoveRecord `json:"moves"`
	Board          *chess.Board `json:"-"`
	IsFinished     bool         `json:"isFinished"`
}

// NewSolitaireSession creates a new Solitaire / Guess-the-move session from a game.
func NewSolitaireSession(game *chess.Game, userSideStr string) (*SolitaireSession, error) {
	if game == nil {
		return nil, fmt.Errorf("nil game")
	}

	userSide := chess.White
	if strings.ToLower(strings.TrimSpace(userSideStr)) == "black" {
		userSide = chess.Black
	}

	board := chess.NewBoard()
	if game.StartFEN != "" && game.StartFEN != chess.StartingFEN {
		if parsed, err := chess.ParseFEN(game.StartFEN); err == nil {
			board = parsed
		}
	}

	// Linearize moves
	var records []MoveRecord
	curr := game.RootMove
	ply := 0
	bClone := board.Clone()

	for curr != nil {
		turnStr := "white"
		if bClone.Turn == chess.Black {
			turnStr = "black"
		}

		fenBefore := bClone.FEN()
		records = append(records, MoveRecord{
			Ply:     ply,
			Turn:    turnStr,
			SAN:     curr.SAN,
			UCI:     curr.Move.String(),
			FEN:     fenBefore,
			Comment: curr.Comment,
			Move:    curr.Move,
		})

		bClone.MakeMove(curr.Move)
		ply++
		curr = curr.Next
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("game has no moves")
	}

	// Calculate total turns user will play
	totalTurns := 0
	for _, r := range records {
		if (userSide == chess.White && r.Turn == "white") || (userSide == chess.Black && r.Turn == "black") {
			totalTurns++
		}
	}

	sess := &SolitaireSession{
		White:         game.Tag("White"),
		Black:         game.Tag("Black"),
		Event:         game.Tag("Event"),
		Date:          game.Tag("Date"),
		Result:        game.Tag("Result"),
		UserSide:      userSide,
		CurrentPlyIdx: 0,
		Score:         0,
		TotalTurns:    totalTurns,
		Moves:         records,
		Board:         board,
	}

	// If user is Black, play White's initial move 1 automatically
	if userSide == chess.Black && len(sess.Moves) > 0 && sess.Moves[0].Turn == "white" {
		sess.Board.MakeMove(sess.Moves[0].Move)
		sess.CurrentPlyIdx = 1
	}

	return sess, nil
}

// CurrentTurnRecord returns the record for the current user move to guess.
func (s *SolitaireSession) CurrentTurnRecord() *MoveRecord {
	if s.CurrentPlyIdx >= len(s.Moves) {
		return nil
	}
	return &s.Moves[s.CurrentPlyIdx]
}

// CurrentFEN returns the board FEN at the current guess point.
func (s *SolitaireSession) CurrentFEN() string {
	if s.Board == nil {
		return chess.StartingFEN
	}
	return s.Board.FEN()
}

// GetHint provides progressive hints for the current move.
func (s *SolitaireSession) GetHint() (string, int) {
	rec := s.CurrentTurnRecord()
	if rec == nil {
		return "Game is completed.", 0
	}

	s.CurrentHints++
	s.HintsUsed++

	movingPiece := s.Board.Squares[rec.Move.From]
	pieceName := movingPiece.Type().String()
	switch movingPiece.Type() {
	case chess.Pawn:
		pieceName = "Pawn"
	case chess.Knight:
		pieceName = "Knight"
	case chess.Bishop:
		pieceName = "Bishop"
	case chess.Rook:
		pieceName = "Rook"
	case chess.Queen:
		pieceName = "Queen"
	case chess.King:
		pieceName = "King"
	}

	switch s.CurrentHints {
	case 1:
		return fmt.Sprintf("Piece type: A %s was moved.", pieceName), 1
	case 2:
		toFile := byte('a' + (rec.Move.To % 8))
		return fmt.Sprintf("The %s moves to the %c-file.", pieceName, toFile), 2
	default:
		toSquareName := rec.Move.To.String()
		return fmt.Sprintf("Destination: The move lands on %s.", toSquareName), 3
	}
}

// SubmitGuess checks the user's guess against the historical game move.
func (s *SolitaireSession) SubmitGuess(guess string) *GuessFeedback {
	rec := s.CurrentTurnRecord()
	if rec == nil {
		s.IsFinished = true
		summary := s.GetSummary()
		return &GuessFeedback{
			IsFinished: true,
			Summary:    &summary,
		}
	}

	cleanGuess := strings.TrimSpace(guess)
	cleanActualSAN := strings.TrimSuffix(strings.TrimSuffix(rec.SAN, "+"), "#")
	cleanGuessSAN := strings.TrimSuffix(strings.TrimSuffix(cleanGuess, "+"), "#")

	isMatch := strings.EqualFold(cleanGuessSAN, cleanActualSAN) || strings.EqualFold(cleanGuess, rec.UCI)

	if !isMatch {
		return &GuessFeedback{
			IsMatch:       false,
			ActualMoveSAN: rec.SAN,
			ActualMoveUCI: rec.UCI,
			PointsEarned:  0,
			TotalScore:    s.Score,
			CurrentFEN:    s.Board.FEN(),
			IsFinished:    false,
		}
	}

	// Guess matched!
	points := 10
	switch s.CurrentHints {
	case 1:
		points = 7
	case 2:
		points = 4
	case 3:
		points = 2
	}

	s.Score += points
	s.CorrectGuesses++
	s.CurrentHints = 0

	// Play user move on board
	s.Board.MakeMove(rec.Move)
	s.CurrentPlyIdx++

	// Now play opponent reply if one exists
	var oppSAN, oppUCI string
	if s.CurrentPlyIdx < len(s.Moves) {
		oppRec := s.Moves[s.CurrentPlyIdx]
		oppSAN = oppRec.SAN
		oppUCI = oppRec.UCI
		s.Board.MakeMove(oppRec.Move)
		s.CurrentPlyIdx++
	}

	if s.CurrentPlyIdx >= len(s.Moves) {
		s.IsFinished = true
	}

	feedback := &GuessFeedback{
		IsMatch:          true,
		ActualMoveSAN:    rec.SAN,
		ActualMoveUCI:    rec.UCI,
		PointsEarned:     points,
		TotalScore:       s.Score,
		CurrentFEN:       s.Board.FEN(),
		OpponentReplySAN: oppSAN,
		OpponentReplyUCI: oppUCI,
		IsFinished:       s.IsFinished,
	}

	if s.IsFinished {
		summary := s.GetSummary()
		feedback.Summary = &summary
	}

	return feedback
}

// RevealAndSkip reveals the actual move and advances to the next turn without awarding points.
func (s *SolitaireSession) RevealAndSkip() *GuessFeedback {
	rec := s.CurrentTurnRecord()
	if rec == nil {
		s.IsFinished = true
		summary := s.GetSummary()
		return &GuessFeedback{
			IsFinished: true,
			Summary:    &summary,
		}
	}

	actualSAN := rec.SAN
	actualUCI := rec.UCI
	s.CurrentHints = 0

	// Play actual move
	s.Board.MakeMove(rec.Move)
	s.CurrentPlyIdx++

	// Play opponent reply if exists
	var oppSAN, oppUCI string
	if s.CurrentPlyIdx < len(s.Moves) {
		oppRec := s.Moves[s.CurrentPlyIdx]
		oppSAN = oppRec.SAN
		oppUCI = oppRec.UCI
		s.Board.MakeMove(oppRec.Move)
		s.CurrentPlyIdx++
	}

	if s.CurrentPlyIdx >= len(s.Moves) {
		s.IsFinished = true
	}

	feedback := &GuessFeedback{
		IsMatch:          false,
		ActualMoveSAN:    actualSAN,
		ActualMoveUCI:    actualUCI,
		PointsEarned:     0,
		TotalScore:       s.Score,
		CurrentFEN:       s.Board.FEN(),
		OpponentReplySAN: oppSAN,
		OpponentReplyUCI: oppUCI,
		IsFinished:       s.IsFinished,
	}

	if s.IsFinished {
		summary := s.GetSummary()
		feedback.Summary = &summary
	}

	return feedback
}

// GetSummary calculates performance statistics and titles.
func (s *SolitaireSession) GetSummary() PerformanceSummary {
	maxScore := s.TotalTurns * 10
	pct := 0.0
	accuracy := 0.0
	if maxScore > 0 {
		pct = float64(s.Score) / float64(maxScore) * 100.0
	}
	if s.TotalTurns > 0 {
		accuracy = float64(s.CorrectGuesses) / float64(s.TotalTurns) * 100.0
	}

	var title, feedback string
	switch {
	case pct >= 90:
		title = "Grandmaster (GM) Level"
		feedback = "Exceptional intuition! You found almost every master move directly without assistance."
	case pct >= 80:
		title = "International Master (IM) Level"
		feedback = "Outstanding play! Strong alignment with grandmaster plans and tactics."
	case pct >= 70:
		title = "FIDE Master (FM) Level"
		feedback = "Very strong tactical and positional awareness across key turning points."
	case pct >= 55:
		title = "Candidate Master / Expert Level"
		feedback = "Solid understanding of the ideas, finding major plans with minimal guidance."
	case pct >= 40:
		title = "Club Player Level"
		feedback = "Good effort! Review the missed moves to see how the master created active counterplay."
	default:
		title = "Developing Player"
		feedback = "Great learning exercise! Replaying master games trains positional pattern recognition."
	}

	return PerformanceSummary{
		Score:          s.Score,
		MaxScore:       maxScore,
		Percentage:     pct,
		AccuracyPct:    accuracy,
		CorrectGuesses: s.CorrectGuesses,
		TotalTurns:     s.TotalTurns,
		HintsUsed:      s.HintsUsed,
		RatingTitle:    title,
		Feedback:       feedback,
	}
}
