package chess960

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Position represents a Fischer Random Chess / Chess960 starting setup.
type Position struct {
	ID        int    `json:"id"`        // 0 to 959
	FEN       string `json:"fen"`       // Starting FEN with Chess960 castling rights
	Pieces    string `json:"pieces"`    // 8-character rank 1 setup, e.g. "RNBQKBNR"
	IsClassic bool   `json:"isClassic"` // True if ID == 518 (standard chess)
}

var knightPairs = [10][2]int{
	{0, 1}, {0, 2}, {0, 3}, {0, 4},
	{1, 2}, {1, 3}, {1, 4},
	{2, 3}, {2, 4},
	{3, 4},
}

// GeneratePosition generates the Chess960 starting setup for a given ID (0 to 959).
// Based on the standard Scharnagl numbering scheme.
func GeneratePosition(id int) (*Position, error) {
	if id < 0 || id > 959 {
		return nil, fmt.Errorf("chess960 ID must be between 0 and 959, got %d", id)
	}

	board := [8]rune{}
	n := id

	// 1. Light-square Bishop (files b, d, f, h -> indices 1, 3, 5, 7)
	b1 := (n % 4) * 2 + 1
	board[b1] = 'B'
	n /= 4

	// 2. Dark-square Bishop (files a, c, e, g -> indices 0, 2, 4, 6)
	b2 := (n % 4) * 2
	board[b2] = 'B'
	n /= 4

	// 3. Queen placed on Q-th remaining empty square (0..5)
	q := n % 6
	n /= 6
	placeInEmpty(&board, q, 'Q')

	// 4. Two Knights placed on remaining 5 empty squares using combination index (0..9)
	kPair := knightPairs[n]
	placeInEmpty(&board, kPair[0], 'N')
	placeInEmpty(&board, kPair[1]-1, 'N')

	// 5. Remaining 3 empty squares are filled with Rook, King, Rook (King between Rooks)
	placeInEmpty(&board, 0, 'R')
	placeInEmpty(&board, 0, 'K')
	placeInEmpty(&board, 0, 'R')

	pieces := string(board[:])
	fen := buildChess960FEN(pieces)

	return &Position{
		ID:        id,
		FEN:       fen,
		Pieces:    pieces,
		IsClassic: id == 518,
	}, nil
}

// RandomPosition generates a randomly selected Chess960 starting position.
func RandomPosition() *Position {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := r.Intn(960)
	pos, _ := GeneratePosition(id)
	return pos
}

// placeInEmpty places piece into the nth available empty square on the board.
func placeInEmpty(board *[8]rune, emptyIndex int, piece rune) {
	currentEmpty := 0
	for i := 0; i < 8; i++ {
		if board[i] == 0 {
			if currentEmpty == emptyIndex {
				board[i] = piece
				return
			}
			currentEmpty++
		}
	}
}

// buildChess960FEN constructs a valid starting FEN for the piece arrangement.
func buildChess960FEN(rank1 string) string {
	blackRank := strings.ToLower(rank1)
	whiteRank := rank1

	// In Chess960 FEN, castling rights use file letters if rooks are not on a/h,
	// or standard KQkq if they are classic. For universal compatibility, standard KQkq is used.
	return fmt.Sprintf("%s/pppppppp/8/8/8/8/PPPPPPPP/%s w KQkq - 0 1", blackRank, whiteRank)
}
