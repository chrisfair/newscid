package training

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

	"newscid/pkg/chess"
	"newscid/pkg/explainer"
)

// ThemeWeaknessStat tracks the user's historical mistakes in online/database games
// and adaptive drill mastery for a specific tactical motif.
type ThemeWeaknessStat struct {
	ThemeKey        string  `json:"themeKey"`
	ThemeName       string  `json:"themeName"`
	Category        string  `json:"category"`
	Description     string  `json:"description"`
	GameMissCount   int     `json:"gameMissCount"`
	TotalEvalLostCp int     `json:"totalEvalLostCp"`
	DrillsAttempted int     `json:"drillsAttempted"`
	DrillsSolved    int     `json:"drillsSolved"`
	DrillsFailed    int     `json:"drillsFailed"`
	CurrentStreak   int     `json:"currentStreak"`
	BestStreak      int     `json:"bestStreak"`
	AccuracyPct     float64 `json:"accuracyPct"`
	MasteryScore    float64 `json:"masteryScore"`   // 0.0 - 100.0
	PriorityWeight  int     `json:"priorityWeight"` // Higher means drilled harder
	CoachingHabit   string  `json:"coachingHabit"`
}

// AdaptiveDrillItem represents a customized, high-instructional tactical exercise
// tailored to the user's diagnosed online game weaknesses.
type AdaptiveDrillItem struct {
	ID                string   `json:"id"`
	SourceType        string   `json:"sourceType"` // "online_game_mistake" | "targeted_reinforcement"
	White             string   `json:"white"`
	Black             string   `json:"black"`
	Event             string   `json:"event"`
	Date              string   `json:"date"`
	Ply               int      `json:"ply"`
	MoveNumber        int      `json:"moveNumber"`
	FEN               string   `json:"fen"`
	Turn              string   `json:"turn"` // "w" | "b"
	SolutionUCI       []string `json:"solutionUci"`
	SolutionSAN       []string `json:"solutionSan"`
	PlayedBlunderUCI  string   `json:"playedBlunderUci,omitempty"`
	PlayedBlunderSAN  string   `json:"playedBlunderSan,omitempty"`
	PrimaryTheme      string   `json:"primaryTheme"`
	ThemeName         string   `json:"themeName"`
	Themes            []string `json:"themes"`
	DifficultyTier    string   `json:"difficultyTier"`
	EstimatedRating   int      `json:"estimatedRating"`
	EvalDeltaCp       int      `json:"evalDeltaCp"`
	DiagnosisHeadline string   `json:"diagnosisHeadline"`
	WhatWentWrong     string   `json:"whatWentWrong"`
	VisualCues        []string `json:"visualCues"`
	StepByStepCoach   string   `json:"stepByStepCoach"`
	PostSolveTakeaway string   `json:"postSolveTakeaway"`
	Attempted         bool     `json:"attempted"`
	SolvedFirstTry    bool     `json:"solvedFirstTry"`
	AttemptsCount     int      `json:"attemptsCount"`
	TimeTakenSec      int      `json:"timeTakenSec"`
}

// CoachingSessionSummary records a completed or in-progress adaptive coaching session for progress tracking.
type CoachingSessionSummary struct {
	SessionID         string   `json:"sessionId"`
	StartedAt         string   `json:"startedAt"`
	CompletedAt       string   `json:"completedAt,omitempty"`
	GamesAnalyzed     int      `json:"gamesAnalyzed"`
	TotalMistakes     int      `json:"totalMistakes"`
	FocusThemes       []string `json:"focusThemes"`
	TotalDrills       int      `json:"totalDrills"`
	CompletedDrills   int      `json:"completedDrills"`
	FirstTrySolved    int      `json:"firstTrySolved"`
	SessionAccuracy   float64  `json:"sessionAccuracy"`
	ReinforcedOnMiss  int      `json:"reinforcedOnMiss"`
	OverallMasteryPct float64  `json:"overallMasteryPct"`
}

// AdaptiveCoachDashboard encapsulates the user's diagnosed weakness profile, active instructional queue, and progress history.
type AdaptiveCoachDashboard struct {
	PlayerIdentifiers []string                 `json:"playerIdentifiers"`
	GamesAnalyzed     int                      `json:"gamesAnalyzed"`
	TotalGameMistakes int                      `json:"totalGameMistakes"`
	OverallMasteryPct float64                  `json:"overallMasteryPct"`
	TotalDrillsSolved int                      `json:"totalDrillsSolved"`
	CoachSummaryText  string                   `json:"coachSummaryText"`
	TopWeaknesses     []ThemeWeaknessStat      `json:"topWeaknesses"`
	AllThemeStats     []ThemeWeaknessStat      `json:"allThemeStats"`
	ActiveSession     *CoachingSessionSummary  `json:"activeSession,omitempty"`
	ActiveQueue       []AdaptiveDrillItem      `json:"activeQueue"`
	SessionHistory    []CoachingSessionSummary `json:"sessionHistory"`
}

// AdaptiveDrillResult is returned when the user submits or completes an adaptive drill.
type AdaptiveDrillResult struct {
	Drill              AdaptiveDrillItem      `json:"drill"`
	UpdatedTheme       ThemeWeaknessStat      `json:"updatedTheme"`
	ReinforcementAdded bool                   `json:"reinforcementAdded"`
	AddedDrill         *AdaptiveDrillItem     `json:"addedDrill,omitempty"`
	Dashboard          AdaptiveCoachDashboard `json:"dashboard"`
}

type adaptiveCoachStore struct {
	PlayerIdentifiers []string                     `json:"playerIdentifiers"`
	GamesAnalyzed     int                          `json:"gamesAnalyzed"`
	TotalGameMistakes int                          `json:"totalGameMistakes"`
	ThemeStats        map[string]ThemeWeaknessStat `json:"themeStats"`
	MinedBank         []AdaptiveDrillItem          `json:"minedBank"`
	ActiveSession     *CoachingSessionSummary      `json:"activeSession,omitempty"`
	ActiveQueue       []AdaptiveDrillItem          `json:"activeQueue"`
	SessionHistory    []CoachingSessionSummary     `json:"sessionHistory"`
}

// AdaptiveCoachManager manages personalized weakness diagnosis, high-instructional drill generation, and persistent progress.
type AdaptiveCoachManager struct {
	mu       sync.RWMutex
	filePath string
	store    adaptiveCoachStore
}

// NewAdaptiveCoachManager initializes or loads the persistent adaptive coaching store.
func NewAdaptiveCoachManager(filePath string) (*AdaptiveCoachManager, error) {
	if strings.TrimSpace(filePath) == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			cfgDir = os.TempDir()
		}
		filePath = filepath.Join(cfgDir, "newscid", "adaptive_coach.json")
	}

	mgr := &AdaptiveCoachManager{
		filePath: filePath,
		store: adaptiveCoachStore{
			ThemeStats:     make(map[string]ThemeWeaknessStat),
			MinedBank:      make([]AdaptiveDrillItem, 0),
			ActiveQueue:    make([]AdaptiveDrillItem, 0),
			SessionHistory: make([]CoachingSessionSummary, 0),
		},
	}

	mgr.load()
	return mgr, nil
}

func (m *AdaptiveCoachManager) load() {
	data, err := os.ReadFile(m.filePath)
	if err != nil || len(data) == 0 {
		return
	}
	var st adaptiveCoachStore
	if err := json.Unmarshal(data, &st); err == nil {
		if st.ThemeStats == nil {
			st.ThemeStats = make(map[string]ThemeWeaknessStat)
		}
		m.store = st
	}
}

func (m *AdaptiveCoachManager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0644)
}

// GetDashboard returns the current adaptive coaching dashboard and progress history.
func (m *AdaptiveCoachManager) GetDashboard() AdaptiveCoachDashboard {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buildDashboardLocked()
}

// ScanAndBuildCurriculum analyzes the provided online/database games for tactical mistakes and hanging pieces,
// updates the user's weakness profile, and builds a customized high-instructional drill queue weighted toward the user's weaknesses.
func (m *AdaptiveCoachManager) ScanAndBuildCurriculum(games []*chess.Game, playerIdentifiers []string, focusTheme string, batchSize int) (*AdaptiveCoachDashboard, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if batchSize <= 0 {
		batchSize = 12
	}
	if batchSize > 40 {
		batchSize = 40
	}

	var cleanIDs []string
	seenID := make(map[string]bool)
	for _, id := range playerIdentifiers {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			low := strings.ToLower(trimmed)
			if !seenID[low] {
				seenID[low] = true
				cleanIDs = append(cleanIDs, trimmed)
			}
		}
	}
	if len(cleanIDs) > 0 {
		m.store.PlayerIdentifiers = cleanIDs
	}

	// Reset game-occurrence counters before re-scanning games while preserving historical drill mastery stats
	for k, st := range m.store.ThemeStats {
		st.GameMissCount = 0
		st.TotalEvalLostCp = 0
		m.store.ThemeStats[k] = st
	}

	minedDrills, gamesMatched := extractCoachingDrillsFromGames(games, cleanIDs)
	m.store.GamesAnalyzed = gamesMatched
	m.store.TotalGameMistakes = len(minedDrills)
	m.store.MinedBank = minedDrills

	// Tally game mistakes per tactical theme
	for _, d := range minedDrills {
		m.recordGameMistakeThemeLocked(d.PrimaryTheme, d.EvalDeltaCp)
		for _, secTheme := range d.Themes {
			if secTheme != d.PrimaryTheme {
				if id, ok := LookupThemeID(secTheme); ok && lichessThemeCatalog[id].Actionable {
					m.recordGameMistakeThemeLocked(secTheme, d.EvalDeltaCp/2)
				}
			}
		}
	}

	// Ensure core themes exist if no mistakes were found yet
	if len(m.store.ThemeStats) == 0 {
		seedKeys := []ThemeID{ThemeHangingPiece, ThemeFork, ThemePin, ThemeBackRankMate, ThemeDiscoveredAttack, ThemeDeflection}
		for _, tid := range seedKeys {
			info := lichessThemeCatalog[tid]
			st := m.ensureThemeStatLocked(info.Key)
			m.store.ThemeStats[info.Key] = st
		}
	}

	// Recalculate priority weights for all themes
	for k, st := range m.store.ThemeStats {
		st.PriorityWeight = computeThemePriorityWeight(st)
		m.store.ThemeStats[k] = st
	}

	// Build customized drill queue weighted heavily toward the user's top missed themes
	m.buildWeightedQueueLocked(focusTheme, batchSize)

	_ = m.saveLocked()
	dash := m.buildDashboardLocked()
	return &dash, nil
}

// RecordDrillAttempt records whether the user solved a customized drill on the first try.
// If the user misses the drill, that tactical theme's priority weight increases immediately and
// an extra reinforcement drill of that exact motif is pushed into the active session queue.
func (m *AdaptiveCoachManager) RecordDrillAttempt(drillID string, solvedFirstTry bool, timeTakenSec int) (*AdaptiveDrillResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i := range m.store.ActiveQueue {
		if m.store.ActiveQueue[i].ID == drillID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("drill %q not found in active coaching queue", drillID)
	}

	drill := m.store.ActiveQueue[idx]
	wasAlreadyAttempted := drill.Attempted
	drill.Attempted = true
	drill.AttemptsCount++
	drill.TimeTakenSec += maxInt(1, timeTakenSec)
	if !wasAlreadyAttempted {
		drill.SolvedFirstTry = solvedFirstTry
	}
	m.store.ActiveQueue[idx] = drill

	// Update ThemeWeaknessStat
	st := m.ensureThemeStatLocked(drill.PrimaryTheme)
	st.DrillsAttempted++
	if solvedFirstTry {
		st.DrillsSolved++
		st.CurrentStreak++
		if st.CurrentStreak > st.BestStreak {
			st.BestStreak = st.CurrentStreak
		}
	} else {
		st.DrillsFailed++
		st.CurrentStreak = 0
	}
	st.AccuracyPct = calcPct(st.DrillsSolved, st.DrillsAttempted)
	st.MasteryScore = computeThemeMasteryScore(st)
	st.PriorityWeight = computeThemePriorityWeight(st)
	m.store.ThemeStats[drill.PrimaryTheme] = st

	// Update ActiveSession summary
	var addedDrill *AdaptiveDrillItem
	reinforcementAdded := false

	if m.store.ActiveSession != nil {
		completed := 0
		firstTry := 0
		for _, item := range m.store.ActiveQueue {
			if item.Attempted {
				completed++
				if item.SolvedFirstTry {
					firstTry++
				}
			}
		}
		m.store.ActiveSession.CompletedDrills = completed
		m.store.ActiveSession.FirstTrySolved = firstTry
		m.store.ActiveSession.SessionAccuracy = calcPct(firstTry, completed)

		// If the user missed this tactical motif, push an extra reinforcement drill of this exact theme into the queue!
		if !solvedFirstTry && len(m.store.ActiveQueue) < 30 {
			extra := m.findReinforcementDrillLocked(drill.PrimaryTheme, drill.FEN)
			if extra != nil {
				m.store.ActiveQueue = append(m.store.ActiveQueue, *extra)
				m.store.ActiveSession.TotalDrills = len(m.store.ActiveQueue)
				m.store.ActiveSession.ReinforcedOnMiss++
				addedDrill = extra
				reinforcementAdded = true
			}
		}

		// Record session into history when all drills in queue are completed
		if m.store.ActiveSession.CompletedDrills >= len(m.store.ActiveQueue) && m.store.ActiveSession.CompletedAt == "" {
			m.store.ActiveSession.CompletedAt = time.Now().UTC().Format(time.RFC3339)
			m.store.ActiveSession.OverallMasteryPct = m.computeOverallMasteryLocked()
			m.store.SessionHistory = append([]CoachingSessionSummary{*m.store.ActiveSession}, m.store.SessionHistory...)
			if len(m.store.SessionHistory) > 25 {
				m.store.SessionHistory = m.store.SessionHistory[:25]
			}
		}
	}

	_ = m.saveLocked()
	dash := m.buildDashboardLocked()

	return &AdaptiveDrillResult{
		Drill:              drill,
		UpdatedTheme:       st,
		ReinforcementAdded: reinforcementAdded,
		AddedDrill:         addedDrill,
		Dashboard:          dash,
	}, nil
}

func (m *AdaptiveCoachManager) ensureThemeStatLocked(themeKey string) ThemeWeaknessStat {
	if st, ok := m.store.ThemeStats[themeKey]; ok {
		return st
	}
	id, ok := LookupThemeID(themeKey)
	info := LichessThemeInfo{
		Key:         themeKey,
		Name:        themeKey,
		Category:    "Motifs",
		Description: "Tactical pattern recognition.",
	}
	if ok {
		info = lichessThemeCatalog[id]
	}
	return ThemeWeaknessStat{
		ThemeKey:       info.Key,
		ThemeName:      info.Name,
		Category:       info.Category,
		Description:    info.Description,
		MasteryScore:   50.0,
		PriorityWeight: 100,
		CoachingHabit:  coachingHabitForTheme(info.Key),
	}
}

func (m *AdaptiveCoachManager) recordGameMistakeThemeLocked(themeKey string, evalDeltaCp int) {
	st := m.ensureThemeStatLocked(themeKey)
	st.GameMissCount++
	st.TotalEvalLostCp += maxInt(100, evalDeltaCp)
	st.MasteryScore = computeThemeMasteryScore(st)
	st.PriorityWeight = computeThemePriorityWeight(st)
	m.store.ThemeStats[themeKey] = st
}

func computeThemeMasteryScore(st ThemeWeaknessStat) float64 {
	base := 55.0
	// Penalty for unaddressed game mistakes
	base -= float64(st.GameMissCount) * 6.0
	// Bonus/penalty from drill performance
	if st.DrillsAttempted > 0 {
		acc := float64(st.DrillsSolved) / float64(st.DrillsAttempted)
		base = (base * 0.35) + (acc * 100.0 * 0.65)
	}
	base += float64(st.CurrentStreak) * 5.0
	base -= float64(st.DrillsFailed) * 4.0
	if base < 5.0 {
		base = 5.0
	}
	if base > 100.0 {
		base = 100.0
	}
	return math.Round(base*10.0) / 10.0
}

func computeThemePriorityWeight(st ThemeWeaknessStat) int {
	weight := 100
	weight += st.GameMissCount * 40
	weight += (st.TotalEvalLostCp / 100) * 5
	weight += st.DrillsFailed * 45
	weight -= st.CurrentStreak * 15
	weight += int(100.0 - st.MasteryScore)
	if weight < 10 {
		weight = 10
	}
	return weight
}

func (m *AdaptiveCoachManager) computeOverallMasteryLocked() float64 {
	if len(m.store.ThemeStats) == 0 {
		return 0.0
	}
	sum := 0.0
	count := 0
	for _, st := range m.store.ThemeStats {
		sum += st.MasteryScore
		count++
	}
	if count == 0 {
		return 0.0
	}
	return math.Round((sum/float64(count))*10.0) / 10.0
}

func (m *AdaptiveCoachManager) buildWeightedQueueLocked(focusTheme string, batchSize int) {
	sortedStats := m.sortedWeaknessesLocked()
	var targetThemes []string
	if strings.TrimSpace(focusTheme) != "" && focusTheme != "auto" {
		targetThemes = append(targetThemes, strings.TrimSpace(focusTheme))
	}
	for _, st := range sortedStats {
		if len(targetThemes) >= 3 {
			break
		}
		if st.ThemeKey != focusTheme {
			targetThemes = append(targetThemes, st.ThemeKey)
		}
	}
	if len(targetThemes) == 0 {
		targetThemes = []string{"hangingPiece", "fork", "pin"}
	}

	var queue []AdaptiveDrillItem
	seenFEN := make(map[string]bool)

	// 1. Pull the user's actual game mistakes matching the priority themes first
	for _, tKey := range targetThemes {
		for _, item := range m.store.MinedBank {
			if len(queue) >= batchSize {
				break
			}
			if seenFEN[item.FEN] {
				continue
			}
			if item.PrimaryTheme == tKey || containsString(item.Themes, tKey) {
				seenFEN[item.FEN] = true
				queue = append(queue, item)
			}
		}
	}

	// 2. Pull any remaining game mistakes from the user's games
	for _, item := range m.store.MinedBank {
		if len(queue) >= batchSize {
			break
		}
		if !seenFEN[item.FEN] {
			seenFEN[item.FEN] = true
			queue = append(queue, item)
		}
	}

	// 3. Push targeted reinforcement drills for the user's #1 and #2 weaknesses from the built-in tactical library
	builtIn := GetBuiltInCyclicalPuzzles()
	for _, tKey := range targetThemes {
		for _, bp := range builtIn {
			if len(queue) >= batchSize {
				break
			}
			if seenFEN[bp.FEN] {
				continue
			}
			if bp.PrimaryTheme == tKey || containsString(bp.Themes, tKey) {
				seenFEN[bp.FEN] = true
				drill := convertCyclicalToAdaptiveDrill(bp, tKey, "targeted_reinforcement")
				queue = append(queue, drill)
			}
		}
	}

	// 4. Fill remaining slots if needed
	for _, bp := range builtIn {
		if len(queue) >= batchSize {
			break
		}
		if !seenFEN[bp.FEN] {
			seenFEN[bp.FEN] = true
			drill := convertCyclicalToAdaptiveDrill(bp, bp.PrimaryTheme, "targeted_reinforcement")
			queue = append(queue, drill)
		}
	}

	m.store.ActiveQueue = queue
	m.store.ActiveSession = &CoachingSessionSummary{
		SessionID:         fmt.Sprintf("coach-%d", time.Now().UnixMilli()),
		StartedAt:         time.Now().UTC().Format(time.RFC3339),
		GamesAnalyzed:     m.store.GamesAnalyzed,
		TotalMistakes:     m.store.TotalGameMistakes,
		FocusThemes:       targetThemes,
		TotalDrills:       len(queue),
		CompletedDrills:   0,
		FirstTrySolved:    0,
		SessionAccuracy:   100.0,
		OverallMasteryPct: m.computeOverallMasteryLocked(),
	}
}

func (m *AdaptiveCoachManager) findReinforcementDrillLocked(themeKey string, excludeFEN string) *AdaptiveDrillItem {
	seenInQueue := make(map[string]bool)
	seenInQueue[excludeFEN] = true
	for _, q := range m.store.ActiveQueue {
		seenInQueue[q.FEN] = true
	}

	for _, item := range m.store.MinedBank {
		if !seenInQueue[item.FEN] && (item.PrimaryTheme == themeKey || containsString(item.Themes, themeKey)) {
			cp := item
			cp.ID = fmt.Sprintf("%s-reinforce-%d", item.ID, len(m.store.ActiveQueue)+1)
			cp.DiagnosisHeadline = fmt.Sprintf("Immediate Reinforcement (%s): Drilled Harder After Miss", cp.ThemeName)
			return &cp
		}
	}

	for _, bp := range GetBuiltInCyclicalPuzzles() {
		if !seenInQueue[bp.FEN] && (bp.PrimaryTheme == themeKey || containsString(bp.Themes, themeKey)) {
			d := convertCyclicalToAdaptiveDrill(bp, themeKey, "targeted_reinforcement")
			d.ID = fmt.Sprintf("%s-reinforce-%d", d.ID, len(m.store.ActiveQueue)+1)
			d.DiagnosisHeadline = fmt.Sprintf("Immediate Reinforcement (%s): Drilled Harder After Miss", d.ThemeName)
			return &d
		}
	}
	return nil
}

func (m *AdaptiveCoachManager) sortedWeaknessesLocked() []ThemeWeaknessStat {
	var list []ThemeWeaknessStat
	for _, st := range m.store.ThemeStats {
		list = append(list, st)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].PriorityWeight != list[j].PriorityWeight {
			return list[i].PriorityWeight > list[j].PriorityWeight
		}
		if list[i].GameMissCount != list[j].GameMissCount {
			return list[i].GameMissCount > list[j].GameMissCount
		}
		return list[i].ThemeKey < list[j].ThemeKey
	})
	return list
}

func (m *AdaptiveCoachManager) buildDashboardLocked() AdaptiveCoachDashboard {
	allStats := m.sortedWeaknessesLocked()
	var topWeak []ThemeWeaknessStat
	for i := 0; i < len(allStats) && i < 4; i++ {
		topWeak = append(topWeak, allStats[i])
	}

	totalSolved := 0
	for _, st := range allStats {
		totalSolved += st.DrillsSolved
	}

	overallMastery := m.computeOverallMasteryLocked()
	summaryText := "Scan your online Lichess/Chess.com or local database games to diagnose recurring tactical blind spots and generate a customized coaching session."
	if len(topWeak) > 0 && (m.store.GamesAnalyzed > 0 || totalSolved > 0) {
		primary := topWeak[0]
		summaryText = fmt.Sprintf(
			"Analyzed %d game(s) (%d tactical mistakes detected). Top priority weakness: %s (%d game misses, %.1f%% mastery). Your training queue is weighted to drill %s harder until mastered.",
			m.store.GamesAnalyzed,
			m.store.TotalGameMistakes,
			primary.ThemeName,
			primary.GameMissCount,
			primary.MasteryScore,
			primary.ThemeName,
		)
	}

	return AdaptiveCoachDashboard{
		PlayerIdentifiers: m.store.PlayerIdentifiers,
		GamesAnalyzed:     m.store.GamesAnalyzed,
		TotalGameMistakes: m.store.TotalGameMistakes,
		OverallMasteryPct: overallMastery,
		TotalDrillsSolved: totalSolved,
		CoachSummaryText:  summaryText,
		TopWeaknesses:     topWeak,
		AllThemeStats:     allStats,
		ActiveSession:     m.store.ActiveSession,
		ActiveQueue:       m.store.ActiveQueue,
		SessionHistory:    m.store.SessionHistory,
	}
}

// extractCoachingDrillsFromGames scans the user's games for both missed tactical shots (especially hanging pieces,
// forks, pins, mates) and moves where the user blundered by leaving a piece hanging or walking into a tactic.
func extractCoachingDrillsFromGames(games []*chess.Game, playerNames []string) ([]AdaptiveDrillItem, int) {
	var normPlayers []string
	for _, n := range playerNames {
		if clean := strings.ToLower(strings.TrimSpace(n)); clean != "" {
			normPlayers = append(normPlayers, clean)
		}
	}

	var drills []AdaptiveDrillItem
	seenFENs := make(map[string]bool)
	gamesMatched := 0

	for gIdx, g := range games {
		if g == nil || g.RootMove == nil {
			continue
		}

		white := g.Tag("White")
		black := g.Tag("Black")
		userPlayWhite := true
		userPlayBlack := true

		if len(normPlayers) > 0 {
			userPlayWhite = matchesAnyPlayerName(white, normPlayers)
			userPlayBlack = matchesAnyPlayerName(black, normPlayers)
			if !userPlayWhite && !userPlayBlack {
				continue
			}
		}
		gamesMatched++

		b := chess.NewBoard()
		if g.StartFEN != "" && g.StartFEN != chess.StartingFEN {
			if parsed, err := chess.ParseFEN(g.StartFEN); err == nil {
				b = parsed
			}
		}

		curr := g.RootMove
		ply := 1
		for curr != nil {
			isUserTurn := (b.Turn == chess.White && userPlayWhite) || (b.Turn == chess.Black && userPlayBlack)
			fenBefore := b.FEN()

			if isUserTurn && !seenFENs[fenBefore] && ply >= 4 {
				if item, ok := detectUserTacticalErrorAtMove(b, curr, g, gIdx+1, ply); ok {
					seenFENs[fenBefore] = true
					drills = append(drills, item)
					if len(drills) >= 60 {
						return drills, gamesMatched
					}
				}
			}

			b.MakeMove(curr.Move)
			curr = curr.Next
			ply++
		}
	}

	return drills, gamesMatched
}

func matchesAnyPlayerName(name string, normTargets []string) bool {
	low := strings.ToLower(strings.TrimSpace(name))
	if low == "" {
		return false
	}
	for _, t := range normTargets {
		if low == t || strings.Contains(low, t) || strings.Contains(t, low) {
			return true
		}
	}
	return false
}

func detectUserTacticalErrorAtMove(b *chess.Board, curr *chess.MoveNode, g *chess.Game, gameNum int, ply int) (AdaptiveDrillItem, bool) {
	us := b.Turn
	them := us.Other()
	legal := b.GenerateLegalMoves()
	if len(legal) <= 1 {
		return AdaptiveDrillItem{}, false
	}

	// Ensure curr.Move is legal on b
	playedLegal := false
	for _, lm := range legal {
		if lm.Equals(curr.Move) {
			curr.Move = lm
			playedLegal = true
			break
		}
	}
	if !playedLegal {
		return AdaptiveDrillItem{}, false
	}

	playedSAN := b.MoveToSAN(curr.Move)
	playedUCI := curr.Move.String()

	// 1. Check if there was a direct tactical shot or hanging piece capture that the user missed
	var bestMove chess.Move
	bestScore := -99999
	for _, m := range legal {
		sc := scoreCoachCandidate(b, m, us, them)
		if sc > bestScore {
			bestScore = sc
			bestMove = m
		}
	}

	playedScore := scoreCoachCandidate(b, curr.Move, us, them)
	evalDelta := bestScore - playedScore

	// 2. Also check if curr.Move blundered by leaving a friendly piece hanging or allowing an immediate opponent tactic
	bAfterPlayed := b.Clone()
	bAfterPlayed.MakeMove(curr.Move)
	oppPunishMove, oppPunishScore := findStrongestOpponentPunishment(bAfterPlayed, them, us)

	bAfterBest := b.Clone()
	bAfterBest.MakeMove(bestMove)
	_, oppScoreAfterBest := findStrongestOpponentPunishment(bAfterBest, them, us)

	blunderedIntoHangingOrTactic := false
	if oppPunishScore >= 240 && (oppPunishScore-oppScoreAfterBest >= 180) && !curr.Move.Equals(bestMove) {
		blunderedIntoHangingOrTactic = true
		if oppPunishScore > evalDelta {
			evalDelta = oppPunishScore
		}
	}

	if curr.Move.Equals(bestMove) || (evalDelta < 200 && !blunderedIntoHangingOrTactic) {
		return AdaptiveDrillItem{}, false
	}

	solUCI := []string{bestMove.String()}
	solSAN := formatSolutionSAN(b.FEN(), solUCI)
	themes, tierID, rating := ClassifyPositionThemes(b.FEN(), solUCI, evalDelta)

	// If the user hung a piece or missed capturing an undefended piece, ensure "hangingPiece" is tagged
	capturedOnBest := b.Squares[bestMove.To]
	if (capturedOnBest != chess.Empty && !isSquareDefendedBy(b, bestMove.To, them)) || blunderedIntoHangingOrTactic {
		if !containsString(themes, "hangingPiece") {
			themes = append([]string{"hangingPiece"}, themes...)
		}
	}

	mask := ThemeMaskFromKeys(themes)
	primary := PrimaryThemeFromMask(mask)
	if containsString(themes, "hangingPiece") && (capturedOnBest != chess.Empty || blunderedIntoHangingOrTactic) {
		primary = "hangingPiece"
	}

	themeInfo := lichessThemeCatalog[ThemeHangingPiece]
	if tid, ok := LookupThemeID(primary); ok {
		themeInfo = lichessThemeCatalog[tid]
	}

	moveNum := (ply + 1) / 2
	whatWentWrong := buildWhatWentWrongText(b, curr.Move, playedSAN, bestMove, solSAN[0], bAfterPlayed, oppPunishMove, oppPunishScore, themeInfo.Name, moveNum)
	visualCues := extractVisualCuesForPosition(b, bestMove, curr.Move)
	stepCoach := buildStepByStepCoachText(primary, themeInfo.Name, solSAN[0], playedSAN)
	postSolve := buildPostSolveTakeaway(b.FEN(), bestMove.String(), evalDelta, primary, themeInfo.Name)

	return AdaptiveDrillItem{
		ID:                fmt.Sprintf("online-game-%d-ply-%d", gameNum, ply),
		SourceType:        "online_game_mistake",
		White:             g.Tag("White"),
		Black:             g.Tag("Black"),
		Event:             g.Tag("Event"),
		Date:              g.Tag("Date"),
		Ply:               ply,
		MoveNumber:        moveNum,
		FEN:               b.FEN(),
		Turn:              turnCodeByColor[us&1],
		SolutionUCI:       solUCI,
		SolutionSAN:       solSAN,
		PlayedBlunderUCI:  playedUCI,
		PlayedBlunderSAN:  playedSAN,
		PrimaryTheme:      primary,
		ThemeName:         themeInfo.Name,
		Themes:            themes,
		DifficultyTier:    tierID.String(),
		EstimatedRating:   rating,
		EvalDeltaCp:       evalDelta,
		DiagnosisHeadline: fmt.Sprintf("Your Game Mistake (%s): Move %d in %s vs %s", themeInfo.Name, moveNum, g.Tag("White"), g.Tag("Black")),
		WhatWentWrong:     whatWentWrong,
		VisualCues:        visualCues,
		StepByStepCoach:   stepCoach,
		PostSolveTakeaway: postSolve,
	}, true
}

func scoreCoachCandidate(b *chess.Board, m chess.Move, us, them chess.Color) int {
	baseMat := coachMaterialDiff(b, us)
	tactBonus := scoreTacticalCandidate(b, m, us, them)

	sim := b.Clone()
	sim.MakeMove(m)
	if sim.IsCheckmate() {
		return 10000
	}

	// If moved piece lands on a square defended by us, enemy king cannot capture it; check fork bonus
	if detectsForkFromSquare(sim, m.To, us) && isSquareDefendedBy(b, m.To, us) {
		if tactBonus < 280 {
			tactBonus += 280
		}
	}

	oppMoves := sim.GenerateLegalMoves()
	if len(oppMoves) == 0 {
		return (coachMaterialDiff(sim, us) - baseMat) + tactBonus
	}

	// Opponent chooses reply that minimizes our material + recapture outcome
	worstNetMat := 99999
	for _, om := range oppMoves {
		b2 := sim.Clone()
		b2.MakeMove(om)
		if b2.IsCheckmate() {
			return -10000
		}
		// Check our best immediate recapture on om.To if opponent captured
		bestAfterOpp := coachMaterialDiff(b2, us) - baseMat
		for _, um2 := range b2.GenerateLegalMoves() {
			if b2.Squares[um2.To] != chess.Empty {
				b3 := b2.Clone()
				b3.MakeMove(um2)
				sc3 := coachMaterialDiff(b3, us) - baseMat
				if sc3 > bestAfterOpp {
					bestAfterOpp = sc3
				}
			}
		}
		if bestAfterOpp < worstNetMat {
			worstNetMat = bestAfterOpp
		}
	}

	if tactBonus > worstNetMat && worstNetMat >= -50 {
		return tactBonus
	}
	return worstNetMat
}

func coachMaterialDiff(b *chess.Board, us chess.Color) int {
	score := 0
	for sq := 0; sq < 64; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty || p.Type() == chess.King {
			continue
		}
		v := pieceValues[p.Type()]
		if p.Color() == us {
			score += v
		} else {
			score -= v
		}
	}
	return score
}

func findStrongestOpponentPunishment(bAfterUser *chess.Board, oppColor, userColor chess.Color) (chess.Move, int) {
	oppMoves := bAfterUser.GenerateLegalMoves()
	var bestOppMove chess.Move
	bestOppScore := 0
	for _, om := range oppMoves {
		sc := scoreTacticalCandidate(bAfterUser, om, oppColor, userColor)
		if sc > bestOppScore {
			bestOppScore = sc
			bestOppMove = om
		}
	}
	return bestOppMove, bestOppScore
}

func buildWhatWentWrongText(b *chess.Board, playedMove chess.Move, playedSAN string, bestMove chess.Move, bestSAN string, bAfterPlayed *chess.Board, oppPunish chess.Move, oppPunishScore int, themeName string, moveNum int) string {
	capturedTarget := b.Squares[bestMove.To]
	if capturedTarget != chess.Empty && !isSquareDefendedBy(b, bestMove.To, b.Turn.Other()) {
		return fmt.Sprintf(
			"On move %d you played %s, missing that the opponent's %s on %s was completely undefended and could be captured immediately with %s.",
			moveNum, playedSAN, pieceTypeLabel(capturedTarget.Type()), bestMove.To.String(), bestSAN,
		)
	}
	if oppPunishScore >= 240 {
		oppSAN := bAfterPlayed.MoveToSAN(oppPunish)
		punishedTarget := bAfterPlayed.Squares[oppPunish.To]
		if punishedTarget != chess.Empty {
			return fmt.Sprintf(
				"On move %d you played %s, which allowed the opponent to play %s capturing your %s on %s. Playing %s avoids this tactical loss.",
				moveNum, playedSAN, oppSAN, pieceTypeLabel(punishedTarget.Type()), oppPunish.To.String(), bestSAN,
			)
		}
		return fmt.Sprintf(
			"On move %d you played %s, allowing the strong tactical refutation %s. Instead, %s seizes the initiative (%s).",
			moveNum, playedSAN, oppSAN, bestSAN, themeName,
		)
	}
	return fmt.Sprintf(
		"On move %d you played %s instead of executing the %s continuation %s.",
		moveNum, playedSAN, themeName, bestSAN,
	)
}

func extractVisualCuesForPosition(b *chess.Board, bestMove chess.Move, playedMove chess.Move) []string {
	var cues []string
	if safety, err := AnalyzeBoardSafety(b.FEN()); err == nil && safety != nil {
		for _, p := range safety.HangingPieces {
			if p.Explanation != "" {
				cues = append(cues, p.Explanation)
			} else {
				cues = append(cues, fmt.Sprintf("Hanging %s %s on %s (%d attacker(s), %d defender(s)).", p.Color, p.Piece, p.Square, len(p.Attackers), len(p.Defenders)))
			}
			if len(cues) >= 2 {
				break
			}
		}
		for _, p := range safety.LoosePieces {
			if len(cues) >= 3 {
				break
			}
			if p.Piece != "P" && p.Piece != "p" && p.Piece != "K" && p.Piece != "k" {
				cues = append(cues, fmt.Sprintf("Loose (undefended) %s %s on %s — potential tactical target.", p.Color, p.Piece, p.Square))
			}
		}
	}

	sim := b.Clone()
	sim.MakeMove(bestMove)
	if sim.InCheck(sim.Turn) {
		cues = append(cues, fmt.Sprintf("Move %s delivers direct check to the enemy king.", b.MoveToSAN(bestMove)))
	}
	if detectsForkFromSquare(sim, bestMove.To, b.Turn) {
		cues = append(cues, fmt.Sprintf("Square %s acts as a double-attack outpost hitting multiple enemy targets.", bestMove.To.String()))
	}

	if len(cues) == 0 {
		cues = append(cues, fmt.Sprintf("Compare forcing candidate %s against %s before committing your move.", b.MoveToSAN(bestMove), b.MoveToSAN(playedMove)))
	}
	return cues
}

func buildStepByStepCoachText(themeKey string, themeName string, bestSAN string, playedSAN string) string {
	habit := coachingHabitForTheme(themeKey)
	if playedSAN != "" {
		return fmt.Sprintf("%s Instead of %s, calculate %s first.", habit, playedSAN, bestSAN)
	}
	return fmt.Sprintf("%s Look for %s to exploit the %s motif.", habit, bestSAN, themeName)
}

func buildPostSolveTakeaway(fen string, bestUCI string, evalDeltaCp int, themeKey string, themeName string) string {
	if expl, err := explainer.ExplainMove(fen, bestUCI, evalDeltaCp, bestUCI); err == nil && expl != nil && expl.DetailedText != "" {
		return fmt.Sprintf("[%s] %s %s", themeName, expl.DetailedText, coachingHabitForTheme(themeKey))
	}
	return fmt.Sprintf("[%s] %s", themeName, coachingHabitForTheme(themeKey))
}

func coachingHabitForTheme(themeKey string) string {
	switch themeKey {
	case "hangingPiece":
		return "Board Vision Protocol (Hanging Pieces): Before every move, scan all undefended enemy pieces you can capture, and verify your own destination square and vacated defense leave zero friendly pieces undefended."
	case "fork":
		return "Double-Attack Scan (Forks): Look for loose pieces or the enemy King and Queen standing on forkable knight squares or shared diagonals/ranks."
	case "pin":
		return "Alignment Scan (Pins): Identify enemy pieces lined up in front of their King or Queen along files, ranks, and diagonals that lack pawn protection."
	case "skewer":
		return "Line-Through Scan (Skewers): Check if a high-value enemy piece (King, Queen, Rook) can be attacked along a line with another valuable target directly behind it."
	case "discoveredAttack", "doubleCheck":
		return "Battery Scan (Discovered Attacks): When one of your pieces sits in front of your Rook, Bishop, or Queen, look for the most destructive forcing move by the front piece."
	case "backRankMate", "mateIn1", "mateIn2", "mateIn3":
		return "Mating Net Check: Verify whether the enemy King has any flight squares off the back rank and calculate all checks to the end."
	case "deflection", "attraction":
		return "Overloaded Defender Scan: Identify which single enemy piece is responsible for guarding a key square or piece, then force it away."
	default:
		return "Forcing Moves Checklist: Always calculate Checks, Captures, and Threats (CCT) for both sides before playing a routine move."
	}
}

func convertCyclicalToAdaptiveDrill(cp CyclicalPuzzle, targetThemeKey string, sourceType string) AdaptiveDrillItem {
	primary := targetThemeKey
	if primary == "" {
		primary = cp.PrimaryTheme
	}
	info := lichessThemeCatalog[ThemeHangingPiece]
	if tid, ok := LookupThemeID(primary); ok {
		info = lichessThemeCatalog[tid]
	}

	b, err := chess.ParseFEN(cp.FEN)
	var cues []string
	bestSAN := ""
	if len(cp.SolutionSAN) > 0 {
		bestSAN = cp.SolutionSAN[0]
	}
	if err == nil && len(cp.SolutionUCI) > 0 {
		if mv, mErr := parseMoveFlexible(b, cp.SolutionUCI[0]); mErr == nil {
			cues = extractVisualCuesForPosition(b, mv, mv)
		}
	}
	if len(cues) == 0 {
		cues = []string{fmt.Sprintf("Target motif: %s (%s).", info.Name, info.Description)}
	}

	bestUCI := ""
	if len(cp.SolutionUCI) > 0 {
		bestUCI = cp.SolutionUCI[0]
	}

	return AdaptiveDrillItem{
		ID:                "coach-" + cp.ID,
		SourceType:        sourceType,
		White:             cp.White,
		Black:             cp.Black,
		Event:             cp.Event,
		Date:              cp.Date,
		Ply:               cp.Ply,
		MoveNumber:        (cp.Ply + 1) / 2,
		FEN:               cp.FEN,
		Turn:              cp.Turn,
		SolutionUCI:       cp.SolutionUCI,
		SolutionSAN:       cp.SolutionSAN,
		PlayedBlunderUCI:  cp.PlayedBlunderUCI,
		PrimaryTheme:      primary,
		ThemeName:         info.Name,
		Themes:            cp.Themes,
		DifficultyTier:    cp.DifficultyTier,
		EstimatedRating:   cp.EstimatedRating,
		EvalDeltaCp:       cp.EvalDeltaCp,
		DiagnosisHeadline: fmt.Sprintf("Targeted Weakness Drill: %s (%s Tier)", info.Name, cp.DifficultyTier),
		WhatWentWrong:     cp.Explanation,
		VisualCues:        cues,
		StepByStepCoach:   buildStepByStepCoachText(primary, info.Name, bestSAN, ""),
		PostSolveTakeaway: buildPostSolveTakeaway(cp.FEN, bestUCI, cp.EvalDeltaCp, primary, info.Name),
	}
}

func pieceTypeLabel(pt chess.PieceType) string {
	switch pt {
	case chess.Pawn:
		return "pawn"
	case chess.Knight:
		return "knight"
	case chess.Bishop:
		return "bishop"
	case chess.Rook:
		return "rook"
	case chess.Queen:
		return "queen"
	case chess.King:
		return "king"
	default:
		return "piece"
	}
}

func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}
