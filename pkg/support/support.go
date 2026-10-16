package support

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Live Stripe Payment Link URLs (shared with bughouse-online) for $5+ supporter tiers.
const (
	StripeLink5USD   = "https://buy.stripe.com/cNidR2e262cEeiOdpCefC01" // $5 Knight Supporter
	StripeLink10USD  = "https://buy.stripe.com/bJe3cogae5oQdeKgBOefC02" // $10 Rook Supporter
	StripeLink35USD  = "https://buy.stripe.com/aFa7sE2joeZqdeK99mefC03" // $35 Queen Supporter
	StripeLink100USD = "https://buy.stripe.com/00w28ke265oQdeK2KYefC04" // $100 Princess Supporter
	StripeLink350USD = "https://buy.stripe.com/6oUbIUf6a04w3Ea71eefC05" // $350 King Lifetime Supporter

	// DefaultStripeSupportURL is the default $5+ Stripe payment link.
	DefaultStripeSupportURL = StripeLink5USD
)

// DefaultMinSupportUSD is the minimum suggested support contribution in USD ($5).
const DefaultMinSupportUSD = 5

// NagCooldownDuration is the 24-hour period that a dismissed support prompt stays hidden before showing again once per day.
const NagCooldownDuration = 24 * time.Hour

// Canonical author online chess handles used to form the author computer exemption ID.
const (
	AuthorChessComPrimary = "christopherfair"
	AuthorChessComAlt     = "chrisfair"
	AuthorLichessPrimary  = "quenyenhighelf"
	AuthorLichessAlt      = "quenyen"
)

// StripeURLForAmount returns the live Stripe checkout link for the selected tier ($5 or higher).
func StripeURLForAmount(amountUSD int, customOverride string) string {
	if clean := strings.TrimSpace(customOverride); clean != "" && clean != DefaultStripeSupportURL {
		return clean
	}
	if envURL := strings.TrimSpace(os.Getenv("NEWSCID_STRIPE_URL")); envURL != "" {
		return envURL
	}
	switch {
	case amountUSD >= 350:
		return StripeLink350USD
	case amountUSD >= 100:
		return StripeLink100USD
	case amountUSD >= 35:
		return StripeLink35USD
	case amountUSD >= 10:
		return StripeLink10USD
	default:
		return StripeLink5USD
	}
}

// BuildCheckoutURL appends an optional client_reference_id if present.
func BuildCheckoutURL(baseURL string, clientRef string) string {
	cleanRef := strings.TrimSpace(clientRef)
	if cleanRef == "" {
		return baseURL
	}
	sep := "?"
	if strings.Contains(baseURL, "?") {
		sep = "&"
	}
	return baseURL + sep + "client_reference_id=" + url.QueryEscape(cleanRef)
}

// SupportConfig is persisted to ~/.config/newscid/support.json.
type SupportConfig struct {
	Supported       bool   `json:"supported"`
	IsAuthor        bool   `json:"isAuthor"`
	AuthorID        string `json:"authorId,omitempty"`
	SupporterTier   string `json:"supporterTier,omitempty"`
	Reference       string `json:"reference,omitempty"`
	SupportedAt     int64  `json:"supportedAt,omitempty"`
	DismissCount    int    `json:"dismissCount"`
	LastDismissedAt int64  `json:"lastDismissedAt,omitempty"`
	StripeURL       string `json:"stripeUrl,omitempty"`
}

// SupportStatus is returned to the frontend UI to determine whether to show the support form.
type SupportStatus struct {
	Supported       bool           `json:"supported"`
	IsAuthor        bool           `json:"isAuthor"`
	AuthorID        string         `json:"authorId,omitempty"`
	DeviceSignature string         `json:"deviceSignature,omitempty"`
	SupporterTier   string         `json:"supporterTier,omitempty"`
	SupportedAt     int64          `json:"supportedAt,omitempty"`
	DismissCount    int            `json:"dismissCount"`
	LastDismissedAt int64          `json:"lastDismissedAt,omitempty"`
	StripeURL       string         `json:"stripeUrl"`
	TierLinks       map[int]string `json:"tierLinks"`
	MinAmountUSD    int            `json:"minAmountUsd"`
	ShouldShowNag   bool           `json:"shouldShowNag"`
}

// Manager manages support/donation status, Stripe link configuration, and author ID exemption.
type Manager struct {
	mu         sync.RWMutex
	configPath string
	config     SupportConfig
}

// NewManager initializes a support manager backed by the given JSON file path.
// If configPath is empty, it defaults to ~/.config/newscid/support.json.
func NewManager(configPath string) (*Manager, error) {
	if strings.TrimSpace(configPath) == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			cfgDir = os.TempDir()
		}
		configPath = filepath.Join(cfgDir, "newscid", "support.json")
	}

	m := &Manager{
		configPath: configPath,
		config: SupportConfig{
			StripeURL: resolveDefaultStripeURL(),
		},
	}

	_ = m.load()
	if strings.TrimSpace(m.config.StripeURL) == "" {
		m.config.StripeURL = resolveDefaultStripeURL()
	}

	return m, nil
}

func resolveDefaultStripeURL() string {
	if envURL := strings.TrimSpace(os.Getenv("NEWSCID_STRIPE_URL")); envURL != "" {
		return envURL
	}
	return DefaultStripeSupportURL
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}
	var cfg SupportConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	m.config = cfg
	return nil
}

func (m *Manager) save() error {
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.configPath, data, 0o644)
}

// BuildAuthorCompositeID constructs a deterministic author identifier from Chess.com and Lichess.org IDs.
// Regular users do NOT need Lichess or Chess.com accounts; this is used solely to recognize the author's computer.
func BuildAuthorCompositeID(chessComIDs []string, lichessIDs []string) (bool, string, string) {
	matchedChessCom := ""
	matchedLichess := ""

	for _, raw := range chessComIDs {
		norm := strings.ToLower(strings.TrimSpace(raw))
		if norm == AuthorChessComPrimary || norm == AuthorChessComAlt {
			matchedChessCom = strings.TrimSpace(raw)
			break
		}
	}

	for _, raw := range lichessIDs {
		norm := strings.ToLower(strings.TrimSpace(raw))
		if norm == AuthorLichessPrimary || norm == AuthorLichessAlt {
			matchedLichess = strings.TrimSpace(raw)
			break
		}
	}

	if matchedChessCom == "" && matchedLichess == "" {
		return false, "", ""
	}

	if matchedChessCom == "" {
		matchedChessCom = "ChristopherFair"
	}
	if matchedLichess == "" {
		matchedLichess = "QuenyenHighElf"
	}

	composite := "chess.com:" + matchedChessCom + "|lichess.org:" + matchedLichess
	sum := sha256.Sum256([]byte(strings.ToLower(composite)))
	sig := hex.EncodeToString(sum[:12])
	return true, composite, sig
}

// GetStatus evaluates current support status, checking both saved supporter state
// and whether the machine has the author's Chess.com / Lichess.org IDs.
func (m *Manager) GetStatus(chessComIDs []string, lichessIDs []string) SupportStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	isAuthor, compositeID, sig := BuildAuthorCompositeID(chessComIDs, lichessIDs)
	if isAuthor && (!m.config.IsAuthor || m.config.AuthorID != compositeID) {
		m.config.IsAuthor = true
		m.config.Supported = true
		m.config.AuthorID = compositeID
		if m.config.SupporterTier == "" {
			m.config.SupporterTier = "Author"
		}
		if m.config.SupportedAt == 0 {
			m.config.SupportedAt = time.Now().UnixMilli()
		}
		_ = m.save()
	}

	return m.buildStatusLocked(sig)
}

// RecordSupport marks the user as having supported NewScid, permanently disabling the startup nag screen.
func (m *Manager) RecordSupport(tier string, reference string) (SupportStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanTier := strings.TrimSpace(tier)
	if cleanTier == "" {
		cleanTier = "$5+ Supporter"
	}
	cleanRef := strings.TrimSpace(reference)

	// Check if the optional reference entered matches one of the author's handles
	isAuthor, compositeID, sig := BuildAuthorCompositeID([]string{cleanRef}, []string{cleanRef})
	if isAuthor {
		m.config.IsAuthor = true
		m.config.AuthorID = compositeID
		cleanTier = "Author"
	}

	m.config.Supported = true
	m.config.SupporterTier = cleanTier
	m.config.Reference = cleanRef
	m.config.SupportedAt = time.Now().UnixMilli()

	err := m.save()
	return m.buildStatusLocked(sig), err
}

// DismissNag records that the user dismissed the support prompt without donating.
// The program remains 100% functional (non-obligatory).
func (m *Manager) DismissNag() (SupportStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.DismissCount++
	m.config.LastDismissedAt = time.Now().UnixMilli()

	err := m.save()
	return m.buildStatusLocked(""), err
}

// SetStripeURL updates the Stripe payment link URL.
func (m *Manager) SetStripeURL(url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clean := strings.TrimSpace(url)
	if clean == "" {
		clean = resolveDefaultStripeURL()
	}
	m.config.StripeURL = clean
	return m.save()
}

func (m *Manager) buildStatusLocked(sig string) SupportStatus {
	if sig == "" && m.config.AuthorID != "" {
		sum := sha256.Sum256([]byte(strings.ToLower(m.config.AuthorID)))
		sig = hex.EncodeToString(sum[:12])
	}
	stripeURL := strings.TrimSpace(m.config.StripeURL)
	if stripeURL == "" {
		stripeURL = resolveDefaultStripeURL()
	}

	exemptOrSupported := m.config.Supported || m.config.IsAuthor

	inDailyCooldown := false
	if !exemptOrSupported && m.config.LastDismissedAt > 0 {
		elapsed := time.Since(time.UnixMilli(m.config.LastDismissedAt))
		if elapsed >= 0 && elapsed < NagCooldownDuration {
			inDailyCooldown = true
		}
	}

	return SupportStatus{
		Supported:       exemptOrSupported,
		IsAuthor:        m.config.IsAuthor,
		AuthorID:        m.config.AuthorID,
		DeviceSignature: sig,
		SupporterTier:   m.config.SupporterTier,
		SupportedAt:     m.config.SupportedAt,
		DismissCount:    m.config.DismissCount,
		LastDismissedAt: m.config.LastDismissedAt,
		StripeURL:       stripeURL,
		TierLinks: map[int]string{
			5:   StripeLink5USD,
			10:  StripeLink10USD,
			35:  StripeLink35USD,
			100: StripeLink100USD,
			350: StripeLink350USD,
		},
		MinAmountUSD:  DefaultMinSupportUSD,
		ShouldShowNag: !exemptOrSupported && !inDailyCooldown,
	}
}
