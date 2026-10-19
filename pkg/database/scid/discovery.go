package scid

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DiscoveredDatabase represents a database found on the user's system.
type DiscoveredDatabase struct {
	Name        string `json:"name"`
	BasePath    string `json:"basePath"` // path without .si4 (or full path for PGN)
	Type        string `json:"type"`     // "scid" or "pgn"
	Category    string `json:"category"` // "Recent Bases", "User Databases", "Training & Practice"
	NumGames    int    `json:"numGames"`
	Description string `json:"description"`
	SizeStr     string `json:"sizeStr"`
	Exists      bool   `json:"exists"`
	WritePolicy string `json:"writePolicy"`
	IsLocked    bool   `json:"isLocked"`
}

// ParseRecentDat extracts file paths from a Scid recent.dat file.
func ParseRecentDat(r io.Reader) []string {
	var paths []string
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "set recentFiles(data)") {
			// Find opening brace { and closing brace }
			start := strings.Index(line, "{")
			end := strings.LastIndex(line, "}")
			if start != -1 && end > start {
				content := strings.TrimSpace(line[start+1 : end])
				if content != "" {
					tokens := strings.Fields(content)
					paths = append(paths, tokens...)
				}
			}
		}
	}

	return paths
}

// FormatFileSize returns a human-readable file size string.
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// InspectDatabase extracts metadata from a Scid base path without loading all records.
func InspectDatabase(basePath string, category string) DiscoveredDatabase {
	cleanPath := strings.TrimSuffix(basePath, ".si4")
	cleanPath = strings.TrimSuffix(cleanPath, ".si3")
	name := filepath.Base(cleanPath)

	db := DiscoveredDatabase{
		Name:     name,
		BasePath: cleanPath,
		Type:     "scid",
		Category: category,
		Exists:   false,
	}

	si4Path := cleanPath + ".si4"
	si4Info, err := os.Stat(si4Path)
	if err != nil {
		// Maybe it's a PGN file
		if strings.HasSuffix(basePath, ".pgn") {
			if pgnInfo, err := os.Stat(basePath); err == nil {
				db.Type = "pgn"
				db.BasePath = basePath
				db.Exists = true
				db.SizeStr = FormatFileSize(pgnInfo.Size())
			}
		}
		policy := InferDefaultWritePolicy(db.BasePath, db.NumGames, db.Description, false)
		db.WritePolicy = policy.String()
		db.IsLocked = !policy.AllowsDirectEdit()
		return db
	}

	db.Exists = true
	totalSize := si4Info.Size()
	if sg4Info, err := os.Stat(cleanPath + ".sg4"); err == nil {
		totalSize += sg4Info.Size()
	}
	if sn4Info, err := os.Stat(cleanPath + ".sn4"); err == nil {
		totalSize += sn4Info.Size()
	}
	db.SizeStr = FormatFileSize(totalSize)

	// Quick read of 182-byte header to get accurate game count and description
	f, err := os.Open(si4Path)
	if err == nil {
		defer f.Close()
		hdr, err := ReadIndexHeader(f)
		if err == nil {
			db.NumGames = int(hdr.NumGames)
			db.Description = hdr.Description
		} else {
			// Fallback: estimate from file size
			if si4Info.Size() > IndexHeaderSize {
				db.NumGames = int((si4Info.Size() - IndexHeaderSize) / IndexEntrySize)
			}
		}
	}

	policy := InferDefaultWritePolicy(db.BasePath, db.NumGames, db.Description, false)
	db.WritePolicy = policy.String()
	db.IsLocked = !policy.AllowsDirectEdit()

	return db
}

// DiscoverScidDatabases searches user Scid configurations, data folders, and standard system paths.
func DiscoverScidDatabases() []DiscoveredDatabase {
	var results []DiscoveredDatabase
	seen := make(map[string]bool)

	home, _ := os.UserHomeDir()

	// 1. Check recent files from Scid config
	if home != "" {
		recentFilesList := []string{
			filepath.Join(home, ".scidvspc", "config", "recent.dat"),
			filepath.Join(home, ".scid5.0", "config", "recent.dat"),
		}
		for _, rf := range recentFilesList {
			f, err := os.Open(rf)
			if err == nil {
				paths := ParseRecentDat(f)
				_ = f.Close()
				for _, p := range paths {
					clean := strings.TrimSuffix(p, ".si4")
					clean = strings.TrimSuffix(clean, ".si3")
					if !seen[clean] {
						seen[clean] = true
						meta := InspectDatabase(p, "Recent Databases")
						if meta.Exists {
							results = append(results, meta)
						}
					}
				}
			}
		}
	}

	// 2. Check ~/ChessDataBases
	if home != "" {
		chessBasesDir := filepath.Join(home, "ChessDataBases")
		if fi, err := os.Stat(chessBasesDir); err == nil && fi.IsDir() {
			_ = filepath.Walk(chessBasesDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info == nil {
					return nil
				}
				if !info.IsDir() && strings.HasSuffix(path, ".si4") {
					clean := strings.TrimSuffix(path, ".si4")
					if !seen[clean] {
						seen[clean] = true
						meta := InspectDatabase(clean, "User Databases")
						if meta.Exists {
							results = append(results, meta)
						}
					}
				}
				return nil
			})
		}
	}

	// 3. Check ~/.scidvspc/data, ~/.local/share/newscid/databases, and persona folders
	if home != "" {
		extraDirs := []struct {
			dir      string
			category string
		}{
			{filepath.Join(home, ".scidvspc", "data"), "User Databases"},
			{filepath.Join(home, ".local", "share", "newscid", "databases"), "User Databases"},
			{filepath.Join(home, ".config", "newscid", "personas"), "Persona Databases"},
			{filepath.Join(home, ".newscid", "personas"), "Persona Databases"},
		}
		for _, ed := range extraDirs {
			if fi, err := os.Stat(ed.dir); err == nil && fi.IsDir() {
				entries, _ := os.ReadDir(ed.dir)
				for _, e := range entries {
					if !e.IsDir() && strings.HasSuffix(e.Name(), ".si4") {
						full := filepath.Join(ed.dir, e.Name())
						clean := strings.TrimSuffix(full, ".si4")
						if !seen[clean] {
							seen[clean] = true
							meta := InspectDatabase(clean, ed.category)
							if meta.Exists {
								results = append(results, meta)
							}
						}
					}
				}
			}
		}
	}

	// 4. System training bases
	systemBases := []struct {
		name string
		path string
	}{
		{"tactics", "/usr/share/scid/bases/tactics"},
		{"matein1", "/usr/share/scid/bases/matein1"},
		{"matein2", "/usr/share/scid/bases/matein2"},
		{"matein3", "/usr/share/scid/bases/matein3"},
		{"matein4andmore", "/usr/share/scid/bases/matein4andmore"},
		{"endings", "/usr/share/scid/bases/endings"},
	}

	for _, sb := range systemBases {
		if !seen[sb.path] {
			if _, err := os.Stat(sb.path + ".si4"); err == nil {
				seen[sb.path] = true
				meta := InspectDatabase(sb.path, "Tutorial & Tactics")
				meta.Name = sb.name
				results = append(results, meta)
			}
		}
	}

	return results
}
