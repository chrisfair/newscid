package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestLivePortalAutoCaptureLichessAndChessCom(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, ".local", "share"))

	const lichessGamePGN = `[Event "Rated Blitz game"]
[Site "https://lichess.org/AbCdEfGh"]
[Date "2026.10.20"]
[White "PortalHero"]
[Black "Challenger1"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0
`

	const chessComGamePGN = `[Event "Live Chess"]
[Site "Chess.com"]
[Date "2026.10.20"]
[White "GrandOpponent"]
[Black "PortalHero"]
[Result "0-1"]
[Link "https://www.chess.com/game/live/987654321"]

1. d4 Nf6 2. c4 g6 3. Nc3 Bg7 0-1
`

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/games/user/"):
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			_, _ = w.Write([]byte(lichessGamePGN))
		case strings.HasPrefix(r.URL.Path, "/game/export/AbCdEfGh"):
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			_, _ = w.Write([]byte(lichessGamePGN))
		case strings.HasSuffix(r.URL.Path, "/games/archives"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"archives":["%s/pub/player/portalhero/games/2026/10"]}`, srv.URL)
		case strings.HasSuffix(r.URL.Path, "/games/2026/10/pgn"):
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			_, _ = w.Write([]byte(chessComGamePGN))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	app := NewApp()
	defer func() {
		_, _ = app.StopLivePortalWatcher()
	}()
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// Configure auto-capture usernames
	status, err := app.ConfigureLivePortalAutoCapture("PortalHero", "PortalHero", false, true)
	if err != nil {
		t.Fatalf("ConfigureLivePortalAutoCapture failed: %v", err)
	}
	if status.LichessUsername != "PortalHero" || status.ChessComUsername != "PortalHero" {
		t.Fatalf("expected PortalHero usernames, got %+v", status)
	}

	// Run CheckAndCaptureLivePortalGames — should automatically create My_Games and import 2 games
	res1, err := app.CheckAndCaptureLivePortalGames()
	if err != nil {
		t.Fatalf("CheckAndCaptureLivePortalGames failed: %v", err)
	}
	if res1.NewGamesImported != 2 {
		t.Fatalf("expected 2 new games imported (1 Lichess + 1 Chess.com), got %d", res1.NewGamesImported)
	}
	if res1.LichessImported != 1 || res1.ChessComImported != 1 {
		t.Fatalf("expected 1 Lichess and 1 Chess.com game, got %+v", res1)
	}
	if res1.LatestGameID != 2 {
		t.Fatalf("expected LatestGameID=2, got %d", res1.LatestGameID)
	}

	// Second poll with identical server games should skip both duplicates (0 new games)
	res2, err := app.CheckAndCaptureLivePortalGames()
	if err != nil {
		t.Fatalf("second CheckAndCaptureLivePortalGames failed: %v", err)
	}
	if res2.NewGamesImported != 0 {
		t.Fatalf("expected 0 new games on duplicate poll, got %d", res2.NewGamesImported)
	}
	if res2.DuplicatesSkipped != 2 {
		t.Fatalf("expected 2 duplicates skipped, got %d", res2.DuplicatesSkipped)
	}

	// Test direct URL capture for a Lichess game URL and Chess.com URL
	resURL, err := app.CaptureOnlineGameFromURLOrPGN("https://lichess.org/AbCdEfGh/black")
	if err != nil {
		t.Fatalf("CaptureOnlineGameFromURLOrPGN(lichess) failed: %v", err)
	}
	if resURL.LatestGamePGN == "" {
		t.Fatalf("expected LatestGamePGN to be populated even when already in DB")
	}
}
