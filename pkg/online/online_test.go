package online

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/database/scid"
)

func TestProfileManager(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "profiles.json")

	// 1. Initial creation with non-existent file
	pm, err := NewProfileManager(configPath)
	require.NoError(t, err)
	assert.Empty(t, pm.GetProfiles())

	// 2. Default config path
	assert.NotEmpty(t, DefaultConfigPath())

	// 3. Save a new profile
	p1 := &Profile{
		Name:             "Main Profile",
		ChessComUsername: "west17",
		LichessUsername:  "QuenyenHighElf",
		RatedOnly:        true,
		AutoDeduplicate:  true,
	}
	err = pm.SaveProfile(p1)
	require.NoError(t, err)
	assert.NotEmpty(t, p1.ID)
	assert.Greater(t, p1.CreatedAt, int64(0))

	// 4. Get profile
	got, err := pm.GetProfile(p1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Main Profile", got.Name)
	assert.Equal(t, "west17", got.ChessComUsername)

	// 5. Get non-existent
	_, err = pm.GetProfile("nonexistent_id")
	assert.Error(t, err)

	// 6. Reload from disk
	pm2, err := NewProfileManager(configPath)
	require.NoError(t, err)
	profiles := pm2.GetProfiles()
	require.Len(t, profiles, 1)
	assert.Equal(t, "Main Profile", profiles[0].Name)

	// 7. Delete profile
	err = pm2.DeleteProfile(p1.ID)
	require.NoError(t, err)
	assert.Empty(t, pm2.GetProfiles())

	// 8. Delete non-existent
	err = pm2.DeleteProfile("already_deleted")
	assert.Error(t, err)

	// 9. Corrupted config file error
	corruptPath := filepath.Join(tmpDir, "corrupted.json")
	err = os.WriteFile(corruptPath, []byte("invalid json {} []"), 0644)
	require.NoError(t, err)
	_, err = NewProfileManager(corruptPath)
	assert.Error(t, err)
}

func TestLichessClient(t *testing.T) {
	client := NewLichessClient()
	assert.NotEmpty(t, client.UserAgent)

	// 1. Empty username error
	_, err := client.FetchUserGames("", 0, "", false, 0)
	assert.Error(t, err)

	// 2. Mock Lichess server
	samplePGN := `[Event "Rated Blitz game"]
[Site "https://lichess.org/abcd1234"]
[Date "2026.05.01"]
[White "QuenyenHighElf"]
[Black "Opponent"]
[Result "1-0"]
[WhiteElo "2150"]
[BlackElo "2100"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "notfound") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.Contains(r.URL.Path, "ratelimited") {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if strings.Contains(r.URL.Path, "servererror") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
			return
		}

		if strings.HasPrefix(r.URL.Path, "/api/user/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"count":{"all":14182,"rated":9977}}`))
			return
		}

		assert.Equal(t, "application/x-chess-pgn", r.Header.Get("Accept"))
		assert.Equal(t, client.UserAgent, r.Header.Get("User-Agent"))

		// Verify query params
		query := r.URL.Query()
		assert.Equal(t, "true", query.Get("clocks"))
		assert.Equal(t, "true", query.Get("evals"))
		assert.Equal(t, "true", query.Get("opening"))
		if query.Get("since") != "" {
			assert.Equal(t, "1000", query.Get("since"))
		}
		if query.Get("max") != "" {
			assert.Equal(t, "50", query.Get("max"))
		}

		if r.Header.Get("Authorization") != "" {
			assert.Equal(t, "Bearer secret_token", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/x-chess-pgn")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(samplePGN))
	}))
	defer server.Close()

	client.BaseURL = server.URL

	// 3. User game count
	assert.Equal(t, 0, client.GetUserGameCount("", false))
	assert.Equal(t, 0, client.GetUserGameCount("notfound", false))
	assert.Equal(t, 14182, client.GetUserGameCount("QuenyenHighElf", false))
	assert.Equal(t, 9977, client.GetUserGameCount("QuenyenHighElf", true))

	// 4. Successful fetch with progress (constrained)
	var progressEvents []string
	progressCallback := func(stage string, current, total int, message string) {
		progressEvents = append(progressEvents, stage)
		assert.NotEmpty(t, message)
	}

	pgn, err := client.FetchUserGames("QuenyenHighElf", 1000, "secret_token", true, 50, progressCallback)
	require.NoError(t, err)
	assert.Contains(t, pgn, "QuenyenHighElf")
	assert.NotEmpty(t, progressEvents)

	// 4b. Successful fetch with progress (unconstrained maxGames = 0)
	pgnAll, err := client.FetchUserGames("QuenyenHighElf", 0, "", false, 0, progressCallback)
	require.NoError(t, err)
	assert.Contains(t, pgnAll, "QuenyenHighElf")

	// 5. Not found
	_, err = client.FetchUserGames("notfound", 0, "", false, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 6. Rate limit
	_, err = client.FetchUserGames("ratelimited", 0, "", false, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")

	// 7. Server error
	_, err = client.FetchUserGames("servererror", 0, "", false, 0)
	assert.Error(t, err)
}

func TestChessComClient(t *testing.T) {
	client := NewChessComClient()
	assert.NotEmpty(t, client.UserAgent)

	// 1. Errors on empty parameters
	_, err := client.FetchArchives("")
	assert.Error(t, err)
	_, err = client.FetchMonthlyPGN("")
	assert.Error(t, err)

	sampleMonthPGN := `[Event "Live Chess"]
[Site "Chess.com"]
[Date "2026.04.15"]
[White "west17"]
[Black "Grandmaster"]
[Result "1-0"]
[WhiteElo "2200"]
[BlackElo "2180"]

1. d4 Nf6 2. c4 e6 3. Nf3 d5 1-0`

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, client.UserAgent, r.Header.Get("User-Agent"))

		if strings.Contains(r.URL.Path, "notfound") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.Contains(r.URL.Path, "ratelimited") {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/archives") {
			w.Header().Set("Content-Type", "application/json")
			resp := ChessComArchivesResponse{
				Archives: []string{
					serverURL + "/pub/player/west17/games/2026/01",
					serverURL + "/pub/player/west17/games/2026/04",
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/pgn") {
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(sampleMonthPGN))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	serverURL = server.URL
	client.BaseURL = server.URL

	// 2. Fetch Archives
	archives, err := client.FetchArchives("west17")
	require.NoError(t, err)
	assert.Len(t, archives, 2)

	// 3. Fetch Monthly PGN
	monthPGN, err := client.FetchMonthlyPGN(archives[0])
	require.NoError(t, err)
	assert.Contains(t, monthPGN, "west17")

	// 4. FetchUserGames with filtering (since 2026/03 -> only 2026/04 included)
	filteredPGN, err := client.FetchUserGames("west17", 2026, 3)
	require.NoError(t, err)
	assert.Contains(t, filteredPGN, "west17")

	// 5. FetchUserGames with maxLimit (reverse chronological processing)
	limitedPGN, err := client.FetchUserGames("west17", 0, 0, 1)
	require.NoError(t, err)
	assert.Contains(t, limitedPGN, "west17")

	// 6. FetchUserGamesWithProgress with callback
	var chessProgressCalled bool
	chessPGN, err := client.FetchUserGamesWithProgress("west17", 0, 0, 1, func(stage string, current, total int, message string) {
		chessProgressCalled = true
		assert.NotEmpty(t, message)
	})
	require.NoError(t, err)
	assert.Contains(t, chessPGN, "west17")
	assert.True(t, chessProgressCalled)

	// 7. User not found
	_, err = client.FetchArchives("notfound")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 6. Rate limited
	_, err = client.FetchArchives("ratelimited")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")
}

func TestSyncManager(t *testing.T) {
	// Setup ProfileManager
	tmpDir := t.TempDir()
	pm, err := NewProfileManager(filepath.Join(tmpDir, "profiles.json"))
	require.NoError(t, err)

	sm := NewSyncManager(pm)

	// 1. Errors on nil profile or closed database
	_, err = sm.SyncProfile(nil, nil)
	assert.Error(t, err)

	closedDb := scid.NewDatabase()
	profile := &Profile{
		Name:             "Sync Test",
		ChessComUsername: "user1",
		LichessUsername:  "user2",
		AutoDeduplicate:  true,
	}
	_, err = sm.SyncProfile(profile, closedDb)
	assert.Error(t, err)

	// 2. Setup mock servers for Lichess and Chess.com
	lichessPGN := `[Event "Lichess Game"]
[Site "https://lichess.org/game1"]
[Date "2026.06.01"]
[White "QuenyenHighElf"]
[Black "Carlsen"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 1-0`

	chessComPGN := `[Event "Chess.com Game"]
[Site "Chess.com"]
[Date "2026.06.02"]
[White "west17"]
[Black "Nakamura"]
[Result "1-0"]

1. d4 d5 2. c4 c6 1-0`

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "lichess") || strings.Contains(r.URL.Path, "/api/games/user") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(lichessPGN))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/archives") {
			resp := ChessComArchivesResponse{
				Archives: []string{serverURL + "/pub/player/west17/games/2026/06"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/pgn") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(chessComPGN))
			return
		}
	}))
	defer server.Close()
	serverURL = server.URL

	sm.Lichess.BaseURL = server.URL
	sm.ChessCom.BaseURL = server.URL

	// 3. Create target SCID database
	targetPath := filepath.Join(tmpDir, "target_db")
	db := scid.NewDatabase()
	err = db.Create(targetPath, "Target Sync DB")
	require.NoError(t, err)
	defer db.Close()

	// 4. Save and sync profile
	err = pm.SaveProfile(profile)
	require.NoError(t, err)

	res, err := sm.SyncProfile(profile, db)
	require.NoError(t, err)
	assert.Equal(t, 2, res.TotalDownloaded)
	assert.Equal(t, 1, res.LichessGames)
	assert.Equal(t, 1, res.ChessComGames)
	assert.Equal(t, 2, res.Imported)
	assert.Equal(t, 0, res.DuplicatesSkipped)
	assert.Equal(t, 2, db.NumGames())

	// Profile sync metrics should be updated
	updatedProfile, err := pm.GetProfile(profile.ID)
	require.NoError(t, err)
	assert.Greater(t, updatedProfile.LastSyncTime, int64(0))
	assert.Equal(t, 2, updatedProfile.LastSyncGames)

	// 5. Test Auto-deduplication on subsequent sync (older Chess.com archive filtered by LastSyncTime)
	res2, err := sm.SyncProfile(profile, db)
	require.NoError(t, err)
	assert.Equal(t, 1, res2.TotalDownloaded)
	assert.Equal(t, 1, res2.DuplicatesSkipped)
}

func TestOnlineEdgeCasesAndErrors(t *testing.T) {
	// 1. NewProfileManager with empty config path uses DefaultConfigPath
	pmDefault, err := NewProfileManager("")
	require.NoError(t, err)
	assert.NotNil(t, pmDefault)

	// 2. LichessClient invalid URL request failure
	badLichess := NewLichessClient()
	badLichess.BaseURL = "http://invalid-nonexistent-domain-12345.xyz"
	_, err = badLichess.FetchUserGames("user", 0, "", false, 0)
	assert.Error(t, err)

	// 3. ChessComClient error cases
	badChessCom := NewChessComClient()
	badChessCom.BaseURL = "http://invalid-nonexistent-domain-12345.xyz"
	_, err = badChessCom.FetchArchives("user")
	assert.Error(t, err)
	_, err = badChessCom.FetchMonthlyPGN("http://invalid-nonexistent-domain-12345.xyz/pgn")
	assert.Error(t, err)

	// 4. ChessComClient server with 404 monthly PGN, 500 monthly PGN, 500 archives, and invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "empty-month") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.Contains(r.URL.Path, "error-month") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("month error"))
			return
		}
		if strings.Contains(r.URL.Path, "invalid-json") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not valid json"))
			return
		}
		if strings.Contains(r.URL.Path, "server-err") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal err"))
			return
		}
		if strings.Contains(r.URL.Path, "empty-archives") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"archives": []}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewChessComClient()
	client.BaseURL = server.URL

	// 4a. Empty month (404) returns empty string and no error
	emptyPGN, err := client.FetchMonthlyPGN(server.URL + "/empty-month")
	require.NoError(t, err)
	assert.Empty(t, emptyPGN)

	// 4b. Error month (500) returns error
	_, err = client.FetchMonthlyPGN(server.URL + "/error-month")
	assert.Error(t, err)

	// 4c. Server error on archives (500)
	_, err = client.FetchArchives("server-err")
	assert.Error(t, err)

	// 4d. Invalid JSON on archives
	_, err = client.FetchArchives("invalid-json")
	assert.Error(t, err)

	// 4e. Empty archives list
	pgnZero, err := client.FetchUserGames("empty-archives", 0, 0)
	require.NoError(t, err)
	assert.Empty(t, pgnZero)

	// 5. SyncManager with 0 games downloaded (empty usernames)
	tmpDir := t.TempDir()
	pm, err := NewProfileManager(filepath.Join(tmpDir, "pm.json"))
	require.NoError(t, err)
	sm := NewSyncManager(pm)

	db := scid.NewDatabase()
	err = db.Create(filepath.Join(tmpDir, "empty_sync_db"), "Empty")
	require.NoError(t, err)
	defer db.Close()

	emptyProfile := &Profile{
		Name: "Empty Usernames",
	}
	res, err := sm.SyncProfile(emptyProfile, db)
	require.NoError(t, err)
	assert.Equal(t, 0, res.TotalDownloaded)

	// 6. SyncManager with errors from both platforms
	failingProfile := &Profile{
		Name:             "Failing Profile",
		ChessComUsername: "server-err",
		LichessUsername:  "notfound",
	}
	sm.ChessCom.BaseURL = server.URL
	sm.Lichess.BaseURL = server.URL + "/notfound"

	resFail, err := sm.SyncProfile(failingProfile, db)
	require.NoError(t, err)
	assert.NotEmpty(t, resFail.Errors)
	assert.Equal(t, 0, resFail.TotalDownloaded)
}

func TestImportPlayerGamesAndBlendedSync(t *testing.T) {
	// Setup mock server for both Lichess and Chess.com
	sampleLichessPGN := `[Event "Lichess Classical"]
[Site "https://lichess.org/12345"]
[Date "2026.01.01"]
[Round "1"]
[White "OpponentGM"]
[Black "HeroPlayer"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0`

	sampleChessComPGN := `[Event "Chess.com Titled Tuesday"]
[Site "Chess.com"]
[Date "2026.02.01"]
[Round "1"]
[White "OpponentGM"]
[Black "RivalMaster"]
[Result "0-1"]

1. d4 Nf6 2. c4 e6 3. Nc3 Bb4 0-1`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/api/games/user/") {
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(sampleLichessPGN))
			return
		}
		if strings.Contains(path, "/games/archives") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			archives := map[string][]string{
				"archives": {"http://" + r.Host + "/pub/player/opponentgm/games/2026/02"},
			}
			_ = json.NewEncoder(w).Encode(archives)
			return
		}
		if strings.Contains(path, "/games/2026/02") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			gamesObj := map[string][]map[string]string{
				"games": {{"pgn": sampleChessComPGN}},
			}
			_ = json.NewEncoder(w).Encode(gamesObj)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	pm, err := NewProfileManager(filepath.Join(tmpDir, "pm.json"))
	require.NoError(t, err)

	sm := NewSyncManager(pm)
	sm.Lichess.BaseURL = server.URL
	sm.ChessCom.BaseURL = server.URL + "/pub/player"

	// 1. Validation error: no usernames
	_, err = sm.ImportPlayerGames(ImportPlayerOptions{}, nil)
	assert.Error(t, err)

	// 2. Active Base Mode
	activeDB := scid.NewDatabase()
	activePath := filepath.Join(tmpDir, "active_base")
	err = activeDB.Create(activePath, "Active Base")
	require.NoError(t, err)
	defer activeDB.Close()

	var reportedImportStages []string
	progressFunc := func(stage string, current, total int, message string) {
		reportedImportStages = append(reportedImportStages, stage)
	}

	resActive, err := sm.ImportPlayerGames(ImportPlayerOptions{
		PlayerName:       "Opponent GM",
		LichessUsername:  "opponentgm",
		ChessComUsername: "opponentgm",
		TargetMode:       TargetActiveBase,
		AutoDeduplicate:  true,
		ProgressFn:       progressFunc,
	}, activeDB)
	require.NoError(t, err)
	assert.Equal(t, 2, resActive.TotalDownloaded)
	assert.Equal(t, 2, resActive.Imported)
	assert.Equal(t, uint32(2), activeDB.Header.NumGames)
	assert.NotEmpty(t, reportedImportStages)

	// 2b. Error: TargetActiveBase with nil active database
	_, err = sm.ImportPlayerGames(ImportPlayerOptions{
		LichessUsername: "opponentgm",
		TargetMode:      TargetActiveBase,
	}, nil)
	assert.Error(t, err)

	// 3. Dedicated Base Mode
	dedicatedPath := filepath.Join(tmpDir, "dedicated_opponent.si4")
	resDedicated, err := sm.ImportPlayerGames(ImportPlayerOptions{
		PlayerName:       "Opponent GM",
		LichessUsername:  "opponentgm",
		ChessComUsername: "opponentgm",
		TargetMode:       TargetDedicatedBase,
		DedicatedPath:    dedicatedPath,
		AutoDeduplicate:  true,
		CreateProfile:    true,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, resDedicated.Imported)

	// Verify profile created
	profs := pm.GetProfiles()
	require.Len(t, profs, 1)
	assert.True(t, profs[0].IsStudyTarget)
	assert.Equal(t, TargetDedicatedBase, profs[0].TargetMode)

	// Verify dedicated database has 2 games
	dedDB := scid.NewDatabase()
	err = dedDB.Open(filepath.Join(tmpDir, "dedicated_opponent"))
	require.NoError(t, err)
	assert.Equal(t, uint32(2), dedDB.Header.NumGames)
	dedDB.Close()

	// 3b. Error: Dedicated path missing
	_, err = sm.ImportPlayerGames(ImportPlayerOptions{
		LichessUsername: "opponentgm",
		TargetMode:      TargetDedicatedBase,
		DedicatedPath:   "",
	}, nil)
	assert.Error(t, err)

	// 4. Blended Base Mode (Dedicated DB + Blended Master DB)
	blendedDedicatedPath := filepath.Join(tmpDir, "hikaru_dedicated.si4")
	blendedMasterPath := filepath.Join(tmpDir, "master_blended.si4")

	resBlended, err := sm.ImportPlayerGames(ImportPlayerOptions{
		PlayerName:       "Hikaru Nakamura",
		LichessUsername:  "opponentgm",
		ChessComUsername: "opponentgm",
		TargetMode:       TargetBlendedBase,
		DedicatedPath:    blendedDedicatedPath,
		BlendedPath:      blendedMasterPath,
		AutoDeduplicate:  true,
		CreateProfile:    true,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, resBlended.Imported) // 2 in dedicated + 2 in blended

	// Verify dedicated base has 2 games
	hDB := scid.NewDatabase()
	err = hDB.Open(filepath.Join(tmpDir, "hikaru_dedicated"))
	require.NoError(t, err)
	assert.Equal(t, uint32(2), hDB.Header.NumGames)
	hDB.Close()

	// Verify blended master base has 2 games
	mDB := scid.NewDatabase()
	err = mDB.Open(filepath.Join(tmpDir, "master_blended"))
	require.NoError(t, err)
	assert.Equal(t, uint32(2), mDB.Header.NumGames)
	mDB.Close()

	// 5. Blended mode with active database fallback
	resBlendedActive, err := sm.ImportPlayerGames(ImportPlayerOptions{
		PlayerName:       "Magnus",
		LichessUsername:  "opponentgm",
		TargetMode:       TargetBlendedBase,
		DedicatedPath:    filepath.Join(tmpDir, "magnus_dedicated.si4"),
		BlendedPath:      "", // empty -> should blend into activeDB!
	}, activeDB)
	require.NoError(t, err)
	assert.Greater(t, resBlendedActive.Imported, 0)

	// 6. Test SyncProfile with TargetBlendedBase
	blendedProf := &Profile{
		Name:             "Blended Sync Test",
		LichessUsername:  "opponentgm",
		TargetMode:       TargetBlendedBase,
		BlendedDatabase:  blendedMasterPath,
		AutoDeduplicate:  true,
	}
	resSyncBlended, err := sm.SyncProfile(blendedProf, activeDB)
	require.NoError(t, err)
	assert.Greater(t, resSyncBlended.Imported, 0)

	// 7. openOrCreateDatabase errors
	_, err = openOrCreateDatabase("", "empty")
	assert.Error(t, err)
}

func TestImportPlayerGamesAdditionalEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	pm, err := NewProfileManager(filepath.Join(tmpDir, "pm2.json"))
	require.NoError(t, err)

	sm := NewSyncManager(pm)
	// Mock server that returns empty string
	emptyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(""))
	}))
	defer emptyServer.Close()

	sm.Lichess.BaseURL = emptyServer.URL
	sm.ChessCom.BaseURL = emptyServer.URL

	// 1. Zero games downloaded
	resZero, err := sm.ImportPlayerGames(ImportPlayerOptions{
		LichessUsername: "nobody",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, resZero.TotalDownloaded)

	// 2. TargetBlendedBase with invalid paths to trigger error branches
	samplePGN := `[Event "Test"]
[White "P1"]
[Black "P2"]
[Result "1-0"]

1. e4 1-0`

	pgnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-chess-pgn")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(samplePGN))
	}))
	defer pgnServer.Close()

	sm.Lichess.BaseURL = pgnServer.URL
	resErr, err := sm.ImportPlayerGames(ImportPlayerOptions{
		LichessUsername: "hasgames",
		TargetMode:      TargetBlendedBase,
		DedicatedPath:   "", // empty dedicated path triggers error recording
		BlendedPath:     "/invalid/nonexistent/dir/file.si4", // invalid blended path
	}, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, resErr.Errors)
}



