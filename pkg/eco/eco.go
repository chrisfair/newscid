package eco

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

// Entry represents an ECO definition.
type Entry struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Moves string `json:"moves"`
}

// Classifier identifies ECO code and opening name from move sequences.
type Classifier struct {
	mu      sync.RWMutex
	entries []*Entry
	byMoves map[string]*Entry
}

func NewClassifier() *Classifier {
	c := &Classifier{
		byMoves: make(map[string]*Entry),
	}
	c.loadDefaults()
	return c
}

func (c *Classifier) loadDefaults() {
	defaults := []Entry{
		{"A00", "Uncommon Opening", ""},
		{"A04", "Reti Opening", "1. Nf3"},
		{"A10", "English Opening", "1. c4"},
		{"B00", "King's Pawn Opening", "1. e4"},
		{"B01", "Scandinavian Defense", "1. e4 d5"},
		{"B07", "Pirc Defense", "1. e4 d6 2. d4 Nf6"},
		{"B10", "Caro-Kann Defense", "1. e4 c6"},
		{"B20", "Sicilian Defense", "1. e4 c5"},
		{"B90", "Sicilian, Najdorf", "1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6"},
		{"C00", "French Defense", "1. e4 e6"},
		{"C20", "King's Pawn Game", "1. e4 e5"},
		{"C42", "Petrov's Defense", "1. e4 e5 2. Nf3 Nf6"},
		{"C50", "Italian Game", "1. e4 e5 2. Nf3 Nc6 3. Bc4"},
		{"C60", "Ruy Lopez", "1. e4 e5 2. Nf3 Nc6 3. Bb5"},
		{"D00", "Queen's Pawn Game", "1. d4 d5"},
		{"D02", "London System", "1. d4 d5 2. Bf4"},
		{"D06", "Queen's Gambit", "1. d4 d5 2. c4"},
		{"D30", "Queen's Gambit Declined", "1. d4 d5 2. c4 e6"},
		{"D80", "Grunfeld Defense", "1. d4 Nf6 2. c4 g6 3. Nc3 d5"},
		{"E00", "Queen's Pawn Game (Catalan/Nimzo)", "1. d4 Nf6 2. c4 e6"},
		{"E60", "King's Indian Defense", "1. d4 Nf6 2. c4 g6"},
	}

	for _, d := range defaults {
		entry := &Entry{Code: d.Code, Name: d.Name, Moves: normalizeMoves(d.Moves)}
		c.entries = append(c.entries, entry)
		c.byMoves[entry.Moves] = entry
	}
}

// LoadFromFile parses a scid.eco format file.
func (c *Classifier) LoadFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	c.mu.Lock()
	defer c.mu.Unlock()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Format: CODE "Name" Moves *
		firstSpace := strings.IndexByte(line, ' ')
		if firstSpace == -1 {
			continue
		}
		code := strings.TrimSpace(line[:firstSpace])

		rest := strings.TrimSpace(line[firstSpace:])
		if !strings.HasPrefix(rest, "\"") {
			continue
		}
		endQuote := strings.Index(rest[1:], "\"")
		if endQuote == -1 {
			continue
		}
		name := rest[1 : endQuote+1]

		movePart := strings.TrimSpace(rest[endQuote+2:])
		movePart = strings.TrimSuffix(movePart, "*")
		moves := normalizeMoves(movePart)

		entry := &Entry{
			Code:  code,
			Name:  name,
			Moves: moves,
		}
		c.entries = append(c.entries, entry)
		c.byMoves[moves] = entry
	}

	return nil
}

// Classify matches the longest move prefix from the classifier table.
func (c *Classifier) Classify(moveSANs []string) *Entry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Build progressive move string
	for i := len(moveSANs); i > 0; i-- {
		candidate := strings.Join(moveSANs[:i], " ")
		if entry, ok := c.byMoves[candidate]; ok {
			return entry
		}
	}

	if entry, ok := c.byMoves[""]; ok {
		return entry
	}
	return &Entry{Code: "A00", Name: "Unknown"}
}

func normalizeMoves(movesStr string) string {
	fields := strings.Fields(movesStr)
	var sans []string
	for _, f := range fields {
		if strings.Contains(f, ".") {
			parts := strings.Split(f, ".")
			val := parts[len(parts)-1]
			if val != "" && val != "*" {
				sans = append(sans, val)
			}
		} else if f != "*" {
			sans = append(sans, f)
		}
	}
	return strings.Join(sans, " ")
}
