package engine

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// EngineConfig represents a configured chess engine discovered from Scid or system.
type EngineConfig struct {
	Name        string `json:"name"`
	Cmd         string `json:"cmd"`
	ResolvedCmd string `json:"resolvedCmd"`
	Args        string `json:"args"`
	Dir         string `json:"dir"`
	UCI         bool   `json:"uci"`
	Elo         int    `json:"elo"`
	Available   bool   `json:"available"`
}

// ParseEnginesDat parses the Scid engines.dat format.
func ParseEnginesDat(r io.Reader) ([]EngineConfig, error) {
	var engines []EngineConfig
	scanner := bufio.NewScanner(r)

	var current *EngineConfig
	inEngineBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "engine {") {
			inEngineBlock = true
			current = &EngineConfig{UCI: true} // default to UCI
			continue
		}

		if inEngineBlock {
			if line == "}" {
				if current != nil && (current.Name != "" || current.Cmd != "") {
					engines = append(engines, *current)
				}
				inEngineBlock = false
				current = nil
				continue
			}

			// Parse key-value pairs (e.g., "Name Stockfish-17", "Cmd /path/to/bin", "UCI 1")
			parts := strings.SplitN(line, " ", 2)
			if len(parts) < 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Strip surrounding braces if any, e.g. "{}" or "{arg1 arg2}"
			if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") && len(val) >= 2 {
				val = val[1 : len(val)-1]
			}

			switch key {
			case "Name":
				current.Name = val
			case "Cmd":
				current.Cmd = val
			case "Args":
				current.Args = val
			case "Dir":
				current.Dir = val
			case "Elo":
				current.Elo, _ = strconv.Atoi(val)
			case "UCI":
				current.UCI = (val == "1" || strings.ToLower(val) == "true")
			}
		}
	}

	return engines, scanner.Err()
}

// CheckEngineAvailability checks whether an engine command is available and executable.
func CheckEngineAvailability(cfg *EngineConfig) {
	if cfg.Cmd == "" {
		cfg.Available = false
		return
	}

	// 1. Direct path check
	if filepath.IsAbs(cfg.Cmd) {
		fi, err := os.Stat(cfg.Cmd)
		if err == nil && !fi.IsDir() && (fi.Mode()&0111 != 0) {
			cfg.Available = true
			cfg.ResolvedCmd = cfg.Cmd
			return
		}
	}

	// 2. Look in PATH
	if resolved, err := exec.LookPath(cfg.Cmd); err == nil {
		cfg.Available = true
		cfg.ResolvedCmd = resolved
		return
	}

	// 3. Expand ~ if path starts with ~/
	if strings.HasPrefix(cfg.Cmd, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			expanded := filepath.Join(home, cfg.Cmd[2:])
			fi, err := os.Stat(expanded)
			if err == nil && !fi.IsDir() && (fi.Mode()&0111 != 0) {
				cfg.Available = true
				cfg.ResolvedCmd = expanded
				return
			}
		}
	}

	cfg.Available = false
}

// DiscoverScidEngines scans user Scid config files and standard locations for installed engines.
func DiscoverScidEngines() []EngineConfig {
	var discovered []EngineConfig
	seenCmds := make(map[string]bool)

	home, _ := os.UserHomeDir()
	var configFiles []string
	if home != "" {
		configFiles = append(configFiles,
			filepath.Join(home, ".scidvspc", "config", "engines.dat"),
			filepath.Join(home, ".scid5.0", "config", "engines.dat"),
		)
	}

	// 1. Load from engines.dat files
	for _, cfgPath := range configFiles {
		f, err := os.Open(cfgPath)
		if err != nil {
			continue
		}
		engs, err := ParseEnginesDat(f)
		_ = f.Close()
		if err == nil {
			for i := range engs {
				CheckEngineAvailability(&engs[i])
				key := engs[i].Cmd
				if key == "" {
					key = engs[i].Name
				}
				if !seenCmds[key] {
					seenCmds[key] = true
					discovered = append(discovered, engs[i])
				}
			}
		}
	}

	// 2. Scan standard system and known user engine paths
	standardEngines := []struct {
		name string
		cmd  string
	}{
		{"Stockfish", "stockfish"},
		{"Stockfish", "/usr/bin/stockfish"},
		{"Stockfish", "/usr/games/stockfish"},
		{"Fruit", "fruit"},
		{"Fruit", "/usr/bin/fruit"},
		{"Phalanx", "phalanx"},
		{"Scidlet", "scidlet"},
		{"Crafty", "crafty"},
	}

	for _, se := range standardEngines {
		resolved, err := exec.LookPath(se.cmd)
		if err == nil {
			if !seenCmds[resolved] && !seenCmds[se.cmd] {
				seenCmds[resolved] = true
				seenCmds[se.cmd] = true
				cfg := EngineConfig{
					Name:        se.name,
					Cmd:         se.cmd,
					ResolvedCmd: resolved,
					UCI:         true,
					Available:   true,
				}
				discovered = append(discovered, cfg)
			}
		}
	}

	return discovered
}
