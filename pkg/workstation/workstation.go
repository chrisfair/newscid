package workstation

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/fairplay"
	"newscid/pkg/online"
	"newscid/pkg/persona"
	"newscid/pkg/scout"
	"newscid/pkg/tournament"
	"newscid/pkg/version"
)

// ToolID is a compact integer constant identifying the workstation companion program.
type ToolID uint8

const (
	ToolImport ToolID = iota
	ToolFairPlay
	ToolScout
)

// ImportOpID is an integer constant for newscid-import operation modes.
type ImportOpID uint8

const (
	OpImportPGN ImportOpID = iota
	OpImportOnline
	OpDeduplicateOnly
)

// ScoutOpID is an integer constant for newscid-scout operation modes.
type ScoutOpID uint8

const (
	OpScoutDossier ScoutOpID = iota
	OpExploitOpening
	OpCrosstable
)

// openOrCreateScidDB opens an existing .si4 database or creates a new one if createIfMissing is true.
func openOrCreateScidDB(dbPath string, createIfMissing bool, description string) (*scid.Database, error) {
	clean := strings.TrimSuffix(strings.TrimSpace(dbPath), ".si4")
	if clean == "" {
		return nil, fmt.Errorf("database path (--db) is required")
	}
	db := scid.NewDatabase()
	if _, err := os.Stat(clean + ".si4"); err == nil {
		if err := db.Open(clean); err != nil {
			return nil, fmt.Errorf("opening database %s: %w", clean, err)
		}
		return db, nil
	}
	if !createIfMissing {
		return nil, fmt.Errorf("database not found: %s.si4", clean)
	}
	if description == "" {
		description = "NewScid Workstation Database"
	}
	if err := db.Create(clean, description); err != nil {
		return nil, fmt.Errorf("creating database %s: %w", clean, err)
	}
	return db, nil
}

// OnlineImporter abstracts online player game importing for testing.
type OnlineImporter interface {
	ImportPlayerGames(opts online.ImportPlayerOptions, activeDB *scid.Database) (*online.SyncResult, error)
}

// NewSyncManagerFn creates an OnlineImporter instance (overridable in tests).
var NewSyncManagerFn = func() OnlineImporter {
	return online.NewSyncManager(nil)
}


func reportErr(stderr io.Writer, prefix string, err error) int {
	fmt.Fprintf(stderr, "%s: %v\n", prefix, err)
	return 1
}

func writeJSON(stdout io.Writer, v any) {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// RunImportCLI executes the standalone `newscid-import` workstation program.
func RunImportCLI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("newscid-import", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dbPath := fs.String("db", "", "Target SCID database path (.si4)")
	pgnPath := fs.String("pgn", "", "PGN file path to import into the target database")
	lichessUser := fs.String("lichess", "", "Lichess.org username for incremental + backfill game sync")
	chesscomUser := fs.String("chesscom", "", "Chess.com username for incremental + backfill game sync")
	maxGames := fs.Int("max", 250, "Maximum new games to fetch when syncing online")
	dedup := fs.Bool("dedup", true, "Run signature-based duplicate removal after import")
	compact := fs.Bool("compact", false, "Physically compact deleted records after import/dedup")
	showVersion := fs.Bool("version", false, "Display version information and exit")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version.String())
		return 0
	}
	if strings.TrimSpace(*dbPath) == "" {
		fmt.Fprintln(stderr, "Usage: newscid-import -db <database.si4> [-pgn <file.pgn>] [-lichess <user>] [-chesscom <user>] [-dedup] [-compact]")
		return 2
	}

	db, err := openOrCreateScidDB(*dbPath, true, "NewScid Imported Database")
	if err != nil {
		return reportErr(stderr, "Error", err)
	}
	defer db.Close()

	var op ImportOpID = OpDeduplicateOnly
	if strings.TrimSpace(*pgnPath) != "" {
		op = OpImportPGN
	} else if strings.TrimSpace(*lichessUser) != "" || strings.TrimSpace(*chesscomUser) != "" {
		op = OpImportOnline
	}

	importedCount := 0
	skippedCount := 0

	switch op {
	case OpImportPGN:
		f, err := os.Open(strings.TrimSpace(*pgnPath))
		if err != nil {
			return reportErr(stderr, "Error opening PGN file", err)
		}
		games, err := chess.ParsePGNDatabase(f)
		_ = f.Close()
		if err != nil {
			return reportErr(stderr, "Error parsing PGN file", err)
		}
		existing := db.BuildGameSignatureSet()
		newGames, skipped := existing.FilterNewGames(games)
		skippedCount += skipped
		if len(newGames) > 0 {
			stats, err := db.ImportGames(newGames)
			if err != nil {
				return reportErr(stderr, "Error importing PGN games", err)
			}
			importedCount += stats.Imported
		}
	case OpImportOnline:
		sm := NewSyncManagerFn()
		res, err := sm.ImportPlayerGames(online.ImportPlayerOptions{
			LichessUsername:  strings.TrimSpace(*lichessUser),
			ChessComUsername: strings.TrimSpace(*chesscomUser),
			TargetMode:       online.TargetActiveBase,
			MaxGames:         *maxGames,
			AutoDeduplicate:  *dedup,
		}, db)
		if err != nil {
			return reportErr(stderr, "Online sync error", err)
		}
		importedCount += res.Imported
		skippedCount += res.DuplicatesSkipped
	case OpDeduplicateOnly:
		// Will execute deduplication/compaction below
	}

	if *dedup && op != OpImportOnline {
		if removed, err := db.DeleteDuplicates(scid.DuplicateCriteria{ExactMoves: true}); err == nil {
			skippedCount += removed
		}
	}
	if *compact {
		_, _ = db.Compact()
	}

	fmt.Fprintf(stdout, "newscid-import complete: imported=%d skipped_existing=%d total_games=%d db=%s.si4\n",
		importedCount, skippedCount, db.NumGames(), db.BasePath)
	return 0
}

// RunFairPlayCLI executes the standalone `newscid-fairplay` workstation program.
func RunFairPlayCLI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("newscid-fairplay", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dbPath := fs.String("db", "", "SCID database path (.si4) to analyze")
	player := fs.String("player", "", "Player name or online username to inspect")
	maxGames := fs.Int("max", 300, "Maximum games to analyze")
	asJSON := fs.Bool("json", false, "Output full statistical dossier as JSON instead of Markdown")
	showVersion := fs.Bool("version", false, "Display version information and exit")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version.String())
		return 0
	}
	if strings.TrimSpace(*dbPath) == "" || strings.TrimSpace(*player) == "" {
		fmt.Fprintln(stderr, "Usage: newscid-fairplay -db <database.si4> -player <PlayerName> [-max 300] [-json]")
		return 2
	}

	db, err := openOrCreateScidDB(*dbPath, false, "")
	if err != nil {
		return reportErr(stderr, "Error", err)
	}
	defer db.Close()

	dossier, err := fairplay.AnalyzePlayerFairPlay(db, *player, *maxGames)
	if err != nil {
		return reportErr(stderr, "Fair-Play analysis error", err)
	}

	if *asJSON {
		writeJSON(stdout, dossier)
	} else {
		fmt.Fprint(stdout, dossier.ReportMarkdown)
	}
	return 0
}

// RunScoutCLI executes the standalone `newscid-scout` workstation program for opponent scouting,
// opening vulnerability exploitation (with Polyglot .bin book export), and tournament crosstables.
func RunScoutCLI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("newscid-scout", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dbPath := fs.String("db", "", "SCID database path (.si4)")
	player := fs.String("player", "", "Opponent player name for scouting or opening exploitation")
	exploitColor := fs.String("exploit-color", "", "Generate Anti-Persona opening exploit tree for color (white|black)")
	bookOut := fs.String("book-out", "", "Optional output path for generated Polyglot (.bin) exploit book")
	crosstableEvent := fs.String("crosstable", "", "Generate tournament crosstable for the specified Event name")
	asJSON := fs.Bool("json", false, "Output report as JSON")
	showVersion := fs.Bool("version", false, "Display version information and exit")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version.String())
		return 0
	}
	if strings.TrimSpace(*dbPath) == "" {
		fmt.Fprintln(stderr, "Usage: newscid-scout -db <database.si4> [-player <Name>] [-exploit-color <white|black>] [-book-out <out.bin>] [-crosstable <EventName>] [-json]")
		return 2
	}

	db, err := openOrCreateScidDB(*dbPath, false, "")
	if err != nil {
		return reportErr(stderr, "Error", err)
	}
	defer db.Close()

	var op ScoutOpID = OpScoutDossier
	if strings.TrimSpace(*crosstableEvent) != "" {
		op = OpCrosstable
	} else if strings.TrimSpace(*exploitColor) != "" {
		op = OpExploitOpening
	}

	switch op {
	case OpCrosstable:
		ct, err := tournament.CalculateCrosstable(db, strings.TrimSpace(*crosstableEvent), "", "")
		if err != nil {
			return reportErr(stderr, "Crosstable error", err)
		}
		if *asJSON {
			writeJSON(stdout, ct)
		} else {
			fmt.Fprintln(stdout, ct.CrossTableText)
		}
		return 0

	case OpExploitOpening:
		if strings.TrimSpace(*player) == "" {
			fmt.Fprintln(stderr, "Error: -player is required with -exploit-color")
			return 2
		}
		cfg := persona.AntiPersonaConfig{
			DisplayName:          strings.TrimSpace(*player),
			Identifiers:          []string{strings.TrimSpace(*player)},
			TargetColor:          strings.ToLower(strings.TrimSpace(*exploitColor)),
			MaxDepth:             16,
			MinGames:             1,
			MinVulnerabilityRate: 40.0,
		}
		rep, err := persona.EvaluatePersonaDatabase(db, cfg)
		if err != nil {
			return reportErr(stderr, "Exploit analysis error", err)
		}
		if strings.TrimSpace(*bookOut) != "" {
			if err := persona.SaveAntiPersonaBook(rep, strings.TrimSpace(*bookOut)); err != nil {
				return reportErr(stderr, "Polyglot book export error", err)
			}
		}
		if *asJSON {
			writeJSON(stdout, rep)
		} else {
			fmt.Fprintf(stdout, "# Opening Exploit Report: %s (%s)\nGames Analyzed: %d | Vulnerable Nodes: %d\n",
				rep.DisplayName, rep.TargetColor, rep.TotalGamesAnalyzed, len(rep.Vulnerabilities))
			for _, line := range rep.TopRefutationLines {
				fmt.Fprintf(stdout, "- %s\n", line)
			}
		}
		return 0

	default:
		if strings.TrimSpace(*player) == "" {
			fmt.Fprintln(stderr, "Error: -player or -crosstable is required")
			return 2
		}
		summaries, _, err := db.GetGameSummaries(0, 100000, scid.SearchCriteria{})
		if err != nil {
			return reportErr(stderr, "Database summary error", err)
		}
		rep := scout.GenerateReport(strings.TrimSpace(*player), summaries)
		if *asJSON {
			writeJSON(stdout, rep)
		} else {
			fmt.Fprintf(stdout, "# Opponent Scouting Dossier: %s\nTotal Games: %d (White: %d [%.1f%%], Black: %d [%.1f%%])\n",
				rep.PlayerName, rep.TotalGames, rep.WhiteRecord.Total, rep.WhiteRecord.WinRate, rep.BlackRecord.Total, rep.BlackRecord.WinRate)
			for _, r := range rep.RepertoireRecommendations {
				fmt.Fprintf(stdout, "- Prep: %s\n", r)
			}
		}
		return 0
	}
}

