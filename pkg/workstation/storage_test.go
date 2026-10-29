package workstation

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetCategoryMethods(t *testing.T) {
	for _, cat := range allCategories {
		if cat.Key() == "unknown" || cat.Key() == "" {
			t.Errorf("unexpected Key() for category %d: %q", cat, cat.Key())
		}
		if cat.Label() == "Unknown Category" || cat.Label() == "" {
			t.Errorf("unexpected Label() for category %d: %q", cat, cat.Label())
		}
		if cat.Description() == "Unrecognized storage category" || cat.Description() == "" {
			t.Errorf("unexpected Description() for category %d: %q", cat, cat.Description())
		}
		parsed, err := ParseAssetCategory(cat.Key())
		if err != nil || parsed != cat {
			t.Errorf("ParseAssetCategory(%q) = (%v, %v), want %v", cat.Key(), parsed, err, cat)
		}
	}

	unknown := AssetCategoryID(255)
	if unknown.Key() != "unknown" {
		t.Errorf("expected unknown Key(), got %q", unknown.Key())
	}
	if unknown.Label() != "Unknown Category" {
		t.Errorf("expected Unknown Category Label(), got %q", unknown.Label())
	}
	if unknown.Description() != "Unrecognized storage category" {
		t.Errorf("expected Unrecognized storage category Description(), got %q", unknown.Description())
	}

	aliases := map[string]AssetCategoryID{
		"database":      CategoryDatabases,
		"persona":       CategoryPersonas,
		"engine":        CategoryEngines,
		"opening":       CategoryOpeningBooks,
		"syzygy":        CategoryEndgameBooks,
		"repertoire":    CategoryRepertoires,
		"settings":      CategoryConfig,
		"backup":        CategoryBackups,
	}
	for k, want := range aliases {
		got, err := ParseAssetCategory(k)
		if err != nil || got != want {
			t.Errorf("ParseAssetCategory(%q) = (%v, %v), want %v", k, got, err, want)
		}
	}
	if _, err := ParseAssetCategory("not_a_category"); err == nil {
		t.Errorf("expected error for invalid category key")
	}
}

func TestFormatByteSize(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}
	for _, tc := range cases {
		if got := FormatByteSize(tc.bytes); got != tc.want {
			t.Errorf("FormatByteSize(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestDefaultDirsAndInspectStorage(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, "xdg_data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, "xdg_config"))

	dataRoot, cfgRoot := DefaultWorkstationDirs()
	if !strings.Contains(dataRoot, "newscid") || !strings.Contains(cfgRoot, "newscid") {
		t.Fatalf("unexpected default dirs: data=%q cfg=%q", dataRoot, cfgRoot)
	}
	leg := DefaultLegacyConfigDir()
	if !strings.Contains(leg, ".newscid") {
		t.Errorf("unexpected legacy dir: %q", leg)
	}

	// Also test without XDG_DATA_HOME set
	t.Setenv("XDG_DATA_HOME", "")
	dataRoot2, _ := DefaultWorkstationDirs()
	if !strings.Contains(dataRoot2, filepath.Join(".local", "share", "newscid")) {
		t.Errorf("expected .local/share/newscid fallback, got %q", dataRoot2)
	}

	// InspectStorage with empty args uses DefaultWorkstationDirs
	rep, err := InspectStorage("", "")
	if err != nil {
		t.Fatalf("InspectStorage failed: %v", err)
	}
	if len(rep.Categories) != 8 {
		t.Errorf("expected 8 categories, got %d", len(rep.Categories))
	}
	if rep.TotalFiles != 0 || rep.TotalBytes != 0 {
		t.Errorf("expected empty initial workstation storage, got files=%d bytes=%d", rep.TotalFiles, rep.TotalBytes)
	}
}

func TestImportAssetToWorkstation(t *testing.T) {
	tmpDir := t.TempDir()
	dataRoot := filepath.Join(tmpDir, "data")
	cfgRoot := filepath.Join(tmpDir, "config")
	extDir := filepath.Join(tmpDir, "external")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a 3-file Scid database bundle (.si4, .sg4, .sn4) in external dir
	scidBase := filepath.Join(extDir, "masters2026")
	if err := os.WriteFile(scidBase+".si4", []byte("SI4_HEADER_DATA"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scidBase+".sg4", []byte("SG4_MOVES_DATA_PAYLOAD"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scidBase+".sn4", []byte("SN4_NAMES_DATA"), 0o644); err != nil {
		t.Fatal(err)
	}

	assetDB, err := ImportAssetToWorkstation("databases", scidBase+".si4", dataRoot, cfgRoot)
	if err != nil {
		t.Fatalf("ImportAssetToWorkstation Scid bundle failed: %v", err)
	}
	if assetDB.Name != "masters2026.si4" {
		t.Errorf("expected masters2026.si4, got %q", assetDB.Name)
	}
	// Verify all 3 companion files were copied into dataRoot/databases/
	for _, ext := range []string{".si4", ".sg4", ".sn4"} {
		copiedPath := filepath.Join(dataRoot, "databases", "masters2026"+ext)
		if _, err := os.Stat(copiedPath); err != nil {
			t.Errorf("expected companion file %s to be copied into NewScid storage: %v", copiedPath, err)
		}
	}

	// Also test importing Scid bundle by base path (without extension) and by copying onto itself (idempotent)
	if _, err := ImportAssetToWorkstation("databases", scidBase, dataRoot, cfgRoot); err != nil {
		t.Errorf("ImportAssetToWorkstation by base path failed: %v", err)
	}
	if _, err := ImportAssetToWorkstation("databases", filepath.Join(dataRoot, "databases", "masters2026.si4"), dataRoot, cfgRoot); err != nil {
		t.Errorf("ImportAssetToWorkstation same-path idempotent copy failed: %v", err)
	}

	// Import an engine binary and verify 0755 executable permissions
	engineSrc := filepath.Join(extDir, "stockfish-18-avx2")
	if err := os.WriteFile(engineSrc, []byte("#!/bin/sh\necho uciok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	assetEng, err := ImportAssetToWorkstation("engines", engineSrc, dataRoot, cfgRoot)
	if err != nil {
		t.Fatalf("ImportAssetToWorkstation engine failed: %v", err)
	}
	st, err := os.Stat(assetEng.Path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o111 == 0 {
		t.Errorf("expected imported engine binary to be executable, got mode %v", st.Mode())
	}

	// Import opening book (.bin), endgame tablebase (.rtbw), persona (.json), repertoire (.json)
	bookSrc := filepath.Join(extDir, "gm_openings.bin")
	_ = os.WriteFile(bookSrc, []byte("POLYGLOT_BOOK_BYTES"), 0o644)
	if _, err := ImportAssetToWorkstation("opening_books", bookSrc, dataRoot, cfgRoot); err != nil {
		t.Errorf("opening book import failed: %v", err)
	}

	tbSrc := filepath.Join(extDir, "KQvK.rtbw")
	_ = os.WriteFile(tbSrc, []byte("SYZYGY_RTBW_BYTES"), 0o644)
	if _, err := ImportAssetToWorkstation("endgame_books", tbSrc, dataRoot, cfgRoot); err != nil {
		t.Errorf("endgame tablebase import failed: %v", err)
	}

	personaSrc := filepath.Join(extDir, "tal_bot.json")
	_ = os.WriteFile(personaSrc, []byte(`{"name":"Mikhail Tal"}`), 0o644)
	if _, err := ImportAssetToWorkstation("personas", personaSrc, dataRoot, cfgRoot); err != nil {
		t.Errorf("persona import failed: %v", err)
	}

	repSrc := filepath.Join(extDir, "najdorf_rep.json")
	_ = os.WriteFile(repSrc, []byte(`{"name":"Najdorf"}`), 0o644)
	if _, err := ImportAssetToWorkstation("repertoires", repSrc, dataRoot, cfgRoot); err != nil {
		t.Errorf("repertoire import failed: %v", err)
	}

	// Inspect storage and verify counts
	report, err := InspectStorage(dataRoot, cfgRoot)
	if err != nil {
		t.Fatalf("InspectStorage failed: %v", err)
	}
	// 3 Scid files + 1 engine + 1 opening book + 1 endgame tb + 1 persona + 1 repertoire = 8 files
	if report.TotalFiles != 8 {
		t.Errorf("expected 8 total files in storage report, got %d", report.TotalFiles)
	}

	// Error paths
	if _, err := ImportAssetToWorkstation("bad_cat", bookSrc, dataRoot, cfgRoot); err == nil {
		t.Errorf("expected error for invalid category")
	}
	if _, err := ImportAssetToWorkstation("databases", "   ", dataRoot, cfgRoot); err == nil {
		t.Errorf("expected error for empty source path")
	}
	if _, err := ImportAssetToWorkstation("databases", filepath.Join(extDir, "missing.pgn"), dataRoot, cfgRoot); err == nil {
		t.Errorf("expected error for missing source file")
	}
	if _, err := ImportAssetToWorkstation("databases", extDir, dataRoot, cfgRoot); err == nil {
		t.Errorf("expected error when source path is a directory")
	}
}

func TestFactoryResetWorkstationWithAndWithoutOptInBackup(t *testing.T) {
	tmpDir := t.TempDir()
	dataRoot := filepath.Join(tmpDir, "data")
	cfgRoot := filepath.Join(tmpDir, "config")
	legacyRoot := filepath.Join(tmpDir, "legacy")

	dirs, err := EnsureWorkstationDirs(dataRoot, cfgRoot)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(legacyRoot, "personas"), 0o755)
	_ = os.MkdirAll(filepath.Join(legacyRoot, "repertoires"), 0o755)

	// Populate sample files across all categories and config dirs
	_ = os.WriteFile(filepath.Join(dirs[CategoryDatabases], "club.si4"), []byte("scid_idx"), 0o644)
	_ = os.WriteFile(filepath.Join(dirs[CategoryPersonas], "kasparov.json"), []byte("{}"), 0o644)
	_ = os.WriteFile(filepath.Join(dirs[CategoryEngines], "stockfish"), []byte("elf"), 0o755)
	_ = os.WriteFile(filepath.Join(dirs[CategoryOpeningBooks], "book.bin"), []byte("bin"), 0o644)
	_ = os.WriteFile(filepath.Join(dirs[CategoryEndgameBooks], "tb.rtbw"), []byte("rtbw"), 0o644)
	_ = os.WriteFile(filepath.Join(dirs[CategoryRepertoires], "e4.json"), []byte("{}"), 0o644)
	_ = os.WriteFile(filepath.Join(cfgRoot, "preferences.json"), []byte(`{"boardTheme":"wood"}`), 0o644)
	_ = os.WriteFile(filepath.Join(legacyRoot, "blunder_vault.json"), []byte(`[]`), 0o644)
	_ = os.WriteFile(filepath.Join(legacyRoot, "personas", "old_bot.json"), []byte(`{}`), 0o644)

	// Test FactoryResetWorkstation WITH Opt-In Backup AND KeepEngines/KeepBooks = true
	resBackup, err := FactoryResetWorkstation(FactoryResetOptions{
		CreateOptInBackup: true,
		KeepEngines:       true,
		KeepBooks:         true,
	}, dataRoot, cfgRoot, legacyRoot)
	if err != nil {
		t.Fatalf("FactoryResetWorkstation with backup failed: %v", err)
	}
	if !resBackup.BackupCreated || resBackup.BackupArchivePath == "" {
		t.Fatalf("expected backup archive to be created")
	}

	// Verify the .tar.gz archive is valid and contains the backed-up files
	f, err := os.Open(resBackup.BackupArchivePath)
	if err != nil {
		t.Fatalf("opening backup archive failed: %v", err)
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader failed: %v", err)
	}
	tr := tar.NewReader(gz)
	archivedFiles := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next failed: %v", err)
		}
		archivedFiles[hdr.Name] = true
	}
	_ = gz.Close()
	_ = f.Close()

	if !archivedFiles["data/databases/club.si4"] || !archivedFiles["config/preferences.json"] || !archivedFiles["legacy/blunder_vault.json"] {
		t.Errorf("expected key files in backup tar.gz, got %v", archivedFiles)
	}

	// Verify engines and books were kept when KeepEngines & KeepBooks were true
	if _, err := os.Stat(filepath.Join(dirs[CategoryEngines], "stockfish")); err != nil {
		t.Errorf("expected stockfish engine to be preserved when KeepEngines=true")
	}
	if _, err := os.Stat(filepath.Join(dirs[CategoryOpeningBooks], "book.bin")); err != nil {
		t.Errorf("expected opening book to be preserved when KeepBooks=true")
	}
	// Verify databases and preferences were wiped
	if _, err := os.Stat(filepath.Join(dirs[CategoryDatabases], "club.si4")); !os.IsNotExist(err) {
		t.Errorf("expected club.si4 to be removed by factory reset")
	}
	if _, err := os.Stat(filepath.Join(cfgRoot, "preferences.json")); !os.IsNotExist(err) {
		t.Errorf("expected preferences.json to be removed by factory reset")
	}

	// Test FactoryResetWorkstation WITHOUT Backup (default opt-out) and full wipe (KeepEngines=false, KeepBooks=false)
	resNoBackup, err := FactoryResetWorkstation(FactoryResetOptions{
		CreateOptInBackup: false,
		KeepEngines:       false,
		KeepBooks:         false,
	}, dataRoot, cfgRoot, legacyRoot)
	if err != nil {
		t.Fatalf("FactoryResetWorkstation without backup failed: %v", err)
	}
	if resNoBackup.BackupCreated {
		t.Errorf("expected BackupCreated=false when CreateOptInBackup=false")
	}
	if _, err := os.Stat(filepath.Join(dirs[CategoryEngines], "stockfish")); !os.IsNotExist(err) {
		t.Errorf("expected stockfish engine to be removed when KeepEngines=false")
	}
	if _, err := os.Stat(filepath.Join(dirs[CategoryOpeningBooks], "book.bin")); !os.IsNotExist(err) {
		t.Errorf("expected opening book to be removed when KeepBooks=false")
	}

	// Test FactoryResetWorkstation with default empty roots (using HOME env)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpHome, "xdg_data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, "xdg_cfg"))
	if _, err := FactoryResetWorkstation(FactoryResetOptions{}, "", "", ""); err != nil {
		t.Errorf("FactoryResetWorkstation with default roots failed: %v", err)
	}
}

func TestStorageErrorPaths(t *testing.T) {
	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "blocker")
	_ = os.WriteFile(blockingFile, []byte("x"), 0o644)

	// EnsureWorkstationDirs fails when dataRoot is a regular file
	if _, err := EnsureWorkstationDirs(blockingFile, filepath.Join(tmpDir, "cfg")); err == nil {
		t.Errorf("expected error when dataRoot is a regular file")
	}
	if _, err := InspectStorage(blockingFile, filepath.Join(tmpDir, "cfg")); err == nil {
		t.Errorf("expected InspectStorage error when dataRoot is a regular file")
	}
	if _, err := ImportAssetToWorkstation("databases", blockingFile, blockingFile, filepath.Join(tmpDir, "cfg")); err == nil {
		t.Errorf("expected ImportAssetToWorkstation error when dataRoot is a regular file")
	}
	if _, err := FactoryResetWorkstation(FactoryResetOptions{}, blockingFile, filepath.Join(tmpDir, "cfg"), ""); err == nil {
		t.Errorf("expected FactoryResetWorkstation error when dataRoot is a regular file")
	}
	// Backup destination inside blockingFile should fail
	validData := filepath.Join(tmpDir, "valid_data")
	validCfg := filepath.Join(tmpDir, "valid_cfg")
	if _, err := FactoryResetWorkstation(FactoryResetOptions{
		CreateOptInBackup: true,
		BackupDestDir:     blockingFile,
	}, validData, validCfg, ""); err == nil {
		t.Errorf("expected error when BackupDestDir is a regular file")
	}
}
