package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"newscid/pkg/blundervault"
	"newscid/pkg/chess"
	"newscid/pkg/chess960"
	"newscid/pkg/database/scid"
	"newscid/pkg/eco"
	"newscid/pkg/epd"
	"newscid/pkg/explainer"
	"newscid/pkg/fileformat"
	"newscid/pkg/gameexport"
	"newscid/pkg/imbalance"
	"newscid/pkg/online"
	"newscid/pkg/persona"
	"newscid/pkg/playerbot"
	"newscid/pkg/polyglot"
	"newscid/pkg/scout"
	"newscid/pkg/stats"
	"newscid/pkg/tablebase"
	"newscid/pkg/tournament"
	"newscid/pkg/training"
	"newscid/pkg/workstation"
)

// Persona Management & Multi-Alias Aggregation
// GetPersonas returns all configured personas sorted by display name, ensuring each has a dedicated .si4 database.
func (a *App) GetPersonas() ([]*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	personas := a.personaMgr.GetPersonas()
	for _, p := range personas {
		_, _ = a.personaMgr.EnsurePersonaDatabase(p)
	}
	return personas, nil
}

// GetPersona retrieves a persona by its unique ID.
func (a *App) GetPersona(id string) (*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	return a.personaMgr.GetPersona(id)
}

// SavePersona creates or updates a unified persona.
func (a *App) SavePersona(p *persona.Persona) (*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	saved, err := a.personaMgr.SavePersona(p)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	delete(a.personaBotCache, saved.ID)
	a.mu.Unlock()
	_, _ = a.personaMgr.EnsurePersonaDatabase(saved)
	return saved, nil
}

// DeletePersona removes a persona by ID.
func (a *App) DeletePersona(id string) error {
	if a.personaMgr == nil {
		return fmt.Errorf("persona manager not initialized")
	}
	a.mu.Lock()
	delete(a.personaBotCache, id)
	a.mu.Unlock()
	return a.personaMgr.DeletePersona(id)
}

// AddPersonaIdentifier adds a new database designator to the persona.
func (a *App) AddPersonaIdentifier(personaID string, identifier string) (*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	return a.personaMgr.AddIdentifier(personaID, identifier)
}

// RemovePersonaIdentifier removes a database designator from the persona.
func (a *App) RemovePersonaIdentifier(personaID string, identifier string) (*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	return a.personaMgr.RemoveIdentifier(personaID, identifier)
}

// SuggestPersonaIdentifiers queries the active database for player names matching query.
func (a *App) SuggestPersonaIdentifiers(query string, existing []string) ([]string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return []string{}, nil
	}
	return persona.SuggestIdentifiers(db, query, existing), nil
}

// GetPersonaStatistics calculates aggregated statistics across all identifiers for a persona from active DB or dedicated base.
func (a *App) GetPersonaStatistics(personaID string) (*persona.PersonaStatistics, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	// If no database is open, try opening persona's dedicated database
	if (db == nil || !db.IsOpen) && p.DatabasePath != "" {
		dedicatedDB := &scid.Database{}
		if err := dedicatedDB.Open(p.DatabasePath); err == nil {
			defer dedicatedDB.Close()
			return persona.CalculatePersonaStatistics(dedicatedDB, p), nil
		}
	}

	if db == nil || !db.IsOpen {
		return &persona.PersonaStatistics{
			PersonaID:   p.ID,
			DisplayName: p.DisplayName,
		}, nil
	}

	stats := persona.CalculatePersonaStatistics(db, p)
	// If the active database has no games for this persona, check dedicated database
	if stats.TotalGames == 0 && p.DatabasePath != "" {
		dedicatedDB := &scid.Database{}
		if err := dedicatedDB.Open(p.DatabasePath); err == nil {
			defer dedicatedDB.Close()
			dedStats := persona.CalculatePersonaStatistics(dedicatedDB, p)
			if dedStats.TotalGames > 0 {
				return dedStats, nil
			}
		}
	}

	return stats, nil
}

// FetchPersonaOnlineGames fetches games for a persona from Chess.com and/or Lichess directly into their dedicated database.
func (a *App) FetchPersonaOnlineGames(personaID string, maxGames int) (*online.SyncResult, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	if a.syncMgr == nil {
		return nil, fmt.Errorf("sync manager not initialized")
	}

	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}

	chessComUser := strings.TrimSpace(p.ChessComUsername)
	lichessUser := strings.TrimSpace(p.LichessUsername)

	if chessComUser == "" && lichessUser == "" {
		return nil, fmt.Errorf("no Chess.com or Lichess username specified for persona %q", p.DisplayName)
	}

	basePath, err := a.personaMgr.EnsurePersonaDatabase(p)
	if err != nil {
		return nil, fmt.Errorf("ensuring persona database: %w", err)
	}

	opts := online.ImportPlayerOptions{
		PlayerName:       p.DisplayName,
		ChessComUsername: chessComUser,
		LichessUsername:  lichessUser,
		TargetMode:       online.TargetDedicatedBase,
		DedicatedPath:    basePath,
		AutoDeduplicate:  true,
		MaxGames:         maxGames,
		ProgressFn: func(stage string, current, total int, message string) {
			a.emitEvent("online:fetch-progress", map[string]interface{}{
				"personaId": personaID,
				"stage":     stage,
				"current":   current,
				"total":     total,
				"message":   message,
			})
		},
	}

	a.mu.RLock()
	activeDB := a.db
	a.mu.RUnlock()

	res, err := a.syncMgr.ImportPlayerGames(opts, activeDB)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	delete(a.personaBotCache, personaID)
	a.mu.Unlock()

	return res, nil
}

// AggregatePersonaGames extracts all matching games from the active database into the persona's dedicated database.
func (a *App) AggregatePersonaGames(personaID string) (*persona.AggregateResult, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database open to aggregate games from")
	}

	res, err := a.personaMgr.AggregateGames(db, personaID)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	delete(a.personaBotCache, personaID)
	a.mu.Unlock()

	return res, nil
}

// OpenPersonaDatabase opens the persona's dedicated database as the active database in NewScid.
func (a *App) OpenPersonaDatabase(personaID string) (*DatabaseInfo, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}
	basePath, err := a.personaMgr.EnsurePersonaDatabase(p)
	if err != nil {
		return nil, err
	}
	return a.OpenDatabase(basePath)
}

// GetPersonaPersonality builds a playerbot personality using all identifiers of the persona.
func (a *App) GetPersonaPersonality(personaID string) (*playerbot.PlayerPersonality, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	// If active database is open and contains games for this persona, use it
	if db != nil && db.IsOpen {
		personality, err := playerbot.BuildPersonalityFromIdentifiers(db, p.DisplayName, p.Identifiers)
		if err == nil && personality.TotalGames > 0 {
			if p.EstimatedElo > 0 {
				personality.EstimatedElo = p.EstimatedElo
			}
			return personality, nil
		}
	}

	// Otherwise, check if dedicated persona database has games
	if p.DatabasePath != "" {
		dedicatedDB := &scid.Database{}
		if err := dedicatedDB.Open(p.DatabasePath); err == nil {
			defer dedicatedDB.Close()
			personality, err := playerbot.BuildPersonalityFromIdentifiers(dedicatedDB, p.DisplayName, p.Identifiers)
			if err == nil && personality.TotalGames > 0 {
				if p.EstimatedElo > 0 {
					personality.EstimatedElo = p.EstimatedElo
				}
				return personality, nil
			}
		}
	}

	if db != nil && db.IsOpen {
		return playerbot.BuildPersonalityFromIdentifiers(db, p.DisplayName, p.Identifiers)
	}

	return nil, fmt.Errorf("no games found for persona %q in open or dedicated databases", p.DisplayName)
}

func makeBoardState(b *chess.Board) *BoardState {
	legal := b.GenerateLegalMoves()
	legalUCI := make([]string, len(legal))
	for i, m := range legal {
		legalUCI[i] = m.String()
	}

	turnStr := "w"
	if b.Turn == chess.Black {
		turnStr = "b"
	}

	return &BoardState{
		FEN:         b.FEN(),
		Turn:        turnStr,
		InCheck:     b.InCheck(b.Turn),
		IsCheckmate: b.IsCheckmate(),
		IsStalemate: b.IsStalemate(),
		LegalMoves:  legalUCI,
		FullMove:    b.FullMove,
		HalfMove:    b.HalfMove,
	}
}

// GetTournamentEvents returns discovered tournament events in the currently open database.
func (a *App) GetTournamentEvents() ([]*tournament.EventSummary, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database is currently open")
	}

	return tournament.DiscoverEvents(db)
}

// GetTournamentCrosstable calculates standings, matrix, and tiebreak metrics for a given event in the open database.
func (a *App) GetTournamentCrosstable(eventName string, site string, date string) (*tournament.TournamentCrosstable, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database is currently open")
	}

	return tournament.CalculateCrosstable(db, eventName, site, date)
}

// ExportCrosstableText returns the ASCII crosstable text for an event in the open database.
func (a *App) ExportCrosstableText(eventName string, site string, date string) (string, error) {
	ct, err := a.GetTournamentCrosstable(eventName, site, date)
	if err != nil {
		return "", err
	}
	return ct.CrossTableText, nil
}

// GetPolyglotBooks returns discovered Polyglot opening books (.bin) on the system.
func (a *App) GetPolyglotBooks() []string {
	return polyglot.DiscoverBooks()
}

// QueryPolyglotBook returns moves from a Polyglot book for the specified position FEN.
func (a *App) QueryPolyglotBook(fen string, bookPath string) ([]polyglot.BookMove, error) {
	if strings.TrimSpace(bookPath) == "" {
		return []polyglot.BookMove{}, nil
	}

	a.mu.Lock()
	if a.polyglotBooks == nil {
		a.polyglotBooks = make(map[string]*polyglot.Book)
	}
	book, ok := a.polyglotBooks[bookPath]
	var err error
	if !ok {
		book, err = polyglot.OpenBook(bookPath)
		if err != nil {
			a.mu.Unlock()
			return nil, err
		}
		a.polyglotBooks[bookPath] = book
	}
	a.mu.Unlock()

	var b *chess.Board
	if strings.TrimSpace(fen) == "" || fen == chess.StartingFEN {
		b = chess.NewBoard()
	} else {
		parsed, err := chess.ParseFEN(fen)
		if err != nil {
			return nil, fmt.Errorf("parsing FEN %q: %w", fen, err)
		}
		b = parsed
	}

	return book.FindMoves(b)
}

// GetPolyglotBookMove selects a book move according to weights or chooses the best move.
func (a *App) GetPolyglotBookMove(fen string, bookPath string, pickBest bool) (*polyglot.BookMove, error) {
	moves, err := a.QueryPolyglotBook(fen, bookPath)
	if err != nil {
		return nil, err
	}
	if len(moves) == 0 {
		return nil, nil
	}
	if pickBest || len(moves) == 1 {
		return &moves[0], nil
	}

	totalWeight := 0
	for _, m := range moves {
		totalWeight += m.Weight
	}
	if totalWeight <= 0 {
		return &moves[0], nil
	}

	r := rand.Intn(totalWeight)
	running := 0
	for i := range moves {
		running += moves[i].Weight
		if r < running {
			return &moves[i], nil
		}
	}
	return &moves[0], nil
}

// GetECOSections returns the 5 primary ECO classifications with game/line counts.
func (a *App) GetECOSections() []eco.SectionSummary {
	a.mu.RLock()
	classifier := a.eco
	a.mu.RUnlock()

	if classifier == nil {
		return nil
	}
	return classifier.GetSections()
}

// SearchECO queries ECO openings by section (A-E or ALL) and optional keywords.
func (a *App) SearchECO(section string, query string, limit int) []*eco.Entry {
	a.mu.RLock()
	classifier := a.eco
	a.mu.RUnlock()

	if classifier == nil {
		return nil
	}
	return classifier.Search(section, query, limit)
}

// ClassifyMoves identifies the ECO code and opening name for a sequence of moves.
func (a *App) ClassifyMoves(sans []string) *eco.Entry {
	a.mu.RLock()
	classifier := a.eco
	a.mu.RUnlock()

	if classifier == nil {
		return &eco.Entry{Code: "A00", Name: "Unknown"}
	}
	return classifier.Classify(sans)
}

// ProbeTablebase queries Syzygy endgame tablebases for a position (up to 7 pieces).
func (a *App) ProbeTablebase(fen string) (*tablebase.TablebaseResult, error) {
	a.mu.RLock()
	client := a.tablebaseClient
	a.mu.RUnlock()
	if client == nil {
		client = tablebase.NewClient()
	}
	return client.Probe(fen)
}

// StartGuessTheMove initializes a Solitaire Chess session for the given game and side.
func (a *App) StartGuessTheMove(gameID int, userSide string) (*training.SolitaireSession, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var g *chess.Game
	if a.db != nil && a.db.IsOpen && gameID > 0 {
		game, err := a.db.GetGame(gameID)
		if err != nil {
			return nil, fmt.Errorf("loading game %d: %w", gameID, err)
		}
		g = game
	} else if a.activeGame != nil {
		g = a.activeGame
	} else {
		return nil, fmt.Errorf("no game loaded for Guess-the-Move training")
	}

	session, err := training.NewSolitaireSession(g, userSide)
	if err != nil {
		return nil, err
	}
	a.solitaireSession = session
	return session, nil
}

// GetGuessTheMoveState returns the current Solitaire Chess session if active.
func (a *App) GetGuessTheMoveState() (*training.SolitaireSession, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.solitaireSession == nil {
		return nil, fmt.Errorf("no active guess-the-move session")
	}
	return a.solitaireSession, nil
}

// SubmitGuessTheMove submits the user's move guess and returns evaluation feedback.
func (a *App) SubmitGuessTheMove(guess string) (*training.GuessFeedback, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.solitaireSession == nil {
		return nil, fmt.Errorf("no active guess-the-move session")
	}
	feedback := a.solitaireSession.SubmitGuess(guess)
	return feedback, nil
}

// GetGuessTheMoveHint requests a progressive hint for the current turn.
func (a *App) GetGuessTheMoveHint() (string, int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.solitaireSession == nil {
		return "", 0, fmt.Errorf("no active guess-the-move session")
	}
	hintText, level := a.solitaireSession.GetHint()
	return hintText, level, nil
}

// SkipGuessTheMove reveals the actual move and advances to the next turn.
func (a *App) SkipGuessTheMove() (*training.GuessFeedback, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.solitaireSession == nil {
		return nil, fmt.Errorf("no active guess-the-move session")
	}
	feedback := a.solitaireSession.RevealAndSkip()
	return feedback, nil
}

// GetRatingEvolution calculates chronological rating progression for a player or persona.
func (a *App) GetRatingEvolution(playerOrPersona string) (*stats.RatingTrackerResult, error) {
	a.mu.RLock()
	db := a.db
	pMgr := a.personaMgr
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	aliases := []string{playerOrPersona}
	if pMgr != nil {
		if p, err := pMgr.GetPersona(playerOrPersona); err == nil && p != nil {
			aliases = append([]string{p.DisplayName}, p.Identifiers...)
			if p.ChessComUsername != "" {
				aliases = append(aliases, p.ChessComUsername)
			}
			if p.LichessUsername != "" {
				aliases = append(aliases, p.LichessUsername)
			}
		}
	}

	return stats.CalculateRatingTracker(db, aliases)
}

// GetDatabaseDemographics returns demographic distributions for the active database.
func (a *App) GetDatabaseDemographics() (*stats.DatabaseDemographics, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return stats.CalculateDatabaseDemographics(db)
}

// StandardizePlayerNames applies spelling standardizations across the active database.
func (a *App) StandardizePlayerNames() (*scid.SpellCheckSummary, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.checkActiveDBEditableLocked(); err != nil {
		return nil, err
	}

	return a.db.StandardizePlayerNames()
}

// ExportCurrentGameHTML exports the currently loaded game as standalone HTML.
func (a *App) ExportCurrentGameHTML(opts gameexport.HTMLExportOptions) (string, error) {
	a.mu.RLock()
	g := a.activeGame
	a.mu.RUnlock()

	if g == nil {
		return "", fmt.Errorf("no active game loaded")
	}

	return gameexport.ExportGameHTML(g, opts)
}

// ExportGameHTML exports a specific game by index as standalone HTML.
func (a *App) ExportGameHTML(gameNum int, opts gameexport.HTMLExportOptions) (string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return "", fmt.Errorf("no database open")
	}

	g, err := db.GetGame(gameNum)
	if err != nil {
		return "", fmt.Errorf("failed to load game %d: %w", gameNum, err)
	}

	return gameexport.ExportGameHTML(g, opts)
}

// ExportCurrentGameLaTeX exports the currently loaded game as a LaTeX document.
func (a *App) ExportCurrentGameLaTeX(opts gameexport.LaTeXExportOptions) (string, error) {
	a.mu.RLock()
	g := a.activeGame
	a.mu.RUnlock()

	if g == nil {
		return "", fmt.Errorf("no active game loaded")
	}

	return gameexport.ExportGameLaTeX(g, opts)
}

// ExportGameLaTeX exports a specific game by index as a LaTeX document.
func (a *App) ExportGameLaTeX(gameNum int, opts gameexport.LaTeXExportOptions) (string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return "", fmt.Errorf("no database open")
	}

	g, err := db.GetGame(gameNum)
	if err != nil {
		return "", fmt.Errorf("failed to load game %d: %w", gameNum, err)
	}

	return gameexport.ExportGameLaTeX(g, opts)
}

// SaveExportFile writes exported content (HTML, LaTeX, PGN, etc.) to a file on disk.
func (a *App) SaveExportFile(filePath, content string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}
// Chess960 (Fischer Random Chess)
// GetChess960Position generates the Chess960 position for a given index (0-959).
func (a *App) GetChess960Position(id int) (*chess960.Position, error) {
	return chess960.GeneratePosition(id)
}

// GenerateRandomChess960 returns a randomly rolled Chess960 starting position.
func (a *App) GenerateRandomChess960() (*chess960.Position, error) {
	pos := chess960.RandomPosition()
	return pos, nil
}
// EPD (Extended Position Description)
// ParseEPD parses an EPD string into structured fields with opcodes.
func (a *App) ParseEPD(epdString string) (*epd.EPDRecord, error) {
	return epd.Parse(epdString)
}

// ExportEPD formats a position and opcodes into standard EPD format.
func (a *App) ExportEPD(fen string, opcodes map[string]string) (string, error) {
	return epd.Format(fen, opcodes), nil
}
// Positional & Material Imbalance
// GetPositionalImbalance analyzes the material, pawn structure, king safety, and territory control.
func (a *App) GetPositionalImbalance(fen string) (*imbalance.Report, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	return imbalance.AnalyzeBoard(b), nil
}
// Natural Language Move Explainer
// ExplainMove generates plain-English coaching explanations of a move and its motifs.
func (a *App) ExplainMove(beforeFEN, moveUCI string, scoreCp int, bestMove string) (*explainer.Explanation, error) {
	return explainer.ExplainMove(beforeFEN, moveUCI, scoreCp, bestMove)
}
// Personal Blunder Spaced-Repetition Vault
// GetDueBlunderPuzzles returns blunder flashcards due for spaced-repetition review.
func (a *App) GetDueBlunderPuzzles() ([]*blundervault.Puzzle, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.blunderVault == nil {
		return []*blundervault.Puzzle{}, nil
	}
	return a.blunderVault.GetDuePuzzles(), nil
}

// GetAllBlunderPuzzles returns all blunder flashcards in the vault.
func (a *App) GetAllBlunderPuzzles() ([]*blundervault.Puzzle, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.blunderVault == nil {
		return []*blundervault.Puzzle{}, nil
	}
	return a.blunderVault.GetAllPuzzles(), nil
}

// RecordBlunderReview records a review grade (0-5) and recalculates SM-2 interval.
func (a *App) RecordBlunderReview(id string, quality int) (*blundervault.Puzzle, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.blunderVault == nil {
		return nil, fmt.Errorf("blunder vault not initialized")
	}
	return a.blunderVault.RecordReview(id, quality)
}

// AddBlunderToVault adds a new blunder puzzle into the vault.
func (a *App) AddBlunderToVault(p *blundervault.Puzzle) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.blunderVault == nil {
		return fmt.Errorf("blunder vault not initialized")
	}
	return a.blunderVault.AddPuzzle(p)
}
// Opponent Scouting & Prep Matrix
// GenerateOpponentScoutingReport builds an in-depth dossier on a player from the current database.
func (a *App) GenerateOpponentScoutingReport(playerName string) (*scout.Report, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database is currently open")
	}

	games, _, err := a.db.GetGameSummaries(0, 100000, scid.SearchCriteria{})
	if err != nil {
		return nil, err
	}
	return scout.GenerateReport(playerName, games), nil
}
// Opponent Opening Exploiter & Custom Book Generator
// EvaluateOpponentWeaknesses performs opening tree evaluation for any player in the active database.
func (a *App) EvaluateOpponentWeaknesses(playerName string, targetColor string, maxDepth int, minGames int) (*persona.AntiPersonaReport, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database is currently open")
	}

	cleanName := strings.TrimSpace(playerName)
	if cleanName == "" {
		return nil, fmt.Errorf("player name cannot be empty")
	}

	cfg := persona.AntiPersonaConfig{
		DisplayName:          cleanName,
		Identifiers:          []string{cleanName},
		TargetColor:          targetColor,
		MaxDepth:             maxDepth,
		MinGames:             minGames,
		MinVulnerabilityRate: 45.0,
	}

	return persona.EvaluatePersonaDatabase(db, cfg)
}

// EvaluatePersonaWeaknesses performs deep opening tree evaluation for a persona across active and dedicated databases.
func (a *App) EvaluatePersonaWeaknesses(personaID string, targetColor string, maxDepth int, minGames int) (*persona.AntiPersonaReport, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}

	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}

	cfg := persona.AntiPersonaConfig{
		PersonaID:            p.ID,
		DisplayName:          p.DisplayName,
		Identifiers:          p.Identifiers,
		TargetColor:          targetColor,
		MaxDepth:             maxDepth,
		MinGames:             minGames,
		MinVulnerabilityRate: 45.0,
	}

	a.mu.RLock()
	activeDB := a.db
	a.mu.RUnlock()

	var allGames []*chess.Game
	seenSigs := make(map[string]bool)

	addGamesFromDB := func(db *scid.Database) {
		if db == nil || !db.IsOpen {
			return
		}
		nameLookup := make(map[string]bool)
		for _, id := range p.Identifiers {
			t := strings.ToLower(strings.TrimSpace(id))
			if t != "" {
				nameLookup[t] = true
			}
		}
		if p.DisplayName != "" {
			nameLookup[strings.ToLower(strings.TrimSpace(p.DisplayName))] = true
		}

		numGames := db.NumGames()
		for i := 0; i < numGames; i++ {
			db.RLock()
			if i >= len(db.Entries) {
				db.RUnlock()
				break
			}
			entry := db.Entries[i]
			if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
				db.RUnlock()
				continue
			}

			w := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.WhiteID))
			b := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.BlackID))
			entryDate := entry.Date
			entryResult := entry.Result
			entryHalfMoves := entry.NumHalfMoves
			db.RUnlock()

			isW := matchPlayerLookup(w, nameLookup)
			isB := matchPlayerLookup(b, nameLookup)
			if !isW && !isB {
				continue
			}

			sig := fmt.Sprintf("%s|%s|%s|%s|%d", entryDate, w, b, entryResult, entryHalfMoves)
			if seenSigs[sig] {
				continue
			}
			seenSigs[sig] = true

			g, err := db.GetGame(i + 1)
			if err == nil && g != nil {
				allGames = append(allGames, g)
			}
		}
	}

	if activeDB != nil && activeDB.IsOpen {
		addGamesFromDB(activeDB)
	}

	if p.DatabasePath != "" {
		dedicatedDB := &scid.Database{}
		if err := dedicatedDB.Open(p.DatabasePath); err == nil {
			defer dedicatedDB.Close()
			addGamesFromDB(dedicatedDB)
		}
	}

	return persona.EvaluateGames(allGames, cfg)
}

func matchPlayerLookup(name string, lookup map[string]bool) bool {
	if lookup[name] {
		return true
	}
	for id := range lookup {
		if strings.Contains(name, id) || strings.Contains(id, name) {
			return true
		}
	}
	return false
}

// ExportAntiPersonaPolyglotBook writes the generated Polyglot opening book to disk.
func (a *App) ExportAntiPersonaPolyglotBook(report *persona.AntiPersonaReport, filePath string) (string, error) {
	if report == nil {
		return "", fmt.Errorf("nil report")
	}

	target := strings.TrimSpace(filePath)
	if target == "" {
		dataRoot, _ := workstation.DefaultWorkstationDirs()
		booksDir := filepath.Join(dataRoot, "books", "opening")
		_ = os.MkdirAll(booksDir, 0755)

		slug := strings.ReplaceAll(report.DisplayName, " ", "_")
		slug = strings.ReplaceAll(slug, ",", "")
		target = filepath.Join(booksDir, fmt.Sprintf("Anti_%s%s", slug, fileformat.ExtPolyglotBook))
	}

	if err := persona.SaveAntiPersonaBook(report, target); err != nil {
		return "", err
	}

	if report.PersonaID != "" && a.personaMgr != nil {
		if p, pErr := a.personaMgr.GetPersona(report.PersonaID); pErr == nil && p != nil {
			p.AntiBookPath = target
			_, _ = a.personaMgr.SavePersona(p)
		}
	}

	a.mu.Lock()
	if a.polyglotBooks != nil {
		delete(a.polyglotBooks, target)
	}
	a.mu.Unlock()

	return target, nil
}

// ExportAntiPersonaPGN writes the generated PGN repertoire to disk.
func (a *App) ExportAntiPersonaPGN(report *persona.AntiPersonaReport, filePath string) (string, error) {
	if report == nil {
		return "", fmt.Errorf("nil report")
	}

	target := strings.TrimSpace(filePath)
	if target == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = os.TempDir()
		}
		repDir := filepath.Join(homeDir, ".local", "share", fileformat.AppDirName, fileformat.DirRepertoires)
		_ = os.MkdirAll(repDir, 0755)

		slug := strings.ReplaceAll(report.DisplayName, " ", "_")
		slug = strings.ReplaceAll(slug, ",", "")
		target = filepath.Join(repDir, fmt.Sprintf("Anti_%s%s", slug, fileformat.ExtPGN))
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(target, []byte(report.PGNRepertoire), 0644); err != nil {
		return "", err
	}

	return target, nil
}

// SaveBookFileDialog prompts user to choose a destination for opening book (.bin).
func (a *App) SaveBookFileDialog(defaultFilename string) (string, error) {
	return a.saveFileDialog("Save Polyglot Opening Book", defaultFilename,
		runtime.FileFilter{DisplayName: "Polyglot Opening Book (" + fileformat.GlobPolyglotBook + ")", Pattern: fileformat.GlobPolyglotBook},
	)
}

// SavePGNFileDialog prompts user to choose a destination for PGN repertoire.
func (a *App) SavePGNFileDialog(defaultFilename string) (string, error) {
	return a.saveFileDialog("Save Repertoire PGN", defaultFilename,
		runtime.FileFilter{DisplayName: "PGN Chess Repertoire (" + fileformat.GlobPGN + ")", Pattern: fileformat.GlobPGN},
	)
}
