//go:build windows

package support

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	// APPMODEL_ERROR_NO_PACKAGE is returned by GetCurrentPackageFullName for unpackaged (compiled) Win32 apps.
	appModelErrorNoPackage = 15700
	// ERROR_INSUFFICIENT_BUFFER is returned when querying required length.
	errorInsufficientBuffer = 122
)

// StoreAddOnStoreID can be injected at build time via:
//   -ldflags "-X newscid/pkg/support.StoreAddOnStoreID=9NXXXXXXXXXX"
// or at runtime via NEWSCID_STORE_ADDON_ID.
var StoreAddOnStoreID = ""

var (
	storeStateMu     sync.Mutex
	storeStateLoaded bool
	cachedStoreState StoreLicenseInfo
)

func resolveStoreAddOnID() string {
	if env := strings.TrimSpace(os.Getenv("NEWSCID_STORE_ADDON_ID")); env != "" {
		return env
	}
	return strings.TrimSpace(StoreAddOnStoreID)
}

// getCurrentPackageFullName calls kernel32!GetCurrentPackageFullName.
// Unpackaged (compiled) Windows executables return ("", false).
func getCurrentPackageFullName() (string, bool) {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	proc := k32.NewProc("GetCurrentPackageFullName")
	if err := proc.Find(); err != nil {
		return "", false
	}

	var length uint32 = 0
	r1, _, _ := proc.Call(uintptr(unsafe.Pointer(&length)), 0)
	if r1 == appModelErrorNoPackage || (r1 != errorInsufficientBuffer && r1 != 0) || length == 0 {
		return "", false
	}

	buf := make([]uint16, length)
	r2, _, _ := proc.Call(uintptr(unsafe.Pointer(&length)), uintptr(unsafe.Pointer(&buf[0])))
	if r2 != 0 {
		return "", false
	}
	return strings.TrimSpace(syscall.UTF16ToString(buf)), true
}

type winRTStoreLicensePayload struct {
	IsActive        bool   `json:"isActive"`
	IsTrial         bool   `json:"isTrial"`
	HasAddOnLicense bool   `json:"hasAddOnLicense"`
	Origin          string `json:"origin"`
}

// queryWindowsStoreLicense queries WinRT Windows.Services.Store.StoreContext.GetAppLicenseAsync()
// and Appx SignatureKind.
// For the Free Store edition + $9.99 Permanent Unlock durable add-on:
//   - Free edition without the add-on returns purchased=false so the nag screen stays active.
//   - Purchasing the $9.99 durable add-on (matching StoreAddOnStoreID or DefaultStoreAddOnOfferToken)
//     sets HasAddOnLicense=true -> purchased=true (100% permanent unlock, no nag).
func queryWindowsStoreLicense(pkgFullName string, addonID string) (purchased bool, isTrial bool, note string) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	script := `
$ErrorActionPreference = 'Stop'
$pkg = Get-AppxPackage -Name '*NewScid*' | Where-Object { $_.PackageFullName -eq $env:NEWSCID_PKG_FULLNAME } | Select-Object -First 1
$origin = if ($pkg) { [string]$pkg.SignatureKind } else { 'Unknown' }
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Services.Store.StoreContext, Windows.Services.Store, ContentType=WindowsRuntime]
$ctx = [Windows.Services.Store.StoreContext]::GetDefault()
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
  $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`1" + `'
})[0]
$op = $ctx.GetAppLicenseAsync()
$task = $asTaskGeneric.MakeGenericMethod([Windows.Services.Store.StoreAppLicense]).Invoke($null, @($op))
$task.Wait(3000) | Out-Null
$lic = $task.Result
$hasAddon = $false
$targetId = $env:NEWSCID_ADDON_ID
$targetToken = $env:NEWSCID_ADDON_TOKEN
if ($lic -and $lic.AddOnLicenses) {
  foreach ($kv in $lic.AddOnLicenses) {
    $al = $kv.Value
    if ($al.IsActive) {
      if (
        ($targetId -and $kv.Key -like "*$targetId*") -or
        ($al.InAppOfferToken -eq $targetToken) -or
        ($al.SkuStoreId -and $targetId -and $al.SkuStoreId -like "*$targetId*")
      ) {
        $hasAddon = $true
        break
      }
    }
  }
}
@{ isActive = [bool]$lic.IsActive; isTrial = [bool]$lic.IsTrial; hasAddOnLicense = $hasAddon; origin = $origin } | ConvertTo-Json -Compress
`
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.Env = append(
		cmd.Environ(),
		"NEWSCID_PKG_FULLNAME="+pkgFullName,
		"NEWSCID_ADDON_ID="+addonID,
		"NEWSCID_ADDON_TOKEN="+DefaultStoreAddOnOfferToken,
	)
	out, err := cmd.Output()
	if err != nil {
		return false, false, "store license verification unavailable"
	}

	var payload winRTStoreLicensePayload
	if err := json.Unmarshal(out, &payload); err != nil {
		return false, false, "store license parse failed"
	}

	isStoreOrigin := strings.EqualFold(strings.TrimSpace(payload.Origin), "Store")
	if !isStoreOrigin {
		return false, false, "unverified MSIX origin: " + payload.Origin
	}
	if payload.HasAddOnLicense {
		return true, false, "verified Microsoft Store $9.99 permanent license"
	}
	if payload.IsTrial {
		return false, true, "Microsoft Store trial"
	}
	return false, false, "Microsoft Store Free Edition (upgrade available for $9.99)"
}

func checkPlatformStoreLicense() StoreLicenseInfo {
	storeStateMu.Lock()
	defer storeStateMu.Unlock()
	if storeStateLoaded {
		return cachedStoreState
	}

	pkgFullName, packaged := getCurrentPackageFullName()
	if !packaged {
		cachedStoreState = StoreLicenseInfo{
			IsWindows:        true,
			IsPackaged:       false,
			IsStorePurchased: false,
			CanPurchaseInApp: false,
			VerificationNote: "compiled unpackaged Windows binary",
		}
		storeStateLoaded = true
		return cachedStoreState
	}

	addonID := resolveStoreAddOnID()
	purchased, isTrial, note := queryWindowsStoreLicense(pkgFullName, addonID)
	cachedStoreState = StoreLicenseInfo{
		IsWindows:        true,
		IsPackaged:       true,
		IsStorePurchased: purchased,
		IsTrial:          isTrial,
		CanPurchaseInApp: !purchased && addonID != "",
		AddOnStoreID:     addonID,
		PackageFullName:  pkgFullName,
		VerificationNote: note,
	}
	storeStateLoaded = true
	return cachedStoreState
}

func purchasePlatformStoreUnlock(ctx context.Context) StorePurchaseResult {
	pkgFullName, packaged := getCurrentPackageFullName()
	if !packaged {
		return StorePurchaseResult{
			Purchased: false,
			Status:    "Unavailable",
			Message:   "Compiled Windows builds use voluntary Stripe support; Microsoft Store checkout requires the Store package.",
		}
	}
	addonID := resolveStoreAddOnID()
	if addonID == "" {
		return StorePurchaseResult{
			Purchased: false,
			Status:    "Unavailable",
			Message:   "Microsoft Store add-on ID is not configured in this build.",
		}
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
	}

	script := `
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Services.Store.StoreContext, Windows.Services.Store, ContentType=WindowsRuntime]
$ctx = [Windows.Services.Store.StoreContext]::GetDefault()
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
  $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`1" + `'
})[0]
$op = $ctx.RequestPurchaseAsync($env:NEWSCID_ADDON_ID)
$task = $asTaskGeneric.MakeGenericMethod([Windows.Services.Store.StorePurchaseResult]).Invoke($null, @($op))
$task.Wait(55000) | Out-Null
$res = $task.Result
@{ status = [string]$res.Status } | ConvertTo-Json -Compress
`
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.Env = append(cmd.Environ(), "NEWSCID_PKG_FULLNAME="+pkgFullName, "NEWSCID_ADDON_ID="+addonID)
	out, err := cmd.Output()
	if err != nil {
		return StorePurchaseResult{
			Purchased: false,
			Status:    "ServerError",
			Message:   "Microsoft Store purchase flow could not be launched: " + err.Error(),
		}
	}

	var parsed struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(out, &parsed)
	status := strings.TrimSpace(parsed.Status)
	if status == "Succeeded" || status == "AlreadyPurchased" {
		// Invalidate cache so next Check() picks up the newly granted permanent license.
		storeStateMu.Lock()
		storeStateLoaded = false
		storeStateMu.Unlock()
		return StorePurchaseResult{
			Purchased: true,
			Status:    status,
			Message:   "Microsoft Store $9.99 Permanent Unlock verified! Thank you for supporting NewScid.",
		}
	}
	if status == "" {
		status = "NotPurchased"
	}
	return StorePurchaseResult{
		Purchased: false,
		Status:    status,
		Message:   "Purchase was not completed (" + status + ").",
	}
}
