package fileformat

import (
	"path/filepath"
	"strings"
)

// Canonical chess database, book, tablebase, export, and archive file extensions.
const (
	// Scid v4 3-file database bundle extensions
	ExtScidIndex    = ".si4"
	ExtScidGameData = ".sg4"
	ExtScidNames    = ".sn4"

	// Legacy Scid v3 database extensions
	ExtScidLegacyIndex    = ".si3"
	ExtScidLegacyGameData = ".sg3"
	ExtScidLegacyNames    = ".sn3"

	// Auxiliary Scid file extensions
	ExtScidRepertoire = ".sor"
	ExtScidSpellcheck = ".ssp"
	ExtScidPhoto      = ".spf"
	ExtScidDat        = ".dat"

	// Atomic write temporary & backup suffixes
	ExtTemp               = ".tmp"
	ExtBackup             = ".bak"
	ExtScidIndexTemp      = ExtScidIndex + ExtTemp
	ExtScidGameDataTemp   = ExtScidGameData + ExtTemp
	ExtScidNamesTemp      = ExtScidNames + ExtTemp
	ExtScidIndexBackup    = ExtScidIndex + ExtBackup
	ExtScidGameDataBackup = ExtScidGameData + ExtBackup
	ExtScidNamesBackup    = ExtScidNames + ExtBackup

	// Portable / text chess formats
	ExtPGN = ".pgn"
	ExtEPD = ".epd"
	ExtFEN = ".fen"
	ExtECO = ".eco"

	// Opening books & endgame tablebases
	ExtPolyglotBook = ".bin"
	ExtSyzygyWDL    = ".rtbw"
	ExtSyzygyDTZ    = ".rtbz"

	// Structured config & document export extensions
	ExtJSON  = ".json"
	ExtHTML  = ".html"
	ExtLaTeX = ".tex"
	ExtCSV   = ".csv"

	// Compressed archive extensions
	ExtZst    = ".zst"
	ExtPGNZst = ExtPGN + ExtZst
	ExtZip    = ".zip"
	Ext7z     = ".7z"
	ExtSI47z  = "_si4" + Ext7z
	ExtTarGz  = ".tar.gz"
)

// File dialog filter glob patterns.
const (
	GlobScidIndex    = "*" + ExtScidIndex
	GlobPGN          = "*" + ExtPGN
	GlobPolyglotBook = "*" + ExtPolyglotBook
	GlobEPD          = "*" + ExtEPD
	GlobJSON         = "*" + ExtJSON
)

// Canonical workstation directory and configuration file names.
const (
	AppDirName        = "newscid"
	LegacyDotDirName  = ".newscid"
	LegacyScidVsPCDir = ".scidvspc"
	LegacyScid5Dir    = ".scid5.0"

	DirDatabases    = "databases"
	DirPersonas     = "personas"
	DirEngines      = "engines"
	DirBooks        = "books"
	DirOpeningBooks = "opening"
	DirEndgameBooks = "endgame"
	DirRepertoires  = "repertoires"
	DirBackups      = "backups"

	DefaultMyGamesDBName = "My_Games"

	FilePreferencesJSON   = "preferences" + ExtJSON
	FileWorkspaceSyncJSON = "workspace_sync" + ExtJSON
	FileBlunderVaultJSON  = "blunder_vault" + ExtJSON
	FileSupportJSON       = "support" + ExtJSON
	FileCyclicalDrillJSON = "cyclical_drill" + ExtJSON
	FileAdaptiveCoachJSON = "adaptive_coach" + ExtJSON
	FileSyncProfilesJSON  = "sync_profiles" + ExtJSON
	FilePersonasJSON      = "personas" + ExtJSON

	FileScidRecentDat  = "recent" + ExtScidDat
	FileScidEnginesDat = "engines" + ExtScidDat
	FileScidECO        = "scid" + ExtECO
	SystemScidECOPath  = "/usr/share/scid/data/" + FileScidECO

	BookPerformanceBin = "Performance" + ExtPolyglotBook
	BookElo2400Bin     = "Elo2400" + ExtPolyglotBook
)

// ScidV4Extensions returns the 3 companion extensions of a Scid v4 database (.si4, .sg4, .sn4).
func ScidV4Extensions() []string {
	return []string{ExtScidIndex, ExtScidGameData, ExtScidNames}
}

// AllDatabaseExtensions returns the Scid v4 companion extensions plus .pgn.
func AllDatabaseExtensions() []string {
	return []string{ExtScidIndex, ExtScidGameData, ExtScidNames, ExtPGN}
}

// TrimScidExtensions strips any Scid v4 or v3 database suffix (.si4, .sg4, .sn4, .si3, .sg3, .sn3)
// from the end of path (case-insensitive), returning the clean database base path.
func TrimScidExtensions(path string) string {
	clean := path
	for _, ext := range []string{
		ExtScidIndex,
		ExtScidGameData,
		ExtScidNames,
		ExtScidLegacyIndex,
		ExtScidLegacyGameData,
		ExtScidLegacyNames,
	} {
		if strings.HasSuffix(strings.ToLower(clean), ext) {
			clean = clean[:len(clean)-len(ext)]
		}
	}
	return clean
}

// TrimDatabaseExtensions strips any Scid database extension (.si4, .sg4, .sn4, .si3, .sg3, .sn3)
// or .pgn suffix from path.
func TrimDatabaseExtensions(path string) string {
	clean := TrimScidExtensions(path)
	if strings.HasSuffix(strings.ToLower(clean), ExtPGN) {
		clean = clean[:len(clean)-len(ExtPGN)]
	}
	return clean
}

// ScidIndexPath returns the .si4 index file path for a Scid database base path.
func ScidIndexPath(basePath string) string {
	return TrimScidExtensions(basePath) + ExtScidIndex
}

// ScidGameDataPath returns the .sg4 game data file path for a Scid database base path.
func ScidGameDataPath(basePath string) string {
	return TrimScidExtensions(basePath) + ExtScidGameData
}

// ScidNamesPath returns the .sn4 namebase file path for a Scid database base path.
func ScidNamesPath(basePath string) string {
	return TrimScidExtensions(basePath) + ExtScidNames
}

// ScidPaths returns the (.si4, .sn4, .sg4) file paths for a Scid database base path.
func ScidPaths(basePath string) (si4Path, sn4Path, sg4Path string) {
	base := TrimScidExtensions(basePath)
	return base + ExtScidIndex, base + ExtScidNames, base + ExtScidGameData
}

// IsPGNPath reports whether path ends with the .pgn extension (case-insensitive).
func IsPGNPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(path)), ExtPGN)
}

// IsScidIndexPath reports whether path ends with the .si4 extension (case-insensitive).
func IsScidIndexPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(path)), ExtScidIndex)
}

// IsScidComponentExt reports whether ext is one of .si4, .sg4, or .sn4 (case-insensitive).
func IsScidComponentExt(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ExtScidIndex, ExtScidGameData, ExtScidNames:
		return true
	default:
		return false
	}
}

// IsPolyglotBookPath reports whether path ends with the .bin extension (case-insensitive).
func IsPolyglotBookPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(path)), ExtPolyglotBook)
}

// IsJSONPath reports whether path ends with the .json extension (case-insensitive).
func IsJSONPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(path)), ExtJSON)
}

// HasExtension reports whether path ends with the given extension (case-insensitive).
func HasExtension(path, ext string) bool {
	return strings.EqualFold(filepath.Ext(strings.TrimSpace(path)), ext)
}
