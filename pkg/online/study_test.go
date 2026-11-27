package online

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/database/scid"
)

func TestParseLichessStudyID(t *testing.T) {
	id, err := ParseLichessStudyID("https://lichess.org/study/AbCd1234/XyZ98765?foo=1")
	require.NoError(t, err)
	assert.Equal(t, "AbCd1234", id)

	id, err = ParseLichessStudyID("AbCd1234")
	require.NoError(t, err)
	assert.Equal(t, "AbCd1234", id)

	_, err = ParseLichessStudyID("")
	assert.Error(t, err)

	_, err = ParseLichessStudyID("not-a-valid-study")
	assert.Error(t, err)
}

func TestFetchLichessStudyAndImport(t *testing.T) {
	samplePGN := `[Event "My Study: Chapter 1 - Najdorf"]
[Site "https://lichess.org/study/AbCd1234/11111111"]
[White "Theory"]
[Black "Najdorf"]
[Result "*"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 { Main line } *

[Event "My Study: Chapter 2 - Lucena"]
[Site "https://lichess.org/study/AbCd1234/22222222"]
[FEN "1K1k4/1P6/8/8/8/8/r7/2R5 w - - 0 1"]
[Result "1-0"]

1. Rd1+ Ke7 2. Rd4 1-0
`

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path == "/api/study/NotFound.pgn" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-chess-pgn")
		_, _ = w.Write([]byte(samplePGN))
	}))
	defer srv.Close()

	client := NewLichessClient()
	client.BaseURL = srv.URL

	db := scid.NewDatabase()
	require.NoError(t, db.Create(filepath.Join(t.TempDir(), "study"), "Study"))
	defer db.Close()

	res, err := client.FetchLichessStudy(context.Background(), "https://lichess.org/study/AbCd1234", "lip_test", db, true)
	require.NoError(t, err)
	assert.Equal(t, "Bearer lip_test", gotAuth)
	assert.Equal(t, "AbCd1234", res.StudyID)
	assert.Equal(t, 2, res.TotalChapters)
	assert.Equal(t, 2, res.ImportedCount)
	assert.Equal(t, 2, db.NumGames())
	assert.Equal(t, 1, res.Chapters[0].GameID)
	assert.Equal(t, 2, res.Chapters[1].GameID)

	// 404 handling
	_, err = client.FetchLichessStudy(context.Background(), "NotFound", "", nil, false)
	assert.Error(t, err)
}
