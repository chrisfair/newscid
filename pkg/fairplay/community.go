package fairplay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// DefaultCommunityAPIBase is the shared FairPlay Radar / NewScid community telemetry server.
const DefaultCommunityAPIBase = "https://bughouse-online.com/fairplay-telemetry"

// CommunitySuspectIntel mirrors FairPlay Radar's CommunitySuspectIntel payload.
type CommunitySuspectIntel struct {
	Platform            string `json:"platform"`
	Username            string `json:"username"`
	CommunityFlagsCount int    `json:"communityFlagsCount"`
	ScansLast24h        int    `json:"scansLast24h"`
	AvgReportedRiskPct  int    `json:"avgReportedRiskPct"`
	ConfirmedBanned     bool   `json:"confirmedBanned"`
	BanReason           string `json:"banReason,omitempty"`
	FirstFlaggedAtMs    int64  `json:"firstFlaggedAtMs,omitempty"`
	LastSeenAtMs        int64  `json:"lastSeenAtMs"`
	CrowdVerdict        string `json:"crowdVerdict"` // "CLEAN" | "WATCHED" | "COMMUNITY_HOTLIST" | "CONFIRMED_BANNED"
	IsOfflineEstimate   bool   `json:"isOfflineEstimate,omitempty"`
}

// CommunityTelemetryClient talks to GET /v1/intel, POST /v1/flag, and GET /v1/hotlist.
type CommunityTelemetryClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewCommunityTelemetryClient creates a client pointing at the shared telemetry server.
func NewCommunityTelemetryClient(baseURL string) *CommunityTelemetryClient {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		if env := strings.TrimSpace(os.Getenv("NEWSCID_TELEMETRY_API")); env != "" {
			base = env
		} else {
			base = DefaultCommunityAPIBase
		}
	}
	return &CommunityTelemetryClient{
		BaseURL: strings.TrimRight(base, "/"),
		HTTPClient: &http.Client{
			Timeout: 2500 * time.Millisecond,
		},
	}
}

// FetchIntel queries GET /v1/intel and falls back to a deterministic offline estimate if unreachable.
func (c *CommunityTelemetryClient) FetchIntel(ctx context.Context, platform, username string, fallbackRiskPct int, confirmedBanned bool) CommunitySuspectIntel {
	plat := strings.ToLower(strings.TrimSpace(platform))
	if plat == "" {
		plat = "lichess"
	}
	user := strings.TrimSpace(username)
	if ctx == nil {
		ctx = context.Background()
	}

	if user != "" && c != nil && c.HTTPClient != nil {
		endpoint := fmt.Sprintf("%s/v1/intel?platform=%s&username=%s", c.BaseURL, url.QueryEscape(plat), url.QueryEscape(user))
		if req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil); err == nil {
			req.Header.Set("Accept", "application/json")
			req.Header.Set("User-Agent", "NewScid/1.3")
			if resp, err := c.HTTPClient.Do(req); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var remote CommunitySuspectIntel
					if decErr := json.NewDecoder(resp.Body).Decode(&remote); decErr == nil && remote.Username != "" {
						return remote
					}
				}
			}
		}
	}

	verdict := "CLEAN"
	flags := 0
	if confirmedBanned {
		verdict = "CONFIRMED_BANNED"
	} else if fallbackRiskPct >= 75 {
		verdict = "COMMUNITY_HOTLIST"
		flags = 3
	} else if fallbackRiskPct >= 50 {
		verdict = "WATCHED"
		flags = 1
	}

	return CommunitySuspectIntel{
		Platform:            plat,
		Username:            user,
		CommunityFlagsCount: flags,
		ScansLast24h:        1,
		AvgReportedRiskPct:  fallbackRiskPct,
		ConfirmedBanned:     confirmedBanned,
		LastSeenAtMs:        time.Now().UnixMilli(),
		CrowdVerdict:        verdict,
		IsOfflineEstimate:   true,
	}
}

// SubmitFlag reports a suspect flag or confirmed ban to POST /v1/flag.
func (c *CommunityTelemetryClient) SubmitFlag(ctx context.Context, platform, username string, riskPct int, confirmedBanned bool, banReason string) (*CommunitySuspectIntel, error) {
	plat := strings.ToLower(strings.TrimSpace(platform))
	if plat == "" {
		plat = "lichess"
	}
	user := strings.TrimSpace(username)
	if user == "" {
		return nil, fmt.Errorf("username is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	payload := map[string]any{
		"platform":        plat,
		"username":        user,
		"riskPct":         riskPct,
		"confirmedBanned": confirmedBanned,
		"banReason":       strings.TrimSpace(banReason),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/flag", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NewScid/1.3")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telemetry flag returned HTTP %d", resp.StatusCode)
	}
	var out CommunitySuspectIntel
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// FetchHotlist retrieves GET /v1/hotlist from the shared telemetry network.
func (c *CommunityTelemetryClient) FetchHotlist(ctx context.Context) ([]CommunitySuspectIntel, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/hotlist", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NewScid/1.3")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hotlist returned HTTP %d", resp.StatusCode)
	}
	var wrapper struct {
		OK      bool                    `json:"ok"`
		Hotlist []CommunitySuspectIntel `json:"hotlist"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}
	return wrapper.Hotlist, nil
}
