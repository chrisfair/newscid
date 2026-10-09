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
		assert.Equal(t, "application/x-chess-pgn", r.Header.Get("Accept"))
		assert.Equal(t, client.UserAgent, r.Header.Get("User-Agent"))

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

	// 3. Successful fetch
	pgn, err := client.FetchUserGames("QuenyenHighElf", 1000, "secret_token", true, 50)
	require.NoError(t, err)
	assert.Contains(t, pgn, "QuenyenHighElf")

	// 4. Not found
	_, err = client.FetchUserGames("notfound", 0, "", false, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 5. Rate limit
	_, err = client.FetchUserGames("ratelimited", 0, "", false, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")

	// 6. Server error
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

	// 5. User not found
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

