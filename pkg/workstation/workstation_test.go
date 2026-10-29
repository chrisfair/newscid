package workstation

import (
	"bytes"
	"errors"
	"newscid/pkg/database/scid"
	"newscid/pkg/online"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockOnlineImporter struct {
	res *online.SyncResult
	err error
}

func (m *mockOnlineImporter) ImportPlayerGames(opts online.ImportPlayerOptions, activeDB *scid.Database) (*online.SyncResult, error) {
	return m.res, m.err
}


const sampleWorkstationPGN = `
[Event "Workstation Open"]
[Site "Chicago"]
[Date "2026.10.14"]
[Round "1"]
[White "AlphaPlayer"]
[Black "BetaPlayer"]
[Result "1-0"]
[WhiteElo "1800"]
[BlackElo "1750"]
[ECO "C50"]
[TimeControl "600+0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6 5. d4 exd4 6. cxd4 Bb4+ 7. Nc3 Nxe4 8. O-O Bxc3 9. bxc3 d5 10. Bd3 O-O 11. Qc2 Bf5 12. Ba3 Re8 13. Rfe1 Bg6 14. Nd2 Nxd2 15. Qxd2 Rxe1+ 16. Rxe1 Qd7 1-0

[Event "Workstation Open"]
[Site "Chicago"]
[Date "2026.10.15"]
[Round "2"]
[White "BetaPlayer"]
[Black "AlphaPlayer"]
[Result "0-1"]
[WhiteElo "1750"]
[BlackElo "1800"]
[ECO "B20"]
[TimeControl "180+2"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 6. Be2 e5 7. Nb3 Be7 8. O-O O-O 9. Be3 Be6 10. f3 Nbd7 11. Qd2 b5 12. Rfd1 Nb6 13. Qe1 Qc7 14. Qf2 Nc4 15. Bxc4 bxc4 16. Nd2 Rab8 0-1
`

func TestWorkstationCLIs(t *testing.T) {
	tmpDir := t.TempDir()
	dbBase := filepath.Join(tmpDir, "workstation_db")
	pgnFile := filepath.Join(tmpDir, "sample.pgn")
	if err := os.WriteFile(pgnFile, []byte(sampleWorkstationPGN), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var out, errBuf bytes.Buffer

	// Version flags
	if code := RunImportCLI([]string{"-version"}, &out, &errBuf); code != 0 {
		t.Errorf("RunImportCLI -version failed: %d", code)
	}
	if code := RunFairPlayCLI([]string{"-version"}, &out, &errBuf); code != 0 {
		t.Errorf("RunFairPlayCLI -version failed: %d", code)
	}
	if code := RunScoutCLI([]string{"-version"}, &out, &errBuf); code != 0 {
		t.Errorf("RunScoutCLI -version failed: %d", code)
	}

	// Argument validation errors
	if code := RunImportCLI([]string{"-badflag"}, &out, &errBuf); code != 2 {
		t.Errorf("expected exit code 2 for bad flag, got %d", code)
	}
	if code := RunImportCLI([]string{}, &out, &errBuf); code != 2 {
		t.Errorf("expected exit code 2 for missing -db, got %d", code)
	}
	if code := RunFairPlayCLI([]string{"-badflag"}, &out, &errBuf); code != 2 {
		t.Errorf("expected exit code 2 for bad flag, got %d", code)
	}
	if code := RunFairPlayCLI([]string{"-db", dbBase}, &out, &errBuf); code != 2 {
		t.Errorf("expected exit code 2 for missing -player, got %d", code)
	}
	if code := RunScoutCLI([]string{"-badflag"}, &out, &errBuf); code != 2 {
		t.Errorf("expected exit code 2 for bad flag, got %d", code)
	}
	if code := RunScoutCLI([]string{}, &out, &errBuf); code != 2 {
		t.Errorf("expected exit code 2 for missing -db, got %d", code)
	}

	// Missing database errors for read-only tools
	if code := RunFairPlayCLI([]string{"-db", filepath.Join(tmpDir, "nonexistent"), "-player", "AlphaPlayer"}, &out, &errBuf); code != 1 {
		t.Errorf("expected exit code 1 for nonexistent db, got %d", code)
	}
	if code := RunScoutCLI([]string{"-db", filepath.Join(tmpDir, "nonexistent"), "-player", "AlphaPlayer"}, &out, &errBuf); code != 1 {
		t.Errorf("expected exit code 1 for nonexistent db, got %d", code)
	}

	// Import PGN into new database, then re-import (testing incremental signature deduplication) + compact
	out.Reset()
	if code := RunImportCLI([]string{"-db", dbBase, "-pgn", pgnFile, "-dedup=true", "-compact=true"}, &out, &errBuf); code != 0 {
		t.Fatalf("RunImportCLI PGN failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "imported=2") {
		t.Errorf("expected imported=2 in output, got %q", out.String())
	}

	// Re-import same PGN (should skip existing)
	out.Reset()
	if code := RunImportCLI([]string{"-db", dbBase, "-pgn", pgnFile}, &out, &errBuf); code != 0 {
		t.Fatalf("RunImportCLI re-import failed: %d", code)
	}
	if !strings.Contains(out.String(), "imported=0 skipped_existing=2") {
		t.Errorf("expected imported=0 skipped_existing=2, got %q", out.String())
	}

	// Deduplicate-only mode on existing DB
	if code := RunImportCLI([]string{"-db", dbBase, "-dedup=true", "-compact=true"}, &out, &errBuf); code != 0 {
		t.Errorf("RunImportCLI dedup-only failed: %d", code)
	}

	// Missing PGN file error
	if code := RunImportCLI([]string{"-db", dbBase, "-pgn", filepath.Join(tmpDir, "missing.pgn")}, &out, &errBuf); code != 1 {
		t.Errorf("expected exit code 1 for missing pgn file, got %d", code)
	}

	// RunFairPlayCLI (Markdown and JSON modes, plus missing player error)
	out.Reset()
	if code := RunFairPlayCLI([]string{"-db", dbBase, "-player", "AlphaPlayer"}, &out, &errBuf); code != 0 {
		t.Fatalf("RunFairPlayCLI markdown failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "AlphaPlayer") {
		t.Errorf("expected AlphaPlayer in fairplay markdown output")
	}

	out.Reset()
	if code := RunFairPlayCLI([]string{"-db", dbBase, "-player", "AlphaPlayer", "-json"}, &out, &errBuf); code != 0 {
		t.Fatalf("RunFairPlayCLI JSON failed: %d", code)
	}
	if !strings.Contains(out.String(), `"playerName": "AlphaPlayer"`) {
		t.Errorf("expected JSON playerName in fairplay output")
	}

	if code := RunFairPlayCLI([]string{"-db", dbBase, "-player", "NobodyHere"}, &out, &errBuf); code != 1 {
		t.Errorf("expected exit code 1 for missing player in fairplay")
	}

	// RunScoutCLI: Dossier (Markdown & JSON), Exploit Opening (Markdown, JSON, & Polyglot book), and Crosstable (Markdown & JSON)
	out.Reset()
	if code := RunScoutCLI([]string{"-db", dbBase, "-player", "AlphaPlayer"}, &out, &errBuf); code != 0 {
		t.Fatalf("RunScoutCLI dossier failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Opponent Scouting Dossier: AlphaPlayer") {
		t.Errorf("unexpected scout dossier output: %q", out.String())
	}

	out.Reset()
	if code := RunScoutCLI([]string{"-db", dbBase, "-player", "AlphaPlayer", "-json"}, &out, &errBuf); code != 0 {
		t.Fatalf("RunScoutCLI dossier JSON failed: %d", code)
	}

	bookPath := filepath.Join(tmpDir, "exploit_beta.bin")
	out.Reset()
	if code := RunScoutCLI([]string{"-db", dbBase, "-player", "BetaPlayer", "-exploit-color", "white", "-book-out", bookPath}, &out, &errBuf); code != 0 {
		t.Fatalf("RunScoutCLI exploit failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Opening Exploit Report: BetaPlayer") {
		t.Errorf("unexpected exploit report output: %q", out.String())
	}

	out.Reset()
	if code := RunScoutCLI([]string{"-db", dbBase, "-player", "BetaPlayer", "-exploit-color", "white", "-json"}, &out, &errBuf); code != 0 {
		t.Fatalf("RunScoutCLI exploit JSON failed: %d", code)
	}

	if code := RunScoutCLI([]string{"-db", dbBase, "-exploit-color", "white"}, &out, &errBuf); code != 2 {
		t.Errorf("expected exit code 2 when -player is missing with -exploit-color")
	}
	if code := RunScoutCLI([]string{"-db", dbBase}, &out, &errBuf); code != 2 {
		t.Errorf("expected exit code 2 when neither -player nor -crosstable is provided")
	}

	out.Reset()
	if code := RunScoutCLI([]string{"-db", dbBase, "-crosstable", "Workstation Open"}, &out, &errBuf); code != 0 {
		t.Fatalf("RunScoutCLI crosstable failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "AlphaPlayer") {
		t.Errorf("expected AlphaPlayer in crosstable output: %q", out.String())
	}

	out.Reset()
	if code := RunScoutCLI([]string{"-db", dbBase, "-crosstable", "Workstation Open", "-json"}, &out, &errBuf); code != 0 {
		t.Fatalf("RunScoutCLI crosstable JSON failed: %d", code)
	}

	if _, err := openOrCreateScidDB("", true, ""); err == nil {
		t.Errorf("expected error for empty dbPath")
	}

	// Online import via NewSyncManagerFn
	if sm := NewSyncManagerFn(); sm == nil {
		t.Errorf("expected default NewSyncManagerFn to return non-nil SyncManager")
	}
	origFn := NewSyncManagerFn
	defer func() { NewSyncManagerFn = origFn }()

	NewSyncManagerFn = func() OnlineImporter {
		return &mockOnlineImporter{
			res: &online.SyncResult{Imported: 5, DuplicatesSkipped: 3},
		}
	}
	out.Reset()
	if code := RunImportCLI([]string{"-db", dbBase, "-lichess", "AlphaPlayer", "-chesscom", "AlphaPlayer"}, &out, &errBuf); code != 0 {
		t.Fatalf("RunImportCLI online failed (%d): %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "imported=5 skipped_existing=3") {
		t.Errorf("unexpected online import output: %q", out.String())
	}

	NewSyncManagerFn = func() OnlineImporter {
		return &mockOnlineImporter{
			err: errors.New("network unreachable"),
		}
	}
	if code := RunImportCLI([]string{"-db", dbBase, "-lichess", "AlphaPlayer"}, &out, &errBuf); code != 1 {
		t.Errorf("expected exit code 1 when online sync returns error, got %d", code)
	}

	// Error path when book-out is an unwritable directory & exploit-color black with refutation lines
	if code := RunScoutCLI([]string{"-db", dbBase, "-player", "BetaPlayer", "-exploit-color", "black", "-book-out", bookPath}, &out, &errBuf); code != 0 {
		t.Errorf("expected exit code 0 for BetaPlayer black exploit, got %d", code)
	}
	if code := RunScoutCLI([]string{"-db", dbBase, "-player", "BetaPlayer", "-exploit-color", "white", "-book-out", tmpDir}, &out, &errBuf); code != 1 {
		t.Errorf("expected exit code 1 when book-out is a directory, got %d", code)
	}

	// Corrupt database file and uncreatable directory error paths and default description
	corruptBase := filepath.Join(tmpDir, "corrupt_db")
	_ = os.WriteFile(corruptBase+".si4", []byte("corrupt"), 0o644)
	if code := RunImportCLI([]string{"-db", corruptBase}, &out, &errBuf); code != 1 {
		t.Errorf("expected exit code 1 for corrupt db in RunImportCLI, got %d", code)
	}
	if code := RunImportCLI([]string{"-db", filepath.Join(tmpDir, "nodir", "sub", "db")}, &out, &errBuf); code != 1 {
		t.Errorf("expected exit code 1 for uncreatable db path in RunImportCLI, got %d", code)
	}
	if db2, err := openOrCreateScidDB(filepath.Join(tmpDir, "default_desc_db"), true, ""); err != nil {
		t.Errorf("unexpected error creating default_desc_db: %v", err)
	} else {
		_ = db2.Close()
	}
}



