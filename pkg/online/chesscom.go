package online

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
func (c *ChessComClient) FetchUserGames(username string, sinceYear, sinceMonth int) (string, error) {
	archives, err := c.FetchArchives(username)
	if err != nil {
		return "", err
	}

	if len(archives) == 0 {
		return "", nil
	}

	var pgnBuilder strings.Builder

	for _, archURL := range archives {
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

		pgnText, err := c.FetchMonthlyPGN(archURL)
		if err != nil {
			return "", fmt.Errorf("fetching archive %s: %w", archURL, err)
		}

		if strings.TrimSpace(pgnText) != "" {
			pgnBuilder.WriteString(pgnText)
			pgnBuilder.WriteString("\n\n")
		}
	}

	return pgnBuilder.String(), nil
}
