<#
.SYNOPSIS
  Build and package NewScid as a Microsoft Store MSIX bundle (Free Edition + $9.99 Permanent Unlock add-on).
.PARAMETER AddOnStoreId
  Partner Center Store ID for the $9.99 durable add-on (offer token: newscid_pro_permanent).
.PARAMETER Version
  Four-part MSIX version string (e.g. 1.3.45.0).
.PARAMETER Publisher
  Exact Partner Center Publisher CN string (e.g. CN=XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX).
.PARAMETER BundledStockfishPath
  Optional path to stockfish.exe (GPLv3) to bundle inside the MSIX package.
#>
param(
  [string]$AddOnStoreId = "",
  [string]$Version = "1.3.46.0",
  [string]$Publisher = "CN=REPLACE-WITH-PARTNER-CENTER-PUBLISHER-ID",
  [string]$BundledStockfishPath = ""
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$StageDir = Join-Path $RepoRoot "build\msix-stage"
$OutDir   = Join-Path $RepoRoot "build\dist"
$MsixOut  = Join-Path $OutDir "NewScid-$Version-x64.msix"

Write-Host "==> Building Vue frontend..."
Push-Location (Join-Path $RepoRoot "frontend")
npm ci
npm run build
Pop-Location

Write-Host "==> Building Windows x64 NewScid binary (Wails)..."
$LdFlags = "-H windowsgui -s -w"
if ($AddOnStoreId) {
  $LdFlags += " -X newscid/pkg/support.StoreAddOnStoreID=$AddOnStoreId"
}
Push-Location $RepoRoot
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -tags "desktop,production" -ldflags $LdFlags -o (Join-Path $RepoRoot "build\bin\newscid.exe") .
Pop-Location

Write-Host "==> Staging MSIX layout..."
if (Test-Path $StageDir) { Remove-Item $StageDir -Recurse -Force }
New-Item -ItemType Directory -Path $StageDir | Out-Null
New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

Copy-Item (Join-Path $RepoRoot "build\bin\newscid.exe") (Join-Path $StageDir "newscid.exe")
Copy-Item (Join-Path $PSScriptRoot "Assets") (Join-Path $StageDir "Assets") -Recurse

if ($BundledStockfishPath -and (Test-Path $BundledStockfishPath)) {
  Write-Host "==> Bundling Stockfish engine ($BundledStockfishPath)..."
  New-Item -ItemType Directory -Path (Join-Path $StageDir "engines") -Force | Out-Null
  Copy-Item $BundledStockfishPath (Join-Path $StageDir "engines\stockfish.exe")
}

$Manifest = Get-Content (Join-Path $PSScriptRoot "AppxManifest.xml") -Raw
$Manifest = $Manifest -replace 'Version="[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"', "Version=`"$Version`""
$Manifest = $Manifest -replace 'Publisher="[^"]+"', "Publisher=`"$Publisher`""
Set-Content (Join-Path $StageDir "AppxManifest.xml") $Manifest -Encoding UTF8

Write-Host "==> Packing MSIX with MakeAppx.exe..."
& makeappx.exe pack /d $StageDir /p $MsixOut /o
Write-Host "DONE: $MsixOut"
Write-Host "Note: Upload unsigned MSIX directly to Microsoft Partner Center (Microsoft signs Store submissions automatically)."
