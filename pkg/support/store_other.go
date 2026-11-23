//go:build !windows

package support

import "context"

func checkPlatformStoreLicense() StoreLicenseInfo {
	return StoreLicenseInfo{
		IsWindows:        false,
		IsPackaged:       false,
		IsStorePurchased: false,
		CanPurchaseInApp: false,
		VerificationNote: "non-windows build",
	}
}

func purchasePlatformStoreUnlock(_ context.Context) StorePurchaseResult {
	return StorePurchaseResult{
		Purchased: false,
		Status:    "Unavailable",
		Message:   "Microsoft Store in-app purchase is only available in the Windows Store edition.",
	}
}
