package repertoire

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"newscid/pkg/chess"
)

// RepertoireMove represents an approved move in an opening repertoire.
type RepertoireMove struct {
	UCI      string `json:"uci"`
	SAN      string `json:"san"`
	NextFEN  string `json:"nextFen"`
	Comment  string `json:"comment,omitempty"`
	Practice int    `json:"practice"`
	Correct  int    `json:"correct"`
}

// Repertoire represents a player's curated opening repertoire for White or Black.
type Repertoire struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Color       string                      `json:"color"` // "w" or "b"
	Description string                      `json:"description"`
	Positions   map[string][]RepertoireMove `json:"positions"` // Key: Core FEN
}

// RepertoireSummary provides lightweight listing information.
type RepertoireSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	NumMoves    int    `json:"numMoves"`
}

// Manager handles saving, loading, and querying repertoires.
type Manager struct {
	mu      sync.RWMutex
	dir     string
	inMem   map[string]*Repertoire
}

func NewManager(storageDir string) *Manager {
	if storageDir == "" {
		home, _ := os.UserHomeDir()
		storageDir = filepath.Join(home, ".newscid", "repertoires")
	}
	_ = os.MkdirAll(storageDir, 0755)

	m := &Manager{
		dir:   storageDir,
		inMem: make(map[string]*Repertoire),
	}
	m.loadAll()
	m.ensureStarterRepertoires()
	return m
}

func (m *Manager) loadAll() {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			path := filepath.Join(m.dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var rep Repertoire
			if err := json.Unmarshal(data, &rep); err == nil {
				if rep.Positions == nil {
					rep.Positions = make(map[string][]RepertoireMove)
				}
				m.inMem[rep.ID] = &rep
			}
		}
	}
}

func (m *Manager) ensureStarterRepertoires() {
	if len(m.inMem) > 0 {
		return
	}

	// 1. White Starter: Italian Game & Vienna Repertoire
	repWhite := &Repertoire{
		ID:          "white-openings-italian",
		Name:        "White: Italian Game & Open Repertoire",
		Color:       "w",
		Description: "Classical 1.e4 opening lines featuring the Italian Game, Guioco Piano, and Fried Liver attack.",
		Positions:   make(map[string][]RepertoireMove),
	}
	m.populateLine(repWhite, []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Bc5", "c3", "Nf6", "d4"}, "Italian Game: Giuoco Piano")
	m.populateLine(repWhite, []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nf6", "Ng5", "d5", "exd5"}, "Two Knights: Fried Liver Attack")
	m.populateLine(repWhite, []string{"e4", "c5", "Nf3", "d6", "d4", "cxd4", "Nxd4"}, "Open Sicilian Defense")
	m.populateLine(repWhite, []string{"e4", "e6", "d4", "d5", "Nc3"}, "French Defense: Classical")
	m.populateLine(repWhite, []string{"e4", "c6", "d4", "d5", "Nc3"}, "Caro-Kann: Classical")
	_ = m.SaveRepertoire(repWhite)

	// 2. Black Starter: Sicilian Najdorf & King's Indian
	repBlack := &Repertoire{
		ID:          "black-openings-sicilian-kid",
		Name:        "Black: Sicilian Najdorf & King's Indian",
		Color:       "b",
		Description: "Dynamic, combative fighting repertoires against 1.e4 (Najdorf) and 1.d4 (King's Indian).",
		Positions:   make(map[string][]RepertoireMove),
	}
	m.populateLine(repBlack, []string{"e4", "c5", "Nf3", "d6", "d4", "cxd4", "Nxd4", "Nf6", "Nc3", "a6"}, "Sicilian Najdorf")
	m.populateLine(repBlack, []string{"d4", "Nf6", "c4", "g6", "Nc3", "Bg7", "e4", "d6", "Nf3", "O-O"}, "King's Indian Defense")
	_ = m.SaveRepertoire(repBlack)
}

func parseRepColor(c string) chess.Color {
	if len(c) > 0 && (c[0] == 'b' || c[0] == 'B') {
		return chess.Black
	}
	return chess.White
}

func (m *Manager) populateLine(rep *Repertoire, sanMoves []string, comment string) {
	b := chess.NewBoard()
	repColor := parseRepColor(rep.Color)
	for _, san := range sanMoves {
		moverColor := b.Turn
		mv, err := b.ParseSAN(san)
		if err != nil {
			break
		}
		core := CoreFEN(b.FEN())
		uci := mv.String()

		b.MakeMove(mv)
		nextCore := CoreFEN(b.FEN())

		// Only add to repertoire if this move was played by the repertoire side
		if moverColor == repColor {
			existing := rep.Positions[core]
			found := false
			for _, ex := range existing {
				if ex.UCI == uci {
					found = true
					break
				}
			}
			if !found {
				rep.Positions[core] = append(existing, RepertoireMove{
					UCI:     uci,
					SAN:     san,
					NextFEN: nextCore,
					Comment: comment,
				})
			}
		}
	}
}

// CoreFEN strips clock, fullmove count, and ephemeral en-passant for reliable position lookup.
func CoreFEN(fen string) string {
	start := 0
	for start < len(fen) && fen[start] == ' ' {
		start++
	}
	end := len(fen)
	for end > start && fen[end-1] == ' ' {
		end--
	}
	spaces := 0
	for i := start; i < end; i++ {
		if fen[i] == ' ' {
			spaces++
			if spaces == 3 {
				return fen[start:i]
			}
		}
	}
	return fen[start:end]
}

// ListRepertoires returns summary metadata of all repertoires.
func (m *Manager) ListRepertoires() []RepertoireSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []RepertoireSummary
	for _, r := range m.inMem {
		numMoves := 0
		for _, moves := range r.Positions {
			numMoves += len(moves)
		}
		list = append(list, RepertoireSummary{
			ID:          r.ID,
			Name:        r.Name,
			Color:       r.Color,
			Description: r.Description,
			NumMoves:    numMoves,
		})
	}
	return list
}

// GetRepertoire fetches a repertoire by ID.
func (m *Manager) GetRepertoire(id string) (*Repertoire, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rep, ok := m.inMem[id]
	if !ok {
		return nil, fmt.Errorf("repertoire %q not found", id)
	}
	return rep, nil
}

// SaveRepertoire writes a repertoire to memory and disk.
func (m *Manager) SaveRepertoire(rep *Repertoire) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveRepertoireLocked(rep)
}

func (m *Manager) saveRepertoireLocked(rep *Repertoire) error {
	if rep.ID == "" {
		slug := strings.ToLower(rep.Name)
		slug = strings.ReplaceAll(slug, " ", "-")
		rep.ID = fmt.Sprintf("%s-%s", rep.Color, slug)
	}
	if rep.Positions == nil {
		rep.Positions = make(map[string][]RepertoireMove)
	}

	m.inMem[rep.ID] = rep

	data, _ := json.MarshalIndent(rep, "", "  ")
	filePath := filepath.Join(m.dir, rep.ID+".json")
	return os.WriteFile(filePath, data, 0644)
}

// DeleteRepertoire deletes a repertoire.
func (m *Manager) DeleteRepertoire(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.inMem, id)
	filePath := filepath.Join(m.dir, id+".json")
	_ = os.Remove(filePath)
	return nil
}

// VerifyMove checks if user move from given FEN is in the repertoire.
func (m *Manager) VerifyMove(repID, fen, uci string) (bool, *RepertoireMove, []RepertoireMove, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rep, ok := m.inMem[repID]
	if !ok {
		return false, nil, nil, fmt.Errorf("repertoire %q not found", repID)
	}

	core := CoreFEN(fen)
	options := rep.Positions[core]
	for _, opt := range options {
		if strings.EqualFold(opt.UCI, uci) {
			return true, &opt, options, nil
		}
	}

	return false, nil, options, nil
}

// RecordPractice tracks mastery statistics when a user plays a repertoire move.
func (m *Manager) RecordPractice(repID, fen, uci string, wasCorrect bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.inMem[repID]
	if !ok {
		return
	}

	core := CoreFEN(fen)
	moves := rep.Positions[core]
	for i := range moves {
		if strings.EqualFold(moves[i].UCI, uci) {
			moves[i].Practice++
			if wasCorrect {
				moves[i].Correct++
			}
			rep.Positions[core] = moves
			_ = m.saveRepertoireLocked(rep)
			return
		}
	}
}

