package repertoire

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/fileformat"
	"newscid/pkg/polyglot"
)

// RepertoireMove represents an approved move in an opening repertoire.
type RepertoireMove struct {
	UCI           string `json:"uci"`
	SAN           string `json:"san"`
	NextFEN       string `json:"nextFen"`
	Comment       string `json:"comment,omitempty"`
	Practice      int    `json:"practice"`
	Correct       int    `json:"correct"`
	Streak        int    `json:"streak"`
	LastPracticed int64  `json:"lastPracticed,omitempty"`
	Mastered      bool   `json:"mastered"`
}

// LineStep represents a single half-move (ply) along a repertoire variation line.
type LineStep struct {
	Ply        int    `json:"ply"`
	FEN        string `json:"fen"`
	CoreFEN    string `json:"coreFen"`
	UCI        string `json:"uci"`
	SAN        string `json:"san"`
	NextFEN    string `json:"nextFen"`
	Comment    string `json:"comment,omitempty"`
	IsUserMove bool   `json:"isUserMove"`
	Practice   int    `json:"practice"`
	Correct    int    `json:"correct"`
	Streak     int    `json:"streak"`
	Mastered   bool   `json:"mastered"`
}

// RepertoireLine represents one complete variation branch from the starting position to a leaf node.
type RepertoireLine struct {
	Index          int        `json:"index"`
	Name           string     `json:"name"`
	SANLine        string     `json:"sanLine"`
	Steps          []LineStep `json:"steps"`
	UserMovesCount int        `json:"userMovesCount"`
	MasteredCount  int        `json:"masteredCount"`
	MasteryPct     float64    `json:"masteryPct"`
	IsMastered     bool       `json:"isMastered"`
	KeyComment     string     `json:"keyComment,omitempty"`
}

// RepertoireMasteryReport summarizes systematic training coverage across all lines and user positions.
type RepertoireMasteryReport struct {
	RepertoireID        string           `json:"repertoireId"`
	RepertoireName      string           `json:"repertoireName"`
	Color               string           `json:"color"`
	TotalUserNodes      int              `json:"totalUserNodes"`
	MasteredUserNodes   int              `json:"masteredUserNodes"`
	UntestedUserNodes   int              `json:"untestedUserNodes"`
	NeedsWorkUserNodes  int              `json:"needsWorkUserNodes"`
	MasteryPct          float64          `json:"masteryPct"`
	TotalLines          int              `json:"totalLines"`
	MasteredLines       int              `json:"masteredLines"`
	NextTargetLineIndex int              `json:"nextTargetLineIndex"`
	Lines               []RepertoireLine `json:"lines"`
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
		storageDir = filepath.Join(home, fileformat.LegacyDotDirName, fileformat.DirRepertoires)
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
		if !entry.IsDir() && fileformat.IsJSONPath(entry.Name()) {
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

	// White Starter: Italian Game & Vienna Repertoire
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

	// Black Starter: Sicilian Najdorf & King's Indian
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
		canonicalSAN := b.MoveToSAN(mv)
		core := CoreFEN(b.FEN())
		uci := mv.String()

		b.MakeMove(mv)
		nextCore := CoreFEN(b.FEN())

		moveComment := ""
		if moverColor == repColor {
			moveComment = comment
		}

		existing := rep.Positions[core]
		found := false
		for i := range existing {
			if strings.EqualFold(existing[i].UCI, uci) {
				if existing[i].Comment == "" && moveComment != "" {
					existing[i].Comment = moveComment
				}
				rep.Positions[core] = existing
				found = true
				break
			}
		}
		if !found {
			rep.Positions[core] = append(existing, RepertoireMove{
				UCI:     uci,
				SAN:     canonicalSAN,
				NextFEN: nextCore,
				Comment: moveComment,
			})
		}
	}
}

// FindMovesForBoard returns the repertoire moves defined at board b.
// For opponent turns in legacy repertoires that only stored user-turn keys, it also discovers legal opponent moves leading into a known repertoire position.
func (rep *Repertoire) FindMovesForBoard(b *chess.Board) []RepertoireMove {
	if rep == nil || b == nil {
		return nil
	}
	core := CoreFEN(b.FEN())
	if moves, ok := rep.Positions[core]; ok && len(moves) > 0 {
		return moves
	}
	repColor := parseRepColor(rep.Color)
	if b.Turn != repColor {
		var inferred []RepertoireMove
		for _, lm := range b.GenerateLegalMoves() {
			nb := b.Clone()
			nb.MakeMove(lm)
			nextCore := CoreFEN(nb.FEN())
			if _, exists := rep.Positions[nextCore]; exists {
				inferred = append(inferred, RepertoireMove{
					UCI:     lm.String(),
					SAN:     b.MoveToSAN(lm),
					NextFEN: nextCore,
				})
			}
		}
		return inferred
	}
	return nil
}

func isMoveMastered(rm RepertoireMove) bool {
	if rm.Mastered {
		return true
	}
	if rm.Practice <= 0 {
		return false
	}
	if rm.Streak >= 2 {
		return true
	}
	return rm.Correct == rm.Practice && rm.Correct >= 1
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
	filePath := filepath.Join(m.dir, rep.ID+fileformat.ExtJSON)
	return os.WriteFile(filePath, data, 0644)
}

// DeleteRepertoire deletes a repertoire.
func (m *Manager) DeleteRepertoire(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.inMem, id)
	filePath := filepath.Join(m.dir, id+fileformat.ExtJSON)
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

// RecordPractice tracks mastery statistics when a user plays a move at fen.
func (m *Manager) RecordPractice(repID, fen, uci string, wasCorrect bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.inMem[repID]
	if !ok {
		return
	}

	core := CoreFEN(fen)
	moves := rep.Positions[core]
	if len(moves) == 0 {
		return
	}

	now := time.Now().UnixMilli()
	matchedIdx := -1
	for i := range moves {
		if strings.EqualFold(moves[i].UCI, uci) || strings.EqualFold(moves[i].SAN, uci) {
			matchedIdx = i
			break
		}
	}

	if matchedIdx == -1 && !wasCorrect {
		matchedIdx = 0
	}
	if matchedIdx < 0 {
		return
	}

	moves[matchedIdx].Practice++
	moves[matchedIdx].LastPracticed = now
	if wasCorrect {
		moves[matchedIdx].Correct++
		moves[matchedIdx].Streak++
		if moves[matchedIdx].Streak >= 2 || (moves[matchedIdx].Correct == moves[matchedIdx].Practice && moves[matchedIdx].Correct >= 1) {
			moves[matchedIdx].Mastered = true
		}
	} else {
		moves[matchedIdx].Streak = 0
		moves[matchedIdx].Mastered = false
	}

	rep.Positions[core] = moves
	_ = m.saveRepertoireLocked(rep)
}

var moveNumTokenRegex = regexp.MustCompile(`^\d+\.+$`)

// AddLineSAN parses a PGN or SAN/UCI move sequence from the starting position and adds all moves to the repertoire.
func (m *Manager) AddLineSAN(repID string, moveText string, comment string) (*Repertoire, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.inMem[repID]
	if !ok {
		return nil, fmt.Errorf("repertoire %q not found", repID)
	}

	cleaned := strings.TrimSpace(moveText)
	if cleaned == "" {
		return nil, fmt.Errorf("move sequence cannot be empty")
	}

	// Strip PGN headers if pasted
	var bodyLines []string
	for _, line := range strings.Split(cleaned, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			continue
		}
		bodyLines = append(bodyLines, trimmed)
	}
	cleaned = strings.Join(bodyLines, " ")

	// Extract inline brace comments if any and strip parenthetical variations
	cleaned = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(cleaned, " ")
	// Separate attached move numbers like "1.e4" or "1...c5"
	cleaned = regexp.MustCompile(`(\d+\.+)`).ReplaceAllString(cleaned, " $1 ")

	rawTokens := strings.Fields(cleaned)
	var movesSAN []string
	for _, tok := range rawTokens {
		if moveNumTokenRegex.MatchString(tok) {
			continue
		}
		if tok == "1-0" || tok == "0-1" || tok == "1/2-1/2" || tok == "*" {
			continue
		}
		if strings.HasPrefix(tok, "$") {
			continue
		}
		movesSAN = append(movesSAN, tok)
	}

	if len(movesSAN) == 0 {
		return nil, fmt.Errorf("no valid moves found in input")
	}

	b := chess.NewBoard()
	repColor := parseRepColor(rep.Color)
	addedCount := 0

	for idx, tok := range movesSAN {
		moverColor := b.Turn
		mv, err := b.ParseSAN(tok)
		if err != nil {
			mv, err = b.ParseUCI(tok)
			if err != nil {
				return nil, fmt.Errorf("invalid move %q at ply %d: %w", tok, idx+1, err)
			}
		}
		san := b.MoveToSAN(mv)
		uci := mv.String()
		core := CoreFEN(b.FEN())

		b.MakeMove(mv)
		nextCore := CoreFEN(b.FEN())

		moveComment := ""
		// Apply note to user moves (especially the deepest user move in the line)
		if moverColor == repColor && comment != "" {
			// Check if this is the last user move in the sequence
			isLastUserMove := true
			if idx+2 < len(movesSAN) {
				isLastUserMove = false
			}
			if isLastUserMove {
				moveComment = comment
			}
		}

		existing := rep.Positions[core]
		found := false
		for i := range existing {
			if strings.EqualFold(existing[i].UCI, uci) {
				if moveComment != "" {
					existing[i].Comment = moveComment
				}
				rep.Positions[core] = existing
				found = true
				break
			}
		}
		if !found {
			rep.Positions[core] = append(existing, RepertoireMove{
				UCI:     uci,
				SAN:     san,
				NextFEN: nextCore,
				Comment: moveComment,
			})
		}
		addedCount++
	}

	if addedCount == 0 {
		return nil, fmt.Errorf("no moves added")
	}

	if err := m.saveRepertoireLocked(rep); err != nil {
		return nil, err
	}
	return rep, nil
}

// DeleteMove removes a specific move from a position in the repertoire.
func (m *Manager) DeleteMove(repID, fen, uci string) (*Repertoire, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.inMem[repID]
	if !ok {
		return nil, fmt.Errorf("repertoire %q not found", repID)
	}

	core := CoreFEN(fen)
	existing := rep.Positions[core]
	var filtered []RepertoireMove
	for _, rm := range existing {
		if !strings.EqualFold(rm.UCI, uci) && !strings.EqualFold(rm.SAN, uci) {
			filtered = append(filtered, rm)
		}
	}
	if len(filtered) == 0 {
		delete(rep.Positions, core)
	} else {
		rep.Positions[core] = filtered
	}

	if err := m.saveRepertoireLocked(rep); err != nil {
		return nil, err
	}
	return rep, nil
}

// UpdateMoveComment updates the strategic note for a move in the repertoire.
func (m *Manager) UpdateMoveComment(repID, fen, uci, comment string) (*Repertoire, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.inMem[repID]
	if !ok {
		return nil, fmt.Errorf("repertoire %q not found", repID)
	}

	core := CoreFEN(fen)
	existing := rep.Positions[core]
	updated := false
	for i := range existing {
		if strings.EqualFold(existing[i].UCI, uci) || strings.EqualFold(existing[i].SAN, uci) {
			existing[i].Comment = strings.TrimSpace(comment)
			updated = true
			break
		}
	}
	if !updated {
		return nil, fmt.Errorf("move %q not found at position", uci)
	}
	rep.Positions[core] = existing

	if err := m.saveRepertoireLocked(rep); err != nil {
		return nil, err
	}
	return rep, nil
}

// ResetMastery clears all practice, correct, streak, and mastery flags in a repertoire for a fresh 0->100% training cycle.
func (m *Manager) ResetMastery(repID string) (*Repertoire, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rep, ok := m.inMem[repID]
	if !ok {
		return nil, fmt.Errorf("repertoire %q not found", repID)
	}

	for core, moves := range rep.Positions {
		for i := range moves {
			moves[i].Practice = 0
			moves[i].Correct = 0
			moves[i].Streak = 0
			moves[i].LastPracticed = 0
			moves[i].Mastered = false
		}
		rep.Positions[core] = moves
	}

	if err := m.saveRepertoireLocked(rep); err != nil {
		return nil, err
	}
	return rep, nil
}

// GetMasteryReport enumerates all lines and user positions in the repertoire to track 0-100% systematic mastery.
func (m *Manager) GetMasteryReport(repID string) (*RepertoireMasteryReport, error) {
	m.mu.RLock()
	rep, ok := m.inMem[repID]
	m.mu.RUnlock()
	if !ok || rep == nil {
		return nil, fmt.Errorf("repertoire %q not found", repID)
	}

	repColor := parseRepColor(rep.Color)
	report := &RepertoireMasteryReport{
		RepertoireID:        rep.ID,
		RepertoireName:      rep.Name,
		Color:               rep.Color,
		NextTargetLineIndex: -1,
		Lines:               make([]RepertoireLine, 0),
	}

	type userNodeKey struct {
		core string
		uci  string
	}
	seenUserNodes := make(map[userNodeKey]RepertoireMove)

	var dfs func(b *chess.Board, steps []LineStep, visited map[string]bool)
	dfs = func(b *chess.Board, steps []LineStep, visited map[string]bool) {
		core := CoreFEN(b.FEN())
		if visited[core] || len(steps) >= 40 {
			if len(steps) > 0 {
				report.Lines = append(report.Lines, buildRepertoireLine(len(report.Lines), steps))
			}
			return
		}

		moves := rep.FindMovesForBoard(b)
		if len(moves) == 0 {
			if len(steps) > 0 {
				report.Lines = append(report.Lines, buildRepertoireLine(len(report.Lines), steps))
			}
			return
		}

		nextVisited := make(map[string]bool, len(visited)+1)
		for k, v := range visited {
			nextVisited[k] = v
		}
		nextVisited[core] = true

		isUser := b.Turn == repColor
		expandedAny := false
		for _, rm := range moves {
			var mv chess.Move
			var err error
			if rm.UCI != "" {
				mv, err = b.ParseUCI(rm.UCI)
			} else if rm.SAN != "" {
				mv, err = b.ParseSAN(rm.SAN)
			}
			if err != nil {
				continue
			}

			san := rm.SAN
			if san == "" {
				san = b.MoveToSAN(mv)
			}
			uci := mv.String()
			mastered := isMoveMastered(rm)

			if isUser {
				rmCopy := rm
				rmCopy.Mastered = mastered
				seenUserNodes[userNodeKey{core: core, uci: strings.ToLower(uci)}] = rmCopy
			}

			nb := b.Clone()
			nb.MakeMove(mv)
			step := LineStep{
				Ply:        len(steps) + 1,
				FEN:        b.FEN(),
				CoreFEN:    core,
				UCI:        uci,
				SAN:        san,
				NextFEN:    nb.FEN(),
				Comment:    rm.Comment,
				IsUserMove: isUser,
				Practice:   rm.Practice,
				Correct:    rm.Correct,
				Streak:     rm.Streak,
				Mastered:   mastered,
			}

			expandedAny = true
			nextSteps := make([]LineStep, len(steps)+1)
			copy(nextSteps, steps)
			nextSteps[len(steps)] = step
			dfs(nb, nextSteps, nextVisited)
		}

		if !expandedAny && len(steps) > 0 {
			report.Lines = append(report.Lines, buildRepertoireLine(len(report.Lines), steps))
		}
	}

	startBoard := chess.NewBoard()
	dfs(startBoard, nil, make(map[string]bool))

	// Also count any disconnected user-turn positions in rep.Positions not reached from starting FEN
	for core, moves := range rep.Positions {
		parts := strings.Fields(core)
		if len(parts) >= 2 {
			turnColor := chess.White
			if parts[1] == "b" {
				turnColor = chess.Black
			}
			if turnColor == repColor {
				for _, rm := range moves {
					key := userNodeKey{core: core, uci: strings.ToLower(rm.UCI)}
					if _, exists := seenUserNodes[key]; !exists {
						rmCopy := rm
						rmCopy.Mastered = isMoveMastered(rm)
						seenUserNodes[key] = rmCopy
					}
				}
			}
		}
	}

	for _, rm := range seenUserNodes {
		report.TotalUserNodes++
		if rm.Mastered {
			report.MasteredUserNodes++
		} else if rm.Practice == 0 {
			report.UntestedUserNodes++
		} else {
			report.NeedsWorkUserNodes++
		}
	}

	if report.TotalUserNodes > 0 {
		report.MasteryPct = math.Round((float64(report.MasteredUserNodes)/float64(report.TotalUserNodes))*1000.0) / 10.0
	}

	report.TotalLines = len(report.Lines)
	for i := range report.Lines {
		if report.Lines[i].IsMastered {
			report.MasteredLines++
		} else if report.NextTargetLineIndex == -1 {
			report.NextTargetLineIndex = i
		}
	}
	if report.NextTargetLineIndex == -1 && len(report.Lines) > 0 {
		report.NextTargetLineIndex = 0
	}

	return report, nil
}

func buildRepertoireLine(index int, steps []LineStep) RepertoireLine {
	var sanParts []string
	userMoves := 0
	masteredMoves := 0
	keyComment := ""

	for _, s := range steps {
		if s.Ply%2 == 1 {
			moveNum := (s.Ply + 1) / 2
			sanParts = append(sanParts, fmt.Sprintf("%d.%s", moveNum, s.SAN))
		} else {
			sanParts = append(sanParts, s.SAN)
		}
		if s.IsUserMove {
			userMoves++
			if s.Mastered {
				masteredMoves++
			}
		}
		if s.Comment != "" {
			keyComment = s.Comment
		}
	}

	pct := 0.0
	isMastered := false
	if userMoves > 0 {
		pct = math.Round((float64(masteredMoves)/float64(userMoves))*1000.0) / 10.0
		isMastered = masteredMoves == userMoves
	}

	name := keyComment
	if name == "" {
		name = fmt.Sprintf("Variation #%d", index+1)
	}

	return RepertoireLine{
		Index:          index,
		Name:           name,
		SANLine:        strings.Join(sanParts, " "),
		Steps:          steps,
		UserMovesCount: userMoves,
		MasteredCount:  masteredMoves,
		MasteryPct:     pct,
		IsMastered:     isMastered,
		KeyComment:     keyComment,
	}
}

// CompileToPolyglotBook compiles a repertoire into a standard Polyglot (.bin) opening book at targetPath.
func (m *Manager) CompileToPolyglotBook(repID string, targetPath string) (int, error) {
	m.mu.RLock()
	rep, ok := m.inMem[repID]
	m.mu.RUnlock()
	if !ok || rep == nil {
		return 0, fmt.Errorf("repertoire %q not found", repID)
	}

	var entries []polyglot.BookEntry
	for coreFEN, moves := range rep.Positions {
		fullFEN := coreFEN + " - 0 1"
		b, err := chess.ParseFEN(fullFEN)
		if err != nil {
			continue
		}
		hash := polyglot.HashBoard(b)
		for idx, rm := range moves {
			var mv chess.Move
			var parseErr error
			if rm.UCI != "" {
				mv, parseErr = b.ParseUCI(rm.UCI)
			} else if rm.SAN != "" {
				mv, parseErr = b.ParseSAN(rm.SAN)
			}
			if parseErr != nil {
				continue
			}
			raw := polyglot.EncodeRawMove(b, mv)
			weight := uint16(100 - idx*15)
			if idx >= 6 {
				weight = 10
			}
			if rm.Correct > 0 {
				bonus := rm.Correct * 5
				if int(weight)+bonus <= 65535 {
					weight += uint16(bonus)
				}
			}
			entries = append(entries, polyglot.BookEntry{
				Key:     hash,
				RawMove: raw,
				Weight:  weight,
				Learn:   uint32(rm.Practice),
			})
		}
	}

	if len(entries) == 0 {
		return 0, fmt.Errorf("repertoire %q has no valid moves to compile", rep.Name)
	}

	if err := polyglot.WriteBook(targetPath, entries); err != nil {
		return 0, err
	}
	return len(entries), nil
}


