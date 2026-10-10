package tablebase

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TablebaseMove represents a candidate move evaluated by the endgame tablebase.
type TablebaseMove struct {
	UCI       string `json:"uci"`
	SAN       string `json:"san"`
	DTZ       int    `json:"dtz"`
	DTM       int    `json:"dtm"`
	Category  string `json:"category"` // "win", "loss", "draw", "unknown"
	IsZeroing bool   `json:"zeroing"`
}

// TablebaseResult represents the full tablebase evaluation of a position.
type TablebaseResult struct {
	FEN         string          `json:"fen"`
	PieceCount  int             `json:"pieceCount"`
	Category    string          `json:"category"` // "win", "loss", "draw", "checkmate", "stalemate"
	DTZ         int             `json:"dtz"`      // Distance to zeroing move (pawn move or capture)
	DTM         int             `json:"dtm"`      // Distance to checkmate (if known)
	Checkmate   bool            `json:"checkmate"`
	Stalemate   bool            `json:"stalemate"`
	SummaryText string          `json:"summaryText"`
	Moves       []TablebaseMove `json:"moves"`
}

// Client probes Syzygy endgame tablebases with caching.
type Client struct {
	mu         sync.RWMutex
	cache      map[string]*TablebaseResult
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new tablebase client.
func NewClient() *Client {
	return &Client{
		cache: make(map[string]*TablebaseResult),
		httpClient: &http.Client{
			Timeout: 4 * time.Second,
		},
		baseURL: "https://tablebase.lichess.ovh/standard",
	}
}

// CountPieces returns the total number of pieces on the board for a FEN.
func CountPieces(fen string) int {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) == 0 {
		return 0
	}
	count := 0
	for _, ch := range parts[0] {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			count++
		}
	}
	return count
}

type apiResponse struct {
	Checkmate bool `json:"checkmate"`
	Stalemate bool `json:"stalemate"`
	DTZ       int  `json:"dtz"`
	DTM       int  `json:"dtm"`
	Category  string `json:"category"`
	Moves     []struct {
		UCI      string `json:"uci"`
		SAN      string `json:"san"`
		DTZ      int    `json:"dtz"`
		DTM      int    `json:"dtm"`
		Category string `json:"category"`
		Zeroing  bool   `json:"zeroing"`
	} `json:"moves"`
}

// Probe evaluates an endgame position with 7 pieces or fewer.
func (c *Client) Probe(fen string) (*TablebaseResult, error) {
	cleanFEN := strings.TrimSpace(fen)
	if cleanFEN == "" {
		return nil, fmt.Errorf("empty FEN")
	}

	pieceCount := CountPieces(cleanFEN)
	if pieceCount > 7 {
		return nil, fmt.Errorf("position has %d pieces (tablebases support up to 7 pieces)", pieceCount)
	}

	// Check cache
	c.mu.RLock()
	if cached, ok := c.cache[cleanFEN]; ok {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	reqURL := fmt.Sprintf("%s?fen=%s", c.baseURL, url.QueryEscape(cleanFEN))
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("querying syzygy tablebase: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("syzygy API returned HTTP %d", resp.StatusCode)
	}

	var api apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, fmt.Errorf("decoding tablebase response: %w", err)
	}

	res := &TablebaseResult{
		FEN:        cleanFEN,
		PieceCount: pieceCount,
		Category:   api.Category,
		DTZ:        api.DTZ,
		DTM:        api.DTM,
		Checkmate:  api.Checkmate,
		Stalemate:  api.Stalemate,
	}

	for _, m := range api.Moves {
		res.Moves = append(res.Moves, TablebaseMove{
			UCI:       m.UCI,
			SAN:       m.SAN,
			DTZ:       m.DTZ,
			DTM:       m.DTM,
			Category:  m.Category,
			IsZeroing: m.Zeroing,
		})
	}

	res.SummaryText = formatSummary(res)

	// Save to cache
	c.mu.Lock()
	c.cache[cleanFEN] = res
	c.mu.Unlock()

	return res, nil
}

func formatSummary(res *TablebaseResult) string {
	if res.Checkmate {
		return "Checkmate"
	}
	if res.Stalemate {
		return "Stalemate (Draw)"
	}

	var sb strings.Builder
	cat := strings.ToLower(res.Category)
	switch cat {
	case "win":
		sb.WriteString("Win")
		if res.DTM != 0 {
			sb.WriteString(fmt.Sprintf(" in %d (DTM)", abs(res.DTM)))
		} else if res.DTZ != 0 {
			sb.WriteString(fmt.Sprintf(" in %d moves (DTZ)", abs(res.DTZ)))
		}
	case "loss":
		sb.WriteString("Loss")
		if res.DTM != 0 {
			sb.WriteString(fmt.Sprintf(" in %d (DTM)", abs(res.DTM)))
		} else if res.DTZ != 0 {
			sb.WriteString(fmt.Sprintf(" in %d moves (DTZ)", abs(res.DTZ)))
		}
	case "draw":
		sb.WriteString("Tablebase Draw")
	default:
		if len(res.Category) > 0 {
			sb.WriteString(strings.ToUpper(res.Category[:1]) + strings.ToLower(res.Category[1:]))
		}
	}

	if len(res.Moves) > 0 {
		best := res.Moves[0]
		if best.SAN != "" {
			sb.WriteString(fmt.Sprintf(" • Best: %s", best.SAN))
		}
	}

	return sb.String()
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
