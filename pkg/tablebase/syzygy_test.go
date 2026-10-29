package tablebase

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountPieces(t *testing.T) {
	// Starting position (32 pieces)
	assert.Equal(t, 32, CountPieces("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"))

	// Endgame KPK (3 pieces)
	assert.Equal(t, 3, CountPieces("8/8/8/4k3/8/8/4P3/4K3 w - - 0 1"))

	// Empty string
	assert.Equal(t, 0, CountPieces(""))
}

func TestSyzygyProbeMock(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fen := r.URL.Query().Get("fen")
		if fen == "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"checkmate": false,
				"stalemate": false,
				"dtz": 14,
				"dtm": 18,
				"category": "win",
				"moves": [
					{"uci": "e2e4", "san": "e4", "dtz": -13, "dtm": -17, "category": "loss", "zeroing": true}
				]
			}`))
			return
		}

		if fen == "draw_fen" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"checkmate": false,
				"stalemate": false,
				"dtz": 0,
				"dtm": 0,
				"category": "draw",
				"moves": [
					{"uci": "e1f1", "san": "Kf1", "dtz": 0, "dtm": 0, "category": "draw", "zeroing": false}
				]
			}`))
			return
		}

		if fen == "bad_json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{invalid json`))
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	// Probe winning position
	fenWin := "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
	res, err := client.Probe(fenWin)
	require.NoError(t, err)
	assert.Equal(t, "win", res.Category)
	assert.Equal(t, 14, res.DTZ)
	assert.Equal(t, 18, res.DTM)
	assert.Contains(t, res.SummaryText, "Win in 18")
	assert.Contains(t, res.SummaryText, "Best: e4")
	require.Len(t, res.Moves, 1)
	assert.Equal(t, "e4", res.Moves[0].SAN)

	// Cache test (second probe should hit cache, callCount remains 1)
	resCached, err := client.Probe(fenWin)
	require.NoError(t, err)
	assert.Equal(t, res, resCached)
	assert.Equal(t, 1, callCount)

	// Draw position
	resDraw, err := client.Probe("draw_fen")
	require.NoError(t, err)
	assert.Equal(t, "draw", resDraw.Category)
	assert.Contains(t, resDraw.SummaryText, "Tablebase Draw")

	// Over piece limit (> 7 pieces)
	_, err = client.Probe("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tablebases support up to 7 pieces")

	// Empty FEN
	_, err = client.Probe("")
	assert.Error(t, err)

	// HTTP 404
	_, err = client.Probe("404_fen")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")

	// Invalid JSON
	_, err = client.Probe("bad_json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decoding tablebase response")

	// Network error (closed server)
	clientBadURL := NewClient()
	clientBadURL.baseURL = "http://127.0.0.1:1"
	_, err = clientBadURL.Probe("4k3/8/8/8/8/8/8/4K3 w - - 0 1")
	assert.Error(t, err)
}

func TestFormatSummaryCheckmateAndStalemate(t *testing.T) {
	cmRes := &TablebaseResult{
		Checkmate: true,
	}
	assert.Equal(t, "Checkmate", formatSummary(cmRes))

	smRes := &TablebaseResult{
		Stalemate: true,
	}
	assert.Equal(t, "Stalemate (Draw)", formatSummary(smRes))

	lossRes := &TablebaseResult{
		Category: "loss",
		DTZ:      -10,
	}
	assert.Contains(t, formatSummary(lossRes), "Loss in 10 moves (DTZ)")

	lossDTM := &TablebaseResult{
		Category: "loss",
		DTM:      -6,
	}
	assert.Contains(t, formatSummary(lossDTM), "Loss in 6 (DTM)")

	winDTZ := &TablebaseResult{
		Category: "win",
		DTZ:      12,
	}
	assert.Contains(t, formatSummary(winDTZ), "Win in 12 moves (DTZ)")

	otherCat := &TablebaseResult{
		Category: "cursed-win",
	}
	assert.Contains(t, formatSummary(otherCat), "Cursed-win")

	assert.Equal(t, 2, CountPieces("   4k3/8/8/8/8/8/8/4K3 w - - 0 1"))
}
