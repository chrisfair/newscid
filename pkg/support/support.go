package support

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"newscid/pkg/fileformat"
)

// Live Stripe Payment Link URLs (shared with bughouse-online) for the 5 classic chess piece tiers.
const (
	StripeLink1USD   = "https://buy.stripe.com/14A7sEgae18A5Mi3P2efC00" // $1 Pawn (3 Days)
	StripeLink5USD   = "https://buy.stripe.com/cNidR2e262cEeiOdpCefC01" // $5 Knight (1 Month)
	StripeLink10USD  = "https://buy.stripe.com/bJe3cogae5oQdeKgBOefC02" // $10 Rook (3 Months)
	StripeLink35USD  = "https://buy.stripe.com/aFa7sE2joeZqdeK99mefC03" // $35 Queen (1 Year)
	StripeLink350USD = "https://buy.stripe.com/6oUbIUf6a04w3Ea71eefC05" // $350 King (Lifetime)

	// DefaultStripeSupportURL is the default $5 Knight Stripe payment link.
	DefaultStripeSupportURL = StripeLink5USD
)

// DefaultMinSupportUSD is the minimum support contribution in USD ($1).
const DefaultMinSupportUSD = 1

// NagCooldownDuration is the 24-hour period that a dismissed support prompt stays hidden before showing again once per day.
const NagCooldownDuration = 24 * time.Hour

// Canonical author online chess handles used to form the author computer exemption ID.
const (
	AuthorChessComPrimary = "christopherfair"
	AuthorChessComAlt     = "chrisfair"
	AuthorLichessPrimary  = "quenyenhighelf"
	AuthorLichessAlt      = "quenyen"
)

// StripeURLForAmount returns the live Stripe checkout link for the selected tier ($1, $5, $10, $35, $350).
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
	case amountUSD >= 35:
		return StripeLink35USD
	case amountUSD >= 10:
		return StripeLink10USD
	case amountUSD >= 5:
		return StripeLink5USD
	default:
		return StripeLink1USD
	}
}

// CooldownForAmount returns the nag-free duration for a given support amount in USD,
// plus a human-readable tier label. A duration of 0 with lifetime=true means permanent.
func CooldownForAmount(amountUSD int) (time.Duration, string, bool) {
	switch {
	case amountUSD >= 350:
		return 0, "$350 King (Lifetime)", true
	case amountUSD >= 35:
		return 365 * 24 * time.Hour, "$35 Queen (1 Year)", false
	case amountUSD >= 10:
		return 90 * 24 * time.Hour, "$10 Rook (3 Months)", false
	case amountUSD >= 5:
		return 30 * 24 * time.Hour, "$5 Knight (1 Month)", false
	default:
		return 3 * 24 * time.Hour, "$1 Pawn (3 Days)", false
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
	Supported                bool   `json:"supported"`
	IsLifetime               bool   `json:"isLifetime,omitempty"`
	IsAuthor                 bool   `json:"isAuthor"`
	AuthorID                 string `json:"authorId,omitempty"`
	SupporterHandle          string `json:"supporterHandle,omitempty"`
	SupporterTier            string `json:"supporterTier,omitempty"`
	Reference                string `json:"reference,omitempty"`
	SupportedAt              int64  `json:"supportedAt,omitempty"`
	SupportedUntil           int64  `json:"supportedUntil,omitempty"`
	DismissCount             int    `json:"dismissCount"`
	LastDismissedAt          int64  `json:"lastDismissedAt,omitempty"`
	StripeURL                string `json:"stripeUrl,omitempty"`
	BughouseOnlineDonorToken string `json:"bughouseOnlineDonorToken,omitempty"`
	RemoteSynced             bool   `json:"remoteSynced,omitempty"`
}

// SupportStatus is returned to the frontend UI to determine whether to show the support form.
type SupportStatus struct {
	Supported                bool           `json:"supported"`
	IsLifetime               bool           `json:"isLifetime"`
	IsAuthor                 bool           `json:"isAuthor"`
	IsWindows                bool           `json:"isWindows"`
	IsPackagedApp            bool           `json:"isPackagedApp"`
	IsStorePurchase          bool           `json:"isStorePurchase"`
	CanPurchaseInApp         bool           `json:"canPurchaseInApp"`
	StorePriceUSD            float64        `json:"storePriceUsd"`
	StoreVerificationNote    string         `json:"storeVerificationNote,omitempty"`
	RemoteSynced             bool           `json:"remoteSynced"`
	SupporterHandle          string         `json:"supporterHandle,omitempty"`
	AuthorID                 string         `json:"authorId,omitempty"`
	DeviceSignature          string         `json:"deviceSignature,omitempty"`
	SupporterTier            string         `json:"supporterTier,omitempty"`
	SupportedAt              int64          `json:"supportedAt,omitempty"`
	SupportedUntil           int64          `json:"supportedUntil,omitempty"`
	DismissCount             int            `json:"dismissCount"`
	LastDismissedAt          int64          `json:"lastDismissedAt,omitempty"`
	StripeURL                string         `json:"stripeUrl"`
	BughouseOnlineDonorToken string         `json:"bughouseOnlineDonorToken,omitempty"`
	TierLinks                map[int]string `json:"tierLinks"`
	MinAmountUSD             int            `json:"minAmountUsd"`
	ShouldShowNag            bool           `json:"shouldShowNag"`
}

// FeatureEntitlement describes whether a premium training/analytics feature (such as Cyclical Tactical Drill)
// is unlocked, and tracks readiness for future bughouse-online supporter synchronization.
type FeatureEntitlement struct {
	FeatureID          string `json:"featureId"`
	Unlocked           bool   `json:"unlocked"`
	EnforceGate        bool   `json:"enforceGate"`
	SupporterActive    bool   `json:"supporterActive"`
	BughouseOnlineSync bool   `json:"bughouseOnlineSync"`
	TierRequired       string `json:"tierRequired"`
	Message            string `json:"message"`
}

// Manager manages support/donation status, Stripe link configuration, Microsoft Store license verification,
// shared supporter database sync, and author ID exemption.
type Manager struct {
	mu           sync.RWMutex
	configPath   string
	config       SupportConfig
	storeChecker StoreLicenseChecker
	remoteClient *RemoteSupporterClient
}

// NewManager initializes a support manager backed by the given JSON file path.
// If configPath is empty, it defaults to ~/.config/newscid/support.json.
func NewManager(configPath string) (*Manager, error) {
	if strings.TrimSpace(configPath) == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			cfgDir = os.TempDir()
		}
		configPath = filepath.Join(cfgDir, fileformat.AppDirName, fileformat.FileSupportJSON)
	}

	m := &Manager{
		configPath: configPath,
		config: SupportConfig{
			StripeURL: resolveDefaultStripeURL(),
		},
		storeChecker: DefaultStoreLicenseChecker(),
		remoteClient: NewRemoteSupporterClient(""),
	}

	_ = m.load()
	if strings.TrimSpace(m.config.StripeURL) == "" {
		m.config.StripeURL = resolveDefaultStripeURL()
	}

	return m, nil
}

// SetStoreLicenseChecker overrides the Microsoft Store license checker (primarily for unit tests).
func (m *Manager) SetStoreLicenseChecker(checker StoreLicenseChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if checker == nil {
		m.storeChecker = DefaultStoreLicenseChecker()
		return
	}
	m.storeChecker = checker
}

// SetRemoteClient overrides the remote supporter database client (primarily for unit tests).
func (m *Manager) SetRemoteClient(client *RemoteSupporterClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.remoteClient = client
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

// parseTierAmount extracts the USD integer from a tier string like "$10 Support" or "$35 Queen".
func parseTierAmount(tier string) int {
	switch {
	case strings.Contains(tier, "350") || strings.Contains(strings.ToLower(tier), "lifetime") || strings.Contains(strings.ToLower(tier), "king"):
		return 350
	case strings.Contains(tier, "35") || strings.Contains(strings.ToLower(tier), "queen"):
		return 35
	case strings.Contains(tier, "10") || strings.Contains(strings.ToLower(tier), "rook"):
		return 10
	case strings.Contains(tier, "1") && !strings.Contains(tier, "10") && !strings.Contains(tier, "15") || strings.Contains(strings.ToLower(tier), "pawn"):
		return 1
	default:
		return 5
	}
}

// RecordSupport marks the user as having supported NewScid for the duration corresponding to their tier.
func (m *Manager) RecordSupport(tier string, reference string) (SupportStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanTier := strings.TrimSpace(tier)
	amt := parseTierAmount(cleanTier)
	dur, canonicalLabel, isLifetime := CooldownForAmount(amt)
	if cleanTier == "" {
		cleanTier = canonicalLabel
	}
	cleanRef := strings.TrimSpace(reference)

	now := time.Now()
	// Check if the optional reference entered matches one of the author's handles
	isAuthor, compositeID, sig := BuildAuthorCompositeID([]string{cleanRef}, []string{cleanRef})
	if isAuthor {
		m.config.IsAuthor = true
		m.config.IsLifetime = true
		m.config.AuthorID = compositeID
		cleanTier = "Author"
		m.config.SupportedUntil = 0
	} else if isLifetime {
		m.config.IsLifetime = true
		m.config.SupportedUntil = 0
	} else {
		m.config.IsLifetime = false
		m.config.SupportedUntil = now.Add(dur).UnixMilli()
	}

	m.config.Supported = true
	m.config.SupporterTier = cleanTier
	m.config.Reference = cleanRef
	m.config.SupportedAt = now.UnixMilli()

	// Sync contribution status to the shared FairPlay Radar / NewScid SQLite database when a handle/reference is provided.
	syncHandle := cleanRef
	if syncHandle == "" {
		syncHandle = strings.TrimSpace(m.config.SupporterHandle)
	}
	if !isAuthor && syncHandle != "" && syncHandle != "stripe_checkout" && m.remoteClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		rec, rerr := m.remoteClient.RegisterContribution(ctx, syncHandle, cleanTier, "newscid_support")
		cancel()
		if rerr == nil && rec != nil && rec.Supported {
			m.config.SupporterHandle = rec.SupporterID
			m.config.RemoteSynced = true
		}
	}

	err := m.save()
	return m.buildStatusLocked(sig), err
}

// SyncRemoteSupporter queries the shared FairPlay Radar / NewScid supporter database for a handle/email/receipt
// and caches verified supporter status locally so the nag screen stays disabled across devices.
func (m *Manager) SyncRemoteSupporter(handle string) (SupportStatus, error) {
	clean := strings.TrimSpace(handle)
	if clean == "" {
		return m.GetStatus(nil, nil), fmt.Errorf("supporter handle or email is required")
	}

	m.mu.RLock()
	client := m.remoteClient
	m.mu.RUnlock()
	if client == nil {
		return m.GetStatus(nil, nil), fmt.Errorf("remote supporter client is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rec, err := client.FetchSupporterStatus(ctx, clean)
	if err != nil {
		return m.GetStatus(nil, nil), err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if rec.Supported {
		m.config.Supported = true
		m.config.SupporterHandle = rec.SupporterID
		if m.config.SupporterHandle == "" {
			m.config.SupporterHandle = clean
		}
		if strings.TrimSpace(rec.TierLabel) != "" {
			m.config.SupporterTier = rec.TierLabel
		} else if m.config.SupporterTier == "" {
			m.config.SupporterTier = "Shared Supporter"
		}
		if rec.SupportedAtMs > 0 {
			m.config.SupportedAt = rec.SupportedAtMs
		} else if m.config.SupportedAt == 0 {
			m.config.SupportedAt = time.Now().UnixMilli()
		}
		m.config.RemoteSynced = true
		_ = m.save()
	}
	return m.buildStatusLocked(""), nil
}

// PurchaseMicrosoftStoreUnlock invokes the native Windows Store $9.99 permanent durable add-on checkout.
// When Succeeded or AlreadyPurchased, permanent lifetime supporter access is persisted locally (and synced
// to the shared database if a handle is configured).
func (m *Manager) PurchaseMicrosoftStoreUnlock(ctx context.Context) (StorePurchaseResult, SupportStatus) {
	m.mu.RLock()
	checker := m.storeChecker
	m.mu.RUnlock()

	if checker == nil {
		checker = DefaultStoreLicenseChecker()
	}
	res := checker.PurchasePermanentUnlock(ctx)
	if res.Purchased {
		m.mu.Lock()
		m.config.Supported = true
		m.config.IsLifetime = true
		m.config.SupportedUntil = 0
		m.config.SupporterTier = "Microsoft Store ($9.99 Permanent)"
		if m.config.SupportedAt == 0 {
			m.config.SupportedAt = time.Now().UnixMilli()
		}
		if h := strings.TrimSpace(m.config.SupporterHandle); h != "" && m.remoteClient != nil {
			rctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			rec, rerr := m.remoteClient.RegisterContribution(rctx, h, m.config.SupporterTier, "msstore_999_permanent")
			cancel()
			if rerr == nil && rec != nil && rec.Supported {
				m.config.RemoteSynced = true
			}
		}
		_ = m.save()
		st := m.buildStatusLocked("")
		m.mu.Unlock()
		return res, st
	}

	return res, m.GetStatus(nil, nil)
}

// DismissNag records that the user dismissed the support prompt without donating (24-hour cooldown).
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

// SetBughouseOnlineToken stores a donor synchronization token from bughouse-online and marks Supported = true when non-empty.
func (m *Manager) SetBughouseOnlineToken(token string) (SupportStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	clean := strings.TrimSpace(token)
	m.config.BughouseOnlineDonorToken = clean
	if clean != "" {
		m.config.Supported = true
		if m.config.SupporterTier == "" {
			m.config.SupporterTier = "bughouse-online Supporter"
		}
		if m.config.SupportedAt == 0 {
			m.config.SupportedAt = time.Now().UnixMilli()
		}
	}

	err := m.save()
	return m.buildStatusLocked(""), err
}

// CheckFeatureEntitlement checks whether a feature (e.g., "cyclical-tactical-drill") is unlocked for the user.
// While enforceGate is false, features are unlocked for all users with an informational readiness message.
func (m *Manager) CheckFeatureEntitlement(featureID string, enforceGate bool, chessComIDs, lichessIDs []string) FeatureEntitlement {
	status := m.GetStatus(chessComIDs, lichessIDs)

	m.mu.RLock()
	donorToken := strings.TrimSpace(m.config.BughouseOnlineDonorToken)
	m.mu.RUnlock()

	bughouseSync := donorToken != ""
	supporterActive := status.Supported || status.IsLifetime || status.IsAuthor || bughouseSync

	unlocked := true
	msg := "Included in NewScid Workstation (Future bughouse-online Supporter Feature)"
	if enforceGate {
		unlocked = supporterActive
		if unlocked {
			msg = "Unlocked via Active Supporter / bughouse-online Entitlement"
		} else {
			msg = "Requires Active Supporter or bughouse-online Donor Entitlement"
		}
	}

	return FeatureEntitlement{
		FeatureID:          featureID,
		Unlocked:           unlocked,
		EnforceGate:        enforceGate,
		SupporterActive:    supporterActive,
		BughouseOnlineSync: bughouseSync,
		TierRequired:       "$5 Knight (1 Month) or bughouse-online Donor",
		Message:            msg,
	}
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

	storeInfo := StoreLicenseInfo{}
	if m.storeChecker != nil {
		storeInfo = m.storeChecker.Check()
	}
	isStorePurchase := storeInfo.IsWindows && storeInfo.IsStorePurchased

	nowMs := time.Now().UnixMilli()
	activeSupport := false
	supporterTier := m.config.SupporterTier
	if isStorePurchase {
		activeSupport = true
		if supporterTier == "" {
			supporterTier = "Microsoft Store ($9.99 Permanent)"
		}
	} else if m.config.IsAuthor || m.config.IsLifetime || strings.TrimSpace(m.config.BughouseOnlineDonorToken) != "" {
		activeSupport = true
	} else if m.config.Supported {
		if m.config.SupportedUntil > 0 {
			activeSupport = nowMs < m.config.SupportedUntil
		} else {
			activeSupport = true
		}
	}

	inDailyCooldown := false
	if !activeSupport && m.config.LastDismissedAt > 0 {
		elapsed := time.Since(time.UnixMilli(m.config.LastDismissedAt))
		if elapsed >= 0 && elapsed < NagCooldownDuration {
			inDailyCooldown = true
		}
	}

	return SupportStatus{
		Supported:                activeSupport,
		IsLifetime:               m.config.IsLifetime || m.config.IsAuthor || isStorePurchase,
		IsAuthor:                 m.config.IsAuthor,
		IsWindows:                storeInfo.IsWindows,
		IsPackagedApp:            storeInfo.IsPackaged,
		IsStorePurchase:          isStorePurchase,
		CanPurchaseInApp:         storeInfo.CanPurchaseInApp && !isStorePurchase,
		StorePriceUSD:            StorePermanentPriceUSD,
		StoreVerificationNote:    storeInfo.VerificationNote,
		RemoteSynced:             m.config.RemoteSynced,
		SupporterHandle:          m.config.SupporterHandle,
		AuthorID:                 m.config.AuthorID,
		DeviceSignature:          sig,
		SupporterTier:            supporterTier,
		SupportedAt:              m.config.SupportedAt,
		SupportedUntil:           m.config.SupportedUntil,
		DismissCount:             m.config.DismissCount,
		LastDismissedAt:          m.config.LastDismissedAt,
		StripeURL:                stripeURL,
		BughouseOnlineDonorToken: m.config.BughouseOnlineDonorToken,
		TierLinks: map[int]string{
			1:   StripeLink1USD,
			5:   StripeLink5USD,
			10:  StripeLink10USD,
			35:  StripeLink35USD,
			350: StripeLink350USD,
		},
		MinAmountUSD:  DefaultMinSupportUSD,
		ShouldShowNag: !activeSupport && !inDailyCooldown,
	}
}


