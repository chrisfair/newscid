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

	// Dismissing nag increments dismiss count while keeping program non-obligatory
	stDismiss, err := mgr.DismissNag()
	require.NoError(t, err)
	assert.Equal(t, 1, stDismiss.DismissCount)
	assert.Greater(t, stDismiss.LastDismissedAt, int64(0))
	assert.True(t, stDismiss.ShouldShowNag)

	// Reload from disk to verify persistence
	mgr2, err := NewManager(cfgPath)
	require.NoError(t, err)
	st2 := mgr2.GetStatus(nil, nil)
	assert.Equal(t, 1, st2.DismissCount)
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
