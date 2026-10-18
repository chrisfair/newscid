package online

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"newscid/pkg/database/scid"
)

func createTestZip(t *testing.T, filename, content string) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	w, err := zw.Create(filename)
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write zip content: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	return buf.Bytes()
}

const sampleTWICPGN = `[Event "World Championship Match"]
[Site "London ENG"]
[Date "2018.11.09"]
[Round "1"]
[White "Caruana, Fabiano"]
[Black "Carlsen, Magnus"]
[Result "1/2-1/2"]
[WhiteElo "2832"]
[BlackElo "2835"]
[ECO "B31"]

1. e4 c5 2. Nf3 Nc6 3. Bb5 g6 4. Bxc6 dxc6 1/2-1/2

[Event "World Championship Match"]
[Site "London ENG"]
[Date "2018.11.12"]
[Round "2"]
[White "Carlsen, Magnus"]
[Black "Caruana, Fabiano"]
[Result "1/2-1/2"]
[WhiteElo "2835"]
[BlackElo "2832"]
[ECO "D37"]

1. d4 Nf6 2. c4 e6 3. Nf3 d5 4. Nc3 Be7 1/2-1/2
`

func TestTWICClient_Basics(t *testing.T) {
	client := NewTWICClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	url := client.GetURL(1560)
	expected := "https://theweekinchess.com/zips/twic1560g.zip"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestTWICClient_DownloadAndExtract(t *testing.T) {
	zipData := createTestZip(t, "twic1560.pgn", sampleTWICPGN)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/twic1560g.zip" {
			w.Header().Set("Content-Type", "application/zip")
			w.WriteHeader(http.StatusOK)
			w.Write(zipData)
			return
		}
		if r.URL.Path == "/twic404g.zip" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &TWICClient{
		BaseURLTemplate: server.URL + "/twic%dg.zip",
		HTTPClient:      server.Client(),
		UserAgent:       "NewScidTest/1.0",
	}

	ctx := context.Background()

	// Test invalid issue
	_, err := client.DownloadIssue(ctx, 0)
	if err == nil {
		t.Error("expected error for issue <= 0")
	}

	// Test 404
	_, err = client.DownloadIssue(ctx, 404)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("404")) {
		t.Errorf("expected 404 error, got %v", err)
	}

	// Test 500
	_, err = client.DownloadIssue(ctx, 500)
	if err == nil {
		t.Error("expected error for 500 response")
	}

	// Test Successful Download & Extract
	games, err := client.DownloadIssue(ctx, 1560)
	if err != nil {
		t.Fatalf("unexpected error downloading issue 1560: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("expected 2 games, got %d", len(games))
	}
	if games[0].Tags["White"] != "Caruana, Fabiano" {
		t.Errorf("expected Caruana, Fabiano, got %s", games[0].Tags["White"])
	}
}

func TestTWICClient_ExtractErrors(t *testing.T) {
	client := NewTWICClient()

	// 1. Invalid zip bytes
	_, err := client.extractGamesFromZip([]byte("not-a-zip-file"), 100)
	if err == nil {
		t.Error("expected error for invalid zip data")
	}

	// 2. Zip with no PGN file inside
	nonPgnZip := createTestZip(t, "readme.txt", "Some text file without PGN")
	_, err = client.extractGamesFromZip(nonPgnZip, 100)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("no .pgn file found")) {
		t.Errorf("expected no pgn file error, got %v", err)
	}
}

func TestTWICClient_ImportIssue(t *testing.T) {
	zipData := createTestZip(t, "twic1000.pgn", sampleTWICPGN)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write(zipData)
	}))
	defer server.Close()

	client := &TWICClient{
		BaseURLTemplate: server.URL + "/twic%dg.zip",
		HTTPClient:      server.Client(),
		UserAgent:       "NewScidTest/1.0",
	}

	// Nil database test
	ctx := context.Background()
	_, err := client.ImportIssue(ctx, 1000, nil)
	if err == nil {
		t.Error("expected error for nil database")
	}

	// Open test db
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "twic_import_test")
	db := scid.NewDatabase()
	if err := db.Create(dbPath, "TWIC Import Test"); err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	defer db.Close()

	res, err := client.ImportIssue(ctx, 1000, db)
	if err != nil {
		t.Fatalf("unexpected error importing issue: %v", err)
	}
	if res.IssueNumber != 1000 {
		t.Errorf("expected issue 1000, got %d", res.IssueNumber)
	}
	if res.GamesDownloaded != 2 {
		t.Errorf("expected 2 games downloaded, got %d", res.GamesDownloaded)
	}
	if res.GamesImported != 2 {
		t.Errorf("expected 2 games imported, got %d", res.GamesImported)
	}
	if db.NumGames() != 2 {
		t.Errorf("expected db to have 2 games, got %d", db.NumGames())
	}
}

func TestTWICClient_ImportRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issuePGN := strings.ReplaceAll(sampleTWICPGN, `[White "Caruana, Fabiano"]`, fmt.Sprintf(`[White "Caruana, Fabiano %s"]`, r.URL.Path))
		issuePGN = strings.ReplaceAll(issuePGN, `[Black "Caruana, Fabiano"]`, fmt.Sprintf(`[Black "Caruana, Fabiano %s"]`, r.URL.Path))
		zipData := createTestZip(t, "games.pgn", issuePGN)
		w.WriteHeader(http.StatusOK)
		w.Write(zipData)
	}))
	defer server.Close()

	client := &TWICClient{
		BaseURLTemplate: server.URL + "/twic%dg.zip",
		HTTPClient:      server.Client(),
		UserAgent:       "NewScidTest/1.0",
	}

	ctx := context.Background()

	// Invalid range
	_, err := client.ImportRange(ctx, -1, 5, nil, nil)
	if err == nil {
		t.Error("expected error for invalid range")
	}

	// Closed DB
	_, err = client.ImportRange(ctx, 10, 12, nil, nil)
	if err == nil {
		t.Error("expected error for nil DB")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "twic_range_test")
	db := scid.NewDatabase()
	if err := db.Create(dbPath, "TWIC Range Test"); err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	var progressCalls int
	progressFn := func(currentIssue, processedIssues, totalIssues, gamesImported int) {
		progressCalls++
	}

	// Test reversed range 102 to 100 (should auto-swap to 100..102)
	batch, err := client.ImportRange(ctx, 102, 100, db, progressFn)
	if err != nil {
		t.Fatalf("unexpected error in ImportRange: %v", err)
	}
	if batch.StartIssue != 100 || batch.EndIssue != 102 {
		t.Errorf("expected range 100..102, got %d..%d", batch.StartIssue, batch.EndIssue)
	}
	if len(batch.Issues) != 3 {
		t.Errorf("expected 3 issues, got %d", len(batch.Issues))
	}
	if batch.TotalImported != 6 {
		t.Errorf("expected 6 games imported, got %d", batch.TotalImported)
	}
	if progressCalls != 3 {
		t.Errorf("expected 3 progress callbacks, got %d", progressCalls)
	}
}

func TestTWICClient_CheckIssueExistsAndFindLatest(t *testing.T) {
	// Let issues 1..1550 exist, issues >= 1551 return 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var issueNum int
		fmt.Sscanf(r.URL.Path, "/twic%dg.zip", &issueNum)
		if issueNum >= 1 && issueNum <= 1550 {
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock zip content"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &TWICClient{
		BaseURLTemplate: server.URL + "/twic%dg.zip",
		HTTPClient:      server.Client(),
		UserAgent:       "NewScidTest/1.0",
	}

	ctx := context.Background()

	// Check invalid issue
	_, err := client.CheckIssueExists(ctx, -5)
	if err == nil {
		t.Error("expected error for issue -5")
	}

	exists1540, err := client.CheckIssueExists(ctx, 1540)
	if err != nil || !exists1540 {
		t.Errorf("expected issue 1540 to exist, got %v, err=%v", exists1540, err)
	}

	exists1560, err := client.CheckIssueExists(ctx, 1560)
	if err != nil || exists1560 {
		t.Errorf("expected issue 1560 not to exist, got %v, err=%v", exists1560, err)
	}

	// FindLatestIssue starting near 1500
	latest, err := client.FindLatestIssue(ctx, 1500)
	if err != nil {
		t.Fatalf("unexpected error finding latest issue: %v", err)
	}
	if latest != 1550 {
		t.Errorf("expected latest issue to be 1550, got %d", latest)
	}
}

func TestTWICClient_CheckIssueExistsFallback(t *testing.T) {
	// Server returns 405 MethodNotAllowed on HEAD, but 200 on GET
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte("data"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &TWICClient{
		BaseURLTemplate: server.URL + "/twic%dg.zip",
		HTTPClient:      server.Client(),
		UserAgent:       "NewScidTest/1.0",
	}

	exists, err := client.CheckIssueExists(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("expected exists to be true with GET fallback")
	}
}
