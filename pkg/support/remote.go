package support

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

// DefaultSupporterAPIBase is the shared SQLite supporter ledger used by FairPlay Radar (cheatdetectorplugin) and NewScid.
const DefaultSupporterAPIBase = "https://bughouse-online.com/fairplay-telemetry"

// RemoteSupporterRecord matches the JSON response from GET/POST /v1/supporter.
type RemoteSupporterRecord struct {
	Supported     bool   `json:"supported"`
	SupporterID   string `json:"supporterId"`
	PlanID        string `json:"planId"`
	TierLabel     string `json:"tierLabel"`
	SupportedAtMs int64  `json:"supportedAtMs,omitempty"`
	Verified      bool   `json:"verified,omitempty"`
	AppSource     string `json:"appSource,omitempty"`
	Error         string `json:"error,omitempty"`
}

// RemoteSupporterClient communicates with the shared FairPlay Radar / NewScid supporter database.
type RemoteSupporterClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewRemoteSupporterClient returns a client configured for the shared supporter database.
func NewRemoteSupporterClient(baseURL string) *RemoteSupporterClient {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		if env := strings.TrimSpace(os.Getenv("NEWSCID_SUPPORTER_API")); env != "" {
			base = env
		} else {
			base = DefaultSupporterAPIBase
		}
	}
	return &RemoteSupporterClient{
		BaseURL: strings.TrimRight(base, "/"),
		HTTPClient: &http.Client{
			Timeout: 2500 * time.Millisecond,
		},
	}
}

// FetchSupporterStatus queries GET /v1/supporter?handle=<handle>&source=newscid.
func (c *RemoteSupporterClient) FetchSupporterStatus(ctx context.Context, handle string) (*RemoteSupporterRecord, error) {
	clean := strings.TrimSpace(handle)
	if clean == "" {
		return nil, fmt.Errorf("empty handle")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	endpoint := fmt.Sprintf("%s/v1/supporter?handle=%s&source=newscid", c.BaseURL, url.QueryEscape(clean))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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
		return nil, fmt.Errorf("supporter api returned HTTP %d", resp.StatusCode)
	}
	var rec RemoteSupporterRecord
	if err := json.NewDecoder(resp.Body).Decode(&rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// RegisterContribution posts a NewScid contribution record to POST /v1/supporter with app="newscid".
func (c *RemoteSupporterClient) RegisterContribution(ctx context.Context, handle, tierLabel, stripeRef string) (*RemoteSupporterRecord, error) {
	clean := strings.TrimSpace(handle)
	if clean == "" {
		return nil, fmt.Errorf("empty handle")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	payload := map[string]any{
		"handle":    clean,
		"app":       "newscid",
		"planId":    "newscid_supporter",
		"tierLabel": strings.TrimSpace(tierLabel),
		"stripeRef": strings.TrimSpace(stripeRef),
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/supporter", bytes.NewReader(bodyBytes))
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

	var rec RemoteSupporterRecord
	_ = json.NewDecoder(resp.Body).Decode(&rec)
	if resp.StatusCode != http.StatusOK {
		if rec.Error != "" {
			return &rec, fmt.Errorf("%s", rec.Error)
		}
		return nil, fmt.Errorf("supporter api returned HTTP %d", resp.StatusCode)
	}
	return &rec, nil
}
