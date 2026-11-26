package academy

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"newscid/pkg/database/scid"
	"newscid/pkg/playerbot"
)

// OpponentRecord tracks the user's career score against one personality.
type OpponentRecord struct {
	Wins       int    `json:"wins"`
	Draws      int    `json:"draws"`
	Losses     int    `json:"losses"`
	LastPlayed string `json:"lastPlayed,omitempty"`
}

// BossExamState tracks a 2-game promotion exam (White + Black) against a Tier Boss.
type BossExamState struct {
	Active       bool    `json:"active"`
	GamesPlayed  int     `json:"gamesPlayed"`
	Score        float64 `json:"score"` // need >= 1.5 out of 2.0 to promote
	Passed       bool    `json:"passed"`
	NextColor    string  `json:"nextColor"` // "white" or "black"
	LastFinished string  `json:"lastFinished,omitempty"`
}

// CareerProfile holds the user's persistent Academy & Career Ladder progression.
type CareerProfile struct {
	PlayerName          string                    `json:"playerName"`
	Rating              int                       `json:"rating"`
	PeakRating          int                       `json:"peakRating"`
	CurrentTier         int                       `json:"currentTier"`
	UnlockedTiers       []int                     `json:"unlockedTiers"`
	Records             map[string]OpponentRecord `json:"records"`
	CompletedLessons    map[string]bool           `json:"completedLessons"`
	BossExams           map[int]BossExamState     `json:"bossExams"`
	CustomPersonalities []Personality             `json:"customPersonalities"`
}

// CareerOverview is the combined payload sent to the Academy & Career UI.
type CareerOverview struct {
	Profile       CareerProfile `json:"profile"`
	Tiers         []TierInfo    `json:"tiers"`
	Personalities []Personality `json:"personalities"`
	Lessons       []Lesson      `json:"lessons"`
}

// GameResultUpdate summarizes what changed after recording a Career game.
type GameResultUpdate struct {
	OldRating    int    `json:"oldRating"`
	NewRating    int    `json:"newRating"`
	RatingDelta  int    `json:"ratingDelta"`
	Promoted     bool   `json:"promoted"`
	NewTier      int    `json:"newTier"`
	ExamMessage  string `json:"examMessage,omitempty"`
	Profile      CareerProfile `json:"profile"`
}

// Manager manages the persistent CareerProfile and custom/DB personalities.
type Manager struct {
	mu       sync.RWMutex
	filePath string
	profile  CareerProfile
}

// DefaultCareerPath returns ~/.config/newscid/career.json.
func DefaultCareerPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "career.json"
	}
	return filepath.Join(home, ".config", "newscid", "career.json")
}

func defaultProfile() CareerProfile {
	return CareerProfile{
		PlayerName:          "Challenger",
		Rating:              900,
		PeakRating:          900,
		CurrentTier:         1,
		UnlockedTiers:       []int{1},
		Records:             make(map[string]OpponentRecord),
		CompletedLessons:    make(map[string]bool),
		BossExams:           make(map[int]BossExamState),
		CustomPersonalities: []Personality{},
	}
}

// NewManager loads or initializes the Career profile at filePath.
func NewManager(filePath string) *Manager {
	if strings.TrimSpace(filePath) == "" {
		filePath = DefaultCareerPath()
	}
	m := &Manager{
		filePath: filePath,
		profile:  defaultProfile(),
	}
	m.load()
	return m
}

func (m *Manager) load() {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return
	}
	var p CareerProfile
	if err := json.Unmarshal(data, &p); err != nil {
		return
	}
	if p.Rating <= 0 {
		p.Rating = 900
	}
	if p.PeakRating < p.Rating {
		p.PeakRating = p.Rating
	}
	if p.CurrentTier < 1 || p.CurrentTier > 6 {
		p.CurrentTier = 1
	}
	if len(p.UnlockedTiers) == 0 {
		p.UnlockedTiers = []int{1}
	}
	if p.Records == nil {
		p.Records = make(map[string]OpponentRecord)
	}
	if p.CompletedLessons == nil {
		p.CompletedLessons = make(map[string]bool)
	}
	if p.BossExams == nil {
		p.BossExams = make(map[int]BossExamState)
	}
	if p.CustomPersonalities == nil {
		p.CustomPersonalities = []Personality{}
	}
	m.profile = p
}

func (m *Manager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0o644)
}

// GetOverview returns the complete Academy & Career state for the frontend.
func (m *Manager) GetOverview() CareerOverview {
	m.mu.RLock()
	defer m.mu.RUnlock()

	all := BuiltinPersonalities()
	all = append(all, m.profile.CustomPersonalities...)

	return CareerOverview{
		Profile:       m.profile,
		Tiers:         Tiers(),
		Personalities: all,
		Lessons:       Curriculum(),
	}
}

// FindPersonality finds either a built-in or custom/database personality by ID.
func (m *Manager) FindPersonality(id string) (Personality, bool) {
	if p, ok := FindBuiltinPersonality(id); ok {
		return p, true
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.profile.CustomPersonalities {
		if p.ID == id {
			return p, true
		}
	}
	return Personality{}, false
}

// RecordGameResult updates Elo, W/D/L record, and Tier Boss promotion exam state.
// outcome must be "win", "draw", or "loss" (from the user's perspective).
func (m *Manager) RecordGameResult(personalityID, outcome string) (GameResultUpdate, error) {
	p, ok := m.FindPersonality(personalityID)
	if !ok {
		return GameResultUpdate{}, fmt.Errorf("unknown personality: %s", personalityID)
	}

	var score float64
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "win", "1-0", "1":
		score = 1.0
	case "draw", "1/2-1/2", "0.5":
		score = 0.5
	case "loss", "0-1", "0":
		score = 0.0
	default:
		return GameResultUpdate{}, fmt.Errorf("invalid outcome: %s", outcome)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	oldRating := m.profile.Rating
	expected := 1.0 / (1.0 + math.Pow(10.0, float64(p.Elo-oldRating)/400.0))
	kFactor := 32.0
	if oldRating >= 2000 {
		kFactor = 20.0
	}
	delta := int(math.Round(kFactor * (score - expected)))
	newRating := oldRating + delta
	if newRating < 400 {
		newRating = 400
	}
	m.profile.Rating = newRating
	if newRating > m.profile.PeakRating {
		m.profile.PeakRating = newRating
	}

	rec := m.profile.Records[personalityID]
	switch score {
	case 1.0:
		rec.Wins++
	case 0.5:
		rec.Draws++
	default:
		rec.Losses++
	}
	rec.LastPlayed = time.Now().UTC().Format(time.RFC3339)
	m.profile.Records[personalityID] = rec

	promoted := false
	examMsg := ""

	// Check Tier Boss Promotion Exam
	if p.IsBoss && p.Tier >= 1 && p.Tier <= 6 {
		exam := m.profile.BossExams[p.Tier]
		if !exam.Active && !exam.Passed {
			exam.Active = true
			exam.NextColor = "white"
		}
		if exam.Active {
			exam.GamesPlayed++
			exam.Score += score
			if exam.NextColor == "white" {
				exam.NextColor = "black"
			} else {
				exam.NextColor = "white"
			}
			if exam.Score >= 1.5 {
				exam.Passed = true
				exam.Active = false
				exam.LastFinished = time.Now().UTC().Format(time.RFC3339)
				if p.Tier < 6 {
					nextTier := p.Tier + 1
					m.unlockTierLocked(nextTier)
					if m.profile.CurrentTier < nextTier {
						m.profile.CurrentTier = nextTier
					}
					promoted = true
					examMsg = fmt.Sprintf("Promotion Exam PASSED (%.1f/2)! Tier %d unlocked!", exam.Score, nextTier)
				} else {
					promoted = true
					examMsg = fmt.Sprintf("Grandmaster Hall Championship WON (%.1f/2)!", exam.Score)
				}
			} else if exam.GamesPlayed >= 2 {
				exam.Active = false
				exam.LastFinished = time.Now().UTC().Format(time.RFC3339)
				examMsg = fmt.Sprintf("Promotion Exam finished %.1f/2 (need 1.5). Review Tier %d lessons and retry!", exam.Score, p.Tier)
				exam.GamesPlayed = 0
				exam.Score = 0
			} else {
				examMsg = fmt.Sprintf("Promotion Exam Game 1/2 complete (%.1f pts). Play Game 2 as %s!", exam.Score, exam.NextColor)
			}
			m.profile.BossExams[p.Tier] = exam
		}
	}

	_ = m.saveLocked()
	return GameResultUpdate{
		OldRating:   oldRating,
		NewRating:   newRating,
		RatingDelta: newRating - oldRating,
		Promoted:    promoted,
		NewTier:     m.profile.CurrentTier,
		ExamMessage: examMsg,
		Profile:     m.profile,
	}, nil
}

func (m *Manager) unlockTierLocked(tier int) {
	if tier < 1 || tier > 6 {
		return
	}
	for _, t := range m.profile.UnlockedTiers {
		if t == tier {
			return
		}
	}
	m.profile.UnlockedTiers = append(m.profile.UnlockedTiers, tier)
	sort.Ints(m.profile.UnlockedTiers)
}

// CompleteLesson marks a Chess University lesson as completed and persists progress.
func (m *Manager) CompleteLesson(lessonID string) (CareerProfile, error) {
	if _, ok := FindLesson(lessonID); !ok {
		return CareerProfile{}, fmt.Errorf("unknown lesson: %s", lessonID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profile.CompletedLessons[lessonID] = true
	_ = m.saveLocked()
	return m.profile, nil
}

// TestOutToTier allows experienced players to unlock directly up to targetTier (1..6).
func (m *Manager) TestOutToTier(targetTier int) (CareerProfile, error) {
	if targetTier < 1 || targetTier > 6 {
		return CareerProfile{}, fmt.Errorf("tier must be between 1 and 6")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for t := 1; t <= targetTier; t++ {
		m.unlockTierLocked(t)
	}
	if m.profile.CurrentTier < targetTier {
		m.profile.CurrentTier = targetTier
	}
	tierInfo := Tiers()[targetTier-1]
	if m.profile.Rating < tierInfo.TargetElo-200 {
		m.profile.Rating = tierInfo.TargetElo - 150
		if m.profile.Rating > m.profile.PeakRating {
			m.profile.PeakRating = m.profile.Rating
		}
	}
	_ = m.saveLocked()
	return m.profile, nil
}

// SaveCustomPersonality creates or updates a custom slider-based personality.
func (m *Manager) SaveCustomPersonality(p Personality) (Personality, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return Personality{}, fmt.Errorf("personality name cannot be empty")
	}
	if p.Elo < 500 {
		p.Elo = 500
	}
	if p.Elo > 3000 {
		p.Elo = 3000
	}
	if p.Tier < 1 || p.Tier > 6 {
		p.Tier = eloToTier(p.Elo)
	}
	if p.Source == "" {
		p.Source = "custom"
	}
	if p.AvatarColor == "" {
		p.AvatarColor = "#10b981"
	}
	if p.AvatarSymbol == "" {
		p.AvatarSymbol = "♘"
	}
	if p.Title == "" {
		p.Title = "Custom Opponent"
	}
	if p.StyleTag == "" {
		p.StyleTag = deriveStyleTag(p.Sliders)
	}
	if p.ID == "" {
		slug := strings.ToLower(strings.ReplaceAll(p.Name, " ", "_"))
		p.ID = fmt.Sprintf("custom_%s_%d", slug, time.Now().UnixMilli()%100000)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	replaced := false
	for i, existing := range m.profile.CustomPersonalities {
		if existing.ID == p.ID {
			m.profile.CustomPersonalities[i] = p
			replaced = true
			break
		}
	}
	if !replaced {
		m.profile.CustomPersonalities = append(m.profile.CustomPersonalities, p)
	}
	_ = m.saveLocked()
	return p, nil
}

// DeleteCustomPersonality removes a custom or database-built personality by ID.
func (m *Manager) DeleteCustomPersonality(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := -1
	for i, p := range m.profile.CustomPersonalities {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("custom personality not found: %s", id)
	}
	m.profile.CustomPersonalities = append(m.profile.CustomPersonalities[:idx], m.profile.CustomPersonalities[idx+1:]...)
	return m.saveLocked()
}

// BuildPersonalityFromDB extracts a player's games from an open Scid database,
// derives their Elo, opening book, and playing style, and saves them as a playable Personality.
func (m *Manager) BuildPersonalityFromDB(db *scid.Database, playerName string) (Personality, error) {
	pb, err := playerbot.BuildPersonality(db, playerName)
	if err != nil {
		return Personality{}, err
	}

	elo := pb.EstimatedElo
	if elo <= 0 {
		elo = 1600
	}
	agg := int(math.Round(pb.AggressionRating))
	if agg < 10 {
		agg = 50
	}
	if agg > 100 {
		agg = 100
	}
	trade := 50
	if pb.AvgGamePlies > 85 {
		trade = 72
	} else if pb.AvgGamePlies > 0 && pb.AvgGamePlies < 55 {
		trade = 32
	}
	sharp := clampInt(elo/25, 20, 98)
	solid := clampInt(100-agg/3, 40, 95)
	blunder := clampInt((2500-elo)/22, 2, 75)

	book := make(map[string][]string)
	mergeBook := func(src map[string][]playerbot.BookMove) {
		for fenKey, moves := range src {
			boardOnly := strings.Split(fenKey, " ")[0]
			if len(book) > 400 {
				break
			}
			for i, bm := range moves {
				if i >= 3 {
					break
				}
				book[boardOnly] = append(book[boardOnly], bm.UCI)
			}
		}
	}
	mergeBook(pb.WhiteBook)
	mergeBook(pb.BlackBook)

	ecoSummary := "Balanced repertoire"
	if len(pb.TopWhiteECOs) > 0 || len(pb.TopBlackECOs) > 0 {
		parts := []string{}
		if len(pb.TopWhiteECOs) > 0 {
			parts = append(parts, "White: "+strings.Join(pb.TopWhiteECOs[:min(3, len(pb.TopWhiteECOs))], ", "))
		}
		if len(pb.TopBlackECOs) > 0 {
			parts = append(parts, "Black: "+strings.Join(pb.TopBlackECOs[:min(3, len(pb.TopBlackECOs))], ", "))
		}
		ecoSummary = strings.Join(parts, " | ")
	}

	bio := fmt.Sprintf("Built from %d database games (%.1f%% score, peak %d Elo). %s.",
		pb.TotalGames, pb.WinRate, pb.PeakElo, ecoSummary)

	slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(pb.Name), " ", "_"))
	slug = strings.ReplaceAll(slug, ",", "")
	pers := Personality{
		ID:           "db_" + slug,
		Name:         pb.Name,
		Title:        "Database Clone",
		Tier:         eloToTier(elo),
		Elo:          elo,
		AvatarColor:  "#06b6d4",
		AvatarSymbol: "♜",
		StyleTag:     pb.PlayStyle,
		Bio:          bio,
		Source:       "database",
		SourcePlayer: pb.Name,
		SourceGames:  pb.TotalGames,
		Sliders: PersonalitySliders{
			Aggression:         agg,
			TradeTendency:      trade,
			TacticalSharpness:  sharp,
			PositionalSolidity: solid,
			BlunderRate:        blunder,
		},
		OpeningBook: book,
	}

	return m.SaveCustomPersonality(pers)
}

func eloToTier(elo int) int {
	switch {
	case elo < 1000:
		return 1
	case elo < 1300:
		return 2
	case elo < 1600:
		return 3
	case elo < 1900:
		return 4
	case elo < 2200:
		return 5
	default:
		return 6
	}
}

func deriveStyleTag(s PersonalitySliders) string {
	switch {
	case s.Aggression >= 75:
		return "Aggressive Attacker"
	case s.TradeTendency >= 72:
		return "Endgame Grinder"
	case s.PositionalSolidity >= 78:
		return "Solid Positional"
	case s.TacticalSharpness >= 75:
		return "Sharp Tactician"
	default:
		return "Balanced All-Rounder"
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
