package online

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DatabaseTargetMode specifies where downloaded player games should be merged.
type DatabaseTargetMode string

const (
	TargetActiveBase    DatabaseTargetMode = "active"    // Merge into currently active database
	TargetDedicatedBase DatabaseTargetMode = "dedicated" // Maintain dedicated standalone database
	TargetBlendedBase   DatabaseTargetMode = "blended"   // Dedicated database + also blend into master database
)

// Profile represents a user synchronization configuration mapping online platform accounts
// to local target database(s).
type Profile struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	TargetDatabase   string             `json:"targetDatabase"` // Full path or empty for active database
	TargetMode       DatabaseTargetMode `json:"targetMode"`     // "active", "dedicated", or "blended"
	BlendedDatabase  string             `json:"blendedDatabase"`// Master database for blended mode
	IsStudyTarget    bool               `json:"isStudyTarget"`  // Opponent, Grandmaster, or study target
	ChessComUsername string             `json:"chessComUsername"`
	LichessUsername  string             `json:"lichessUsername"`
	LichessToken     string             `json:"lichessToken,omitempty"` // Optional API bearer token
	LastSyncTime     int64              `json:"lastSyncTime"`           // Unix milliseconds
	LastSyncGames    int                `json:"lastSyncGames"`
	RatedOnly        bool               `json:"ratedOnly"`
	AutoDeduplicate  bool               `json:"autoDeduplicate"`
	CreatedAt        int64              `json:"createdAt"`
	UpdatedAt        int64              `json:"updatedAt"`
}

// ProfileManager manages loading, saving, and querying user sync profiles.
type ProfileManager struct {
	mu         sync.RWMutex
	configPath string
	profiles   map[string]*Profile
}

// DefaultConfigPath returns the standard user configuration path for NewScid sync profiles.
func DefaultConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	return filepath.Join(configDir, "newscid", "sync_profiles.json")
}

// NewProfileManager creates a profile manager persisting data at configPath.
func NewProfileManager(configPath string) (*ProfileManager, error) {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	pm := &ProfileManager{
		configPath: configPath,
		profiles:   make(map[string]*Profile),
	}

	if err := pm.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading sync profiles: %w", err)
	}

	return pm, nil
}

func (pm *ProfileManager) load() error {
	data, err := os.ReadFile(pm.configPath)
	if err != nil {
		return err
	}

	var list []*Profile
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	pm.profiles = make(map[string]*Profile)
	for _, p := range list {
		if p.ID != "" {
			pm.profiles[p.ID] = p
		}
	}
	return nil
}

func (pm *ProfileManager) save() error {
	dir := filepath.Dir(pm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	list := make([]*Profile, 0, len(pm.profiles))
	for _, p := range pm.profiles {
		list = append(list, p)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(pm.configPath, data, 0644)
}

// GetProfiles returns all configured sync profiles.
func (pm *ProfileManager) GetProfiles() []*Profile {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]*Profile, 0, len(pm.profiles))
	for _, p := range pm.profiles {
		clone := *p
		result = append(result, &clone)
	}
	return result
}

// GetProfile retrieves a profile by ID.
func (pm *ProfileManager) GetProfile(id string) (*Profile, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	p, ok := pm.profiles[id]
	if !ok {
		return nil, fmt.Errorf("profile with ID %q not found", id)
	}
	clone := *p
	return &clone, nil
}

// SaveProfile creates or updates a profile.
func (pm *ProfileManager) SaveProfile(p *Profile) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now().UnixMilli()
	if p.ID == "" {
		p.ID = fmt.Sprintf("profile_%d", now)
		p.CreatedAt = now
	}
	if p.TargetMode == "" {
		p.TargetMode = TargetActiveBase
	}
	p.UpdatedAt = now

	clone := *p
	pm.profiles[p.ID] = &clone

	return pm.save()
}

// DeleteProfile removes a profile by ID.
func (pm *ProfileManager) DeleteProfile(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.profiles[id]; !ok {
		return fmt.Errorf("profile with ID %q not found", id)
	}

	delete(pm.profiles, id)
	return pm.save()
}
