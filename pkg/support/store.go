package support

import "context"

// DefaultStoreAddOnOfferToken is the Partner Center In-App Offer Token for the $9.99 permanent unlock add-on.
// Override the Partner Center Store ID at build time or via NEWSCID_STORE_ADDON_ID.
const (
	DefaultStoreAddOnOfferToken = "newscid_pro_permanent"
	StorePermanentPriceUSD      = 9.99
)

// StoreLicenseInfo describes the runtime Windows / Microsoft Store packaging and license state.
type StoreLicenseInfo struct {
	IsWindows        bool   `json:"isWindows"`
	IsPackaged       bool   `json:"isPackaged"`
	IsStorePurchased bool   `json:"isStorePurchased"`
	IsTrial          bool   `json:"isTrial"`
	CanPurchaseInApp bool   `json:"canPurchaseInApp"`
	AddOnStoreID     string `json:"addOnStoreId,omitempty"`
	PackageFullName  string `json:"packageFullName,omitempty"`
	VerificationNote string `json:"verificationNote,omitempty"`
}

// StorePurchaseResult describes the outcome of a native Microsoft Store $9.99 permanent add-on purchase.
type StorePurchaseResult struct {
	Purchased bool   `json:"purchased"`
	Status    string `json:"status"` // "Succeeded" | "AlreadyPurchased" | "NotPurchased" | "NetworkError" | "ServerError" | "Unavailable"
	Message   string `json:"message"`
}

// StoreLicenseChecker verifies whether the current executable is running as a Microsoft Store MSIX
// package with either an active $9.99 permanent add-on license or paid Store app license.
// Compiled/unpackaged binaries and Free Store installs without the add-on always report IsStorePurchased=false
// so the nag screen remains intact.
type StoreLicenseChecker interface {
	Check() StoreLicenseInfo
	PurchasePermanentUnlock(ctx context.Context) StorePurchaseResult
}

type platformStoreChecker struct{}

func (platformStoreChecker) Check() StoreLicenseInfo {
	return checkPlatformStoreLicense()
}

func (platformStoreChecker) PurchasePermanentUnlock(ctx context.Context) StorePurchaseResult {
	return purchasePlatformStoreUnlock(ctx)
}

// DefaultStoreLicenseChecker returns the OS-native Store license checker.
func DefaultStoreLicenseChecker() StoreLicenseChecker {
	return platformStoreChecker{}
}
