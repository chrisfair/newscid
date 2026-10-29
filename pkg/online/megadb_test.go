package online

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

const sampleMegaPGN = `[Event "Titled Tuesday"]
[Site "Chess.com"]
[Date "2026.09.01"]
[Round "1"]
[White "Carlsen, Magnus"]
[Black "Nakamura, Hikaru"]
[Result "1-0"]
[WhiteElo "2882"]
[BlackElo "2875"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0

[Event "Club Match"]
[Site "Lichess"]
[Date "2026.09.02"]
[Round "2"]
[White "ClubPlayerA"]
[Black "ClubPlayerB"]
[Result "0-1"]
[WhiteElo "1650"]
[BlackElo "1700"]

1. d4 d5 2. c4 e6 0-1

[Event "FIDE Grand Swiss"]
[Site "Isle of Man"]
[Date "2026.09.03"]
[Round "3"]
[White "Caruana, Fabiano"]
[Black "Firouzja, Alireza"]
[Result "1/2-1/2"]
[WhiteElo "2805"]
[BlackElo "2785"]

1. c4 e5 2. Nc3 Nf6 1/2-1/2
`

func TestMegaDatabaseCatalogAndLookup(t *testing.T) {
	client := NewMegaDatabaseClient()
	cat := client.GetCatalog()
	require.NotEmpty(t, cat)

	latest := client.GetLatestCatalog()
	require.NotEmpty(t, latest)
	assert.Less(t, len(latest), len(cat))
	for _, s := range latest {
		assert.True(t, s.IsLatest || s.Year >= 2026)
	}

	src, ok := client.FindSource("lumbras-otb-2026")
	require.True(t, ok)
	assert.Equal(t, "LumbrasGigaBase", src.Provider)
	assert.Equal(t, "7z-pgn", src.Format)
	assert.True(t, src.IsLatest)
	assert.Equal(t, 2026, src.Year)

	srcLichess, ok := client.FindSource("lichess-open-2026-09")
	require.True(t, ok)
	assert.Equal(t, "Lichess Open", srcLichess.Provider)
	assert.Equal(t, "pgn.zst", srcLichess.Format)

	_, ok = client.FindSource("non-existent-id")
	assert.False(t, ok)
}

func TestMegaDatabase_ZstdStreamWithMinEloAndMaxGames(t *testing.T) {
	// Compress sampleMegaPGN using zstd
	var zstBuf bytes.Buffer
	zw, err := zstd.NewWriter(&zstBuf)
	require.NoError(t, err)
	_, err = zw.Write([]byte(sampleMegaPGN))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zstd")
		_, _ = w.Write(zstBuf.Bytes())
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "megatest")
	db := scid.NewDatabase()
	require.NoError(t, db.Create(dbPath, "Mega Test"))
	defer db.Close()

	client := NewMegaDatabaseClient()
	var lastProgress MegaImportProgress

	// Filter MinElo=2400 -> should skip the 1650/1700 club game and import the 2 GM games
	res, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		CustomURL: srv.URL + "/lichess_db_broadcast.pgn.zst",
		MinElo:    2400,
		MaxGames:  10,
	}, db, tmpDir, func(p MegaImportProgress) {
		lastProgress = p
	})
	require.NoError(t, err)
	assert.Equal(t, 3, res.GamesScanned)
	assert.Equal(t, 2, res.GamesImported)
	assert.Equal(t, "completed", lastProgress.Phase)
}

func TestMegaDatabase_ZipArchiveImport(t *testing.T) {
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	_, _ = zw.Create("readme.txt")
	fw1, err := zw.Create("part1.pgn")
	require.NoError(t, err)
	_, err = fw1.Write([]byte(sampleMegaPGN))
	require.NoError(t, err)
	fw2, err := zw.Create("part2.pgn")
	require.NoError(t, err)
	_, err = fw2.Write([]byte(sampleMegaPGN))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(zipBuf.Bytes())
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "ziptest")
	db := scid.NewDatabase()
	require.NoError(t, db.Create(dbPath, "Zip Test"))
	defer db.Close()

	client := NewMegaDatabaseClient()
	var progCount int
	res, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		CustomURL: srv.URL + "/lichess_elite_2025-11.zip",
		MaxGames:  2, // Limit to first 2 games across files
	}, db, tmpDir, func(p MegaImportProgress) {
		progCount++
	})
	require.NoError(t, err)
	assert.Equal(t, 2, res.GamesImported)
	assert.Greater(t, progCount, 0)
}

func TestMegaDatabase_DirectPGNStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleMegaPGN))
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "pgntest")
	db := scid.NewDatabase()
	require.NoError(t, db.Create(dbPath, "PGN Test"))
	defer db.Close()

	client := NewMegaDatabaseClient()
	var progCount int
	res, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		CustomURL: srv.URL + "/games.pgn",
	}, db, tmpDir, func(p MegaImportProgress) {
		progCount++
	})
	require.NoError(t, err)
	assert.Equal(t, 3, res.GamesImported)
	assert.Greater(t, progCount, 0)
}

func TestDetectFormatFromURL(t *testing.T) {
	assert.Equal(t, "pgn.zst", detectFormatFromURL("https://database.lichess.org/standard/lichess_db_standard_rated_2026-09.pgn.zst"))
	assert.Equal(t, "zip-pgn", detectFormatFromURL("https://database.nikonoel.fr/lichess_elite_2025-11.zip"))
	assert.Equal(t, "7z-si4", detectFormatFromURL("https://huggingface.co/LumbrasGigabase_Online_si4.7z"))
	assert.Equal(t, "7z-pgn", detectFormatFromURL("https://huggingface.co/LumbrasGigabase_OTB_2026.7z"))
	assert.Equal(t, "pgn", detectFormatFromURL("https://example.com/master.pgn"))

	assert.Equal(t, formatUnknown, parseArchiveFormat("bad"))
	assert.Equal(t, "", formatUnknown.String())
	assert.Equal(t, "", importPhase(99).String())
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestMegaDatabase_FetchLichessFeedAndCatalogImport(t *testing.T) {
	client := NewMegaDatabaseClient()

	// Compress sampleMegaPGN using zstd for catalog source test
	var zstBuf bytes.Buffer
	zw, err := zstd.NewWriter(&zstBuf)
	require.NoError(t, err)
	_, _ = zw.Write([]byte(sampleMegaPGN))
	_ = zw.Close()

	client.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/list.txt") {
			if strings.Contains(req.URL.Path, "standard") {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("err")),
				}, nil
			}
			body := "https://database.lichess.org/broadcast/b1.pgn.zst\nhttps://database.lichess.org/broadcast/b2.pgn.zst\nhttps://database.lichess.org/broadcast/b3.pgn.zst\n"
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			ContentLength: int64(zstBuf.Len()),
			Body:          io.NopCloser(bytes.NewReader(zstBuf.Bytes())),
		}, nil
	})

	// FetchLichessFeed with default feedType and limit
	urls, err := client.FetchLichessFeed(context.Background(), "other", 2)
	require.NoError(t, err)
	assert.Len(t, urls, 2)

	urlsDefaultLimit, err := client.FetchLichessFeed(context.Background(), "broadcast", 0)
	require.NoError(t, err)
	assert.Len(t, urlsDefaultLimit, 3)

	// FetchLichessFeed non-200 error
	_, err = client.FetchLichessFeed(context.Background(), "standard", 5)
	assert.Error(t, err)

	// DownloadAndImport using catalog SourceID ("lichess-open-2026-09" with MaxGames=0 -> defaults to 25000)
	tmpDir := t.TempDir()
	db := scid.NewDatabase()
	require.NoError(t, db.Create(filepath.Join(tmpDir, "cat_db"), "Catalog DB"))
	defer db.Close()

	res, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		SourceID: "lichess-open-2026-09",
	}, db, tmpDir, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, res.GamesImported)

	// Re-import into same DB to test existingSigs duplicate skipping in streamImportPGN
	resDup, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		SourceID: "lichess-open-2026-09",
	}, db, tmpDir, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, resDup.GamesImported)
}

func TestMegaDatabase_7zPGNAnd7zNativeSI4(t *testing.T) {
	bsdtarPath, err := exec.LookPath("bsdtar")
	require.NoError(t, err, "bsdtar is required to build test 7z fixtures")

	tmpDir := t.TempDir()

	// Build a .7z archive containing games.pgn and readme.txt
	srcPgnDir := filepath.Join(tmpDir, "src_pgn")
	require.NoError(t, os.MkdirAll(srcPgnDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcPgnDir, "games.pgn"), []byte(sampleMegaPGN), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcPgnDir, "readme.txt"), []byte("ignore"), 0644))

	sevenZipPGN := filepath.Join(tmpDir, "test_games.7z")
	cmd1 := exec.Command(bsdtarPath, "--format", "7zip", "-cf", sevenZipPGN, "-C", srcPgnDir, "games.pgn", "readme.txt")
	require.NoError(t, cmd1.Run())
	sevenZipPGNBytes, err := os.ReadFile(sevenZipPGN)
	require.NoError(t, err)

	// Build a .7z archive containing a native Scid database (.si4, .sn4, .sg4) and readme.txt
	srcSi4Dir := filepath.Join(tmpDir, "src_si4")
	require.NoError(t, os.MkdirAll(srcSi4Dir, 0755))
	srcDB := scid.NewDatabase()
	require.NoError(t, srcDB.Create(filepath.Join(srcSi4Dir, "native_mega"), "Native Mega"))
	require.NoError(t, srcDB.Close())
	require.NoError(t, os.WriteFile(filepath.Join(srcSi4Dir, "readme.txt"), []byte("ignore"), 0644))

	sevenZipSI4 := filepath.Join(tmpDir, "native_si4.7z")
	cmd2 := exec.Command(bsdtarPath, "--format", "7zip", "-cf", sevenZipSI4, "-C", srcSi4Dir, "native_mega.si4", "native_mega.sn4", "native_mega.sg4", "readme.txt")
	require.NoError(t, cmd2.Run())
	sevenZipSI4Bytes, err := os.ReadFile(sevenZipSI4)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test_games.7z":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(sevenZipPGNBytes)
		case "/native_si4.7z":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(sevenZipSI4Bytes)
		case "/bad_si4.7z":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(sevenZipPGNBytes) // no .si4 inside!
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewMegaDatabaseClient()

	// Test 7z-pgn import with MaxGames=2 and progress callback
	targetDB := scid.NewDatabase()
	require.NoError(t, targetDB.Create(filepath.Join(tmpDir, "target_7z"), "Target 7z"))
	defer targetDB.Close()

	var progressCalled bool
	resPGN, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		CustomURL: srv.URL + "/test_games.7z",
		MaxGames:  2,
	}, targetDB, tmpDir, func(p MegaImportProgress) {
		progressCalled = true
	})
	require.NoError(t, err)
	assert.Equal(t, 2, resPGN.GamesImported)
	assert.True(t, progressCalled)

	// Test 7z-si4 extraction (via bsdtar fast path)
	destFast := filepath.Join(tmpDir, "dest_fast")
	resSI4Fast, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		CustomURL: srv.URL + "/native_si4.7z",
	}, nil, destFast, func(p MegaImportProgress) {})
	require.NoError(t, err)
	assert.Contains(t, resSI4Fast.DatabasePath, "native_mega")

	// Test 7z-si4 extraction via pure-Go sevenzip reader (by temporarily clearing PATH)
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	destPure := filepath.Join(tmpDir, "dest_pure")
	resSI4Pure, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		CustomURL: srv.URL + "/native_si4.7z",
	}, nil, destPure, nil)
	require.NoError(t, err)
	assert.Contains(t, resSI4Pure.DatabasePath, "native_mega")

	// Test 7z-si4 when archive has no .si4 files inside
	_, err = client.DownloadAndImport(context.Background(), MegaImportOptions{
		CustomURL: srv.URL + "/bad_si4.7z",
	}, nil, filepath.Join(tmpDir, "dest_bad"), nil)
	assert.Error(t, err)
	t.Setenv("PATH", origPath)

	// Test extractAndImport7zPGN with cancelled context
	ctxCancel, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = extractAndImport7zPGN(ctxCancel, sevenZipPGN, targetDB, 10, 0, nil)
	assert.Error(t, err)
}

func TestMegaDatabase_ErrorsAndMinEloHelpers(t *testing.T) {
	// 1. gameMeetsMinElo edge cases
	assert.False(t, gameMeetsMinElo(nil, 2000))
	assert.True(t, gameMeetsMinElo(&chess.Game{Tags: map[string]string{"WhiteElo": "2000", "BlackElo": "2450"}}, 2400))
	assert.False(t, gameMeetsMinElo(&chess.Game{Tags: map[string]string{"WhiteElo": "2000", "BlackElo": "2100"}}, 2400))

	// Error paths in DownloadAndImport, extractAndImportZip, extractAndImport7zPGN, extract7zNativeSI4
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "404") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("corrupt archive bytes"))
	}))
	defer srv.Close()

	client := NewMegaDatabaseClient()
	client.Catalog = append(client.Catalog, MegaDatabaseSource{
		ID:     "bad-format",
		Name:   "Bad Format",
		Format: "rar",
		URL:    srv.URL + "/file.rar",
	})

	ctx := context.Background()
	tmpDir := t.TempDir()
	db := scid.NewDatabase()
	require.NoError(t, db.Create(filepath.Join(tmpDir, "err_db"), "Err DB"))
	defer db.Close()

	// Unknown source ID
	_, err := client.DownloadAndImport(ctx, MegaImportOptions{SourceID: "missing"}, db, tmpDir, nil)
	assert.Error(t, err)

	// Unsupported archive format
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{SourceID: "bad-format"}, db, tmpDir, nil)
	assert.Error(t, err)

	// Closed/nil DB errors for pgn.zst, pgn, zip-pgn, 7z-pgn
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/a.pgn.zst"}, nil, tmpDir, nil)
	assert.Error(t, err)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/a.pgn"}, nil, tmpDir, nil)
	assert.Error(t, err)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/a.zip"}, nil, tmpDir, nil)
	assert.Error(t, err)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/a.7z"}, nil, tmpDir, nil)
	assert.Error(t, err)

	// Non-200 HTTP errors for pgn.zst, pgn, and downloadToTempFile
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/404.pgn.zst"}, db, tmpDir, nil)
	assert.Error(t, err)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/404.pgn"}, db, tmpDir, nil)
	assert.Error(t, err)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/404.zip"}, db, tmpDir, nil)
	assert.Error(t, err)

	// Corrupt zip and 7z archive errors
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/corrupt.zip"}, db, tmpDir, nil)
	assert.Error(t, err)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/corrupt.7z"}, db, tmpDir, nil)
	assert.Error(t, err)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/corrupt_si4.7z"}, nil, tmpDir, nil)
	assert.Error(t, err)

	// Cancelled context in streamImportPGN and downloadToTempFile
	ctxCancel, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err = streamImportPGN(ctxCancel, strings.NewReader(sampleMegaPGN), db, 10, 0, nil)
	assert.Error(t, err)
	_, _, err = client.downloadToTempFile(ctxCancel, srv.URL+"/corrupt.zip", nil)
	assert.Error(t, err)

	// Mid-stream read error and context cancellation inside downloadToTempFile loop
	ctxMid, cancelMid := context.WithCancel(ctx)
	errClient := NewMegaDatabaseClient()
	callIdx := 0
	errClient.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		callIdx++
		if callIdx == 1 {
			// Return 1 byte and cancel context so next loop iteration hits <-ctx.Done()
			return &http.Response{
				StatusCode:    http.StatusOK,
				ContentLength: 100,
				Body: io.NopCloser(&funcReader{fn: func(p []byte) (int, error) {
					p[0] = 'x'
					cancelMid()
					return 1, nil
				}}),
			}, nil
		}
		if callIdx == 2 {
			// Return read error
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(&funcReader{fn: func(p []byte) (int, error) {
					return 0, io.ErrUnexpectedEOF
				}}),
			}, nil
		}
		return nil, io.ErrUnexpectedEOF
	})
	_, _, err = errClient.downloadToTempFile(ctxMid, "https://example.com/a.zip", func(dl, total int64) {})
	assert.Error(t, err)
	_, _, err = errClient.downloadToTempFile(ctx, "https://example.com/b.zip", nil)
	assert.Error(t, err)

	// Transport errors for FetchLichessFeed, pgn.zst, pgn, and downloadToTempFile
	_, err = errClient.FetchLichessFeed(ctx, "broadcast", 5)
	assert.Error(t, err)
	_, err = errClient.DownloadAndImport(ctx, MegaImportOptions{CustomURL: "https://example.com/a.pgn.zst"}, db, tmpDir, nil)
	assert.Error(t, err)
	_, err = errClient.DownloadAndImport(ctx, MegaImportOptions{CustomURL: "https://example.com/a.pgn"}, db, tmpDir, nil)
	assert.Error(t, err)
	_, err = errClient.DownloadAndImport(ctx, MegaImportOptions{CustomURL: "https://example.com/a.zip"}, db, tmpDir, nil)
	assert.Error(t, err)

	// Invalid URLs (http.NewRequestWithContext error)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: "://bad.pgn.zst"}, db, tmpDir, nil)
	assert.Error(t, err)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: "://bad.pgn"}, db, tmpDir, nil)
	assert.Error(t, err)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: "://bad.zip"}, db, tmpDir, nil)
	assert.Error(t, err)

	// extract7zNativeSI4 error when destDir is under a regular file
	regularFile := filepath.Join(tmpDir, "regfile")
	require.NoError(t, os.WriteFile(regularFile, []byte("x"), 0644))
	_, err = extract7zNativeSI4("dummy.7z", filepath.Join(regularFile, "sub"))
	assert.Error(t, err)

	// format7zSI4 with empty destDir (uses HOME env var)
	t.Setenv("HOME", tmpDir)
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/corrupt_si4.7z"}, nil, "", nil)
	assert.Error(t, err)

	// streamImportPGN with 500 games:
	// 1) minElo=2000 where all 500 have WhiteElo 1500 -> triggers scanned%500 == 0 progress callback
	// 2) minElo=0 -> triggers len(batch) >= 250 flushBatch
	var bulkPGN strings.Builder
	for i := 1; i <= 500; i++ {
		bulkPGN.WriteString(fmt.Sprintf("[Event \"Bulk %d\"]\n[Site \"S%d\"]\n[Date \"2026.01.01\"]\n[Round \"%d\"]\n[White \"W%d\"]\n[Black \"B%d\"]\n[WhiteElo \"1500\"]\n[Result \"1-0\"]\n\n1. e4 e5 1-0\n\n", i, i, i, i, i))
	}
	var scanProgCalled bool
	sc500, imp500, err := streamImportPGN(ctx, strings.NewReader(bulkPGN.String()), db, 0, 2000, func(sc, imp int) {
		scanProgCalled = true
	})
	require.NoError(t, err)
	assert.Equal(t, 500, sc500)
	assert.Equal(t, 0, imp500)
	assert.True(t, scanProgCalled)

	scBulk, impBulk, err := streamImportPGN(ctx, strings.NewReader(bulkPGN.String()), db, 0, 0, func(sc, imp int) {})
	require.NoError(t, err)
	assert.Equal(t, 500, scBulk)
	assert.Equal(t, 500, impBulk)

	// Corrupt zstd stream in DownloadAndImport (formatPGNZst) and read error in formatPGN
	_, err = client.DownloadAndImport(ctx, MegaImportOptions{CustomURL: srv.URL + "/corrupt.pgn.zst"}, db, tmpDir, nil)
	assert.Error(t, err)

	pgnErrClient := NewMegaDatabaseClient()
	pgnErrClient.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(&funcReader{fn: func(p []byte) (int, error) {
				return 0, io.ErrUnexpectedEOF
			}}),
		}, nil
	})
	_, err = pgnErrClient.DownloadAndImport(ctx, MegaImportOptions{CustomURL: "https://example.com/err.pgn"}, db, tmpDir, nil)
	assert.Error(t, err)

	// streamImportPGN where pgnMeetsMinElo is true via spoofed/malformed tag but gameMeetsMinElo is false
	spoofPGN := "[Event \"Spoof\"]\n[WhiteElo \"2600\"]\n[WhiteElo \"1500\"]\n[BlackElo \"1500\"]\n\n1. e4 e5 1-0\n"
	_, impSpoof, err := streamImportPGN(ctx, strings.NewReader(spoofPGN), db, 0, 2400, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, impSpoof)
}

type funcReader struct {
	fn func(p []byte) (int, error)
}

func (r *funcReader) Read(p []byte) (int, error) {
	return r.fn(p)
}


