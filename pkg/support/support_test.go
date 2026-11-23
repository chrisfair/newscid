package support

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupportManager_DefaultAndDismiss(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "support.json")

	mgr, err := NewManager(cfgPath)
	require.NoError(t, err)

	// Initially not supported and not author -> should show nag
	st := mgr.GetStatus([]string{"RandomUser"}, []string{"AnotherUser"})
	assert.False(t, st.Supported)
	assert.False(t, st.IsAuthor)
	assert.True(t, st.ShouldShowNag)
	assert.Equal(t, DefaultMinSupportUSD, st.MinAmountUSD)
	assert.Equal(t, DefaultStripeSupportURL, st.StripeURL)
	assert.Equal(t, 0, st.DismissCount)

	// Dismissing nag increments dismiss count and suppresses the prompt for 24 hours (one full day)
	stDismiss, err := mgr.DismissNag()
	require.NoError(t, err)
	assert.Equal(t, 1, stDismiss.DismissCount)
	assert.Greater(t, stDismiss.LastDismissedAt, int64(0))
	assert.False(t, stDismiss.ShouldShowNag, "prompt should stay dismissed for 24 hours")

	// Reload from disk to verify persistence during the 24h window
	mgr2, err := NewManager(cfgPath)
	require.NoError(t, err)
	st2 := mgr2.GetStatus(nil, nil)
	assert.Equal(t, 1, st2.DismissCount)
	assert.False(t, st2.ShouldShowNag)

	// Simulate 25 hours having passed since dismissal -> should show once per day again
	mgr2.config.LastDismissedAt -= (25 * 3600 * 1000)
	require.NoError(t, mgr2.save())
	stAfterDay := mgr2.GetStatus(nil, nil)
	assert.True(t, stAfterDay.ShouldShowNag, "prompt should appear again after 24 hours if not supported")
}

func TestSupportManager_RecordSupportDisablesNag(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "support.json")

	mgr, err := NewManager(cfgPath)
	require.NoError(t, err)

	st, err := mgr.RecordSupport("$10 Supporter", "stripe_rcpt_123")
	require.NoError(t, err)
	assert.True(t, st.Supported)
	assert.False(t, st.ShouldShowNag)
	assert.Equal(t, "$10 Supporter", st.SupporterTier)
	assert.Greater(t, st.SupportedAt, int64(0))

	// Reload and confirm nag remains disabled permanently
	mgrReloaded, err := NewManager(cfgPath)
	require.NoError(t, err)
	stReloaded := mgrReloaded.GetStatus(nil, nil)
	assert.True(t, stReloaded.Supported)
	assert.False(t, stReloaded.ShouldShowNag)
	assert.Equal(t, "$10 Supporter", stReloaded.SupporterTier)
}

func TestSupportManager_AuthorExemptionByChessComAndLichessIDs(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "support.json")

	mgr, err := NewManager(cfgPath)
	require.NoError(t, err)

	// Passing the author's Chess.com and Lichess.org IDs automatically exempts the machine
	st := mgr.GetStatus([]string{"ChristopherFair"}, []string{"QuenyenHighElf"})
	assert.True(t, st.IsAuthor)
	assert.True(t, st.Supported)
	assert.False(t, st.ShouldShowNag)
	assert.Equal(t, "chess.com:ChristopherFair|lichess.org:QuenyenHighElf", st.AuthorID)
	assert.NotEmpty(t, st.DeviceSignature)
	assert.Equal(t, "Author", st.SupporterTier)

	// Even if queried later with empty lists, persisted author status remains active
	mgrReloaded, err := NewManager(cfgPath)
	require.NoError(t, err)
	stReloaded := mgrReloaded.GetStatus(nil, nil)
	assert.True(t, stReloaded.IsAuthor)
	assert.True(t, stReloaded.Supported)
	assert.False(t, stReloaded.ShouldShowNag)
	assert.NotEmpty(t, stReloaded.DeviceSignature)
}

func TestSupportManager_RecordSupportWithAuthorHandle(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "support.json")

	mgr, err := NewManager(cfgPath)
	require.NoError(t, err)

	// Entering QuenyenHighElf or ChristopherFair in the support form also activates author mode
	st, err := mgr.RecordSupport("", "QuenyenHighElf")
	require.NoError(t, err)
	assert.True(t, st.IsAuthor)
	assert.True(t, st.Supported)
	assert.False(t, st.ShouldShowNag)
	assert.Equal(t, "Author", st.SupporterTier)
}

func TestSupportManager_CustomStripeURLAndEnv(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "support.json")

	t.Setenv("NEWSCID_STRIPE_URL", "https://buy.stripe.com/custom-env-link")
	mgr, err := NewManager(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "https://buy.stripe.com/custom-env-link", mgr.GetStatus(nil, nil).StripeURL)

	err = mgr.SetStripeURL("https://buy.stripe.com/explicit-link")
	require.NoError(t, err)
	assert.Equal(t, "https://buy.stripe.com/explicit-link", mgr.GetStatus(nil, nil).StripeURL)

	// Resetting to empty falls back to default/env
	err = mgr.SetStripeURL("   ")
	require.NoError(t, err)
	assert.Equal(t, "https://buy.stripe.com/custom-env-link", mgr.GetStatus(nil, nil).StripeURL)

	// Default path constructor when empty string is passed
	defaultMgr, err := NewManager("")
	require.NoError(t, err)
	assert.NotNil(t, defaultMgr)

	// Corrupt JSON file is handled gracefully
	corruptPath := filepath.Join(tmpDir, "corrupt.json")
	require.NoError(t, os.WriteFile(corruptPath, []byte("{invalid"), 0o644))
	corruptMgr, err := NewManager(corruptPath)
	require.NoError(t, err)
	assert.NotNil(t, corruptMgr)

	// JSON file with empty stripeUrl falls back to default
	emptyURLPath := filepath.Join(tmpDir, "empty_url.json")
	require.NoError(t, os.WriteFile(emptyURLPath, []byte(`{"supported":true,"stripeUrl":""}`), 0o644))
	emptyURLMgr, err := NewManager(emptyURLPath)
	require.NoError(t, err)
	emptyURLMgr.config.StripeURL = ""
	assert.NotEmpty(t, emptyURLMgr.GetStatus(nil, nil).StripeURL)

	// Partial match with only Chess.com handle fills in canonical Lichess handle
	ok, comp, sig := BuildAuthorCompositeID([]string{"ChristopherFair"}, nil)
	assert.True(t, ok)
	assert.Equal(t, "chess.com:ChristopherFair|lichess.org:QuenyenHighElf", comp)
	assert.NotEmpty(t, sig)
}

func TestStripeURLForAmountAndCheckoutBuilder(t *testing.T) {
	t.Setenv("NEWSCID_STRIPE_URL", "")
	assert.Equal(t, StripeLink1USD, StripeURLForAmount(1, ""))
	assert.Equal(t, StripeLink5USD, StripeURLForAmount(5, ""))
	assert.Equal(t, StripeLink10USD, StripeURLForAmount(10, ""))
	assert.Equal(t, StripeLink35USD, StripeURLForAmount(35, ""))
	assert.Equal(t, StripeLink350USD, StripeURLForAmount(350, ""))
	assert.Equal(t, "https://buy.stripe.com/custom-override", StripeURLForAmount(10, "https://buy.stripe.com/custom-override"))

	assert.Equal(t, StripeLink5USD, BuildCheckoutURL(StripeLink5USD, ""))
	assert.Equal(t, StripeLink5USD+"?client_reference_id=user123", BuildCheckoutURL(StripeLink5USD, "user123"))
	assert.Equal(t, StripeLink5USD+"?prefilled_promo=1&client_reference_id=user123", BuildCheckoutURL(StripeLink5USD+"?prefilled_promo=1", "user123"))

	tmpDir := t.TempDir()
	mgr, err := NewManager(filepath.Join(tmpDir, "tiers.json"))
	require.NoError(t, err)

	// Test $1 (3 days), $5 (1 month), $10 (3 months), $35 (1 year)
	for _, tier := range []string{"$1 Support", "$5 Support", "$10 Support", "$35 Support"} {
		st, err := mgr.RecordSupport(tier, "rcpt")
		require.NoError(t, err)
		assert.True(t, st.Supported)
		assert.False(t, st.IsLifetime)
		assert.Greater(t, st.SupportedUntil, st.SupportedAt)
	}

	// Simulate $1 tier cooldown expiring -> ShouldShowNag becomes true again
	mgr.config.SupportedUntil = mgr.config.SupportedAt - 1000
	require.NoError(t, mgr.save())
	stExpired := mgr.GetStatus(nil, nil)
	assert.False(t, stExpired.Supported)
	assert.True(t, stExpired.ShouldShowNag)

	// Test $350 King Lifetime never expires
	stLife, err := mgr.RecordSupport("$350 Support", "rcpt_king")
	require.NoError(t, err)
	assert.True(t, stLife.Supported)
	assert.True(t, stLife.IsLifetime)
	assert.False(t, stLife.ShouldShowNag)
}

func TestBughouseOnlineTokenAndFeatureEntitlement(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "entitlement.json")

	mgr, err := NewManager(cfgPath)
	require.NoError(t, err)

	// Before support, enforceGate = false unlocks feature with readiness message
	entFree := mgr.CheckFeatureEntitlement("cyclical-tactical-drill", false, nil, nil)
	assert.Equal(t, "cyclical-tactical-drill", entFree.FeatureID)
	assert.True(t, entFree.Unlocked)
	assert.False(t, entFree.EnforceGate)
	assert.False(t, entFree.SupporterActive)
	assert.False(t, entFree.BughouseOnlineSync)
	assert.Equal(t, "Included in NewScid Workstation (Future bughouse-online Supporter Feature)", entFree.Message)

	// Before support, enforceGate = true locks feature
	entGated := mgr.CheckFeatureEntitlement("cyclical-tactical-drill", true, nil, nil)
	assert.False(t, entGated.Unlocked)
	assert.True(t, entGated.EnforceGate)
	assert.False(t, entGated.SupporterActive)
	assert.Contains(t, entGated.Message, "Requires Active Supporter")

	// Setting a non-empty bughouse-online donor token marks Supported = true and unlocks gated feature
	stToken, err := mgr.SetBughouseOnlineToken("bh_donor_token_abc123")
	require.NoError(t, err)
	assert.True(t, stToken.Supported)
	assert.Equal(t, "bh_donor_token_abc123", stToken.BughouseOnlineDonorToken)
	assert.Equal(t, "bughouse-online Supporter", stToken.SupporterTier)

	entAfterToken := mgr.CheckFeatureEntitlement("cyclical-tactical-drill", true, nil, nil)
	assert.True(t, entAfterToken.Unlocked)
	assert.True(t, entAfterToken.SupporterActive)
	assert.True(t, entAfterToken.BughouseOnlineSync)
	assert.Contains(t, entAfterToken.Message, "Unlocked via Active Supporter")

	// Clearing the token
	stCleared, err := mgr.SetBughouseOnlineToken("   ")
	require.NoError(t, err)
	assert.Empty(t, stCleared.BughouseOnlineDonorToken)
}




type mockStoreChecker struct {
	info        StoreLicenseInfo
	purchaseRes StorePurchaseResult
}

func (m mockStoreChecker) Check() StoreLicenseInfo {
	return m.info
}

func (m mockStoreChecker) PurchasePermanentUnlock(_ context.Context) StorePurchaseResult {
	return m.purchaseRes
}

func TestSupportManager_WindowsStorePurchaseVsCompiledBinary(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "support.json")

	mgr, err := NewManager(cfgPath)
	require.NoError(t, err)

	// 1. Compiled (unpackaged) Windows .exe: nag MUST remain intact normally.
	mgr.SetStoreLicenseChecker(mockStoreChecker{
		info: StoreLicenseInfo{
			IsWindows:        true,
			IsPackaged:       false,
			IsStorePurchased: false,
			VerificationNote: "compiled unpackaged Windows binary",
		},
	})
	stCompiled := mgr.GetStatus(nil, nil)
	assert.True(t, stCompiled.IsWindows)
	assert.False(t, stCompiled.IsPackagedApp)
	assert.False(t, stCompiled.IsStorePurchase)
	assert.False(t, stCompiled.Supported)
	assert.True(t, stCompiled.ShouldShowNag, "compiled Windows binary must keep nag screen intact")

	// 2. Microsoft Store trial MSIX: nag MUST still show.
	mgr.SetStoreLicenseChecker(mockStoreChecker{
		info: StoreLicenseInfo{
			IsWindows:        true,
			IsPackaged:       true,
			IsStorePurchased: false,
			IsTrial:          true,
			PackageFullName:  "FairComputing.NewScid_1.3.45.0_x64__abc",
			VerificationNote: "Microsoft Store trial",
		},
	})
	stTrial := mgr.GetStatus(nil, nil)
	assert.True(t, stTrial.IsPackagedApp)
	assert.False(t, stTrial.IsStorePurchase)
	assert.True(t, stTrial.ShouldShowNag)

	// 3. Verified Microsoft Store $9.99 permanent purchase (Store-signed MSIX + active add-on license): nag suppressed.
	mgr.SetStoreLicenseChecker(mockStoreChecker{
		info: StoreLicenseInfo{
			IsWindows:        true,
			IsPackaged:       true,
			IsStorePurchased: true,
			PackageFullName:  "FairComputing.NewScid_1.3.45.0_x64__abc",
			VerificationNote: "verified Microsoft Store $9.99 permanent license",
		},
	})
	stStore := mgr.GetStatus(nil, nil)
	assert.True(t, stStore.IsWindows)
	assert.True(t, stStore.IsPackagedApp)
	assert.True(t, stStore.IsStorePurchase)
	assert.True(t, stStore.Supported)
	assert.True(t, stStore.IsLifetime)
	assert.Equal(t, "Microsoft Store ($9.99 Permanent)", stStore.SupporterTier)
	assert.False(t, stStore.ShouldShowNag, "verified Microsoft Store purchase must disable nag screen")

	// 4. Free Store Edition without add-on -> nag stays; completing $9.99 IAP permanently unlocks.
	cfgPathFree := filepath.Join(tmpDir, "support_free_store.json")
	mgrFree, err := NewManager(cfgPathFree)
	require.NoError(t, err)
	mgrFree.SetStoreLicenseChecker(mockStoreChecker{
		info: StoreLicenseInfo{
			IsWindows:        true,
			IsPackaged:       true,
			IsStorePurchased: false,
			CanPurchaseInApp: true,
			AddOnStoreID:     "9NTESTADDON12",
			VerificationNote: "Microsoft Store Free Edition (upgrade available for $9.99)",
		},
		purchaseRes: StorePurchaseResult{
			Purchased: true,
			Status:    "Succeeded",
			Message:   "Microsoft Store $9.99 Permanent Unlock verified!",
		},
	})
	stFree := mgrFree.GetStatus(nil, nil)
	assert.True(t, stFree.IsPackagedApp)
	assert.True(t, stFree.CanPurchaseInApp)
	assert.False(t, stFree.IsStorePurchase)
	assert.True(t, stFree.ShouldShowNag, "Free Store edition without $9.99 purchase must keep nag intact")

	pRes, stAfterBuy := mgrFree.PurchaseMicrosoftStoreUnlock(context.Background())
	assert.True(t, pRes.Purchased)
	assert.Equal(t, "Succeeded", pRes.Status)
	assert.True(t, stAfterBuy.Supported)
	assert.True(t, stAfterBuy.IsLifetime)
	assert.False(t, stAfterBuy.ShouldShowNag)
	assert.Equal(t, "Microsoft Store ($9.99 Permanent)", stAfterBuy.SupporterTier)

	// Reset checker to default (nil fallback)
	mgr.SetStoreLicenseChecker(nil)
	stDefault := mgr.GetStatus(nil, nil)
	assert.False(t, stDefault.IsStorePurchase)
}

func TestSupportManager_SharedRemoteSupporterDBSync(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "support.json")

	registered := map[string]RemoteSupporterRecord{}
	srv := newTestSupporterServer(t, registered)
	defer srv.Close()

	mgr, err := NewManager(cfgPath)
	require.NoError(t, err)
	mgr.SetRemoteClient(NewRemoteSupporterClient(srv.URL))

	// Record a contribution with handle -> pushes to shared remote supporter DB
	st, err := mgr.RecordSupport("$10 Rook", "chessfan42")
	require.NoError(t, err)
	assert.True(t, st.Supported)
	assert.True(t, st.RemoteSynced)
	assert.Equal(t, "chessfan42", st.SupporterHandle)
	assert.False(t, st.ShouldShowNag)

	// New machine (fresh local config) restores supporter status from shared DB
	cfgPath2 := filepath.Join(tmpDir, "support_machine2.json")
	mgr2, err := NewManager(cfgPath2)
	require.NoError(t, err)
	mgr2.SetRemoteClient(NewRemoteSupporterClient(srv.URL))

	stRestored, err := mgr2.SyncRemoteSupporter("chessfan42")
	require.NoError(t, err)
	assert.True(t, stRestored.Supported)
	assert.True(t, stRestored.RemoteSynced)
	assert.Equal(t, "chessfan42", stRestored.SupporterHandle)
	assert.False(t, stRestored.ShouldShowNag)

	// Empty handle error
	_, err = mgr2.SyncRemoteSupporter("   ")
	assert.Error(t, err)

	// Unregistered handle returns Supported=false without error
	cfgPath3 := filepath.Join(tmpDir, "support_machine3.json")
	mgr3, err := NewManager(cfgPath3)
	require.NoError(t, err)
	mgr3.SetRemoteClient(NewRemoteSupporterClient(srv.URL))
	stUnknown, err := mgr3.SyncRemoteSupporter("nobody_here")
	require.NoError(t, err)
	assert.False(t, stUnknown.Supported)
	assert.True(t, stUnknown.ShouldShowNag)

	// Nil remote client error
	mgr3.SetRemoteClient(nil)
	_, err = mgr3.SyncRemoteSupporter("chessfan42")
	assert.Error(t, err)
}

func TestSupport_PlatformStoreAndRemoteEdgeCases(t *testing.T) {
	// DefaultStoreLicenseChecker().PurchasePermanentUnlock on non-windows
	checker := DefaultStoreLicenseChecker()
	info := checker.Check()
	assert.False(t, info.IsStorePurchased)
	res := checker.PurchasePermanentUnlock(context.Background())
	assert.False(t, res.Purchased)
	assert.Equal(t, "Unavailable", res.Status)

	// NewRemoteSupporterClient env override and default
	t.Setenv("NEWSCID_SUPPORTER_API", "https://example.invalid/api/")
	envClient := NewRemoteSupporterClient("")
	assert.Equal(t, "https://example.invalid/api", envClient.BaseURL)
	t.Setenv("NEWSCID_SUPPORTER_API", "")
	defClient := NewRemoteSupporterClient("")
	assert.Equal(t, DefaultSupporterAPIBase, defClient.BaseURL)

	// StripeURLForAmount with env var
	t.Setenv("NEWSCID_STRIPE_URL", "https://buy.stripe.com/env-tier")
	assert.Equal(t, "https://buy.stripe.com/env-tier", StripeURLForAmount(10, ""))
	t.Setenv("NEWSCID_STRIPE_URL", "")

	// RemoteSupporterClient error paths
	var mode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch mode {
		case "500":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{}`))
		case "400_msg":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad handle"}`))
		case "bad_json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not-json`))
		case "minimal_ok":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"supported":true}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"supported":true,"supporterId":"alice","tierLabel":"$10 Rook"}`))
		}
	}))
	defer srv.Close()

	client := NewRemoteSupporterClient(srv.URL)

	// Empty handle errors
	_, err := client.FetchSupporterStatus(context.Background(), "   ")
	assert.Error(t, err)
	_, err = client.RegisterContribution(context.Background(), "   ", "tier", "ref")
	assert.Error(t, err)

	// nil context works
	mode = "minimal_ok"
	rec, err := client.FetchSupporterStatus(nil, "alice")
	require.NoError(t, err)
	assert.True(t, rec.Supported)

	rec, err = client.RegisterContribution(nil, "alice", "$5 Knight", "ref1")
	require.NoError(t, err)
	assert.True(t, rec.Supported)

	// HTTP 500
	mode = "500"
	_, err = client.FetchSupporterStatus(context.Background(), "alice")
	assert.Error(t, err)
	_, err = client.RegisterContribution(context.Background(), "alice", "$5 Knight", "ref1")
	assert.Error(t, err)

	// HTTP 400 with error message
	mode = "400_msg"
	recErr, err := client.RegisterContribution(context.Background(), "alice", "$5 Knight", "ref1")
	assert.ErrorContains(t, err, "bad handle")
	assert.NotNil(t, recErr)

	// Bad JSON on GET
	mode = "bad_json"
	_, err = client.FetchSupporterStatus(context.Background(), "alice")
	assert.Error(t, err)

	// Manager.SyncRemoteSupporter with minimal response (empty SupporterID, TierLabel, SupportedAtMs)
	tmpDir := t.TempDir()
	mgr, err := NewManager(filepath.Join(tmpDir, "sync_min.json"))
	require.NoError(t, err)
	mgr.SetRemoteClient(client)
	mode = "minimal_ok"
	st, err := mgr.SyncRemoteSupporter("bob_handle")
	require.NoError(t, err)
	assert.True(t, st.Supported)
	assert.Equal(t, "bob_handle", st.SupporterHandle)
	assert.Equal(t, "Shared Supporter", st.SupporterTier)
	assert.Greater(t, st.SupportedAt, int64(0))

	// Manager.SyncRemoteSupporter server error
	mode = "500"
	_, err = mgr.SyncRemoteSupporter("bob_handle")
	assert.Error(t, err)

	// Manager.RecordSupport with empty reference reuses existing SupporterHandle
	mode = "ok"
	st2, err := mgr.RecordSupport("$5 Knight", "")
	require.NoError(t, err)
	assert.True(t, st2.RemoteSynced)

	// Manager.PurchaseMicrosoftStoreUnlock with existing handle syncs remotely, and failed purchase path
	mgrStore, err := NewManager(filepath.Join(tmpDir, "store_sync.json"))
	require.NoError(t, err)
	mgrStore.config.SupporterHandle = "store_buyer"
	mgrStore.SetRemoteClient(client)
	mgrStore.SetStoreLicenseChecker(mockStoreChecker{
		purchaseRes: StorePurchaseResult{Purchased: true, Status: "Succeeded"},
	})
	pRes, stStore := mgrStore.PurchaseMicrosoftStoreUnlock(context.Background())
	assert.True(t, pRes.Purchased)
	assert.True(t, stStore.RemoteSynced)

	// Not purchased path & nil checker fallback
	mgrStore.storeChecker = nil
	pResFail, _ := mgrStore.PurchaseMicrosoftStoreUnlock(context.Background())
	assert.False(t, pResFail.Purchased)
	assert.Equal(t, "Unavailable", pResFail.Status)
}
