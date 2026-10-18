package online

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

const (
	DefaultChessComBaseURL = "https://api.chess.com"
)

// ChessComArchivesResponse maps the JSON response from /pub/player/{username}/games/archives.
type ChessComArchivesResponse struct {
	Archives []string `json:"archives"`
}

// ChessComClient provides an HTTP client for retrieving games from chess.com.
type ChessComClient struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

// NewChessComClient creates a client configured with default timeout and user agent.
func NewChessComClient() *ChessComClient {
	return &ChessComClient{
		BaseURL: DefaultChessComBaseURL,
		HTTPClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
		},
		UserAgent: DefaultOnlineUserAgent,
	}
}

// FetchArchives retrieves the list of monthly game archive URLs for a user.
func (c *ChessComClient) FetchArchives(username string) ([]string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("chess.com username cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/pub/player/%s/games/archives", strings.TrimRight(c.BaseURL, "/"), url.PathEscape(strings.ToLower(username)))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating chess.com archives request: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chess.com archives request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("chess.com user %q not found", username)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("chess.com rate limit exceeded, please wait before retrying")
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("chess.com API returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var archivesResp ChessComArchivesResponse
	if err := json.NewDecoder(resp.Body).Decode(&archivesResp); err != nil {
		return nil, fmt.Errorf("decoding chess.com archives json: %w", err)
	}

	return archivesResp.Archives, nil
}

// FetchMonthlyPGN downloads raw PGN games for a specific monthly archive URL.
func (c *ChessComClient) FetchMonthlyPGN(archiveURL string) (string, error) {
	archiveURL = strings.TrimSpace(archiveURL)
	if archiveURL == "" {
		return "", fmt.Errorf("archive URL cannot be empty")
	}

	pgnURL := strings.TrimRight(archiveURL, "/") + "/pgn"

	req, err := http.NewRequest(http.MethodGet, pgnURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating monthly pgn request: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/x-chess-pgn")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("monthly pgn request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // Empty month
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("chess.com monthly pgn HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading chess.com monthly pgn: %w", err)
	}

	return string(body), nil
}

// FetchUserGames retrieves games for a Chess.com player across relevant monthly archives.
// If sinceYear > 0 and sinceMonth > 0, older archives are skipped.
// If maxLimit is provided and > 0, archives are processed from most recent backwards until maxLimit games are retrieved.
func (c *ChessComClient) FetchUserGames(username string, sinceYear, sinceMonth int, maxLimit ...int) (string, error) {
	maxGames := 0
	if len(maxLimit) > 0 && maxLimit[0] > 0 {
		maxGames = maxLimit[0]
	}
	return c.FetchUserGamesWithProgress(username, sinceYear, sinceMonth, maxGames, nil)
}

// FetchUserGamesWithProgress retrieves games for a Chess.com player with real-time progress callbacks.
func (c *ChessComClient) FetchUserGamesWithProgress(username string, sinceYear, sinceMonth int, maxGames int, progress ProgressFn) (string, error) {
	archives, err := c.FetchArchives(username)
	if err != nil {
		return "", err
	}

	if len(archives) == 0 {
		return "", nil
	}

	// If maxGames > 0 and no specific sinceYear/Month filter, process archives newest-first
	// so the user gets recent games immediately without waiting for hundreds of old archives.
	targetArchives := append([]string(nil), archives...)
	if maxGames > 0 && sinceYear == 0 && sinceMonth == 0 {
		for i, j := 0, len(targetArchives)-1; i < j; i, j = i+1, j-1 {
			targetArchives[i], targetArchives[j] = targetArchives[j], targetArchives[i]
		}
	}

	var pgnBuilder strings.Builder
	gameCount := 0

	for idx, archURL := range targetArchives {
		// Filter by sinceYear and sinceMonth if specified
		if sinceYear > 0 && sinceMonth > 0 {
			// Expected URL structure: .../games/{YYYY}/{MM}
			parts := strings.Split(strings.TrimRight(archURL, "/"), "/")
			if len(parts) >= 2 {
				var y, m int
				fmt.Sscanf(parts[len(parts)-2], "%d", &y)
				fmt.Sscanf(parts[len(parts)-1], "%d", &m)

				if y > 0 && m > 0 {
					if y < sinceYear || (y == sinceYear && m < sinceMonth) {
						continue // Skip older archive
					}
				}
			}
		}

		if progress != nil {
			msg := fmt.Sprintf("Downloading from Chess.com archive %d/%d...", idx+1, len(targetArchives))
			progress("downloading", idx+1, len(targetArchives), msg)
		}

		pgnText, err := c.FetchMonthlyPGN(archURL)
		if err != nil {
			return "", fmt.Errorf("fetching archive %s: %w", archURL, err)
		}

		if strings.TrimSpace(pgnText) != "" {
			pgnBuilder.WriteString(pgnText)
			pgnBuilder.WriteString("\n\n")
			added := strings.Count(pgnText, "[Event ")
			gameCount += added
			if progress != nil {
				msg := fmt.Sprintf("Chess.com: %d games downloaded (%d/%d archives)", gameCount, idx+1, len(targetArchives))
				progress("downloading", gameCount, maxGames, msg)
			}
			if maxGames > 0 && gameCount >= maxGames {
				break
			}
		}
	}

	return pgnBuilder.String(), nil
}

// FetchUserGamesIncremental retrieves games for a Chess.com player starting from the most recent monthly archive
// and automatically walking backwards into older archives if needed, skipping any game already present in existingSigs
// WITHOUT counting existing games toward maxNewGames.
func (c *ChessComClient) FetchUserGamesIncremental(
	username string,
	maxNewGames int,
	existingSigs *scid.GameSignatureSet,
	progress ProgressFn,
) ([]*chess.Game, int, error) {
	archives, err := c.FetchArchives(username)
	if err != nil {
		return nil, 0, err
	}
	if len(archives) == 0 {
		return nil, 0, nil
	}

	var newGames []*chess.Game
	skippedExisting := 0
	totalArchives := len(archives)

	// Walk from most recent monthly archive backwards to oldest archive
	for step := 0; step < totalArchives; step++ {
		archIdx := totalArchives - 1 - step
		archURL := archives[archIdx]

		if progress != nil {
			msg := fmt.Sprintf("Checking Chess.com archive %d/%d (%d new, %d existing skipped)...", step+1, totalArchives, len(newGames), skippedExisting)
			progress("downloading", len(newGames), maxNewGames, msg)
		}

		pgnText, err := c.FetchMonthlyPGN(archURL)
		if err != nil {
			return newGames, skippedExisting, fmt.Errorf("fetching archive %s: %w", archURL, err)
		}
		if strings.TrimSpace(pgnText) == "" {
			continue
		}

		monthGames, err := chess.ParsePGNDatabase(strings.NewReader(pgnText))
		if err != nil || len(monthGames) == 0 {
			continue
		}

		// Process newest games in the month first
		for gIdx := len(monthGames) - 1; gIdx >= 0; gIdx-- {
			g := monthGames[gIdx]
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
				break
			}
		}

		if progress != nil {
			msg := fmt.Sprintf("Chess.com update: %d new games found (%d existing skipped, %d/%d archives)", len(newGames), skippedExisting, step+1, totalArchives)
			progress("downloading", len(newGames), maxNewGames, msg)
		}

		if maxNewGames > 0 && len(newGames) >= maxNewGames {
			break
		}
	}

	return newGames, skippedExisting, nil
}

