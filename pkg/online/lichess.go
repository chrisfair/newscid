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
