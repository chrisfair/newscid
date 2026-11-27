package online

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// StudyChapterSummary summarizes a single imported chapter from a Lichess Study.
type StudyChapterSummary struct {
	ChapterIndex int    `json:"chapterIndex"`
	Title        string `json:"title"`
	White        string `json:"white"`
	Black        string `json:"black"`
	FEN          string `json:"fen"`
	PlyCount     int    `json:"plyCount"`
	PGN          string `json:"pgn"`
	GameID       int    `json:"gameId,omitempty"`
}

// LichessStudyImportResult holds the chapters fetched and optionally imported from a Lichess Study.
type LichessStudyImportResult struct {
	StudyID       string                `json:"studyId"`
	StudyURL      string                `json:"studyUrl"`
	TotalChapters int                   `json:"totalChapters"`
	ImportedCount int                   `json:"importedCount"`
	Chapters      []StudyChapterSummary `json:"chapters"`
}

var lichessStudyIDRe = regexp.MustCompile(`(?i)(?:lichess\.org/study/)?([a-z0-9]{8})(?:/[a-z0-9]{8})?`)

// ParseLichessStudyID extracts the 8-character Lichess study ID from a URL or raw ID.
func ParseLichessStudyID(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("study URL or ID is required")
	}
	// Strip query/fragment
	if idx := strings.IndexAny(trimmed, "?#"); idx != -1 {
		trimmed = trimmed[:idx]
	}
	trimmed = strings.TrimRight(trimmed, "/")

	// Direct 8-char alphanumeric ID
	if len(trimmed) == 8 && !strings.Contains(trimmed, "/") {
		for _, ch := range trimmed {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
				return "", fmt.Errorf("invalid Lichess study ID: %q", input)
			}
		}
		return trimmed, nil
	}

	m := lichessStudyIDRe.FindStringSubmatch(trimmed)
	if len(m) < 2 {
		return "", fmt.Errorf("could not extract 8-character Lichess study ID from %q", input)
	}
	return m[1], nil
}

// FetchLichessStudy downloads all chapters of a public (or token-authorized) Lichess Study
// and optionally imports them directly into targetDB when importIntoDB is true.
func (c *LichessClient) FetchLichessStudy(ctx context.Context, studyInput string, token string, targetDB *scid.Database, importIntoDB bool) (*LichessStudyImportResult, error) {
	studyID, err := ParseLichessStudyID(studyInput)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://lichess.org"
	}
	reqURL := fmt.Sprintf("%s/api/study/%s.pgn?comments=true&variations=true&clocks=true&source=true", baseURL, studyID)

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building Lichess study request: %w", err)
	}
	req.Header.Set("Accept", "application/x-chess-pgn")
	req.Header.Set("User-Agent", "NewScid/2.1 (+https://github.com/chrisfair/newscid)")
	if t := strings.TrimSpace(token); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Lichess study %s: %w", studyID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("Lichess study %q not found (or private — add a Lichess token if needed)", studyID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Lichess study API returned HTTP %d", resp.StatusCode)
	}

	rawBytes, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("reading study PGN: %w", err)
	}

	games, err := chess.ParsePGNDatabase(strings.NewReader(string(rawBytes)))
	if err != nil {
		return nil, fmt.Errorf("parsing study PGN: %w", err)
	}

	res := &LichessStudyImportResult{
		StudyID:       studyID,
		StudyURL:      fmt.Sprintf("https://lichess.org/study/%s", studyID),
		TotalChapters: len(games),
		Chapters:      make([]StudyChapterSummary, 0, len(games)),
	}

	for idx, g := range games {
		title := g.Tag("Event")
		if title == "" || title == "?" {
			title = fmt.Sprintf("Chapter %d", idx+1)
		}
		summary := StudyChapterSummary{
			ChapterIndex: idx + 1,
			Title:        title,
			White:        g.Tag("White"),
			Black:        g.Tag("Black"),
			FEN:          g.StartFEN,
			PlyCount:     g.PlyCount,
			PGN:          g.ExportPGN(),
		}
		if importIntoDB && targetDB != nil && targetDB.IsOpen {
			if id, saveErr := targetDB.SaveGame(g); saveErr == nil {
				summary.GameID = id
				res.ImportedCount++
			}
		}
		res.Chapters = append(res.Chapters, summary)
	}

	return res, nil
}
