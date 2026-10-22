package workstation

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AssetCategoryID is a compact hot-path integer constant identifying a workstation storage category.
type AssetCategoryID uint8

const (
	CategoryDatabases AssetCategoryID = iota
	CategoryPersonas
	CategoryEngines
	CategoryOpeningBooks
	CategoryEndgameBooks
	CategoryRepertoires
	CategoryConfig
	CategoryBackups
)

var allCategories = []AssetCategoryID{
	CategoryDatabases,
	CategoryPersonas,
	CategoryEngines,
	CategoryOpeningBooks,
	CategoryEndgameBooks,
	CategoryRepertoires,
	CategoryConfig,
	CategoryBackups,
}

// Key returns the machine-readable category key string.
func (c AssetCategoryID) Key() string {
	switch c {
	case CategoryDatabases:
		return "databases"
	case CategoryPersonas:
		return "personas"
	case CategoryEngines:
		return "engines"
	case CategoryOpeningBooks:
		return "opening_books"
	case CategoryEndgameBooks:
		return "endgame_books"
	case CategoryRepertoires:
		return "repertoires"
	case CategoryConfig:
		return "config"
	case CategoryBackups:
		return "backups"
	default:
		return "unknown"
	}
}

// Label returns the human-readable category title.
func (c AssetCategoryID) Label() string {
	switch c {
	case CategoryDatabases:
		return "Chess Databases"
	case CategoryPersonas:
		return "Opponent Personas & Bots"
	case CategoryEngines:
		return "UCI Chess Engines"
	case CategoryOpeningBooks:
		return "Opening Books (Polyglot)"
	case CategoryEndgameBooks:
		return "Endgame Tablebases (Syzygy)"
	case CategoryRepertoires:
		return "Opening Repertoires"
	case CategoryConfig:
		return "Workstation Config & State"
	case CategoryBackups:
		return "Workstation Backups"
	default:
		return "Unknown Category"
	}
}

// Description returns a short description of what is stored in the category.
func (c AssetCategoryID) Description() string {
	switch c {
	case CategoryDatabases:
		return "Managed Scid (.si4/.sg4/.sn4) and PGN (.pgn) chess game databases"
	case CategoryPersonas:
		return "Opponent persona profiles, stylometric models, and bot definitions"
	case CategoryEngines:
		return "Local UCI chess engine binaries managed by NewScid"
	case CategoryOpeningBooks:
		return "Polyglot (.bin) opening books for preparation and analysis"
	case CategoryEndgameBooks:
		return "Syzygy (.rtbw/.rtbz) endgame tablebase files"
	case CategoryRepertoires:
		return "Spaced-repetition opening repertoire trees and training decks"
	case CategoryConfig:
		return "User preferences, blunder vault, workspace sync, and application settings"
	case CategoryBackups:
		return "Opt-in compressed workstation backup archives (.tar.gz)"
	default:
		return "Unrecognized storage category"
	}
}

// ParseAssetCategory resolves a string key into an AssetCategoryID.
func ParseAssetCategory(key string) (AssetCategoryID, error) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "databases", "database", "db":
		return CategoryDatabases, nil
	case "personas", "persona":
		return CategoryPersonas, nil
	case "engines", "engine":
		return CategoryEngines, nil
	case "opening_books", "opening_book", "openingbooks", "opening":
		return CategoryOpeningBooks, nil
	case "endgame_books", "endgame_book", "endgamebooks", "endgame", "tablebases", "syzygy":
		return CategoryEndgameBooks, nil
	case "repertoires", "repertoire":
		return CategoryRepertoires, nil
	case "config", "configuration", "settings":
		return CategoryConfig, nil
	case "backups", "backup":
		return CategoryBackups, nil
	default:
		return 0, fmt.Errorf("unknown asset category %q", key)
	}
}

// StoredAssetInfo describes an individual file stored inside NewScid's workstation directories.
type StoredAssetInfo struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Category      string `json:"category"`
	SizeBytes     int64  `json:"sizeBytes"`
	SizeFormatted string `json:"sizeFormatted"`
	ModifiedAt    string `json:"modifiedAt"`
}

// CategoryStorageInfo summarizes disk usage and files for a single workstation category.
type CategoryStorageInfo struct {
	CategoryID    AssetCategoryID   `json:"categoryId"`
	Key           string            `json:"key"`
	Label         string            `json:"label"`
	Path          string            `json:"path"`
	Description   string            `json:"description"`
	FileCount     int               `json:"fileCount"`
	TotalBytes    int64             `json:"totalBytes"`
	SizeFormatted string            `json:"sizeFormatted"`
	Assets        []StoredAssetInfo `json:"assets"`
}

// StorageReport represents the complete disk layout and asset inventory of the NewScid workstation.
type StorageReport struct {
	DataRootDir   string                `json:"dataRootDir"`
	ConfigRootDir string                `json:"configRootDir"`
	TotalBytes    int64                 `json:"totalBytes"`
	TotalSizeStr  string                `json:"totalSizeStr"`
	TotalFiles    int                   `json:"totalFiles"`
	Categories    []CategoryStorageInfo `json:"categories"`
}

// FormatByteSize formats a byte count into a human-readable string (B, KB, MB, GB).
func FormatByteSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	const unit = 1024
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit && exp < 3; n /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), suffixes[exp])
}

// DefaultWorkstationDirs returns the standard XDG / OS-aware data and config directories for NewScid.
func DefaultWorkstationDirs() (dataRoot string, configRoot string) {
	home, _ := os.UserHomeDir()
	if xdgData := os.Getenv("XDG_DATA_HOME"); strings.TrimSpace(xdgData) != "" {
		dataRoot = filepath.Join(xdgData, "newscid")
	} else {
		dataRoot = filepath.Join(home, ".local", "share", "newscid")
	}

	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		configRoot = filepath.Join(cfgDir, "newscid")
	} else {
		configRoot = filepath.Join(home, ".config", "newscid")
	}
	return dataRoot, configRoot
}

// DefaultLegacyConfigDir returns the legacy ~/.newscid directory path.
func DefaultLegacyConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".newscid")
}

// resolveRoots fills in default dataRoot and configRoot if either is empty.
func resolveRoots(dataRoot, configRoot string) (string, string) {
	defData, defCfg := DefaultWorkstationDirs()
	if strings.TrimSpace(dataRoot) == "" {
		dataRoot = defData
	}
	if strings.TrimSpace(configRoot) == "" {
		configRoot = defCfg
	}
	return dataRoot, configRoot
}

// EnsureWorkstationDirs creates and returns the directory paths for all 8 workstation asset categories.
func EnsureWorkstationDirs(dataRoot, configRoot string) (map[AssetCategoryID]string, error) {
	dataRoot, configRoot = resolveRoots(dataRoot, configRoot)

	dirs := map[AssetCategoryID]string{
		CategoryDatabases:    filepath.Join(dataRoot, "databases"),
		CategoryPersonas:     filepath.Join(dataRoot, "personas"),
		CategoryEngines:      filepath.Join(dataRoot, "engines"),
		CategoryOpeningBooks: filepath.Join(dataRoot, "books", "opening"),
		CategoryEndgameBooks: filepath.Join(dataRoot, "books", "endgame"),
		CategoryRepertoires:  filepath.Join(dataRoot, "repertoires"),
		CategoryConfig:       configRoot,
		CategoryBackups:      filepath.Join(dataRoot, "backups"),
	}

	for _, cat := range allCategories {
		dirPath := dirs[cat]
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return nil, fmt.Errorf("creating workstation directory for %s (%s): %w", cat.Key(), dirPath, err)
		}
	}

	return dirs, nil
}

// InspectStorage walks all workstation asset directories and returns a complete StorageReport.
func InspectStorage(dataRoot, configRoot string) (*StorageReport, error) {
	dataRoot, configRoot = resolveRoots(dataRoot, configRoot)
	dirs, err := EnsureWorkstationDirs(dataRoot, configRoot)
	if err != nil {
		return nil, err
	}

	report := &StorageReport{
		DataRootDir:   dataRoot,
		ConfigRootDir: configRoot,
		Categories:    make([]CategoryStorageInfo, 0, len(allCategories)),
	}

	for _, cat := range allCategories {
		catDir := dirs[cat]
		catInfo := CategoryStorageInfo{
			CategoryID:  cat,
			Key:         cat.Key(),
			Label:       cat.Label(),
			Path:        catDir,
			Description: cat.Description(),
			Assets:      []StoredAssetInfo{},
		}

		_ = filepath.WalkDir(catDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			info, statErr := d.Info()
			if statErr != nil {
				return nil
			}
			size := info.Size()
			catInfo.Assets = append(catInfo.Assets, StoredAssetInfo{
				Name:          d.Name(),
				Path:          path,
				Category:      cat.Key(),
				SizeBytes:     size,
				SizeFormatted: FormatByteSize(size),
				ModifiedAt:    info.ModTime().UTC().Format(time.RFC3339),
			})
			catInfo.FileCount++
			catInfo.TotalBytes += size
			return nil
		})

		sort.Slice(catInfo.Assets, func(i, j int) bool {
			return catInfo.Assets[i].Name < catInfo.Assets[j].Name
		})
		catInfo.SizeFormatted = FormatByteSize(catInfo.TotalBytes)

		report.TotalFiles += catInfo.FileCount
		report.TotalBytes += catInfo.TotalBytes
		report.Categories = append(report.Categories, catInfo)
	}

	report.TotalSizeStr = FormatByteSize(report.TotalBytes)
	return report, nil
}

// copySingleFile copies src to dst with the given file mode and returns the number of bytes written.
func copySingleFile(src, dst string, perm os.FileMode) (int64, error) {
	cleanSrc, _ := filepath.Abs(src)
	cleanDst, _ := filepath.Abs(dst)
	if cleanSrc == cleanDst {
		st, err := os.Stat(cleanDst)
		if err != nil {
			return 0, err
		}
		return st.Size(), nil
	}

	in, err := os.Open(cleanSrc)
	if err != nil {
		return 0, fmt.Errorf("opening source file %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(cleanDst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return 0, fmt.Errorf("creating destination file %s: %w", dst, err)
	}
	written, copyErr := io.Copy(out, in)
	_ = out.Close()
	if copyErr != nil {
		return 0, fmt.Errorf("copying file data to %s: %w", dst, copyErr)
	}
	_ = os.Chmod(cleanDst, perm)
	return written, nil
}

// ImportAssetToWorkstation copies an external file (or Scid 3-file bundle) into NewScid's managed workstation storage.
func ImportAssetToWorkstation(categoryKey string, sourcePath string, dataRoot string, configRoot string) (*StoredAssetInfo, error) {
	cat, err := ParseAssetCategory(categoryKey)
	if err != nil {
		return nil, err
	}

	trimmedSrc := strings.TrimSpace(sourcePath)
	if trimmedSrc == "" {
		return nil, fmt.Errorf("source path is required")
	}

	dirs, err := EnsureWorkstationDirs(dataRoot, configRoot)
	if err != nil {
		return nil, err
	}
	targetDir := dirs[cat]

	// Special handling for Scid databases (.si4, .sg4, .sn4 or base path without extension)
	if cat == CategoryDatabases {
		ext := strings.ToLower(filepath.Ext(trimmedSrc))
		if ext == ".si4" || ext == ".sg4" || ext == ".sn4" || ext == "" {
			baseSrc := strings.TrimSuffix(trimmedSrc, filepath.Ext(trimmedSrc))
			if _, statErr := os.Stat(baseSrc + ".si4"); statErr == nil {
				baseName := filepath.Base(baseSrc)
				var totalSize int64
				for _, scidExt := range []string{".si4", ".sg4", ".sn4"} {
					companionSrc := baseSrc + scidExt
					if st, err := os.Stat(companionSrc); err == nil && !st.IsDir() {
						dstPath := filepath.Join(targetDir, baseName+scidExt)
						n, err := copySingleFile(companionSrc, dstPath, 0o644)
						if err != nil {
							return nil, err
						}
						totalSize += n
					}
				}
				primaryDst := filepath.Join(targetDir, baseName+".si4")
				st, _ := os.Stat(primaryDst)
				return &StoredAssetInfo{
					Name:          baseName + ".si4",
					Path:          primaryDst,
					Category:      cat.Key(),
					SizeBytes:     totalSize,
					SizeFormatted: FormatByteSize(totalSize),
					ModifiedAt:    st.ModTime().UTC().Format(time.RFC3339),
				}, nil
			}
		}
	}

	srcInfo, err := os.Stat(trimmedSrc)
	if err != nil {
		return nil, fmt.Errorf("source asset not found (%s): %w", trimmedSrc, err)
	}
	if srcInfo.IsDir() {
		return nil, fmt.Errorf("source asset %s is a directory, expected a file", trimmedSrc)
	}

	perm := os.FileMode(0o644)
	if cat == CategoryEngines {
		perm = 0o755
	}

	dstPath := filepath.Join(targetDir, filepath.Base(trimmedSrc))
	written, err := copySingleFile(trimmedSrc, dstPath, perm)
	if err != nil {
		return nil, err
	}

	dstInfo, _ := os.Stat(dstPath)
	return &StoredAssetInfo{
		Name:          filepath.Base(dstPath),
		Path:          dstPath,
		Category:      cat.Key(),
		SizeBytes:     written,
		SizeFormatted: FormatByteSize(written),
		ModifiedAt:    dstInfo.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

// FactoryResetOptions configures a NewScid workstation factory reset operation.
type FactoryResetOptions struct {
	CreateOptInBackup bool   `json:"createOptInBackup"` // Unchecked by default (opt-in!)
	BackupDestDir     string `json:"backupDestDir,omitempty"`
	KeepEngines       bool   `json:"keepEngines,omitempty"`
	KeepBooks         bool   `json:"keepBooks,omitempty"`
}

// FactoryResetResult summarizes the outcome of a workstation factory reset.
type FactoryResetResult struct {
	ResetCompletedAt  string   `json:"resetCompletedAt"`
	BackupCreated     bool     `json:"backupCreated"`
	BackupArchivePath string   `json:"backupArchivePath,omitempty"`
	WipedPaths        []string `json:"wipedPaths"`
	FilesRemoved      int      `json:"filesRemoved"`
	BytesFreed        int64    `json:"bytesFreed"`
	SizeFreedStr      string   `json:"sizeFreedStr"`
	Message           string   `json:"message"`
}

// createWorkstationBackupArchive writes a .tar.gz archive of the workstation directories.
func createWorkstationBackupArchive(archivePath string, roots map[string]string, skipDir string) error {
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return fmt.Errorf("creating backup parent directory: %w", err)
	}
	f, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("creating backup archive %s: %w", archivePath, err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	cleanSkip, _ := filepath.Abs(skipDir)

	prefixes := make([]string, 0, len(roots))
	for prefix := range roots {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)

	for _, prefix := range prefixes {
		rootDir := roots[prefix]
		if _, err := os.Stat(rootDir); err != nil {
			continue
		}
		_ = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if absPath, err := filepath.Abs(path); err == nil && cleanSkip != "" {
				if absPath == cleanSkip || strings.HasPrefix(absPath, cleanSkip+string(filepath.Separator)) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil || !info.Mode().IsRegular() {
				return nil
			}
			rel, _ := filepath.Rel(rootDir, path)
			hdr, _ := tar.FileInfoHeader(info, "")
			hdr.Name = filepath.ToSlash(filepath.Join(prefix, rel))
			_ = tw.WriteHeader(hdr)
			if in, err := os.Open(path); err == nil {
				_, _ = io.Copy(tw, in)
				_ = in.Close()
			}
			return nil
		})
	}
	return nil
}

// wipeDirectoryContents removes all files and subdirectories inside dirPath and returns the number of files and bytes removed.
func wipeDirectoryContents(dirPath string) (int, int64, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	filesRemoved := 0
	var bytesFreed int64

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())
		_ = filepath.WalkDir(fullPath, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr == nil && !d.IsDir() {
				if info, err := d.Info(); err == nil {
					filesRemoved++
					bytesFreed += info.Size()
				}
			}
			return nil
		})
		_ = os.RemoveAll(fullPath)
	}
	return filesRemoved, bytesFreed, nil
}

// FactoryResetWorkstation performs a complete factory reset of NewScid's workstation storage,
// with an opt-in compressed backup archive (.tar.gz) when opts.CreateOptInBackup is true.
func FactoryResetWorkstation(opts FactoryResetOptions, dataRoot, configRoot, legacyConfigDir string) (*FactoryResetResult, error) {
	dataRoot, configRoot = resolveRoots(dataRoot, configRoot)
	if strings.TrimSpace(legacyConfigDir) == "" {
		legacyConfigDir = DefaultLegacyConfigDir()
	}

	dirs, err := EnsureWorkstationDirs(dataRoot, configRoot)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	res := &FactoryResetResult{
		ResetCompletedAt: now.Format(time.RFC3339),
		WipedPaths:       []string{},
	}

	// 1. Optional (opt-in) backup before wiping any data
	if opts.CreateOptInBackup {
		backupDir := strings.TrimSpace(opts.BackupDestDir)
		if backupDir == "" {
			backupDir = dirs[CategoryBackups]
		}
		archiveName := fmt.Sprintf("newscid-factory-backup-%s.tar.gz", now.Format("20060102-150405"))
		archivePath := filepath.Join(backupDir, archiveName)
		roots := map[string]string{
			"data":   dataRoot,
			"config": configRoot,
			"legacy": legacyConfigDir,
		}
		if err := createWorkstationBackupArchive(archivePath, roots, dirs[CategoryBackups]); err != nil {
			return nil, err
		}
		res.BackupCreated = true
		res.BackupArchivePath = archivePath
	}

	// 2. Determine directories to wipe
	targetDirs := []string{
		dirs[CategoryDatabases],
		dirs[CategoryPersonas],
		dirs[CategoryRepertoires],
		filepath.Join(dataRoot, "storage"),
		filepath.Join(dataRoot, "localstorage"),
		filepath.Join(dataRoot, "cache"),
		filepath.Join(dataRoot, "mediakeys"),
		filepath.Join(configRoot, "personas"),
		filepath.Join(configRoot, "repertoires"),
		filepath.Join(legacyConfigDir, "personas"),
		filepath.Join(legacyConfigDir, "repertoires"),
	}
	if !opts.KeepEngines {
		targetDirs = append(targetDirs, dirs[CategoryEngines])
	}
	if !opts.KeepBooks {
		targetDirs = append(targetDirs,
			dirs[CategoryOpeningBooks],
			dirs[CategoryEndgameBooks],
			filepath.Join(dataRoot, "books"),
		)
	}

	for _, d := range targetDirs {
		removed, freed, err := wipeDirectoryContents(d)
		if err != nil {
			return nil, fmt.Errorf("wiping directory %s: %w", d, err)
		}
		res.FilesRemoved += removed
		res.BytesFreed += freed
		res.WipedPaths = append(res.WipedPaths, d)
	}

	// 2b. Wipe loose files directly inside dataRoot (e.g. SQLite databases, temporary files, logs)
	if entries, err := os.ReadDir(dataRoot); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				fp := filepath.Join(dataRoot, entry.Name())
				if info, err := entry.Info(); err == nil {
					res.FilesRemoved++
					res.BytesFreed += info.Size()
				}
				_ = os.Remove(fp)
				res.WipedPaths = append(res.WipedPaths, fp)
			}
		}
	}

	// 3. Remove all JSON and state files in configRoot and legacyConfigDir
	for _, cfgDir := range []string{configRoot, legacyConfigDir} {
		entries, err := os.ReadDir(cfgDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fullPath := filepath.Join(cfgDir, entry.Name())
			if info, err := entry.Info(); err == nil {
				res.FilesRemoved++
				res.BytesFreed += info.Size()
			}
			_ = os.Remove(fullPath)
			res.WipedPaths = append(res.WipedPaths, fullPath)
		}
	}

	// 4. Re-create clean skeleton directories
	_, _ = EnsureWorkstationDirs(dataRoot, configRoot)

	res.SizeFreedStr = FormatByteSize(res.BytesFreed)
	if res.BackupCreated {
		res.Message = fmt.Sprintf("Factory reset complete (%d files removed, %s freed). Opt-in backup saved to %s.",
			res.FilesRemoved, res.SizeFreedStr, res.BackupArchivePath)
	} else {
		res.Message = fmt.Sprintf("Factory reset complete (%d files removed, %s freed). Workstation restored to clean factory defaults.",
			res.FilesRemoved, res.SizeFreedStr)
	}
	return res, nil
}
