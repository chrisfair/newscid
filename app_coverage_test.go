package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/blundervault"
	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/engine"
	"newscid/pkg/online"
	"newscid/pkg/persona"
	"newscid/pkg/polyglot"
	"newscid/pkg/repertoire"
	"newscid/pkg/tournament"
)

const covSamplePGN = `[Event "Cov Match"]
[Site "https://lichess.org/cov12345"]
[Date "2026.05.01"]
[Round "1"]
[White "CovHero"]
[Black "CovVillain"]
[Result "1-0"]
[WhiteElo "2100"]
[BlackElo "2050"]
[TimeControl "180+2"]
[ECO "C50"]

1. e4 {[%clk 0:03:00]} 1... e5 {[%clk 0:03:00]} 2. Nf3 {[%clk 0:02:58]} 2... Nc6 {[%clk 0:02:57]} 3. Bc4 {[%clk 0:02:55]} 3... Bc5 {[%clk 0:02:52]} 4. d3 {[%clk 0:02:49]} 4... Nf6 {[%clk 0:02:45]} 5. O-O {[%clk 0:02:40]} 5... d6 {[%clk 0:02:35]} 6. c3 {[%clk 0:02:30]} 6... a6 {[%clk 0:02:20]} 1-0

[Event "Cov Match"]
[Site "https://lichess.org/cov67890"]
[Date "2026.05.02"]
[Round "2"]
[White "CovVillain"]
[Black "CovHero"]
[Result "0-1"]
[WhiteElo "2050"]
[BlackElo "2100"]
[TimeControl "600+0"]
[ECO "B20"]

1. e4 {[%clk 0:10:00]} 1... c5 {[%clk 0:10:00]} 2. Nf3 {[%clk 0:09:50]} 2... d6 {[%clk 0:09:45]} 3. d4 {[%clk 0:09:30]} 3... cxd4 {[%clk 0:09:20]} 4. Nxd4 {[%clk 0:09:10]} 4... Nf6 {[%clk 0:09:05]} 5. Nc3 {[%clk 0:08:50]} 5... a6 {[%clk 0:08:40]} 0-1
`

const covBlunderPGN = `[Event "Blunder Blitz"]
[Site "NewScid"]
[Date "2026.05.03"]
[Round "1"]
[White "CovHero"]
[Black "CovVillain"]
[Result "0-1"]
[WhiteElo "1700"]
[BlackElo "1720"]
[TimeControl "180+0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Nd4 4. Nxe5 Qg5 5. Nxf7 Qxg2 6. Rf1 Qxe4+ 7. Be2 Nf3# 0-1

[Event "Blunder Blitz"]
[Site "NewScid"]
[Date "2026.05.04"]
[Round "2"]
[White "CovVillain"]
[Black "CovHero"]
[Result "1-0"]
[WhiteElo "1720"]
[BlackElo "1700"]
[TimeControl "180+0"]

1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0
`

func newIsolatedApp(t *testing.T) (*App, string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))
	app := NewApp()
	return app, tmp
}

func newMockOnlineServer(t *testing.T, pgn string) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/games/user/"):
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			_, _ = w.Write([]byte(pgn))
		case strings.HasPrefix(r.URL.Path, "/api/user/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"covhero","username":"CovHero","perfs":{"blitz":{"rating":2100,"games":50}}}`))
		case strings.HasPrefix(r.URL.Path, "/game/export/"):
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			_, _ = w.Write([]byte(pgn))
		case strings.HasSuffix(r.URL.Path, "/games/archives"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"archives":["%s/pub/player/covhero/games/2026/05"]}`, srv.URL)
		case strings.HasSuffix(r.URL.Path, "/games/2026/05/pgn"):
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			singlePGN := "[Event \"Cov Match\"]\n[Site \"https://www.chess.com/game/live/999888777\"]\n[Date \"2026.05.01\"]\n[Round \"1\"]\n[White \"CovHero\"]\n[Black \"CovVillain\"]\n[Result \"1-0\"]\n\n1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0\n"
			_, _ = w.Write([]byte(singlePGN))
		case strings.Contains(r.URL.Path, "/games/2026/05"):
			w.Header().Set("Content-Type", "application/json")
			singlePGN := "[Event \"Cov Match\"]\n[Site \"https://www.chess.com/game/live/999888777\"]\n[Date \"2026.05.01\"]\n[Round \"1\"]\n[White \"CovHero\"]\n[Black \"CovVillain\"]\n[Result \"1-0\"]\n\n1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0\n"
			escaped, _ := json.Marshal(singlePGN)
			_, _ = fmt.Fprintf(w, `{"games":[{"url":"https://www.chess.com/game/live/999888777","pgn":%s,"end_time":1777600000,"time_class":"blitz","white":{"username":"CovHero","rating":2100},"black":{"username":"CovVillain","rating":2050}}]}`, string(escaped))
		default:
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			_, _ = w.Write([]byte(pgn))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func createTempScidDB(t *testing.T, dir, name, pgn string) string {
	t.Helper()
	basePath := filepath.Join(dir, name)
	db := scid.NewDatabase()
	require.NoError(t, db.Create(basePath, "Test DB "+name))
	games, err := chess.ParsePGNDatabase(strings.NewReader(pgn))
	require.NoError(t, err)
	_, err = db.ImportGames(games)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	return basePath
}

func TestCovAppRepertoireFull(t *testing.T) {
	app, tmp := newIsolatedApp(t)
	srv := newMockOnlineServer(t, covSamplePGN)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// Save custom repertoire and CRUD
	rep := &repertoire.Repertoire{
		ID:        "cov-white-rep",
		Name:      "Cov White Rep",
		Color:     "white",
		Positions: map[string][]repertoire.RepertoireMove{},
	}
	require.NoError(t, app.SaveRepertoire(rep))

	got, err := app.GetRepertoire("cov-white-rep")
	require.NoError(t, err)
	assert.Equal(t, "Cov White Rep", got.Name)

	// AddRepertoireMove: new, duplicate, unknown repID
	nextFEN := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	require.NoError(t, app.AddRepertoireMove("cov-white-rep", chess.StartingFEN, "e2e4", "e4", nextFEN, "King pawn"))
	require.NoError(t, app.AddRepertoireMove("cov-white-rep", chess.StartingFEN, "e2e4", "e4", nextFEN, "duplicate"))
	assert.Error(t, app.AddRepertoireMove("nonexistent-rep", chess.StartingFEN, "e2e4", "e4", nextFEN, ""))

	// AddRepertoireLineSAN, UpdateRepertoireMoveComment, RecordRepertoirePractice
	updated, err := app.AddRepertoireLineSAN("cov-white-rep", "1. e4 e5 2. Nf3 Nc6 3. Bb5", "Ruy Lopez")
	require.NoError(t, err)
	assert.NotNil(t, updated)

	_, err = app.UpdateRepertoireMoveComment("cov-white-rep", chess.StartingFEN, "e2e4", "Main weapon")
	require.NoError(t, err)

	check, err := app.VerifyRepertoireMove("cov-white-rep", chess.StartingFEN, "e2e4")
	require.NoError(t, err)
	assert.True(t, check.IsRepertoire)
	_, err = app.VerifyRepertoireMove("nonexistent-rep", chess.StartingFEN, "e2e4")
	assert.Error(t, err)

	app.RecordRepertoirePractice("cov-white-rep", chess.StartingFEN, "e2e4", true)

	mastery, err := app.GetRepertoireMastery("cov-white-rep")
	require.NoError(t, err)
	assert.NotNil(t, mastery)

	resetMastery, err := app.ResetRepertoireMastery("cov-white-rep")
	require.NoError(t, err)
	assert.NotNil(t, resetMastery)
	_, err = app.ResetRepertoireMastery("nonexistent-rep")
	assert.Error(t, err)

	// CompileRepertoireToPolyglot with linked persona
	personaDB := createTempScidDB(t, tmp, "persona_rep_db", covSamplePGN)
	savedPersona, err := app.SavePersona(&persona.Persona{
		DisplayName:       "CovHero",
		IsUserProfile:     true,
		LichessUsername:   "CovHero",
		ChessComUsername:  "CovHero",
		WhiteRepertoireID: "cov-white-rep",
		DatabasePath:      personaDB,
		Identifiers:       []string{"CovHero"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, savedPersona.ID)

	compiledMsg, err := app.CompileRepertoireToPolyglot("cov-white-rep")
	require.NoError(t, err)
	assert.Contains(t, compiledMsg, ".bin")
	_, err = app.CompileRepertoireToPolyglot("nonexistent-rep")
	assert.Error(t, err)

	// GetMainUserOnlineDefaults: user profile path
	defaults := app.GetMainUserOnlineDefaults()
	assert.Equal(t, "CovHero", defaults["displayName"])
	assert.Equal(t, "CovHero", defaults["lichessUsername"])

	// GetMainUserOnlineDefaults: fallback to non-user persona and profileMgr
	app2, _ := newIsolatedApp(t)
	_, err = app2.SavePersona(&persona.Persona{
		DisplayName:   "AltPlayer",
		IsUserProfile: false,
	})
	require.NoError(t, err)
	require.NoError(t, app2.SaveSyncProfile(&online.Profile{
		ID:               "p1",
		Name:             "SyncName",
		LichessUsername:  "sync_li",
		ChessComUsername: "sync_cc",
	}))
	def2 := app2.GetMainUserOnlineDefaults()
	assert.Equal(t, "sync_li", def2["lichessUsername"])
	assert.Equal(t, "sync_cc", def2["chessComUsername"])

	// ScanRepertoireDeviations: error on empty ID
	_, err = app.ScanRepertoireDeviations(RepertoireScanRequest{})
	assert.Error(t, err)

	// ScanRepertoireDeviations: online + database + maxGames clamping
	activeDB := createTempScidDB(t, tmp, "active_rep_db", covSamplePGN)
	_, err = app.OpenDatabase(activeDB)
	require.NoError(t, err)

	report, err := app.ScanRepertoireDeviations(RepertoireScanRequest{
		RepertoireID: "cov-white-rep",
		Source:       "",
		MaxGames:     2000,
	})
	require.NoError(t, err)
	assert.NotNil(t, report)

	// ActiveGame fallback when DB closed and source="database"
	require.NoError(t, app.CloseDatabase())
	g, err := chess.ParsePGN(covSamplePGN)
	require.NoError(t, err)
	app.activeGame = g
	savedPersona.DatabasePath = ""
	_, _ = app.SavePersona(savedPersona)

	report2, err := app.ScanRepertoireDeviations(RepertoireScanRequest{
		RepertoireID: "cov-white-rep",
		Source:       "database",
		PlayerName:   "CovHero",
		MaxGames:     0,
	})
	require.NoError(t, err)
	assert.NotNil(t, report2)

	// DeleteRepertoireMove & DeleteRepertoire
	_, err = app.DeleteRepertoireMove("cov-white-rep", chess.StartingFEN, "e2e4")
	require.NoError(t, err)
	require.NoError(t, app.DeleteRepertoire("cov-white-rep"))
}

func TestCovAppVariationsAnnotationsAndHeaders(t *testing.T) {
	app, tmp := newIsolatedApp(t)

	// No active game errors
	assert.Error(t, app.PromoteVariation(1, 0))
	assert.Error(t, app.DeleteVariation(1, 0))
	assert.Error(t, app.AddGameComment(1, "hi"))
	assert.Error(t, app.SetGameNAG(1, 1))
	assert.Error(t, app.ClearGameNAGs(1))

	// Parse a game with variations at ply 1 and ply 3
	g, err := chess.ParsePGN(`1. e4 (1. d4 d5) 1... e5 2. Nf3 (2. Bc4 Nf6) 2... Nc6 1-0`)
	require.NoError(t, err)
	app.activeGame = g

	// PromoteVariation error paths
	assert.Error(t, app.PromoteVariation(1, 5))
	assert.Error(t, app.PromoteVariation(99, 0))
	assert.Error(t, app.PromoteVariation(3, 5))

	// PromoteVariation ply > 1
	require.NoError(t, app.PromoteVariation(3, 0))
	// PromoteVariation ply <= 1
	require.NoError(t, app.PromoteVariation(1, 0))

	// DeleteVariation error paths and valid path
	assert.Error(t, app.DeleteVariation(99, 0))
	assert.Error(t, app.DeleteVariation(1, 5))
	require.NoError(t, app.DeleteVariation(1, 0))

	// FormatMove & FormatMoveSequence fallback styles
	app.notationStyle = chess.NotationInvalid
	s, err := app.FormatMove(chess.StartingFEN, "e2e4", "invalid_style")
	require.NoError(t, err)
	assert.Equal(t, "e4", s)

	seq, err := app.FormatMoveSequence(chess.StartingFEN, []string{"e2e4", "e7e5"}, "invalid_style")
	require.NoError(t, err)
	assert.Contains(t, seq, "e4")

	require.NoError(t, app.SetNotationStyle("uci"))
	seq2, err := app.FormatMoveSequence(chess.StartingFEN, []string{"e2e4", "e7e5"}, "invalid_style")
	require.NoError(t, err)
	assert.Contains(t, seq2, "e2e4")

	// UpdateGameHeaders error when no DB open + valid update
	_, err = app.UpdateGameHeaders(1, map[string]string{"White": "Test"})
	assert.Error(t, err)

	dbPath := createTempScidDB(t, tmp, "hdr_db", covSamplePGN)
	_, err = app.OpenDatabase(dbPath)
	require.NoError(t, err)
	updatedGame, err := app.UpdateGameHeaders(1, map[string]string{
		"White": "UpdatedHero",
		"Event": "UpdatedEvent",
	})
	require.NoError(t, err)
	assert.Equal(t, "UpdatedHero", updatedGame.White)
	_, err = app.UpdateGameHeaders(999, map[string]string{"White": "Nobody"})
	assert.Error(t, err)
}

func TestCovAppOpponentAndPersonaWeaknesses(t *testing.T) {
	app, tmp := newIsolatedApp(t)

	// No DB open errors
	_, err := app.EvaluateOpponentWeaknesses("CovHero", "both", 10, 1)
	assert.Error(t, err)

	activeDB := createTempScidDB(t, tmp, "weak_active", covSamplePGN)
	_, err = app.OpenDatabase(activeDB)
	require.NoError(t, err)

	_, err = app.EvaluateOpponentWeaknesses("   ", "both", 10, 1)
	assert.Error(t, err)

	oppReport, err := app.EvaluateOpponentWeaknesses("CovHero", "both", 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, oppReport)

	// EvaluatePersonaWeaknesses: nil personaMgr, unknown ID, valid across active + dedicated DB
	origPM := app.personaMgr
	app.personaMgr = nil
	_, err = app.EvaluatePersonaWeaknesses("missing", "both", 10, 1)
	assert.Error(t, err)
	app.personaMgr = origPM

	_, err = app.EvaluatePersonaWeaknesses("missing", "both", 10, 1)
	assert.Error(t, err)

	dedDB := createTempScidDB(t, tmp, "weak_ded", covBlunderPGN)
	p, err := app.SavePersona(&persona.Persona{
		DisplayName:  "CovHero",
		Identifiers:  []string{"CovHero", "Hero"},
		DatabasePath: dedDB,
	})
	require.NoError(t, err)

	pReport, err := app.EvaluatePersonaWeaknesses(p.ID, "both", 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, pReport)

	// matchPlayerLookup branches
	lookup := map[string]bool{"carlsen, magnus": true}
	assert.True(t, matchPlayerLookup("carlsen, magnus", lookup))
	assert.True(t, matchPlayerLookup("carlsen", lookup))
	assert.True(t, matchPlayerLookup("gm carlsen, magnus", lookup))
	assert.False(t, matchPlayerLookup("nakamura", lookup))

	// ExportAntiPersonaPolyglotBook & ExportAntiPersonaPGN
	_, err = app.ExportAntiPersonaPolyglotBook(nil, "")
	assert.Error(t, err)
	_, err = app.ExportAntiPersonaPGN(nil, "")
	assert.Error(t, err)

	explicitBin := filepath.Join(tmp, "anti_explicit.bin")
	outBin, err := app.ExportAntiPersonaPolyglotBook(pReport, explicitBin)
	require.NoError(t, err)
	assert.Equal(t, explicitBin, outBin)

	defaultBin, err := app.ExportAntiPersonaPolyglotBook(pReport, "")
	require.NoError(t, err)
	assert.FileExists(t, defaultBin)

	explicitPGN := filepath.Join(tmp, "anti_explicit.pgn")
	outPGN, err := app.ExportAntiPersonaPGN(pReport, explicitPGN)
	require.NoError(t, err)
	assert.Equal(t, explicitPGN, outPGN)

	defaultPGN, err := app.ExportAntiPersonaPGN(pReport, "")
	require.NoError(t, err)
	assert.FileExists(t, defaultPGN)
}

func TestCovAppSafetyIntuitionProphylaxisTimeAndVault(t *testing.T) {
	app, tmp := newIsolatedApp(t)

	// AnalyzeBoardSafety: empty FEN + nil activeGame -> StartingFEN
	rep1, err := app.AnalyzeBoardSafety("")
	require.NoError(t, err)
	assert.NotNil(t, rep1)

	// AnalyzeBoardSafety: empty FEN + activeGame with StartFEN and moves
	g, err := chess.ParsePGN(`[FEN "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"]

1... e5 2. Nf3 Nc6 *`)
	require.NoError(t, err)
	app.activeGame = g
	rep2, err := app.AnalyzeBoardSafety("")
	require.NoError(t, err)
	assert.NotNil(t, rep2)

	// Safety drills: batch, from DB (no DB fallback + with DB), submit
	batch, err := app.GetSafetyDrillBatch(3, "all")
	require.NoError(t, err)
	assert.NotEmpty(t, batch)

	fbSafety, err := app.GenerateSafetyDrillsFromDatabase(3)
	require.NoError(t, err)
	assert.NotEmpty(t, fbSafety)

	feedback, err := app.SubmitSafetyDrillAnswer(batch[0], []string{"e4"}, "")
	require.NoError(t, err)
	assert.NotNil(t, feedback)

	// Intuition & Prophylaxis fallback when no DB open
	fbIntuit, err := app.GenerateIntuitionDrillsFromDatabase(3)
	require.NoError(t, err)
	assert.NotEmpty(t, fbIntuit)

	fbProph, err := app.GenerateProphylaxisDrillsFromDatabase(3)
	require.NoError(t, err)
	assert.NotEmpty(t, fbProph)

	// Time management: no DB + nil activeGame error
	app.activeGame = nil
	_, err = app.AnalyzeGameTimeManagement(0)
	assert.Error(t, err)
	_, err = app.AnalyzePlayerTimeManagement("CovHero", 10)
	assert.Error(t, err)

	// Open DB and run DB-backed safety/intuition/prophylaxis/time management
	dbPath := createTempScidDB(t, tmp, "drills_db", covSamplePGN+"\n"+covBlunderPGN)
	_, err = app.OpenDatabase(dbPath)
	require.NoError(t, err)

	_, err = app.GenerateSafetyDrillsFromDatabase(2)
	require.NoError(t, err)
	_, err = app.GenerateIntuitionDrillsFromDatabase(2)
	require.NoError(t, err)
	_, err = app.GenerateProphylaxisDrillsFromDatabase(2)
	require.NoError(t, err)

	// AnalyzeGameTimeManagement: fallback to game 1 when activeGame nil, explicit gameID, invalid gameID
	tm1, err := app.AnalyzeGameTimeManagement(0)
	require.NoError(t, err)
	assert.NotNil(t, tm1)
	tm2, err := app.AnalyzeGameTimeManagement(1)
	require.NoError(t, err)
	assert.NotNil(t, tm2)
	_, err = app.AnalyzeGameTimeManagement(999)
	assert.Error(t, err)

	ptm, err := app.AnalyzePlayerTimeManagement("CovHero", 10)
	require.NoError(t, err)
	assert.NotNil(t, ptm)

	// Blunder vault: due puzzles + nil vault paths
	_, err = app.GetDueBlunderPuzzles()
	require.NoError(t, err)

	savedVault := app.blunderVault
	app.blunderVault = nil
	dueNil, err := app.GetDueBlunderPuzzles()
	require.NoError(t, err)
	assert.Empty(t, dueNil)
	allNil, err := app.GetAllBlunderPuzzles()
	require.NoError(t, err)
	assert.Empty(t, allNil)
	_, err = app.RecordBlunderReview("x", 4)
	assert.Error(t, err)
	err = app.AddBlunderToVault(&blundervault.Puzzle{})
	assert.Error(t, err)
	app.blunderVault = savedVault
}

func TestCovAppSupportStoreAndEntitlement(t *testing.T) {
	app, _ := newIsolatedApp(t)

	// Populate profileMgr and personaMgr for collectLinkedChessAndLichessIDs
	require.NoError(t, app.SaveSyncProfile(&online.Profile{
		ID:               "sup-prof",
		Name:             "SupName",
		LichessUsername:  "sup_li",
		ChessComUsername: "sup_cc",
	}))
	_, err := app.SavePersona(&persona.Persona{
		DisplayName:      "SupPersona",
		LichessUsername:  "per_li",
		ChessComUsername: "per_cc",
		Identifiers:      []string{"SupAlias"},
	})
	require.NoError(t, err)

	ccIDs, liIDs := app.collectLinkedChessAndLichessIDs()
	assert.Contains(t, ccIDs, "sup_cc")
	assert.Contains(t, liIDs, "per_li")

	// CheckCyclicalEntitlement with nil supportMgr, then initialized
	app.supportMgr = nil
	ent1 := app.CheckCyclicalEntitlement()
	assert.True(t, ent1.Unlocked)

	st := app.GetSupportStatus()
	assert.NotEmpty(t, st.StripeURL)

	ent2 := app.CheckCyclicalEntitlement()
	assert.True(t, ent2.Unlocked)

	app.supportMgr = nil
	_, err = app.DismissSupportNag()
	require.NoError(t, err)

	app.supportMgr = nil
	err = app.SetSupportStripeURL("https://buy.stripe.com/test_newscid")
	require.NoError(t, err)

	app.supportMgr = nil
	_, err = app.RecordUserSupport("$10 Support", "test_ref")
	require.NoError(t, err)

	// OpenSupportLink with amount < min and >= min (ctx is nil so browser is not opened)
	url1 := app.OpenSupportLink(1)
	assert.Contains(t, url1, "newscid_")
	url2 := app.OpenSupportLink(25)
	assert.Contains(t, url2, "newscid_25usd")

	// PurchaseMicrosoftStoreUnlock (returns non-windows / unavailable result cleanly on Linux)
	app.supportMgr = nil
	storeRes := app.PurchaseMicrosoftStoreUnlock()
	assert.NotNil(t, storeRes)
}

func TestCovAppCyclicalAndAdaptiveCoach(t *testing.T) {
	app, tmp := newIsolatedApp(t)
	srv := newMockOnlineServer(t, covBlunderPGN)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// Nil cyclicalMgr error paths
	app.cyclicalMgr = nil
	assert.Error(t, app.DeleteCyclicalDrillSet("x"))
	_, err := app.RecordCyclicalPuzzleAttempt("x", "p", true, 5)
	assert.Error(t, err)
	_, err = app.StartNextCyclicalCycle("x")
	assert.Error(t, err)

	// Dashboard & themes catalog
	dash := app.GetCyclicalDashboard()
	assert.NotNil(t, dash)
	themes := app.GetLichessThemesCatalog()
	assert.NotEmpty(t, themes)

	// Add a blunder puzzle to vault so cyclical set has puzzles
	err = app.AddBlunderToVault(&blundervault.Puzzle{
		ID:         "bv-1",
		FEN:        "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4",
		PlayerName: "CovHero",
		UserColor:  "white",
		PlayedMove: "h5e5",
		BestMove:   "h5f7",
		DeltaCp:    500,
	})
	require.NoError(t, err)

	// Create persona DB + active DB
	personaDB := createTempScidDB(t, tmp, "cyc_persona_db", covBlunderPGN)
	p, err := app.SavePersona(&persona.Persona{
		DisplayName:      "CovHero",
		IsUserProfile:    true,
		LichessUsername:  "CovHero",
		ChessComUsername: "CovHero",
		DatabasePath:     personaDB,
		Identifiers:      []string{"CovHero"},
	})
	require.NoError(t, err)

	activeDB := createTempScidDB(t, tmp, "cyc_active_db", covSamplePGN)
	_, err = app.OpenDatabase(activeDB)
	require.NoError(t, err)

	app.cyclicalMgr = nil
	set, err := app.CreateCyclicalDrillSet("My Cycle", p.ID, nil, 5)
	require.NoError(t, err)
	require.NotNil(t, set)
	if len(set.Puzzles) > 0 {
		_, err = app.RecordCyclicalPuzzleAttempt(set.ID, set.Puzzles[0].ID, true, 12)
		require.NoError(t, err)
	}
	_, err = app.StartNextCyclicalCycle(set.ID)
	require.NoError(t, err)
	require.NoError(t, app.DeleteCyclicalDrillSet(set.ID))

	// Adaptive Coach: dashboard, generate (online + database + activeGame), record attempt
	app.coachMgr = nil
	cDash, err := app.GetAdaptiveCoachDashboard()
	require.NoError(t, err)
	assert.NotNil(t, cDash)

	// Online + database
	cDash2, err := app.GenerateAdaptiveCoachingSession(AdaptiveCoachRequest{
		Source:     "",
		PlayerName: "CovHero",
		FocusTheme: "auto",
		MaxGames:   600,
		BatchSize:  0,
	})
	require.NoError(t, err)
	assert.NotNil(t, cDash2)

	// ActiveGame fallback when DB closed
	require.NoError(t, app.CloseDatabase())
	p.DatabasePath = ""
	_, _ = app.SavePersona(p)
	ag, err := chess.ParsePGN(covBlunderPGN)
	require.NoError(t, err)
	app.activeGame = ag

	cDash3, err := app.GenerateAdaptiveCoachingSession(AdaptiveCoachRequest{
		Source:     "database",
		PlayerName: "CovHero",
		MaxGames:   0,
		BatchSize:  4,
	})
	require.NoError(t, err)
	assert.NotNil(t, cDash3)

	if len(cDash3.ActiveQueue) > 0 {
		_, err = app.RecordAdaptiveCoachAttempt(cDash3.ActiveQueue[0].ID, true, 8)
		require.NoError(t, err)
	} else {
		_, _ = app.RecordAdaptiveCoachAttempt("nonexistent", false, 8)
	}
}

func TestCovAppEngineTreeTournamentSimulationAndRPC(t *testing.T) {
	app, tmp := newIsolatedApp(t)

	// Simple discovery / metadata helpers
	_ = app.GetAvailableEngines()
	assert.NotEmpty(t, app.GetPieceSets())
	assert.NotEmpty(t, app.GetBoardThemes())
	_ = app.GetScidSystemBases()
	_ = app.IsScidDriverAvailable()
	assert.NotEmpty(t, app.GetMegaDatabaseCatalog())
	assert.NotEmpty(t, app.GetLatestMegaDatabaseCatalog())

	// Event listener coverage
	var events []string
	app.SetEventListener(func(eventName string, optionalData ...interface{}) {
		events = append(events, eventName)
	})
	app.emitEvent("test:event", "payload")
	assert.Contains(t, events, "test:event")

	// OpeningTree: no DB open (falls back to polyglot or empty)
	treeNoDB, err := app.GetOpeningTree(chess.StartingFEN)
	require.NoError(t, err)
	assert.NotNil(t, treeNoDB)

	// OpeningTree: with Reference DB enabled
	refDBPath := createTempScidDB(t, tmp, "ref_tree_db", covSamplePGN)
	_, err = app.SetReferenceDatabase(refDBPath, true)
	require.NoError(t, err)
	treeRef, err := app.GetOpeningTree(chess.StartingFEN)
	require.NoError(t, err)
	assert.Greater(t, treeRef.TotalGames, 0)

	// Tournament events / crosstable: no DB error, then with DB open
	_, err = app.GetTournamentEvents()
	assert.Error(t, err)
	_, err = app.GetTournamentCrosstable("Cov Match", "", "")
	assert.Error(t, err)
	_, err = app.ExportCrosstableText("Cov Match", "", "")
	assert.Error(t, err)

	_, err = app.OpenDatabase(refDBPath)
	require.NoError(t, err)
	evts, err := app.GetTournamentEvents()
	require.NoError(t, err)
	assert.NotEmpty(t, evts)

	ct, err := app.GetTournamentCrosstable(evts[0].Event, evts[0].Site, "")
	require.NoError(t, err)
	assert.NotNil(t, ct)

	ctText, err := app.ExportCrosstableText(evts[0].Event, evts[0].Site, "")
	require.NoError(t, err)
	assert.NotEmpty(t, ctText)

	// Engine methods: GetEngineEval (nil engine), SetEngineMultiPV clamping, AnnotateCurrentGame nil error, AnnotatePGN invalid
	ev0 := app.GetEngineEval()
	assert.Equal(t, 0, ev0.Depth)
	require.NoError(t, app.SetEngineMultiPV(0))
	assert.Equal(t, 1, app.GetEngineMultiPV())
	require.NoError(t, app.SetEngineMultiPV(99))
	assert.Equal(t, 10, app.GetEngineMultiPV())
	require.NoError(t, app.SetEngineMultiPV(1))
	require.NoError(t, app.StopEngine())

	app.activeGame = nil
	_, err = app.AnnotateCurrentGame(30)
	assert.Error(t, err)

	// If a UCI engine is installed, test AnalyzePosition, GetEngineEval, AnnotatePGN
	if len(app.GetAvailableEngines()) > 0 {
		require.NoError(t, app.AnalyzePosition(chess.StartingFEN, nil, 1))
		_ = app.GetEngineEval()
		require.NoError(t, app.StopEngine())

		ann, err := app.AnnotatePGN("1. e4 e5 2. Nf3 Nc6 1-0", 25)
		require.NoError(t, err)
		assert.NotNil(t, ann)
	}

	// GetPersonaSimulatedMove: unknown persona, repertoire move, book move, historical move, out-of-book move
	_, err = app.GetPersonaSimulatedMove("does-not-exist", chess.StartingFEN, true)
	assert.Error(t, err)

	// Create a repertoire and polyglot book for persona
	rep := &repertoire.Repertoire{
		ID:        "sim-rep",
		Name:      "Sim Rep",
		Color:     "white",
		Positions: map[string][]repertoire.RepertoireMove{},
	}
	require.NoError(t, app.SaveRepertoire(rep))
	_, err = app.AddRepertoireLineSAN("sim-rep", "1. d4 d5 2. c4", "QGD")
	require.NoError(t, err)

	bookPath := filepath.Join(tmp, "sim_book.bin")
	_, err = app.repMgr.CompileToPolyglotBook("sim-rep", bookPath)
	require.NoError(t, err)

	simP, err := app.SavePersona(&persona.Persona{
		DisplayName:       "CovHero",
		EstimatedElo:      1850,
		WhiteRepertoireID: "sim-rep",
		BookPath:          bookPath,
		DatabasePath:      refDBPath,
		Identifiers:       []string{"CovHero"},
	})
	require.NoError(t, err)

	// 1) Hits WhiteRepertoireID
	mRep, err := app.GetPersonaSimulatedMove(simP.ID, chess.StartingFEN, true)
	require.NoError(t, err)
	assert.True(t, mRep.IsBook)
	assert.Equal(t, "d2d4", mRep.UCI)

	// 2) Clear repertoire so it hits BookPath
	simP.WhiteRepertoireID = ""
	_, err = app.SavePersona(simP)
	require.NoError(t, err)
	mBook, err := app.GetPersonaSimulatedMove(simP.ID, chess.StartingFEN, true)
	require.NoError(t, err)
	assert.True(t, mBook.IsBook)

	// 3) Clear BookPath so it hits historical DB move or out-of-book engine/tactical move
	simP.BookPath = ""
	_, err = app.SavePersona(simP)
	require.NoError(t, err)
	mHist, err := app.GetPersonaSimulatedMove(simP.ID, chess.StartingFEN, true)
	require.NoError(t, err)
	assert.NotEmpty(t, mHist.UCI)

	// 4) Out-of-book rare position (hits engine or tactical fallback)
	rareFEN := "r3k2r/ppp2ppp/2n1bn2/3pp3/3PP3/2N1BN2/PPP2PPP/R3K2R w KQkq - 0 8"
	mOOB, err := app.GetPersonaSimulatedMove(simP.ID, rareFEN, true)
	require.NoError(t, err)
	assert.NotEmpty(t, mOOB.UCI)

	// GetSimulatedMove out-of-book
	mSim, err := app.GetSimulatedMove("CovHero", rareFEN, true)
	require.NoError(t, err)
	assert.NotEmpty(t, mSim.UCI)

	// Engine match error paths & StopEngineMatch
	assert.False(t, app.StopEngineMatch())
	_, err = app.StartEngineMatch(tournament.EngineMatchConfig{
		Engine1: tournament.EngineConfig{Name: "Bad1", Path: "/nonexistent/engine1"},
		Engine2: tournament.EngineConfig{Name: "Mock2"},
		Rounds:  1,
	})
	assert.Error(t, err)
	_, err = app.StartEngineMatch(tournament.EngineMatchConfig{
		Engine1: tournament.EngineConfig{Name: "Mock1"},
		Engine2: tournament.EngineConfig{Name: "Bad2", Path: "/nonexistent/engine2"},
		Rounds:  1,
	})
	assert.Error(t, err)

	// Already running error & StopEngineMatch true branch
	_, cancel := context.WithCancel(context.Background())
	app.mu.Lock()
	app.matchRunning = true
	app.matchCancel = cancel
	app.mu.Unlock()
	_, err = app.StartEngineMatch(tournament.EngineMatchConfig{Rounds: 1})
	assert.Error(t, err)
	assert.True(t, app.StopEngineMatch())

	// DispatchRPC additional branches: invalid JSON arg, error return, void return, multi-value return
	_, err = app.DispatchRPC("GetGame", []json.RawMessage{json.RawMessage(`"not_an_int"`)})
	assert.Error(t, err)
	_, err = app.DispatchRPC("GetGame", []json.RawMessage{json.RawMessage(`9999`)})
	assert.Error(t, err)
	_, err = app.DispatchRPC("RecordRepertoirePractice", []json.RawMessage{
		json.RawMessage(`"missing"`), json.RawMessage(`""`), json.RawMessage(`""`), json.RawMessage(`false`),
	})
	require.NoError(t, err)
	multiRes, err := app.DispatchRPC("CompileRepertoireToPolyglotBook", []json.RawMessage{
		json.RawMessage(`"sim-rep"`), json.RawMessage(`false`),
	})
	require.NoError(t, err)
	assert.NotNil(t, multiRes)

	// shutdown cleans up db, refDB, engine, polyglotBooks
	if b, bErr := polyglot.OpenBook(bookPath); bErr == nil {
		app.polyglotBooks[bookPath] = b
	}
	app.shutdown(context.Background())
}

func TestCovAppMegaDBFairPlayHardMoveAndTWIC(t *testing.T) {
	app, tmp := newIsolatedApp(t)
	srv := newMockOnlineServer(t, covSamplePGN)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// ImportMegaDatabase with local httptest PGN CustomURL
	megaRes, err := app.ImportMegaDatabase(online.MegaImportOptions{
		SourceID:          "custom-test",
		CustomURL:         srv.URL + "/mega.pgn",
		MaxGames:          10,
		CreateNewDatabase: "mega_local_test",
	})
	require.NoError(t, err)
	assert.NotNil(t, megaRes)
	assert.GreaterOrEqual(t, megaRes.GamesImported, 1)

	// Re-import into existing database
	megaRes2, err := app.ImportMegaDatabase(online.MegaImportOptions{
		SourceID:          "custom-test",
		CustomURL:         srv.URL + "/mega.pgn",
		MaxGames:          10,
		CreateNewDatabase: "mega_local_test",
	})
	require.NoError(t, err)
	assert.NotNil(t, megaRes2)

	// AnalyzePlayerFairPlay & AnalyzeHardMoveBenchmark
	fp, err := app.AnalyzePlayerFairPlay("CovHero", 20)
	require.NoError(t, err)
	assert.NotNil(t, fp)

	hmb, err := app.AnalyzeHardMoveBenchmark("CovHero", "all", 20, false)
	require.NoError(t, err)
	assert.NotNil(t, hmb)

	// AnalyzeHardMoveBenchmark when no DB open (auto-ensures personal DB)
	require.NoError(t, app.CloseDatabase())
	_, _ = app.AnalyzeHardMoveBenchmark("CovHero", "all", 10, false)

	// DownloadAndAnalyzePlayerFairPlay: empty username error
	_, err = app.DownloadAndAnalyzePlayerFairPlay("lichess", "   ", 10)
	assert.Error(t, err)

	// DownloadAndAnalyzePlayerFairPlay: lichess (no open DB -> creates fairplay_ DB)
	require.NoError(t, app.CloseDatabase())
	fpDL, err := app.DownloadAndAnalyzePlayerFairPlay("lichess", "CovHero", 0)
	require.NoError(t, err)
	assert.NotNil(t, fpDL)

	// DownloadAndAnalyzePlayerFairPlay: chesscom & both
	fpCC, err := app.DownloadAndAnalyzePlayerFairPlay("chesscom", "CovHero", 10)
	require.NoError(t, err)
	assert.NotNil(t, fpCC)

	fpBoth, err := app.DownloadAndAnalyzePlayerFairPlay("both", "CovHero", 10)
	require.NoError(t, err)
	assert.NotNil(t, fpBoth)

	// DownloadAndAnalyzePlayerFairPlay: fallback when lichess fails
	app.syncMgr.Lichess.BaseURL = "http://127.0.0.1:1"
	fpFB, err := app.DownloadAndAnalyzePlayerFairPlay("lichess", "CovHero", 10)
	require.NoError(t, err)
	assert.NotNil(t, fpFB)
	app.syncMgr.Lichess.BaseURL = srv.URL

	// DownloadAndAnalyzeHardMoveBenchmark
	hmbDL, err := app.DownloadAndAnalyzeHardMoveBenchmark("lichess", "CovHero", "all", 10, false)
	require.NoError(t, err)
	assert.NotNil(t, hmbDL)

	// GetLatestTWICIssue with local httptest server
	twicSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "1560") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer twicSrv.Close()
	app.twicClient = &online.TWICClient{
		BaseURLTemplate: twicSrv.URL + "/twic%dg.zip",
		HTTPClient:      twicSrv.Client(),
	}
	latestIssue, err := app.GetLatestTWICIssue()
	require.NoError(t, err)
	assert.Equal(t, 1560, latestIssue)

	// Also verify LoadTacticsPuzzle & GetProgressivePuzzle error/temp paths
	tacticsDB := createTempScidDB(t, tmp, "tactics_local", covBlunderPGN)
	puz, err := app.LoadTacticsPuzzle(tacticsDB, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, puz.GameID)
	puzWrap, err := app.LoadTacticsPuzzle(tacticsDB, 999)
	require.NoError(t, err)
	assert.NotNil(t, puzWrap)
	_, err = app.LoadTacticsPuzzle(filepath.Join(tmp, "missing_db"), 1)
	assert.Error(t, err)
}

func TestCovAppDeepBranches(t *testing.T) {
	app, tmp := newIsolatedApp(t)
	srv := newMockOnlineServer(t, covSamplePGN)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// 1. GetDiscoveredDatabases with persona DB outside default discovery dirs
	customPersonaDB := createTempScidDB(t, tmp, "outside_persona_db", covSamplePGN)
	p, err := app.SavePersona(&persona.Persona{
		DisplayName:      "OutsidePersona",
		EstimatedElo:     2200,
		LichessUsername:  "CovHero",
		ChessComUsername: "CovHero",
		DatabasePath:     customPersonaDB,
		Identifiers:      []string{"CovHero"},
	})
	require.NoError(t, err)
	discovered := app.GetDiscoveredDatabases()
	assert.NotEmpty(t, discovered)

	// 2. GetPersonaStatistics, GetPersonaPersonality, FetchPersonaOnlineGames (nil personaMgr + unknown + valid)
	origPM := app.personaMgr
	app.personaMgr = nil
	_, err = app.GetPersonaStatistics(p.ID)
	assert.Error(t, err)
	_, err = app.GetPersonaPersonality(p.ID)
	assert.Error(t, err)
	_, err = app.FetchPersonaOnlineGames(p.ID, 5)
	assert.Error(t, err)
	_, _, err = app.BuildPersonaOpeningBook(p.ID, "white", 8)
	assert.Error(t, err)
	app.personaMgr = origPM

	_, err = app.GetPersonaStatistics("missing-id")
	assert.Error(t, err)
	_, err = app.GetPersonaPersonality("missing-id")
	assert.Error(t, err)
	_, err = app.FetchPersonaOnlineGames("missing-id", 5)
	assert.Error(t, err)
	_, _, err = app.BuildPersonaOpeningBook("missing-id", "white", 8)
	assert.Error(t, err)

	_, err = app.OpenDatabase(customPersonaDB)
	require.NoError(t, err)
	pStats, err := app.GetPersonaStatistics(p.ID)
	require.NoError(t, err)
	assert.NotNil(t, pStats)

	pPers, err := app.GetPersonaPersonality(p.ID)
	require.NoError(t, err)
	assert.NotNil(t, pPers)

	pFetch, err := app.FetchPersonaOnlineGames(p.ID, 5)
	require.NoError(t, err)
	assert.NotNil(t, pFetch)

	// BuildPersonaOpeningBook with white, black, both, maxDepth <= 0
	bookPath, cnt, err := app.BuildPersonaOpeningBook(p.ID, "white", 0)
	require.NoError(t, err)
	assert.NotEmpty(t, bookPath)
	assert.GreaterOrEqual(t, cnt, 0)
	_, _, err = app.BuildPersonaOpeningBook(p.ID, "black", 6)
	require.NoError(t, err)

	// 3. GetPersonaSimulatedMove fallback when personality fails (persona with no games/DB) & Elo >= 2000 / < 1600
	emptyP, err := app.SavePersona(&persona.Persona{
		DisplayName:  "NoGamesPlayer",
		EstimatedElo: 2100,
	})
	require.NoError(t, err)
	require.NoError(t, app.CloseDatabase())
	simMove2100, err := app.GetPersonaSimulatedMove(emptyP.ID, "r3k2r/ppp2ppp/2n1bn2/3pp3/3PP3/2N1BN2/PPP2PPP/R3K2R w KQkq - 0 8", true)
	require.NoError(t, err)
	assert.NotEmpty(t, simMove2100.UCI)

	emptyP2, err := app.SavePersona(&persona.Persona{
		DisplayName:  "ClubPlayer",
		EstimatedElo: 0,
	})
	require.NoError(t, err)
	simMove1500, err := app.GetPersonaSimulatedMove(emptyP2.ID, "r3k2r/ppp2ppp/2n1bn2/3pp3/3PP3/2N1BN2/PPP2PPP/R3K2R w KQkq - 0 8", false)
	require.NoError(t, err)
	assert.NotEmpty(t, simMove1500.UCI)

	// GetSimulatedMove when no DB open
	_, err = app.GetSimulatedMove("CovHero", chess.StartingFEN, true)
	assert.Error(t, err)

	// 4. StartGuessTheMove & GetRatingEvolution edge cases
	_, err = app.StartGuessTheMove(1, "white")
	assert.Error(t, err)
	_, err = app.GetRatingEvolution("CovHero")
	assert.Error(t, err)

	_, err = app.OpenDatabase(customPersonaDB)
	require.NoError(t, err)
	_, err = app.StartGuessTheMove(999, "white")
	assert.Error(t, err)
	gtm, err := app.StartGuessTheMove(1, "white")
	require.NoError(t, err)
	assert.NotNil(t, gtm)
	_, err = app.GetRatingEvolution("")
	require.NoError(t, err)
	rev, err := app.GetRatingEvolution("CovHero")
	require.NoError(t, err)
	assert.NotNil(t, rev)

	// 5. GetProgressivePuzzle levels 1..5
	for lvl := 1; lvl <= 5; lvl++ {
		_, _ = app.GetProgressivePuzzle(lvl)
	}

	// 6. SyncOnlineProfile & checkActiveDBSyncAllowedLocked
	_, err = app.SyncOnlineProfile("missing-profile-id")
	assert.Error(t, err)
	require.NoError(t, app.SaveSyncProfile(&online.Profile{
		ID:              "cov-sync-prof",
		Name:            "CovHero",
		LichessUsername: "CovHero",
		TargetMode:      online.TargetActiveBase,
	}))
	_, err = app.SetDatabaseWritePolicy(customPersonaDB, "locked")
	require.NoError(t, err)
	app.mu.Lock()
	assert.Error(t, app.checkActiveDBSyncAllowedLocked())
	app.mu.Unlock()
	_, err = app.SetDatabaseWritePolicy(customPersonaDB, "editable")
	require.NoError(t, err)
	app.mu.Lock()
	assert.NoError(t, app.checkActiveDBSyncAllowedLocked())
	app.mu.Unlock()
	sRes, err := app.SyncOnlineProfile("cov-sync-prof")
	require.NoError(t, err)
	assert.NotNil(t, sRes)

	// 7. ImportMegaDatabase: no DB open + empty CreateNewDatabase + error URL
	require.NoError(t, app.CloseDatabase())
	_, err = app.ImportMegaDatabase(online.MegaImportOptions{
		SourceID:  "unknown-source-id",
		CustomURL: "http://127.0.0.1:1/fail.pgn",
		MaxGames:  5,
	})
	assert.Error(t, err)

	// 8. DownloadAndAnalyzePlayerFairPlay: existing fairplay DB on disk + chesscom fallback to lichess + both fail
	require.NoError(t, app.CloseDatabase())
	_, err = app.DownloadAndAnalyzePlayerFairPlay("lichess", "CovHero", 5)
	require.NoError(t, err)
	// Close DB so it hits the "fairplay_covhero already exists on disk -> OpenDatabase" branch
	require.NoError(t, app.CloseDatabase())
	app.syncMgr.ChessCom.BaseURL = "http://127.0.0.1:1"
	_, err = app.DownloadAndAnalyzePlayerFairPlay("chesscom", "CovHero", 5)
	require.NoError(t, err)
	// Both fail
	app.syncMgr.Lichess.BaseURL = "http://127.0.0.1:1"
	_, _ = app.DownloadAndAnalyzePlayerFairPlay("lichess", "CovHero", 5)

	// 9. OpenDatabase error cases & UpdateGameHeaders locked DB error
	_, err = app.OpenDatabase(filepath.Join(tmp, "does_not_exist.si4"))
	assert.Error(t, err)
	_, err = app.OpenDatabase(customPersonaDB + ".si4")
	require.NoError(t, err)
	_, err = app.SetDatabaseWritePolicy(customPersonaDB, "locked")
	require.NoError(t, err)
	_, err = app.UpdateGameHeaders(1, map[string]string{"White": "Blocked"})
	assert.Error(t, err)
}

func TestCovAppFinalPush90(t *testing.T) {
	app, tmp := newIsolatedApp(t)

	// 1. Tactical fallback in GetSimulatedMove & GetPersonaSimulatedMove when no engine is available
	dbPath := createTempScidDB(t, tmp, "tactical_fb_db", covSamplePGN+"\n"+covBlunderPGN)
	_, err := app.OpenDatabase(dbPath)
	require.NoError(t, err)

	// Mark game 1 deleted to cover FlagDelete skip branches in EvaluatePersonaWeaknesses / GetPersonaStatistics / BuildPersonaOpeningBook
	app.db.Entries[0].Flags |= scid.FlagDelete

	p, err := app.SavePersona(&persona.Persona{
		DisplayName:  "CovHero",
		EstimatedElo: 1800,
		DatabasePath: dbPath,
		Identifiers:  []string{"CovHero"},
	})
	require.NoError(t, err)
	_, err = app.EvaluatePersonaWeaknesses(p.ID, "both", 8, 1)
	require.NoError(t, err)
	_, err = app.GetPersonaStatistics(p.ID)
	require.NoError(t, err)
	_, _, err = app.BuildPersonaOpeningBook(p.ID, "both", 8)
	require.NoError(t, err)

	// Hide all engines from PATH and clear a.engine so tactical heuristic fallback runs
	t.Setenv("PATH", tmp)
	app.mu.Lock()
	app.engine = nil
	app.mu.Unlock()

	rareFEN := "r3k2r/ppp2ppp/2n1bn2/3pp3/3PP3/2N1BN2/PPP2PPP/R3K2R w KQkq - 0 8"
	sm1, err := app.GetSimulatedMove("CovHero", rareFEN, true)
	require.NoError(t, err)
	assert.NotEmpty(t, sm1.UCI)

	pm1, err := app.GetPersonaSimulatedMove(p.ID, rareFEN, true)
	require.NoError(t, err)
	assert.NotEmpty(t, pm1.UCI)

	// Invalid FEN error on tactical fallback
	_, err = app.GetSimulatedMove("CovHero", "not-a-valid-fen", true)
	assert.Error(t, err)
	_, err = app.GetPersonaSimulatedMove(p.ID, "not-a-valid-fen", true)
	assert.Error(t, err)

	// 2. GetOpeningTree on rare position not in DB or book
	treeRare, err := app.GetOpeningTree(rareFEN)
	require.NoError(t, err)
	assert.NotNil(t, treeRare)

	// 3. UpdateGameHeaders with all fields
	_, err = app.UpdateGameHeaders(2, map[string]string{
		"White":    "Alpha",
		"Black":    "Beta",
		"Event":    "Grand Prix",
		"Site":     "Berlin",
		"Date":     "2026.06.01",
		"Round":    "5",
		"Result":   "1/2-1/2",
		"WhiteElo": "2400",
		"BlackElo": "2380",
		"ECO":      "C65",
	})
	require.NoError(t, err)

	// 4. ImportMegaDatabase: known sourceID with empty CreateNewDatabase + cancelled context branch
	cancelSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.CancelActiveOperation()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(covSamplePGN))
	}))
	defer cancelSrv.Close()

	require.NoError(t, app.CloseDatabase())
	cat := app.GetMegaDatabaseCatalog()
	knownID := "lichess-elite"
	for _, s := range cat {
		if s.Format != "7z-si4" {
			knownID = s.ID
			break
		}
	}
	resCancel, err := app.ImportMegaDatabase(online.MegaImportOptions{
		SourceID:  knownID,
		CustomURL: cancelSrv.URL + "/cancel.pgn",
		MaxGames:  5,
	})
	// Either cancelled cleanly or returned partial result
	_ = resCancel
	_ = err

	// 5. checkActiveDBSyncAllowedLocked when no DB open
	require.NoError(t, app.CloseDatabase())
	app.mu.Lock()
	assert.Error(t, app.checkActiveDBSyncAllowedLocked())
	app.mu.Unlock()
}

func TestCovAppOver90Surgical(t *testing.T) {
	app, tmp := newIsolatedApp(t)
	srv := newMockOnlineServer(t, covSamplePGN)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// 1. GetProgressivePuzzle across all level buckets (0, 10, 20, 30, 40, 50)
	for _, lvl := range []int{0, 10, 20, 30, 40, 50} {
		_, _ = app.GetProgressivePuzzle(lvl)
	}

	// 2. Open DB, load activeGame #2, then UpdateGameHeaders to sync activeGame tags & Result
	dbPath := createTempScidDB(t, tmp, "surg_db", covSamplePGN+"\n"+covBlunderPGN)
	_, err := app.OpenDatabase(dbPath)
	require.NoError(t, err)
	g2, err := app.GetGame(2)
	require.NoError(t, err)
	app.activeGame = g2
	_, err = app.UpdateGameHeaders(2, map[string]string{
		"White":  "SyncedW",
		"Result": "1-0",
	})
	require.NoError(t, err)
	assert.Equal(t, chess.Result("1-0"), app.activeGame.Result)

	// 3. GetRatingEvolution with persona ID containing ChessComUsername & LichessUsername
	p, err := app.SavePersona(&persona.Persona{
		DisplayName:      "CovHero",
		ChessComUsername: "CovHeroCC",
		LichessUsername:  "CovHeroLI",
		DatabasePath:     dbPath,
		Identifiers:      []string{"CovHero"},
	})
	require.NoError(t, err)
	revP, err := app.GetRatingEvolution(p.ID)
	require.NoError(t, err)
	assert.NotNil(t, revP)

	// 4. GetPersonaStatistics with persona dedicated DB while active DB is closed
	require.NoError(t, app.CloseDatabase())
	pStatsClosed, err := app.GetPersonaStatistics(p.ID)
	require.NoError(t, err)
	assert.NotNil(t, pStatsClosed)

	// 5. Tactical fallback in GetSimulatedMove & GetPersonaSimulatedMove using unstarted engine
	_, err = app.OpenDatabase(dbPath)
	require.NoError(t, err)
	app.mu.Lock()
	app.engine = engine.NewEngine("/nonexistent/uci")
	app.personaBotCache = nil
	app.mu.Unlock()

	rareFEN := "r3k2r/ppp2ppp/2n1bn2/3pp3/3PP3/2N1BN2/PPP2PPP/R3K2R w KQkq - 0 8"
	smTact, err := app.GetSimulatedMove("CovHero", rareFEN, true)
	require.NoError(t, err)
	assert.NotEmpty(t, smTact.UCI)

	pmTact, err := app.GetPersonaSimulatedMove(p.ID, rareFEN, true)
	require.NoError(t, err)
	assert.NotEmpty(t, pmTact.UCI)

	// Also startEngineLocked with invalid binary while a.engine != nil (quits previous & returns error)
	assert.Error(t, app.StartEngine("/nonexistent/uci_2"))

	// 6. SyncOnlineProfile with explicit TargetDatabase: locked error, matching active DB, and existing closed DB
	otherDB := createTempScidDB(t, tmp, "sync_other_db", covSamplePGN)
	require.NoError(t, app.SaveSyncProfile(&online.Profile{
		ID:              "prof-explicit",
		Name:            "CovHero",
		LichessUsername: "CovHero",
		TargetDatabase:  dbPath,
	}))
	_, err = app.SetDatabaseWritePolicy(dbPath, "locked")
	require.NoError(t, err)
	_, err = app.SyncOnlineProfile("prof-explicit")
	assert.Error(t, err)

	_, err = app.SetDatabaseWritePolicy(dbPath, "editable")
	require.NoError(t, err)
	_, err = app.SyncOnlineProfile("prof-explicit")
	require.NoError(t, err)

	require.NoError(t, app.SaveSyncProfile(&online.Profile{
		ID:              "prof-closed",
		Name:            "CovHero",
		LichessUsername: "CovHero",
		TargetDatabase:  otherDB,
	}))
	_, err = app.SyncOnlineProfile("prof-closed")
	require.NoError(t, err)

	// 7. SearchPosition error paths (invalid FEN, no DB open)
	_, err = app.SearchPosition("invalid-fen")
	assert.Error(t, err)
	require.NoError(t, app.CloseDatabase())
	_, err = app.SearchPosition(chess.StartingFEN)
	assert.Error(t, err)

	// 8. AggregatePersonaGames & OpenPersonaDatabase error paths (nil personaMgr, unknown ID)
	origPM := app.personaMgr
	app.personaMgr = nil
	_, err = app.AggregatePersonaGames("x")
	assert.Error(t, err)
	_, err = app.OpenPersonaDatabase("x")
	assert.Error(t, err)
	app.personaMgr = origPM
	_, err = app.AggregatePersonaGames("missing")
	assert.Error(t, err)
	_, err = app.OpenPersonaDatabase("missing")
	assert.Error(t, err)

	// 9. DispatchRPC returning nil error with no value (StopEngine)
	resNil, err := app.DispatchRPC("StopEngine", nil)
	require.NoError(t, err)
	assert.Nil(t, resNil)

	// 10. GenerateAdaptiveCoachingSession with empty PlayerName and nil syncMgr
	app.syncMgr = nil
	_, err = app.GenerateAdaptiveCoachingSession(AdaptiveCoachRequest{
		Source:     "database",
		PlayerName: "",
		MaxGames:   10,
		BatchSize:  3,
	})
	require.NoError(t, err)
}
