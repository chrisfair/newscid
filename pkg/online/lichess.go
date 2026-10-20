package online

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

const (
	DefaultLichessBaseURL   = "https://lichess.org"
	DefaultOnlineUserAgent  = "NewScid/1.0 (Modern Chess Information Database)"
	DefaultHTTPTimeout      = 45 * time.Second
)

// ProgressFn is a callback reporting real-time download and processing progress.
type ProgressFn func(stage string, current int, total int, message string)

// LichessClient provides an HTTP client for retrieving games from lichess.org.
type LichessClient struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

// NewLichessClient creates a client configured with default timeout and user agent.
func NewLichessClient() *LichessClient {
	return &LichessClient{
		BaseURL: DefaultLichessBaseURL,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				ResponseHeaderTimeout: 30 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
		UserAgent: DefaultOnlineUserAgent,
	}
}

// GetUserGameCount queries Lichess user endpoint to determine the total number of games for a user.
func (c *LichessClient) GetUserGameCount(username string, ratedOnly bool) int {
	username = strings.TrimSpace(username)
	if username == "" {
		return 0
	}
	endpoint := fmt.Sprintf("%s/api/user/%s", strings.TrimRight(c.BaseURL, "/"), url.PathEscape(username))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTPClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return 0
	}
	defer resp.Body.Close()

	var userProfile struct {
		Count struct {
			All   int `json:"all"`
			Rated int `json:"rated"`
		} `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userProfile); err != nil {
		return 0
	}
	if ratedOnly {
		return userProfile.Count.Rated
	}
	return userProfile.Count.All
}

// FetchUserGames downloads games for the specified Lichess user in standard PGN format.
// If sinceTimeMs > 0, only games played after that millisecond timestamp are returned.
// Progress updates are streamed in real time via optional onProgress callback.
func (c *LichessClient) FetchUserGames(username string, sinceTimeMs int64, token string, ratedOnly bool, maxGames int, onProgress ...ProgressFn) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", fmt.Errorf("lichess username cannot be empty")
	}

	var progress ProgressFn
	if len(onProgress) > 0 && onProgress[0] != nil {
		progress = onProgress[0]
	}

	totalExpected := maxGames
	if maxGames == 0 && progress != nil {
		totalExpected = c.GetUserGameCount(username, ratedOnly)
	}

	endpoint := fmt.Sprintf("%s/api/games/user/%s", strings.TrimRight(c.BaseURL, "/"), url.PathEscape(username))

	queryParams := url.Values{}
	queryParams.Set("clocks", "true")
	queryParams.Set("evals", "true")
	queryParams.Set("opening", "true")

	if sinceTimeMs > 0 {
		queryParams.Set("since", strconv.FormatInt(sinceTimeMs, 10))
	}
	if ratedOnly {
		queryParams.Set("rated", "true")
	}
	if maxGames > 0 {
		queryParams.Set("max", strconv.Itoa(maxGames))
	}

	fullURL := endpoint + "?" + queryParams.Encode()

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating lichess request: %w", err)
	}

	req.Header.Set("Accept", "application/x-chess-pgn")
	req.Header.Set("User-Agent", c.UserAgent)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("lichess request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("lichess user %q not found", username)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("lichess rate limit exceeded, please wait before retrying")
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("lichess API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var pgnBuilder strings.Builder
	reader := bufio.NewReaderSize(resp.Body, 64*1024)
	gameCount := 0
	lastReportTime := time.Now()

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			pgnBuilder.WriteString(line)
			if strings.HasPrefix(line, "[Event ") {
				gameCount++
				if progress != nil && (time.Since(lastReportTime) > 350*time.Millisecond || gameCount%25 == 0) {
					lastReportTime = time.Now()
					msg := fmt.Sprintf("Downloading from Lichess: %d games...", gameCount)
					if totalExpected > 0 {
						pct := (gameCount * 100) / totalExpected
						if pct > 100 {
							pct = 100
						}
						msg = fmt.Sprintf("Downloading from Lichess: %d / %d games (%d%%)...", gameCount, totalExpected, pct)
					}
					progress("downloading", gameCount, totalExpected, msg)
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("reading lichess response: %w", err)
		}
	}

	if progress != nil && gameCount > 0 {
		msg := fmt.Sprintf("Lichess download complete: %d games received", gameCount)
		progress("downloaded", gameCount, totalExpected, msg)
	}

	return pgnBuilder.String(), nil
}

// FetchUserGamesIncremental streams games for username from Lichess starting from the most recent games
// and automatically continuing into older games if necessary, skipping any game already present in existingSigs
// WITHOUT counting existing games toward maxNewGames.
func (c *LichessClient) FetchUserGamesIncremental(
	username string,
	token string,
	ratedOnly bool,
	maxNewGames int,
	existingSigs *scid.GameSignatureSet,
	onProgress ...ProgressFn,
) ([]*chess.Game, int, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, 0, fmt.Errorf("lichess username cannot be empty")
	}

	var progress ProgressFn
	if len(onProgress) > 0 && onProgress[0] != nil {
		progress = onProgress[0]
	}

	endpoint := fmt.Sprintf("%s/api/games/user/%s", strings.TrimRight(c.BaseURL, "/"), url.PathEscape(username))

	queryParams := url.Values{}
	queryParams.Set("clocks", "true")
	queryParams.Set("evals", "true")
	queryParams.Set("opening", "true")
	if ratedOnly {
		queryParams.Set("rated", "true")
	}
	// If we have existing games, request enough games to skip all existing ones and still collect maxNewGames new ones
	if maxNewGames > 0 {
		existingCount := 0
		if existingSigs != nil {
			existingCount = existingSigs.Count()
		}
		if existingCount == 0 {
			queryParams.Set("max", strconv.Itoa(maxNewGames))
		} else {
			queryParams.Set("max", strconv.Itoa(maxNewGames+existingCount+50))
		}
	}

	fullURL := endpoint + "?" + queryParams.Encode()
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("creating lichess request: %w", err)
	}
	req.Header.Set("Accept", "application/x-chess-pgn")
	req.Header.Set("User-Agent", c.UserAgent)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("lichess request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, 0, fmt.Errorf("lichess user %q not found", username)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, 0, fmt.Errorf("lichess rate limit exceeded, please wait before retrying")
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, 0, fmt.Errorf("lichess API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)

	var currentGameLines strings.Builder
	inMoves := false
	var newGames []*chess.Game
	skippedExisting := 0
	lastReportTime := time.Now()

	processOne := func(rawPGN string) bool {
		trimmed := strings.TrimSpace(rawPGN)
		if trimmed == "" {
			return false
		}
		parsed, err := chess.ParsePGNDatabase(strings.NewReader(trimmed))
		if err != nil || len(parsed) == 0 {
			return false
		}
		for _, g := range parsed {
			if g == nil {
				continue
			}
			if existingSigs != nil && existingSigs.ContainsGame(g) {
				skippedExisting++
				continue
			}
			if existingSigs != nil {
				existingSigs.AddGame(g)
			}
			newGames = append(newGames, g)
			if maxNewGames > 0 && len(newGames) >= maxNewGames {
				return true
			}
		}
		if progress != nil && (time.Since(lastReportTime) > 350*time.Millisecond || len(newGames)%20 == 0) {
			lastReportTime = time.Now()
			msg := fmt.Sprintf("Lichess update: %d new games found (%d already in DB skipped)...", len(newGames), skippedExisting)
			progress("downloading", len(newGames), maxNewGames, msg)
		}
		return false
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && inMoves {
			if stop := processOne(currentGameLines.String()); stop {
				currentGameLines.Reset()
				inMoves = false
				break
			}
			currentGameLines.Reset()
			inMoves = false
		} else if trimmed != "" && !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "%") {
			inMoves = true
		}

		currentGameLines.WriteString(line)
		currentGameLines.WriteByte('\n')
	}

	if currentGameLines.Len() > 0 && (maxNewGames <= 0 || len(newGames) < maxNewGames) {
		_ = processOne(currentGameLines.String())
	}

	if progress != nil {
		msg := fmt.Sprintf("Lichess sync complete: %d new games (%d existing skipped)", len(newGames), skippedExisting)
		progress("downloaded", len(newGames), maxNewGames, msg)
	}

	return newGames, skippedExisting, nil
}

// FetchRecentNewGames polls only the most recent `limit` games for a Lichess user
// (ideal for lightweight real-time auto-capture while playing in the Live Portal)
// and returns any games not yet present in existingSigs.
func (c *LichessClient) FetchRecentNewGames(
	username string,
	token string,
	limit int,
	existingSigs *scid.GameSignatureSet,
) ([]*chess.Game, int, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, 0, fmt.Errorf("lichess username cannot be empty")
	}
	if limit <= 0 {
		limit = 8
	}

	endpoint := fmt.Sprintf("%s/api/games/user/%s", strings.TrimRight(c.BaseURL, "/"), url.PathEscape(username))
	queryParams := url.Values{}
	queryParams.Set("clocks", "true")
	queryParams.Set("evals", "true")
	queryParams.Set("opening", "true")
	queryParams.Set("max", strconv.Itoa(limit))

	req, err := http.NewRequest(http.MethodGet, endpoint+"?"+queryParams.Encode(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/x-chess-pgn")
	req.Header.Set("User-Agent", c.UserAgent)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, 0, fmt.Errorf("lichess HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	parsed, err := chess.ParsePGNDatabase(strings.NewReader(string(rawBytes)))
	if err != nil {
		return nil, 0, err
	}

	var newGames []*chess.Game
	skipped := 0
	// Iterate from oldest of the recent batch to newest so chronological order is preserved on import
	for i := len(parsed) - 1; i >= 0; i-- {
		g := parsed[i]
		if g == nil {
			continue
		}
		if existingSigs != nil && existingSigs.ContainsGame(g) {
			skipped++
			continue
		}
		if existingSigs != nil {
			existingSigs.AddGame(g)
		}
		newGames = append(newGames, g)
	}
	return newGames, skipped, nil
}

// FetchGameByID downloads a single Lichess game by its 8-character (or 12-character player URL) ID.
func (c *LichessClient) FetchGameByID(gameID string, token string) (*chess.Game, error) {
	cleanID := strings.TrimSpace(gameID)
	cleanID = strings.TrimPrefix(cleanID, "/")
	if idx := strings.Index(cleanID, "/"); idx > 0 {
		cleanID = cleanID[:idx]
	}
	if idx := strings.IndexAny(cleanID, "?#"); idx > 0 {
		cleanID = cleanID[:idx]
	}
	if len(cleanID) == 12 {
		cleanID = cleanID[:8]
	}
	if len(cleanID) < 8 {
		return nil, fmt.Errorf("invalid lichess game ID %q", gameID)
	}

	endpoint := fmt.Sprintf("%s/game/export/%s?clocks=true&evals=true&opening=true", strings.TrimRight(c.BaseURL, "/"), url.PathEscape(cleanID))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/x-chess-pgn")
	req.Header.Set("User-Agent", c.UserAgent)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("lichess game export HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return chess.ParsePGN(string(rawBytes))
}


