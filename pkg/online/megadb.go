package online

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bodgit/sevenzip"
	"github.com/klauspost/compress/zstd"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// MegaDatabaseSource describes a curated online chess database available for one-click download/import.
type MegaDatabaseSource struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`      // "LumbrasGigaBase", "Lichess Open", "Lichess Elite", "Lichess Broadcast"
	Category      string `json:"category"`      // "OTB Master", "Online Master (2400+)", "Lichess Monthly Stream", "Native .si4"
	Format        string `json:"format"`        // "pgn.zst", "7z-pgn", "7z-si4", "zip-pgn"
	URL           string `json:"url"`
	SizeEstimate  string `json:"sizeEstimate"`
	GamesEstimate string `json:"gamesEstimate"`
	Description   string `json:"description"`
	SupportsLimit bool   `json:"supportsLimit"` // true if streaming allows stopping early after N games
}

// MegaImportOptions configures a download and import operation from a big online database.
type MegaImportOptions struct {
	SourceID          string `json:"sourceId"`
	CustomURL         string `json:"customUrl,omitempty"`
	MaxGames          int    `json:"maxGames"`          // 0 = unlimited, >0 = stop stream after N matching games
	MinElo            int    `json:"minElo"`            // 0 = any rating, >0 = require WhiteElo >= MinElo or BlackElo >= MinElo
	CreateNewDatabase string `json:"createNewDatabase"` // optional name of a new .si4 database to create in user databases dir
}

// MegaImportProgress is emitted via Wails events during download & import.
type MegaImportProgress struct {
	Phase           string `json:"phase"` // "downloading", "streaming", "extracting", "importing", "completed"
	SourceName      string `json:"sourceName"`
	BytesDownloaded int64  `json:"bytesDownloaded"`
	TotalBytes      int64  `json:"totalBytes"`
	GamesScanned    int    `json:"gamesScanned"`
	GamesImported   int    `json:"gamesImported"`
	Percent         int    `json:"percent"`
	Message         string `json:"message"`
}

// MegaImportResult summarizes a completed mega database download/import.
type MegaImportResult struct {
	SourceID        string `json:"sourceId"`
	SourceName      string `json:"sourceName"`
	URL             string `json:"url"`
	GamesScanned    int    `json:"gamesScanned"`
	GamesImported   int    `json:"gamesImported"`
	BytesDownloaded int64  `json:"bytesDownloaded"`
	DatabasePath    string `json:"databasePath,omitempty"`
	DurationMs      int64  `json:"durationMs"`
}

// DefaultMegaDatabaseCatalog returns the built-in catalog of LumbrasGigaBase, Lichess Open, Lichess Broadcast, and Lichess Elite sources.
func DefaultMegaDatabaseCatalog() []MegaDatabaseSource {
	return []MegaDatabaseSource{
		// 1. LumbrasGigaBase (HuggingFace CDN)
		{
			ID:            "lumbras-otb-2026",
			Name:          "Lumbra's GigaBase — OTB 2026 Release",
			Provider:      "LumbrasGigaBase",
			Category:      "OTB Master",
			Format:        "7z-pgn",
			URL:           "https://huggingface.co/datasets/Lumbra74/LumbrasGigabaseCCBYNC40/resolve/main/LumbrasGigabase_OTB_2026_partial_release.7z",
			SizeEstimate:  "127 MB (.7z)",
			GamesEstimate: "~350,000 OTB games",
			Description:   "Latest 2026 Over-The-Board tournament games from Lumbra's GigaBase.",
			SupportsLimit: true,
		},
		{
			ID:            "lumbras-otb-2025",
			Name:          "Lumbra's GigaBase — OTB 2025 Complete",
			Provider:      "LumbrasGigaBase",
			Category:      "OTB Master",
			Format:        "7z-pgn",
			URL:           "https://huggingface.co/datasets/Lumbra74/LumbrasGigabaseCCBYNC40/resolve/main/LumbrasGigabase_OTB_2025.7z",
			SizeEstimate:  "111 MB (.7z)",
			GamesEstimate: "~550,000 OTB games",
			Description:   "Full 2025 Over-The-Board tournament archive from Lumbra's GigaBase.",
			SupportsLimit: true,
		},
		{
			ID:            "lumbras-otb-elo2400",
			Name:          "Lumbra's GigaBase — OTB Elite 2400+ Elo",
			Provider:      "LumbrasGigaBase",
			Category:      "OTB Master",
			Format:        "7z-pgn",
			URL:           "https://huggingface.co/datasets/Lumbra74/LumbrasGigabaseCCBYNC40/resolve/main/LumbrasGigabase_OTB_ELO2400.7z",
			SizeEstimate:  "203 MB (.7z)",
			GamesEstimate: "~1.4M Master/GM games",
			Description:   "All historical & modern OTB master games (2400+ Elo) curated by Lumbra's GigaBase.",
			SupportsLimit: true,
		},
		{
			ID:            "lumbras-otb-2020-2024",
			Name:          "Lumbra's GigaBase — OTB 2020–2024",
			Provider:      "LumbrasGigaBase",
			Category:      "OTB Master",
			Format:        "7z-pgn",
			URL:           "https://huggingface.co/datasets/Lumbra74/LumbrasGigabaseCCBYNC40/resolve/main/LumbrasGigabase_OTB_2020-2024.7z",
			SizeEstimate:  "254 MB (.7z)",
			GamesEstimate: "~1.8M OTB games",
			Description:   "Over-The-Board tournament games from 2020 through 2024.",
			SupportsLimit: true,
		},
		{
			ID:            "lumbras-online-2026",
			Name:          "Lumbra's GigaBase — Online Master 2026",
			Provider:      "LumbrasGigaBase",
			Category:      "Online Master (2400+)",
			Format:        "7z-pgn",
			URL:           "https://huggingface.co/datasets/Lumbra74/LumbrasGigabaseCCBYNC40/resolve/main/LumbrasGigabase_Online_2026_partial_release.7z",
			SizeEstimate:  "870 KB (.7z)",
			GamesEstimate: "~15,000 High-Elo Online games",
			Description:   "Fast download of 2026 titled/master online tournament games (Titled Tuesday, Brawl, etc.).",
			SupportsLimit: true,
		},
		{
			ID:            "lumbras-online-si4",
			Name:          "Lumbra's GigaBase — Native Scid .si4 Online MegaBase",
			Provider:      "LumbrasGigaBase",
			Category:      "Native .si4",
			Format:        "7z-si4",
			URL:           "https://huggingface.co/datasets/Lumbra74/LumbrasGigabaseCCBYNC40/resolve/main/LumbrasGigabase_Online_si4.7z",
			SizeEstimate:  "740 MB (.7z)",
			GamesEstimate: "~9.5M games (Native .si4)",
			Description:   "Pre-compiled Scid vs. PC (.si4/.sn4/.sg4) mega database—extracts directly into your User Databases folder.",
			SupportsLimit: false,
		},
		{
			ID:            "lumbras-otb-complete",
			Name:          "Lumbra's GigaBase — OTB Complete (All History)",
			Provider:      "LumbrasGigaBase",
			Category:      "OTB Master",
			Format:        "7z-pgn",
			URL:           "https://huggingface.co/datasets/Lumbra74/LumbrasGigabaseCCBYNC40/resolve/main/LumbrasGigabase_OTB_Complete.7z",
			SizeEstimate:  "2.46 GB (.7z)",
			GamesEstimate: "~15M+ OTB games",
			Description:   "The complete historical + modern Over-The-Board Lumbra's GigaBase.",
			SupportsLimit: true,
		},

		// 2. Lichess OTB Broadcasts (database.lichess.org/broadcast)
		{
			ID:            "lichess-broadcast-2026-09",
			Name:          "Lichess OTB Master Broadcasts — Sep 2026",
			Provider:      "Lichess Broadcast",
			Category:      "OTB Master",
			Format:        "pgn.zst",
			URL:           "https://database.lichess.org/broadcast/lichess_db_broadcast_2026-09.pgn.zst",
			SizeEstimate:  "~18 MB (.pgn.zst stream)",
			GamesEstimate: "~45,000 OTB Broadcast games",
			Description:   "Official FIDE & top OTB tournament broadcasts relayed on Lichess (Sep 2026). Streams & decompresses on the fly!",
			SupportsLimit: true,
		},
		{
			ID:            "lichess-broadcast-2026-08",
			Name:          "Lichess OTB Master Broadcasts — Aug 2026",
			Provider:      "Lichess Broadcast",
			Category:      "OTB Master",
			Format:        "pgn.zst",
			URL:           "https://database.lichess.org/broadcast/lichess_db_broadcast_2026-08.pgn.zst",
			SizeEstimate:  "~20 MB (.pgn.zst stream)",
			GamesEstimate: "~50,000 OTB Broadcast games",
			Description:   "Official FIDE & top OTB tournament broadcasts relayed on Lichess (Aug 2026).",
			SupportsLimit: true,
		},

		// 3. Lichess Elite Database (database.nikonoel.fr — 2400+ vs 2200+)
		{
			ID:            "lichess-elite-2025-11",
			Name:          "Lichess Elite Database (2400+ Elo) — Nov 2025",
			Provider:      "Lichess Elite",
			Category:      "Online Master (2400+)",
			Format:        "zip-pgn",
			URL:           "https://database.nikonoel.fr/lichess_elite_2025-11.zip",
			SizeEstimate:  "~65 MB (.zip)",
			GamesEstimate: "~280,000 Elite 2400+ games",
			Description:   "NikoNoel's curated Lichess Elite Database containing only games between 2400+ and 2200+ players (no bullet).",
			SupportsLimit: true,
		},
		{
			ID:            "lichess-elite-2025-10",
			Name:          "Lichess Elite Database (2400+ Elo) — Oct 2025",
			Provider:      "Lichess Elite",
			Category:      "Online Master (2400+)",
			Format:        "zip-pgn",
			URL:           "https://database.nikonoel.fr/lichess_elite_2025-10.zip",
			SizeEstimate:  "~65 MB (.zip)",
			GamesEstimate: "~280,000 Elite 2400+ games",
			Description:   "NikoNoel's curated Lichess Elite Database (2400+ vs 2200+ classical/rapid/blitz).",
			SupportsLimit: true,
		},

		// 4. Lichess Open Database Live Zstd Stream (database.lichess.org/standard)
		{
			ID:            "lichess-open-2026-09",
			Name:          "Lichess Open Rated Database Stream — Sep 2026",
			Provider:      "Lichess Open",
			Category:      "Lichess Monthly Stream",
			Format:        "pgn.zst",
			URL:           "https://database.lichess.org/standard/lichess_db_standard_rated_2026-09.pgn.zst",
			SizeEstimate:  "Live HTTP Zstd Stream",
			GamesEstimate: "90M+ games (Stream top N / Elo filter)",
			Description:   "Streams directly from database.lichess.org using on-the-fly zstd decompression! Set Max Games & Min Elo to grab thousands of master games in seconds without downloading 30 GB.",
			SupportsLimit: true,
		},
	}
}

// MegaDatabaseClient handles downloading, streaming, decompressing (.pgn.zst, .zip, .7z), and importing big databases.
type MegaDatabaseClient struct {
	HTTPClient *http.Client
	UserAgent  string
	Catalog    []MegaDatabaseSource
}

// NewMegaDatabaseClient creates a new MegaDatabaseClient.
func NewMegaDatabaseClient() *MegaDatabaseClient {
	return &MegaDatabaseClient{
		HTTPClient: &http.Client{
			Timeout: 0, // Streaming large archives can take several minutes; context cancellation controls lifecycle.
		},
		UserAgent: "NewScid/1.3 (Modern Chess Database; https://github.com/chrisfair/newscid)",
		Catalog:   DefaultMegaDatabaseCatalog(),
	}
}

// GetCatalog returns the available mega database sources.
func (c *MegaDatabaseClient) GetCatalog() []MegaDatabaseSource {
	return c.Catalog
}

// FindSource looks up a source by ID.
func (c *MegaDatabaseClient) FindSource(id string) (MegaDatabaseSource, bool) {
	for _, s := range c.Catalog {
		if s.ID == id {
			return s, true
		}
	}
	return MegaDatabaseSource{}, false
}

// FetchLichessFeed fetches the latest monthly URLs from database.lichess.org (e.g. "broadcast" or "standard").
func (c *MegaDatabaseClient) FetchLichessFeed(ctx context.Context, feedType string, limit int) ([]string, error) {
	if feedType != "broadcast" && feedType != "standard" {
		feedType = "broadcast"
	}
	if limit <= 0 {
		limit = 12
	}
	listURL := fmt.Sprintf("https://database.lichess.org/%s/list.txt", feedType)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lichess feed returned HTTP %d", resp.StatusCode)
	}

	var urls []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && strings.HasPrefix(line, "https://") {
			urls = append(urls, line)
			if len(urls) >= limit {
				break
			}
		}
	}
	return urls, scanner.Err()
}

// DownloadAndImport downloads or streams the selected big database and imports it into targetDB
// (or extracts a native .si4 archive into destDir).
func (c *MegaDatabaseClient) DownloadAndImport(
	ctx context.Context,
	opts MegaImportOptions,
	targetDB *scid.Database,
	destDir string,
	progressFn func(MegaImportProgress),
) (*MegaImportResult, error) {
	start := time.Now()

	var src MegaDatabaseSource
	if strings.TrimSpace(opts.CustomURL) != "" {
		customURL := strings.TrimSpace(opts.CustomURL)
		src = MegaDatabaseSource{
			ID:            "custom",
			Name:          filepath.Base(customURL),
			Provider:      "Custom URL",
			Format:        detectFormatFromURL(customURL),
			URL:           customURL,
			SupportsLimit: true,
		}
	} else {
		found, ok := c.FindSource(opts.SourceID)
		if !ok {
			return nil, fmt.Errorf("unknown mega database source ID: %s", opts.SourceID)
		}
		src = found
	}

	res := &MegaImportResult{
		SourceID:   src.ID,
		SourceName: src.Name,
		URL:        src.URL,
	}

	// Default MaxGames safety cap on Lichess Open 90M-game monthly dump if user left MaxGames == 0
	maxGames := opts.MaxGames
	if src.Provider == "Lichess Open" && maxGames <= 0 {
		maxGames = 25000
	}

	emit := func(phase, msg string, bytesDL, totalB int64, scanned, imported, pct int) {
		if progressFn != nil {
			progressFn(MegaImportProgress{
				Phase:           phase,
				SourceName:      src.Name,
				BytesDownloaded: bytesDL,
				TotalBytes:      totalB,
				GamesScanned:    scanned,
				GamesImported:   imported,
				Percent:         pct,
				Message:         msg,
			})
		}
	}

	switch src.Format {
	case "pgn.zst":
		// On-the-fly HTTP + Zstandard streaming decompression!
		if targetDB == nil || !targetDB.IsOpen {
			return nil, fmt.Errorf("target database is not open")
		}
		emit("streaming", fmt.Sprintf("Connecting to %s zstd stream...", src.Name), 0, 0, 0, 0, 5)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", c.UserAgent)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("connecting to %s: %w", src.URL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("server returned HTTP %d for %s", resp.StatusCode, src.URL)
		}

		counter := &countingReader{r: resp.Body}
		zstdDecoder, err := zstd.NewReader(counter)
		if err != nil {
			return nil, fmt.Errorf("initializing zstd stream decoder: %w", err)
		}
		defer zstdDecoder.Close()

		scanned, imported, err := streamImportPGN(ctx, zstdDecoder, targetDB, maxGames, opts.MinElo, func(sc, imp int) {
			pct := 50
			if maxGames > 0 && imp <= maxGames {
				pct = 10 + (imp*85)/maxGames
			}
			emit("streaming", fmt.Sprintf("Streaming %s: %d games imported (%d scanned)...", src.Name, imp, sc), counter.n, resp.ContentLength, sc, imp, pct)
		})
		res.BytesDownloaded = counter.n
		res.GamesScanned = scanned
		res.GamesImported = imported
		res.DurationMs = time.Since(start).Milliseconds()
		if err != nil {
			return res, err
		}
		emit("completed", fmt.Sprintf("Successfully imported %d games from %s!", imported, src.Name), counter.n, resp.ContentLength, scanned, imported, 100)
		return res, nil

	case "pgn":
		if targetDB == nil || !targetDB.IsOpen {
			return nil, fmt.Errorf("target database is not open")
		}
		emit("streaming", fmt.Sprintf("Streaming PGN from %s...", src.Name), 0, 0, 0, 0, 5)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", c.UserAgent)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("server returned HTTP %d for %s", resp.StatusCode, src.URL)
		}

		counter := &countingReader{r: resp.Body}
		scanned, imported, err := streamImportPGN(ctx, counter, targetDB, maxGames, opts.MinElo, func(sc, imp int) {
			emit("streaming", fmt.Sprintf("Importing %s: %d games imported...", src.Name, imp), counter.n, resp.ContentLength, sc, imp, 50)
		})
		res.BytesDownloaded = counter.n
		res.GamesScanned = scanned
		res.GamesImported = imported
		res.DurationMs = time.Since(start).Milliseconds()
		if err != nil {
			return res, err
		}
		emit("completed", fmt.Sprintf("Successfully imported %d games from %s!", imported, src.Name), counter.n, resp.ContentLength, scanned, imported, 100)
		return res, nil

	case "zip-pgn", "7z-pgn", "7z-si4":
		// Download archive to a temporary file with live byte progress, then extract/stream
		tmpFile, totalBytes, err := c.downloadToTempFile(ctx, src.URL, func(dl, total int64) {
			pct := 25
			if total > 0 {
				pct = int((dl * 50) / total)
			}
			emit("downloading", fmt.Sprintf("Downloading %s (%.1f MB)...", src.Name, float64(dl)/(1024*1024)), dl, total, 0, 0, pct)
		})
		if err != nil {
			return nil, err
		}
		defer os.Remove(tmpFile)
		res.BytesDownloaded = totalBytes

		if src.Format == "zip-pgn" {
			if targetDB == nil || !targetDB.IsOpen {
				return nil, fmt.Errorf("target database is not open")
			}
			scanned, imported, err := extractAndImportZip(ctx, tmpFile, targetDB, maxGames, opts.MinElo, func(sc, imp int) {
				emit("importing", fmt.Sprintf("Importing %s: %d games imported (%d scanned)...", src.Name, imp, sc), totalBytes, totalBytes, sc, imp, 75)
			})
			res.GamesScanned = scanned
			res.GamesImported = imported
			res.DurationMs = time.Since(start).Milliseconds()
			if err != nil {
				return res, err
			}
			emit("completed", fmt.Sprintf("Successfully imported %d games from %s!", imported, src.Name), totalBytes, totalBytes, scanned, imported, 100)
			return res, nil
		}

		// 7z-pgn or 7z-si4
		if src.Format == "7z-si4" {
			if strings.TrimSpace(destDir) == "" {
				home, _ := os.UserHomeDir()
				destDir = filepath.Join(home, ".local", "share", "newscid", "databases")
			}
			emit("extracting", fmt.Sprintf("Extracting native Scid .si4 database from %s...", src.Name), totalBytes, totalBytes, 0, 0, 75)
			si4Path, err := extract7zNativeSI4(tmpFile, destDir)
			res.DatabasePath = si4Path
			res.DurationMs = time.Since(start).Milliseconds()
			if err != nil {
				return res, err
			}
			emit("completed", fmt.Sprintf("Extracted Scid database to %s!", filepath.Base(si4Path)), totalBytes, totalBytes, 0, 0, 100)
			return res, nil
		}

		// 7z-pgn
		if targetDB == nil || !targetDB.IsOpen {
			return nil, fmt.Errorf("target database is not open")
		}
		scanned, imported, err := extractAndImport7zPGN(ctx, tmpFile, targetDB, maxGames, opts.MinElo, func(sc, imp int) {
			emit("importing", fmt.Sprintf("Importing %s: %d games imported (%d scanned)...", src.Name, imp, sc), totalBytes, totalBytes, sc, imp, 80)
		})
		res.GamesScanned = scanned
		res.GamesImported = imported
		res.DurationMs = time.Since(start).Milliseconds()
		if err != nil {
			return res, err
		}
		emit("completed", fmt.Sprintf("Successfully imported %d games from %s!", imported, src.Name), totalBytes, totalBytes, scanned, imported, 100)
		return res, nil

	default:
		return nil, fmt.Errorf("unsupported archive format: %s", src.Format)
	}
}

func detectFormatFromURL(u string) string {
	lower := strings.ToLower(u)
	switch {
	case strings.HasSuffix(lower, ".pgn.zst") || strings.HasSuffix(lower, ".zst"):
		return "pgn.zst"
	case strings.HasSuffix(lower, ".zip"):
		return "zip-pgn"
	case strings.HasSuffix(lower, "_si4.7z"):
		return "7z-si4"
	case strings.HasSuffix(lower, ".7z"):
		return "7z-pgn"
	default:
		return "pgn"
	}
}

type countingReader struct {
	r io.Reader
	n int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}

func (c *MegaDatabaseClient) downloadToTempFile(ctx context.Context, url string, progressFn func(dl, total int64)) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("download returned HTTP %d for %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp("", "newscid-mega-*")
	if err != nil {
		return "", 0, err
	}
	tmpPath := tmp.Name()

	buf := make([]byte, 128*1024)
	var downloaded int64
	lastReport := time.Now()

	for {
		select {
		case <-ctx.Done():
			tmp.Close()
			os.Remove(tmpPath)
			return "", downloaded, ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := tmp.Write(buf[:n]); wErr != nil {
				tmp.Close()
				os.Remove(tmpPath)
				return "", downloaded, wErr
			}
			downloaded += int64(n)
			if progressFn != nil && time.Since(lastReport) >= 250*time.Millisecond {
				progressFn(downloaded, resp.ContentLength)
				lastReport = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return "", downloaded, readErr
		}
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", downloaded, err
	}
	if progressFn != nil {
		progressFn(downloaded, resp.ContentLength)
	}
	return tmpPath, downloaded, nil
}

// streamImportPGN reads a stream of PGN games one game at a time, filters by MinElo if requested,
// and imports into targetDB in batches of 250 games without loading the entire archive into memory.
func streamImportPGN(
	ctx context.Context,
	r io.Reader,
	targetDB *scid.Database,
	maxGames int,
	minElo int,
	progressFn func(scanned, imported int),
) (int, int, error) {
	scanner := bufio.NewScanner(r)
	// Allow up to 2MB per line for deeply annotated PGNs
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)

	var currentGameLines strings.Builder
	inMoves := false
	scanned := 0
	imported := 0
	batch := make([]*chess.Game, 0, 250)

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		stats, err := targetDB.ImportGames(batch)
		if err != nil {
			return err
		}
		imported += stats.Imported
		batch = batch[:0]
		if progressFn != nil {
			progressFn(scanned, imported)
		}
		return nil
	}

	processOneGame := func(rawPGN string) (bool, error) {
		trimmed := strings.TrimSpace(rawPGN)
		if trimmed == "" {
			return false, nil
		}
		scanned++

		// Fast pre-check for MinElo before full move tree parsing if MinElo > 0
		if minElo > 0 && !pgnMeetsMinElo(trimmed, minElo) {
			if scanned%500 == 0 && progressFn != nil {
				progressFn(scanned, imported)
			}
			return false, nil
		}

		games, err := chess.ParsePGNDatabase(strings.NewReader(trimmed))
		if err != nil || len(games) == 0 {
			return false, nil
		}
		for _, g := range games {
			if minElo > 0 && !gameMeetsMinElo(g, minElo) {
				continue
			}
			batch = append(batch, g)
			if maxGames > 0 && (imported+len(batch)) >= maxGames {
				if err := flushBatch(); err != nil {
					return true, err
				}
				return true, nil
			}
			if len(batch) >= 250 {
				if err := flushBatch(); err != nil {
					return true, err
				}
			}
		}
		return false, nil
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			_ = flushBatch()
			return scanned, imported, ctx.Err()
		default:
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && inMoves {
			// A new game's tag section has started; flush the completed game!
			stop, err := processOneGame(currentGameLines.String())
			if err != nil {
				return scanned, imported, err
			}
			if stop {
				return scanned, imported, nil
			}
			currentGameLines.Reset()
			inMoves = false
		} else if trimmed != "" && !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "%") {
			inMoves = true
		}

		currentGameLines.WriteString(line)
		currentGameLines.WriteByte('\n')
	}

	if currentGameLines.Len() > 0 {
		_, err := processOneGame(currentGameLines.String())
		if err != nil {
			return scanned, imported, err
		}
	}
	if err := flushBatch(); err != nil {
		return scanned, imported, err
	}
	return scanned, imported, scanner.Err()
}

func pgnMeetsMinElo(rawPGN string, minElo int) bool {
	// Quick check on [WhiteElo "..."] or [BlackElo "..."]
	for _, line := range strings.Split(rawPGN, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			break
		}
		if strings.HasPrefix(line, "[WhiteElo \"") || strings.HasPrefix(line, "[BlackElo \"") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				if elo, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && elo >= minElo {
					return true
				}
			}
		}
	}
	return false
}

func gameMeetsMinElo(g *chess.Game, minElo int) bool {
	if g == nil || g.Tags == nil {
		return false
	}
	if w, err := strconv.Atoi(strings.TrimSpace(g.Tags["WhiteElo"])); err == nil && w >= minElo {
		return true
	}
	if b, err := strconv.Atoi(strings.TrimSpace(g.Tags["BlackElo"])); err == nil && b >= minElo {
		return true
	}
	return false
}

func extractAndImportZip(
	ctx context.Context,
	zipPath string,
	targetDB *scid.Database,
	maxGames int,
	minElo int,
	progressFn func(scanned, imported int),
) (int, int, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, 0, fmt.Errorf("opening zip archive: %w", err)
	}
	defer zr.Close()

	totalScanned := 0
	totalImported := 0

	for _, f := range zr.File {
		if f.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(f.Name), ".pgn") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return totalScanned, totalImported, err
		}
		remaining := 0
		if maxGames > 0 {
			remaining = maxGames - totalImported
			if remaining <= 0 {
				rc.Close()
				break
			}
		}
		sc, imp, err := streamImportPGN(ctx, rc, targetDB, remaining, minElo, func(s, i int) {
			if progressFn != nil {
				progressFn(totalScanned+s, totalImported+i)
			}
		})
		rc.Close()
		totalScanned += sc
		totalImported += imp
		if err != nil {
			return totalScanned, totalImported, err
		}
		if maxGames > 0 && totalImported >= maxGames {
			break
		}
	}
	return totalScanned, totalImported, nil
}

func extractAndImport7zPGN(
	ctx context.Context,
	sevenZipPath string,
	targetDB *scid.Database,
	maxGames int,
	minElo int,
	progressFn func(scanned, imported int),
) (int, int, error) {
	sz, err := sevenzip.OpenReader(sevenZipPath)
	if err != nil {
		return 0, 0, fmt.Errorf("opening 7z archive: %w", err)
	}
	defer sz.Close()

	totalScanned := 0
	totalImported := 0

	for _, f := range sz.File {
		if f.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(f.Name), ".pgn") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return totalScanned, totalImported, err
		}
		remaining := 0
		if maxGames > 0 {
			remaining = maxGames - totalImported
			if remaining <= 0 {
				rc.Close()
				break
			}
		}
		sc, imp, err := streamImportPGN(ctx, rc, targetDB, remaining, minElo, func(s, i int) {
			if progressFn != nil {
				progressFn(totalScanned+s, totalImported+i)
			}
		})
		rc.Close()
		totalScanned += sc
		totalImported += imp
		if err != nil {
			return totalScanned, totalImported, err
		}
		if maxGames > 0 && totalImported >= maxGames {
			break
		}
	}
	return totalScanned, totalImported, nil
}

func extract7zNativeSI4(sevenZipPath string, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}

	// Fast path: use bsdtar if available on system for multi-threaded C extraction, otherwise pure-Go sevenzip
	if bsdtarPath, err := exec.LookPath("bsdtar"); err == nil {
		cmd := exec.Command(bsdtarPath, "-xf", sevenZipPath, "-C", destDir)
		if err := cmd.Run(); err == nil {
			matches, _ := filepath.Glob(filepath.Join(destDir, "*.si4"))
			if len(matches) > 0 {
				return strings.TrimSuffix(matches[0], ".si4"), nil
			}
		}
	}

	sz, err := sevenzip.OpenReader(sevenZipPath)
	if err != nil {
		return "", fmt.Errorf("opening 7z archive: %w", err)
	}
	defer sz.Close()

	var si4BasePath string
	for _, f := range sz.File {
		if f.FileInfo().IsDir() {
			continue
		}
		baseName := filepath.Base(f.Name)
		ext := strings.ToLower(filepath.Ext(baseName))
		if ext != ".si4" && ext != ".sn4" && ext != ".sg4" {
			continue
		}
		outPath := filepath.Join(destDir, baseName)
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		outFile, err := os.Create(outPath)
		if err != nil {
			rc.Close()
			return "", err
		}
		_, copyErr := io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if ext == ".si4" {
			si4BasePath = strings.TrimSuffix(outPath, ".si4")
		}
	}

	if si4BasePath == "" {
		return "", fmt.Errorf("no .si4 database found in 7z archive")
	}
	return si4BasePath, nil
}
