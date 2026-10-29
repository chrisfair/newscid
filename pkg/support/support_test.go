package support

import (
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



