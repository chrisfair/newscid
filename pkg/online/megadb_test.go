package online

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	src, ok := client.FindSource("lumbras-otb-2026")
	require.True(t, ok)
	assert.Equal(t, "LumbrasGigaBase", src.Provider)
	assert.Equal(t, "7z-pgn", src.Format)

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
	fw, err := zw.Create("lichess_elite_2025-11.pgn")
	require.NoError(t, err)
	_, err = fw.Write([]byte(sampleMegaPGN))
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
	res, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		CustomURL: srv.URL + "/lichess_elite_2025-11.zip",
		MaxGames:  2, // Limit to first 2 games
	}, db, tmpDir, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, res.GamesImported)
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
	res, err := client.DownloadAndImport(context.Background(), MegaImportOptions{
		CustomURL: srv.URL + "/games.pgn",
	}, db, tmpDir, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, res.GamesImported)
}

func TestDetectFormatFromURL(t *testing.T) {
	assert.Equal(t, "pgn.zst", detectFormatFromURL("https://database.lichess.org/standard/lichess_db_standard_rated_2026-09.pgn.zst"))
	assert.Equal(t, "zip-pgn", detectFormatFromURL("https://database.nikonoel.fr/lichess_elite_2025-11.zip"))
	assert.Equal(t, "7z-si4", detectFormatFromURL("https://huggingface.co/LumbrasGigabase_Online_si4.7z"))
	assert.Equal(t, "7z-pgn", detectFormatFromURL("https://huggingface.co/LumbrasGigabase_OTB_2026.7z"))
	assert.Equal(t, "pgn", detectFormatFromURL("https://example.com/master.pgn"))
}
