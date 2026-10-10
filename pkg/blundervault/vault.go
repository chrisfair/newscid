package blundervault

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Puzzle represents a personal mistake / blunder spaced-repetition flashcard.
type Puzzle struct {
	ID           string    `json:"id"`
	FEN          string    `json:"fen"`
	PlayerName   string    `json:"playerName"`
	UserColor    string    `json:"userColor"` // "white" | "black"
	PlayedMove   string    `json:"playedMove"` // UCI
	BestMove     string    `json:"bestMove"`   // UCI
	DeltaCp      int       `json:"deltaCp"`
	Event        string    `json:"event"`
	Date         string    `json:"date"`
	Explanation  string    `json:"explanation,omitempty"`
	
	// Spaced Repetition (SM-2)
	Repetitions  int       `json:"repetitions"`
	IntervalDays int       `json:"intervalDays"`
	EaseFactor   float64   `json:"easeFactor"`
	NextReview   time.Time `json:"nextReview"`
	LastReview   time.Time `json:"lastReview"`
	Status       string    `json:"status"` // "due", "learning", "mastered"
}

// Vault manages personal blunder flashcards persisted to disk.
type Vault struct {
	mu       sync.RWMutex
	filePath string
	puzzles  map[string]*Puzzle
}

// NewVault initializes or loads an existing blunder vault from filePath.
func NewVault(filePath string) (*Vault, error) {
	v := &Vault{
		filePath: filePath,
		puzzles:  make(map[string]*Puzzle),
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}

	if data, err := os.ReadFile(filePath); err == nil {
		var list []*Puzzle
		if err := json.Unmarshal(data, &list); err == nil {
			for _, p := range list {
				v.puzzles[p.ID] = p
			}
		}
	}

	return v, nil
}

// AddPuzzle inserts a new blunder puzzle into the vault.
func (v *Vault) AddPuzzle(p *Puzzle) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if p.ID == "" {
		p.ID = fmt.Sprintf("blunder-%d", time.Now().UnixNano())
	}
	if p.EaseFactor <= 0 {
		p.EaseFactor = 2.5
	}
	if p.IntervalDays <= 0 {
		p.IntervalDays = 1
	}
	if p.NextReview.IsZero() {
		p.NextReview = time.Now()
	}
	if p.Status == "" {
		p.Status = "due"
	}

	v.puzzles[p.ID] = p
	return v.saveLocked()
}

// GetDuePuzzles returns all puzzles due for review now.
func (v *Vault) GetDuePuzzles() []*Puzzle {
	v.mu.RLock()
	defer v.mu.RUnlock()

	now := time.Now()
	var due []*Puzzle
	for _, p := range v.puzzles {
		if p.NextReview.Before(now) || p.NextReview.Equal(now) {
			due = append(due, p)
		}
	}
	return due
}

// GetAllPuzzles returns all puzzles in the vault.
func (v *Vault) GetAllPuzzles() []*Puzzle {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var all []*Puzzle
	for _, p := range v.puzzles {
		all = append(all, p)
	}
	return all
}

// RecordReview applies SuperMemo SM-2 algorithm to adjust interval and ease factor.
// quality: 0 (blackout) to 5 (perfect recall).
func (v *Vault) RecordReview(id string, quality int) (*Puzzle, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	p, ok := v.puzzles[id]
	if !ok {
		return nil, fmt.Errorf("puzzle %s not found", id)
	}

	if quality < 0 {
		quality = 0
	} else if quality > 5 {
		quality = 5
	}

	p.LastReview = time.Now()

	// SM-2 Algorithm Implementation
	if quality >= 3 {
		// Successful review
		if p.Repetitions == 0 {
			p.IntervalDays = 1
		} else if p.Repetitions == 1 {
			p.IntervalDays = 6
		} else {
			p.IntervalDays = int(math.Round(float64(p.IntervalDays) * p.EaseFactor))
		}
		p.Repetitions++
	} else {
		// Failed review: reset repetitions
		p.Repetitions = 0
		p.IntervalDays = 1
	}

	// Update Ease Factor: EF' = EF + (0.1 - (5 - q) * (0.08 + (5 - q) * 0.02))
	p.EaseFactor = p.EaseFactor + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))
	if p.EaseFactor < 1.3 {
		p.EaseFactor = 1.3
	}

	p.NextReview = time.Now().Add(time.Duration(p.IntervalDays) * 24 * time.Hour)

	if p.Repetitions >= 4 {
		p.Status = "mastered"
	} else {
		p.Status = "learning"
	}

	if err := v.saveLocked(); err != nil {
		return nil, err
	}

	return p, nil
}

func (v *Vault) saveLocked() error {
	var list []*Puzzle
	for _, p := range v.puzzles {
		list = append(list, p)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(v.filePath, data, 0644)
}
