package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"newscid/pkg/blundervault"
	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/fairplay"
	"newscid/pkg/fileformat"
	"newscid/pkg/online"
	"newscid/pkg/stats"
	"newscid/pkg/support"
	"newscid/pkg/training"
)

// Piece Safety Scanner & Drill Module
// AnalyzeBoardSafety calculates square safety, attackers, defenders, and hanging pieces for the given FEN.
func (a *App) AnalyzeBoardSafety(fen string) (*training.BoardSafetyReport, error) {
	if strings.TrimSpace(fen) == "" {
		a.mu.RLock()
		if a.activeGame != nil {
			b := chess.NewBoard()
			if a.activeGame.StartFEN != "" {
				if parsed, err := chess.ParseFEN(a.activeGame.StartFEN); err == nil {
					b = parsed
				}
			}
			for curr := a.activeGame.RootMove; curr != nil; curr = curr.Next {
				b.MakeMove(curr.Move)
			}
			fen = b.FEN()
		}
		a.mu.RUnlock()
	}
	if fen == "" {
		fen = chess.StartingFEN
	}
	return training.AnalyzeBoardSafety(fen)
}

// GetSafetyDrillBatch retrieves a set of curated safety challenges.
func (a *App) GetSafetyDrillBatch(count int, difficulty string) ([]training.SafetyDrill, error) {
	drills := training.GetSafetyDrillBatch(count, difficulty)
	return drills, nil
}

// GenerateSafetyDrillsFromDatabase mines blunder positions from the active database.
func (a *App) GenerateSafetyDrillsFromDatabase(count int) ([]training.SafetyDrill, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return training.GetSafetyDrillBatch(count, "all"), nil
	}
	return training.ExtractSafetyDrillsFromDatabase(db, count), nil
}

// SubmitSafetyDrillAnswer evaluates the user's selected squares or move against a safety drill.
func (a *App) SubmitSafetyDrillAnswer(drill training.SafetyDrill, selectedSquares []string, moveUCI string) (*training.SafetyDrillFeedback, error) {
	feedback := training.EvaluateSafetySubmission(&drill, selectedSquares, moveUCI)
	return feedback, nil
}

// GetIntuitionDrillBatch returns a batch of curated rapid evaluation challenges.
func (a *App) GetIntuitionDrillBatch(count int) ([]training.IntuitionChallenge, error) {
	return training.GetIntuitionDrillBatch(count), nil
}

// GenerateIntuitionDrillsFromDatabase extracts positions from the active database for speed assessment.
func (a *App) GenerateIntuitionDrillsFromDatabase(count int) ([]training.IntuitionChallenge, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return training.GetIntuitionDrillBatch(count), nil
	}
	return training.GenerateIntuitionDrillsFromDatabase(db, count)
}

// SubmitIntuitionAnswer evaluates the user's rapid judgment and returns score and speed bonus.
func (a *App) SubmitIntuitionAnswer(challengeID string, userChoice string, elapsedMs int) (*training.IntuitionResult, error) {
	return training.SubmitIntuitionAnswer(challengeID, userChoice, elapsedMs)
}

// AnalyzeGameTimeManagement analyzes clock progression, tanks, panics, and scrambles for a game.
func (a *App) AnalyzeGameTimeManagement(gameID int) (*stats.GameTimeManagementReport, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var targetGame *chess.Game
	if gameID <= 0 || a.db == nil || !a.db.IsOpen {
		targetGame = a.activeGame
	} else {
		g, err := a.db.GetGame(gameID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch game %d: %w", gameID, err)
		}
		targetGame = g
	}

	if targetGame == nil && a.db != nil && a.db.IsOpen && a.db.NumGames() > 0 {
		targetGame, _ = a.db.GetGame(1)
	}
	if targetGame == nil {
		return nil, fmt.Errorf("no game loaded or found to analyze")
	}

	return stats.AnalyzeGameTimeManagement(targetGame)
}

// AnalyzePlayerTimeManagement generates an aggregate time management and tanking profile for a player.
func (a *App) AnalyzePlayerTimeManagement(playerName string, maxGames int) (*stats.PlayerTimeManagementDossier, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	return stats.AnalyzePlayerTimeManagement(db, playerName, maxGames)
}

// GetProphylaxisDrillBatch returns curated quiet move / prophylaxis drill challenges.
func (a *App) GetProphylaxisDrillBatch(count int, category string) ([]training.ProphylaxisDrill, error) {
	return training.GetProphylaxisDrillBatch(count, category), nil
}

// GenerateProphylaxisDrillsFromDatabase mines quiet prophylactic moves from the active database.
func (a *App) GenerateProphylaxisDrillsFromDatabase(count int) ([]training.ProphylaxisDrill, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return training.GetProphylaxisDrillBatch(count, "all"), nil
	}
	return training.GenerateProphylaxisDrillsFromDatabase(db, count)
}

// SubmitProphylaxisAnswer checks the user's quiet prophylactic move.
func (a *App) SubmitProphylaxisAnswer(drillID string, moveUCI string) (*training.ProphylaxisFeedback, error) {
	return training.SubmitProphylaxisAnswer(drillID, moveUCI)
}

func (a *App) collectLinkedChessAndLichessIDs() ([]string, []string) {
	var chessComIDs []string
	var lichessIDs []string

	if a.profileMgr != nil {
		for _, p := range a.profileMgr.GetProfiles() {
			if strings.TrimSpace(p.ChessComUsername) != "" {
				chessComIDs = append(chessComIDs, p.ChessComUsername)
			}
			if strings.TrimSpace(p.LichessUsername) != "" {
				lichessIDs = append(lichessIDs, p.LichessUsername)
			}
			if strings.TrimSpace(p.Name) != "" {
				chessComIDs = append(chessComIDs, p.Name)
				lichessIDs = append(lichessIDs, p.Name)
			}
		}
	}

	if a.personaMgr != nil {
		for _, p := range a.personaMgr.GetPersonas() {
			if strings.TrimSpace(p.ChessComUsername) != "" {
				chessComIDs = append(chessComIDs, p.ChessComUsername)
			}
			if strings.TrimSpace(p.LichessUsername) != "" {
				lichessIDs = append(lichessIDs, p.LichessUsername)
			}
			for _, id := range p.Identifiers {
				chessComIDs = append(chessComIDs, id)
				lichessIDs = append(lichessIDs, id)
			}
		}
	}

	return chessComIDs, lichessIDs
}

// GetSupportStatus returns the current voluntary support status, Stripe URL, and whether the startup prompt should display.
// It automatically exempts the author's computer using the linked Chess.com and Lichess.org IDs.
func (a *App) GetSupportStatus() support.SupportStatus {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	chessComIDs, lichessIDs := a.collectLinkedChessAndLichessIDs()
	return a.supportMgr.GetStatus(chessComIDs, lichessIDs)
}

// RecordUserSupport records that the user has supported NewScid (or verified an author/supporter ID),
// permanently disabling the startup nag dialog.
func (a *App) RecordUserSupport(tier string, reference string) (support.SupportStatus, error) {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	return a.supportMgr.RecordSupport(tier, reference)
}

// DismissSupportNag dismisses the non-obligatory support prompt for the current session.
func (a *App) DismissSupportNag() (support.SupportStatus, error) {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	return a.supportMgr.DismissNag()
}

// OpenSupportLink opens the Stripe support checkout URL in the user's default system browser
// with client_reference_id=newscid_<amount>usd so payments from NewScid are distinguishable from Bughouse Online in Stripe.
func (a *App) OpenSupportLink(amountUSD int) string {
	st := a.GetSupportStatus()
	if amountUSD < support.DefaultMinSupportUSD {
		amountUSD = support.DefaultMinSupportUSD
	}
	baseURL := support.StripeURLForAmount(amountUSD, st.StripeURL)
	targetURL := support.BuildCheckoutURL(baseURL, fmt.Sprintf("newscid_%dusd", amountUSD))
	_, _ = a.RecordUserSupport(fmt.Sprintf("$%d Support", amountUSD), "stripe_checkout")
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, targetURL)
	}
	return targetURL
}

// SetSupportStripeURL updates the Stripe payment link URL used by the Support form.
func (a *App) SetSupportStripeURL(url string) error {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	return a.supportMgr.SetStripeURL(url)
}

// SyncRemoteSupportStatus verifies and restores supporter status from the shared FairPlay Radar / NewScid database.
func (a *App) SyncRemoteSupportStatus(handle string) (support.SupportStatus, error) {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	return a.supportMgr.SyncRemoteSupporter(handle)
}

// PurchaseMicrosoftStoreUnlock triggers the native Microsoft Store $9.99 permanent unlock in-app purchase.
func (a *App) PurchaseMicrosoftStoreUnlock() support.StorePurchaseResult {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	res, _ := a.supportMgr.PurchaseMicrosoftStoreUnlock(a.ctx)
	return res
}

// GetMegaDatabaseCatalog returns the curated catalog of big online chess databases
// (Lumbra's GigaBase, Lichess Open, Lichess Broadcasts, and Lichess Elite).
func (a *App) GetMegaDatabaseCatalog() []online.MegaDatabaseSource {
	client := online.NewMegaDatabaseClient()
	return client.GetCatalog()
}

// GetLatestMegaDatabaseCatalog returns only the most recent (2026 / latest cumulative) online databases.
func (a *App) GetLatestMegaDatabaseCatalog() []online.MegaDatabaseSource {
	client := online.NewMegaDatabaseClient()
	return client.GetLatestCatalog()
}

// FetchLichessDatabaseFeed fetches the latest monthly .pgn.zst archive URLs from database.lichess.org.
func (a *App) FetchLichessDatabaseFeed(feedType string, limit int) ([]string, error) {
	client := online.NewMegaDatabaseClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return client.FetchLichessFeed(ctx, feedType, limit)
}

// ImportMegaDatabase downloads or streams a big online database (LumbrasGigaBase, Lichess Open, Lichess Elite, etc.)
// and imports the games into the active Scid database (or creates a new .si4 database if requested).
func (a *App) ImportMegaDatabase(opts online.MegaImportOptions) (*online.MegaImportResult, error) {
	client := online.NewMegaDatabaseClient()
	home, _ := os.UserHomeDir()
	userDBDir := filepath.Join(home, ".local", "share", fileformat.AppDirName, fileformat.DirDatabases)
	_ = os.MkdirAll(userDBDir, 0o755)

	// Determine whether we need to create/open a new database first
	a.mu.Lock()
	targetDB := a.db
	isOpen := targetDB != nil && targetDB.IsOpen
	a.mu.Unlock()

	src, hasSrc := client.FindSource(opts.SourceID)
	isNativeSI4 := hasSrc && src.Format == "7z-si4"

	if !isNativeSI4 {
		newDBName := strings.TrimSpace(opts.CreateNewDatabase)
		if newDBName == "" && !isOpen {
			if hasSrc {
				newDBName = strings.ReplaceAll(strings.ToLower(src.ID), "-", "_")
			} else {
				newDBName = "mega_imported"
			}
		}
		if newDBName != "" {
			cleanBase := filepath.Base(newDBName)
			cleanBase = fileformat.TrimScidExtensions(cleanBase)
			newDBPath := filepath.Join(userDBDir, cleanBase)
			if _, err := os.Stat(newDBPath + fileformat.ExtScidIndex); err == nil {
				if _, err := a.OpenDatabase(newDBPath); err != nil {
					return nil, fmt.Errorf("opening database %s: %w", newDBPath, err)
				}
			} else {
				if _, err := a.CreateDatabase(newDBPath, cleanBase); err != nil {
					return nil, fmt.Errorf("creating database %s: %w", newDBPath, err)
				}
			}
			a.mu.Lock()
			targetDB = a.db
			a.mu.Unlock()
		}
	}

	opCtx, finishOp := a.startCancellableOperation("mega-import")
	defer finishOp()

	res, err := client.DownloadAndImport(opCtx, opts, targetDB, userDBDir, func(prog online.MegaImportProgress) {
		a.emitEvent("megadb:progress", prog)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			if res == nil {
				res = &online.MegaImportResult{
					SourceID:   opts.SourceID,
					SourceName: src.Name,
				}
			}
			res.Cancelled = true
			return res, nil
		}
		return res, err
	}

	// If a native .si4 archive was extracted, open it automatically and protect it as sync_only!
	if res != nil && res.DatabasePath != "" {
		_, _ = a.SetDatabaseWritePolicy(res.DatabasePath, scid.WritePolicySyncOnly.String())
		_, _ = a.OpenDatabase(res.DatabasePath)
	} else if targetDB != nil && targetDB.BasePath != "" {
		_, _ = a.SetDatabaseWritePolicy(targetDB.BasePath, scid.WritePolicySyncOnly.String())
	}
	return res, nil
}

// AnalyzePlayerFairPlay runs the Longitudinal Fair-Play & Time-Control Consistency Inspector
// across all games of playerName in the currently open database.
func (a *App) AnalyzePlayerFairPlay(playerName string, maxGames int) (*fairplay.PlayerFairPlayDossier, error) {
	a.mu.Lock()
	db := a.db
	a.mu.Unlock()
	return fairplay.AnalyzePlayerFairPlay(db, playerName, maxGames)
}

// FetchFairPlayCommunityIntel queries the shared FairPlay Radar community telemetry ledger.
func (a *App) FetchFairPlayCommunityIntel(platform, username string, fallbackRiskPct int, confirmedBanned bool) fairplay.CommunitySuspectIntel {
	client := fairplay.NewCommunityTelemetryClient("")
	return client.FetchIntel(a.ctx, platform, username, fallbackRiskPct, confirmedBanned)
}

// SubmitFairPlayCommunityFlag reports a suspect flag to the shared FairPlay Radar telemetry ledger.
func (a *App) SubmitFairPlayCommunityFlag(platform, username string, riskPct int, confirmedBanned bool, banReason string) (*fairplay.CommunitySuspectIntel, error) {
	client := fairplay.NewCommunityTelemetryClient("")
	return client.SubmitFlag(a.ctx, platform, username, riskPct, confirmedBanned, banReason)
}

// DownloadAndAnalyzePlayerFairPlay fetches a player's recent games across all time controls from
// Lichess.org and/or Chess.com, imports them into the active (or dedicated) database, and returns
// the complete Longitudinal Multi-Time-Control & Day-by-Day Fair-Play Dossier.
func (a *App) DownloadAndAnalyzePlayerFairPlay(platform string, username string, maxGames int) (*fairplay.PlayerFairPlayDossier, error) {
	user := strings.TrimSpace(username)
	if user == "" {
		return nil, fmt.Errorf("username is required")
	}
	if maxGames <= 0 {
		maxGames = 200
	}

	// Ensure a database is open to store and inspect the downloaded games
	a.mu.Lock()
	db := a.db
	isOpen := db != nil && db.IsOpen
	a.mu.Unlock()

	if !isOpen {
		home, _ := os.UserHomeDir()
		userDBDir := filepath.Join(home, ".local", "share", fileformat.AppDirName, fileformat.DirDatabases)
		_ = os.MkdirAll(userDBDir, 0o755)
		safeName := "fairplay_" + strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(user, "_"))
		dbPath := filepath.Join(userDBDir, safeName)
		if _, err := os.Stat(dbPath + fileformat.ExtScidIndex); err == nil {
			if _, err := a.OpenDatabase(dbPath); err != nil {
				return nil, fmt.Errorf("opening fair-play database: %w", err)
			}
		} else {
			if _, err := a.CreateDatabase(dbPath, "Fair-Play Dossier: "+user); err != nil {
				return nil, fmt.Errorf("creating fair-play database: %w", err)
			}
		}
	}

	plat := strings.ToLower(strings.TrimSpace(platform))
	opts := online.ImportPlayerOptions{
		PlayerName:      user,
		TargetMode:      online.TargetActiveBase,
		MaxGames:        maxGames,
		AutoDeduplicate: true,
	}
	switch plat {
	case "chesscom", "chess.com":
		opts.ChessComUsername = user
	case "both":
		opts.LichessUsername = user
		opts.ChessComUsername = user
	default:
		opts.LichessUsername = user
	}

	syncRes, err := a.ImportPlayerGames(opts)
	if err != nil || (syncRes != nil && syncRes.TotalDownloaded == 0 && len(syncRes.Errors) > 0) {
		// Automatically fall back to the other platform if the user left the dropdown on the wrong site
		if opts.LichessUsername != "" && opts.ChessComUsername == "" {
			fallbackOpts := opts
			fallbackOpts.LichessUsername = ""
			fallbackOpts.ChessComUsername = user
			if fbRes, fbErr := a.ImportPlayerGames(fallbackOpts); fbErr == nil {
				syncRes = fbRes
				err = nil
			}
		} else if opts.ChessComUsername != "" && opts.LichessUsername == "" {
			fallbackOpts := opts
			fallbackOpts.ChessComUsername = ""
			fallbackOpts.LichessUsername = user
			if fbRes, fbErr := a.ImportPlayerGames(fallbackOpts); fbErr == nil {
				syncRes = fbRes
				err = nil
			}
		}
	}
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	db = a.db
	a.mu.Unlock()
	return fairplay.AnalyzePlayerFairPlay(db, user, maxGames)
}

// AnalyzeHardMoveBenchmark runs the "Man vs Chess" Self-Benchmarked Hard-Move Excess Detector
// across all games of benchmarkPlayer in the currently open database (or Default Personal User DB).
func (a *App) AnalyzeHardMoveBenchmark(
	benchmarkPlayer string,
	timeControlFilter string,
	maxGames int,
	checkOnlineBans bool,
) (*fairplay.HardMoveBenchmarkReport, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		if _, err := a.EnsurePersonalUserDatabase(true); err == nil {
			a.mu.RLock()
			db = a.db
			a.mu.RUnlock()
		}
	}
	return fairplay.AnalyzeHardMoveBenchmark(db, benchmarkPlayer, timeControlFilter, maxGames, checkOnlineBans)
}

// DownloadAndAnalyzeHardMoveBenchmark downloads a player's games from Lichess.org / Chess.com into the active
// or personal database and runs the Self-Benchmarked Hard-Move Excess Detector.
func (a *App) DownloadAndAnalyzeHardMoveBenchmark(
	platform string,
	username string,
	timeControlFilter string,
	maxGames int,
	checkOnlineBans bool,
) (*fairplay.HardMoveBenchmarkReport, error) {
	if _, err := a.DownloadAndAnalyzePlayerFairPlay(platform, username, maxGames); err != nil {
		return nil, err
	}
	return a.AnalyzeHardMoveBenchmark(username, timeControlFilter, maxGames, checkOnlineBans)
}
// Cyclical Tactical Drill & Lichess Theme Mastery
// GetCyclicalDashboard returns the user's cyclical drill sets, improvement history, and Lichess-style theme performance radar.
func (a *App) GetCyclicalDashboard() training.CyclicalDashboard {
	if a.cyclicalMgr == nil {
		configDir, _ := os.UserConfigDir()
		a.cyclicalMgr, _ = training.NewCyclicalManager(filepath.Join(configDir, fileformat.AppDirName, fileformat.FileCyclicalDrillJSON))
	}
	return a.cyclicalMgr.GetDashboard()
}

// GetLichessThemesCatalog returns the full catalog of Lichess tactical & phase themes.
func (a *App) GetLichessThemesCatalog() []training.LichessThemeInfo {
	return training.GetLichessThemeCatalog()
}

// CreateCyclicalDrillSet builds a new Easy-to-Hard ordered Cyclical Drill Set from the user's game history and Blunder Vault.
func (a *App) CreateCyclicalDrillSet(name string, playerOrPersonaID string, themeFilter []string, setSize int) (*training.CycleSet, error) {
	if a.cyclicalMgr == nil {
		configDir, _ := os.UserConfigDir()
		a.cyclicalMgr, _ = training.NewCyclicalManager(filepath.Join(configDir, fileformat.AppDirName, fileformat.FileCyclicalDrillJSON))
	}

	resolvedPlayer := strings.TrimSpace(playerOrPersonaID)
	var games []*chess.Game

	// Check if playerOrPersonaID matches a Persona ID or DisplayName
	if a.personaMgr != nil && resolvedPlayer != "" {
		for _, p := range a.personaMgr.GetPersonas() {
			if p.ID == resolvedPlayer || strings.EqualFold(p.DisplayName, resolvedPlayer) {
				resolvedPlayer = p.DisplayName
				if p.DatabasePath != "" {
					pDB := scid.NewDatabase()
					if err := pDB.Open(p.DatabasePath); err == nil {
						total := pDB.NumGames()
						limit := 250
						if total < limit {
							limit = total
						}
						for id := total; id >= 1 && len(games) < limit; id-- {
							if g, err := pDB.GetGame(id); err == nil && g != nil {
								games = append(games, g)
							}
						}
						_ = pDB.Close()
					}
				}
				break
			}
		}
	}

	// Also pull games from the currently open database
	a.mu.RLock()
	db := a.db
	activeGame := a.activeGame
	a.mu.RUnlock()

	if len(games) < 250 && db != nil && db.IsOpen {
		total := db.NumGames()
		for id := total; id >= 1 && len(games) < 250; id-- {
			if g, err := db.GetGame(id); err == nil && g != nil {
				games = append(games, g)
			}
		}
	}
	if len(games) == 0 && activeGame != nil {
		games = append(games, activeGame)
	}

	var vaultPuzzles []*blundervault.Puzzle
	if a.blunderVault != nil {
		vaultPuzzles = a.blunderVault.GetAllPuzzles()
	}

	return a.cyclicalMgr.CreateCycleSet(name, resolvedPlayer, games, vaultPuzzles, themeFilter, setSize)
}

// DeleteCyclicalDrillSet removes a cyclical drill set by ID.
func (a *App) DeleteCyclicalDrillSet(setID string) error {
	if a.cyclicalMgr == nil {
		return fmt.Errorf("cyclical manager not initialized")
	}
	return a.cyclicalMgr.DeleteCycleSet(setID)
}

// RecordCyclicalPuzzleAttempt records the user's attempt on a puzzle in an active cycle and updates Lichess theme stats.
func (a *App) RecordCyclicalPuzzleAttempt(setID string, puzzleID string, solvedFirstTry bool, timeTakenSec int) (*training.CycleSet, error) {
	if a.cyclicalMgr == nil {
		return nil, fmt.Errorf("cyclical manager not initialized")
	}
	return a.cyclicalMgr.RecordCyclePuzzleAttempt(setID, puzzleID, solvedFirstTry, timeTakenSec)
}

// StartNextCyclicalCycle starts or resets a cycle run for the specified drill set.
func (a *App) StartNextCyclicalCycle(setID string) (*training.CycleSet, error) {
	if a.cyclicalMgr == nil {
		return nil, fmt.Errorf("cyclical manager not initialized")
	}
	return a.cyclicalMgr.StartNextCycle(setID)
}

// CheckCyclicalEntitlement checks whether the user is entitled to the Cyclical Tactical Drill module
// (unlocked by default in NewScid, with future bughouse-online donor/supporter readiness).
func (a *App) CheckCyclicalEntitlement() support.FeatureEntitlement {
	chessComIDs, lichessIDs := a.collectLinkedChessAndLichessIDs()
	if a.supportMgr == nil {
		return support.FeatureEntitlement{
			FeatureID:    "cyclical-tactical-drill",
			Unlocked:     true,
			EnforceGate:  false,
			TierRequired: "bughouse-online-supporter",
			Message:      "Included in NewScid Workstation (Future bughouse-online Supporter Feature)",
		}
	}
	return a.supportMgr.CheckFeatureEntitlement("cyclical-tactical-drill", false, chessComIDs, lichessIDs)
}

// AdaptiveCoachRequest configures an online/database scan and custom instructional coaching session.
type AdaptiveCoachRequest struct {
	Source           string `json:"source"` // "both", "online", "database"
	LichessUsername  string `json:"lichessUsername"`
	ChessComUsername string `json:"chessComUsername"`
	PlayerName       string `json:"playerName"`
	FocusTheme       string `json:"focusTheme"` // "auto" or specific theme key like "hangingPiece"
	MaxGames         int    `json:"maxGames"`
	BatchSize        int    `json:"batchSize"`
}

func (a *App) ensureCoachMgr() *training.AdaptiveCoachManager {
	if a.coachMgr == nil {
		configDir, _ := os.UserConfigDir()
		a.coachMgr, _ = training.NewAdaptiveCoachManager(filepath.Join(configDir, fileformat.AppDirName, fileformat.FileAdaptiveCoachJSON))
	}
	return a.coachMgr
}

// GetAdaptiveCoachDashboard returns the current adaptive coaching weakness profile, drill queue, and session history.
func (a *App) GetAdaptiveCoachDashboard() (*training.AdaptiveCoachDashboard, error) {
	mgr := a.ensureCoachMgr()
	if mgr == nil {
		return nil, fmt.Errorf("adaptive coach manager not initialized")
	}
	dash := mgr.GetDashboard()
	return &dash, nil
}

// GenerateAdaptiveCoachingSession scans the user's online (Lichess / Chess.com) and/or local database games,
// diagnoses recurring tactical blind spots (such as missed hanging pieces), and builds a customized high-instructional drill queue.
func (a *App) GenerateAdaptiveCoachingSession(req AdaptiveCoachRequest) (*training.AdaptiveCoachDashboard, error) {
	mgr := a.ensureCoachMgr()
	if mgr == nil {
		return nil, fmt.Errorf("adaptive coach manager not initialized")
	}

	maxGames := req.MaxGames
	if maxGames <= 0 {
		maxGames = 100
	}
	if maxGames > 500 {
		maxGames = 500
	}

	batchSize := req.BatchSize
	if batchSize <= 0 {
		batchSize = 12
	}

	source := strings.ToLower(strings.TrimSpace(req.Source))
	if source == "" {
		source = "both"
	}

	lichessUser := strings.TrimSpace(req.LichessUsername)
	chessComUser := strings.TrimSpace(req.ChessComUsername)
	playerName := strings.TrimSpace(req.PlayerName)

	var identifiers []string
	var personaDBPath string

	if a.personaMgr != nil {
		for _, p := range a.personaMgr.GetPersonas() {
			if p.IsUserProfile ||
				(playerName != "" && strings.EqualFold(p.DisplayName, playerName)) ||
				(lichessUser != "" && strings.EqualFold(p.LichessUsername, lichessUser)) ||
				(chessComUser != "" && strings.EqualFold(p.ChessComUsername, chessComUser)) {
				if lichessUser == "" {
					lichessUser = strings.TrimSpace(p.LichessUsername)
				}
				if chessComUser == "" {
					chessComUser = strings.TrimSpace(p.ChessComUsername)
				}
				if playerName == "" {
					playerName = strings.TrimSpace(p.DisplayName)
				}
				if personaDBPath == "" && p.DatabasePath != "" {
					personaDBPath = p.DatabasePath
				}
				identifiers = append(identifiers, p.DisplayName, p.LichessUsername, p.ChessComUsername)
				identifiers = append(identifiers, p.Identifiers...)
				break
			}
		}
	}

	if lichessUser != "" {
		identifiers = append(identifiers, lichessUser)
	}
	if chessComUser != "" {
		identifiers = append(identifiers, chessComUser)
	}
	if playerName != "" {
		identifiers = append(identifiers, playerName)
	}

	var games []*chess.Game

	if source == "online" || source == "both" {
		if a.syncMgr == nil {
			a.syncMgr = online.NewSyncManager(a.ensureProfileMgr())
		}
		if lichessUser != "" && a.syncMgr.Lichess != nil {
			if pgnText, err := a.syncMgr.Lichess.FetchUserGames(lichessUser, 0, "", false, maxGames); err == nil && strings.TrimSpace(pgnText) != "" {
				if parsed, pErr := chess.ParsePGNDatabase(strings.NewReader(pgnText)); pErr == nil {
					games = append(games, parsed...)
				}
			}
		}
		if chessComUser != "" && a.syncMgr.ChessCom != nil {
			if pgnText, err := a.syncMgr.ChessCom.FetchUserGames(chessComUser, 0, 0, maxGames); err == nil && strings.TrimSpace(pgnText) != "" {
				if parsed, pErr := chess.ParsePGNDatabase(strings.NewReader(pgnText)); pErr == nil {
					games = append(games, parsed...)
				}
			}
		}
	}

	if source == "database" || source == "both" || len(games) == 0 {
		if personaDBPath != "" {
			pDB := scid.NewDatabase()
			if err := pDB.Open(personaDBPath); err == nil {
				total := pDB.NumGames()
				for id := total; id >= 1 && len(games) < maxGames*2; id-- {
					if g, gErr := pDB.GetGame(id); gErr == nil && g != nil {
						games = append(games, g)
					}
				}
				_ = pDB.Close()
			}
		}

		a.mu.RLock()
		db := a.db
		activeGame := a.activeGame
		a.mu.RUnlock()

		if db != nil && db.IsOpen {
			total := db.NumGames()
			for id := total; id >= 1 && len(games) < maxGames*2; id-- {
				if g, gErr := db.GetGame(id); gErr == nil && g != nil {
					games = append(games, g)
				}
			}
		}
		if len(games) == 0 && activeGame != nil {
			games = append(games, activeGame)
		}
	}

	return mgr.ScanAndBuildCurriculum(games, identifiers, req.FocusTheme, batchSize)
}

// RecordAdaptiveCoachAttempt records a drill attempt in the adaptive coach and triggers extra reinforcement drills if missed.
func (a *App) RecordAdaptiveCoachAttempt(drillID string, solvedFirstTry bool, timeTakenSec int) (*training.AdaptiveDrillResult, error) {
	mgr := a.ensureCoachMgr()
	if mgr == nil {
		return nil, fmt.Errorf("adaptive coach manager not initialized")
	}
	return mgr.RecordDrillAttempt(drillID, solvedFirstTry, timeTakenSec)
}
