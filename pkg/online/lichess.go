package online

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultLichessBaseURL   = "https://lichess.org"
	DefaultOnlineUserAgent  = "NewScid/1.0 (Modern Chess Information Database)"
	DefaultHTTPTimeout      = 45 * time.Second
)

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
			Timeout: DefaultHTTPTimeout,
		},
		UserAgent: DefaultOnlineUserAgent,
	}
}

// FetchUserGames downloads games for the specified Lichess user in standard PGN format.
// If sinceTimeMs > 0, only games played after that millisecond timestamp are returned.
func (c *LichessClient) FetchUserGames(username string, sinceTimeMs int64, token string, ratedOnly bool, maxGames int) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", fmt.Errorf("lichess username cannot be empty")
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading lichess response: %w", err)
	}

	return string(body), nil
}
