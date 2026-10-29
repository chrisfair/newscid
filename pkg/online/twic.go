package online

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// TWICIssueResult details the result of downloading and importing a single TWIC issue.
type TWICIssueResult struct {
	IssueNumber     int    `json:"issueNumber"`
	URL             string `json:"url"`
	GamesDownloaded int    `json:"gamesDownloaded"`
	GamesImported   int    `json:"gamesImported"`
	DurationMs      int64  `json:"durationMs"`
	Error           string `json:"error,omitempty"`
}

// TWICBatchResult summarizes a range download of TWIC bulletins.
type TWICBatchResult struct {
	StartIssue      int                `json:"startIssue"`
	EndIssue        int                `json:"endIssue"`
	TotalDownloaded int                `json:"totalDownloaded"`
	TotalImported   int                `json:"totalImported"`
	DurationMs      int64              `json:"durationMs"`
	Issues          []*TWICIssueResult `json:"issues"`
}

// TWICClient manages downloading TWIC (The Week In Chess) bulletins.
type TWICClient struct {
	BaseURLTemplate string // defaults to "https://theweekinchess.com/zips/twic%dg.zip"
	HTTPClient      *http.Client
	UserAgent       string
}

// NewTWICClient creates a new TWICClient with reasonable HTTP timeouts.
func NewTWICClient() *TWICClient {
	return &TWICClient{
		BaseURLTemplate: "https://theweekinchess.com/zips/twic%dg.zip",
		HTTPClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		UserAgent: "NewScid/1.0 (Chess Database; https://github.com/chrisfair/newscid)",
	}
}

// GetURL returns the download URL for a given TWIC issue number.
func (c *TWICClient) GetURL(issue int) string {
	return fmt.Sprintf(c.BaseURLTemplate, issue)
}

// DownloadIssue downloads and extracts all PGN games from a given TWIC issue.
func (c *TWICClient) DownloadIssue(ctx context.Context, issue int) ([]*chess.Game, error) {
	if issue <= 0 {
		return nil, fmt.Errorf("invalid TWIC issue number: %d", issue)
	}

	url := c.GetURL(issue)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for TWIC %d: %w", issue, err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading TWIC issue %d: %w", issue, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("TWIC issue %d not found (HTTP 404)", issue)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TWIC issue %d returned unexpected HTTP %d: %s", issue, resp.StatusCode, resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response for TWIC issue %d: %w", issue, err)
	}

	return c.extractGamesFromZip(bodyBytes, issue)
}

func (c *TWICClient) extractGamesFromZip(data []byte, issue int) ([]*chess.Game, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("opening TWIC %d zip archive: %w", issue, err)
	}

	var allPGNBuilder strings.Builder
	foundPGN := false

	for _, file := range zipReader.File {
		if strings.HasSuffix(strings.ToLower(file.Name), ".pgn") {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("reading %s in TWIC %d zip: %w", file.Name, issue, err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("reading %s content in TWIC %d zip: %w", file.Name, issue, err)
			}
			allPGNBuilder.Write(content)
			allPGNBuilder.WriteString("\n\n")
			foundPGN = true
		}
	}

	if !foundPGN {
		return nil, fmt.Errorf("no .pgn file found inside TWIC %d zip archive", issue)
	}

	games, err := chess.ParsePGNDatabase(strings.NewReader(allPGNBuilder.String()))
	if err != nil {
		return nil, fmt.Errorf("parsing PGN games from TWIC %d: %w", issue, err)
	}

	return games, nil
}

// ImportIssue downloads the specified TWIC issue and imports it directly into targetDB.
func (c *TWICClient) ImportIssue(ctx context.Context, issue int, targetDB *scid.Database) (*TWICIssueResult, error) {
	if targetDB == nil || !targetDB.IsOpen {
		return nil, fmt.Errorf("target database is not open")
	}

	start := time.Now()
	res := &TWICIssueResult{
		IssueNumber: issue,
		URL:         c.GetURL(issue),
	}

	games, err := c.DownloadIssue(ctx, issue)
	if err != nil {
		res.DurationMs = time.Since(start).Milliseconds()
		res.Error = err.Error()
		return res, err
	}

	res.GamesDownloaded = len(games)
	if len(games) == 0 {
		res.DurationMs = time.Since(start).Milliseconds()
		return res, nil
	}

	existingSigs := targetDB.BuildGameSignatureSet()
	newGames, _ := existingSigs.FilterNewGames(games)
	if len(newGames) == 0 {
		res.GamesImported = 0
		res.DurationMs = time.Since(start).Milliseconds()
		return res, nil
	}

	importStats, err := targetDB.ImportGames(newGames)
	if err != nil {
		res.DurationMs = time.Since(start).Milliseconds()
		res.Error = err.Error()
		return res, fmt.Errorf("importing TWIC %d games into database: %w", issue, err)
	}

	res.GamesImported = importStats.Imported
	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}

// ImportRange downloads a range of TWIC issues [startIssue, endIssue] and imports them.
// progressCallback is invoked after each issue is processed.
func (c *TWICClient) ImportRange(
	ctx context.Context,
	startIssue, endIssue int,
	targetDB *scid.Database,
	progressFn func(currentIssue, processedIssues, totalIssues, gamesImported int),
) (*TWICBatchResult, error) {
	if startIssue <= 0 || endIssue <= 0 {
		return nil, fmt.Errorf("invalid issue range: %d..%d", startIssue, endIssue)
	}
	if startIssue > endIssue {
		startIssue, endIssue = endIssue, startIssue
	}

	if targetDB == nil || !targetDB.IsOpen {
		return nil, fmt.Errorf("target database is not open")
	}

	totalIssues := endIssue - startIssue + 1
	start := time.Now()
	batch := &TWICBatchResult{
		StartIssue: startIssue,
		EndIssue:   endIssue,
		Issues:     make([]*TWICIssueResult, 0, totalIssues),
	}

	for issue := startIssue; issue <= endIssue; issue++ {
		select {
		case <-ctx.Done():
			batch.DurationMs = time.Since(start).Milliseconds()
			return batch, ctx.Err()
		default:
		}

		res, err := c.ImportIssue(ctx, issue, targetDB)
		if err != nil {
			if res == nil {
				res = &TWICIssueResult{
					IssueNumber: issue,
					URL:         c.GetURL(issue),
					Error:       err.Error(),
				}
			}
		} else {
			batch.TotalDownloaded += res.GamesDownloaded
			batch.TotalImported += res.GamesImported
		}
		batch.Issues = append(batch.Issues, res)

		if progressFn != nil {
			progressFn(issue, len(batch.Issues), totalIssues, batch.TotalImported)
		}
	}

	batch.DurationMs = time.Since(start).Milliseconds()
	return batch, nil
}

// CheckIssueExists tests whether a given TWIC issue is available on the server.
func (c *TWICClient) CheckIssueExists(ctx context.Context, issue int) (bool, error) {
	if issue <= 0 {
		return false, fmt.Errorf("invalid issue number: %d", issue)
	}

	url := c.GetURL(issue)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	// Some servers return 405 Method Not Allowed for HEAD; fallback to GET with small Range
	if resp.StatusCode == http.StatusMethodNotAllowed {
		getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false, err
		}
		getReq.Header.Set("User-Agent", c.UserAgent)
		getReq.Header.Set("Range", "bytes=0-10")

		getResp, err := c.HTTPClient.Do(getReq)
		if err != nil {
			return false, err
		}
		defer getResp.Body.Close()
		return getResp.StatusCode == http.StatusOK || getResp.StatusCode == http.StatusPartialContent, nil
	}

	return false, nil
}

// FindLatestIssue searches for the highest published TWIC issue starting around probeStart.
func (c *TWICClient) FindLatestIssue(ctx context.Context, probeStart int) (int, error) {
	if probeStart < 1 {
		probeStart = 1500
	}

	// Verify if probeStart exists. If not, binary search downwards.
	exists, err := c.CheckIssueExists(ctx, probeStart)
	if err != nil {
		return 0, fmt.Errorf("probing initial issue %d: %w", probeStart, err)
	}

	low := 1
	high := probeStart

	if exists {
		// Probe forward exponentially: probeStart + 10, + 20, + 40...
		low = probeStart
		step := 10
		curr := probeStart + step
		for {
			select {
			case <-ctx.Done():
				return low, ctx.Err()
			default:
			}

			found, err := c.CheckIssueExists(ctx, curr)
			if err != nil || !found {
				high = curr
				break
			}
			low = curr
			step *= 2
			curr = low + step
		}
	}

	// Binary search between low (exists) and high (does not exist)
	for low < high-1 {
		select {
		case <-ctx.Done():
			return low, ctx.Err()
		default:
		}

		mid := low + (high-low)/2
		found, err := c.CheckIssueExists(ctx, mid)
		if err != nil {
			return low, err
		}
		if found {
			low = mid
		} else {
			high = mid
		}
	}

	return low, nil
}
