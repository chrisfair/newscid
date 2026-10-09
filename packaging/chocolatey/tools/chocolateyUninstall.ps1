$ErrorActionPreference = 'Stop'

$packageArgs = @{
  packageName  = 'newscid'
  softwareName = 'NewScid*'
  fileType     = 'exe'
  silentArgs   = '/S'
  validExitCodes = @(0)
}

$uninstalled = $false
[array]$key = Get-UninstallRegistryKey -SoftwareName $packageArgs['softwareName']

if ($key.Count -eq 1) {
  $key | ForEach-Object {
    $packageArgs['file'] = "$($_.UninstallString)"
    if ($packageArgs['file'] -match '^"([^"]+)"') {
      $packageArgs['file'] = $matches[1]
    }
    Uninstall-ChocolateyPackage @packageArgs
    $uninstalled = $true
  }
}

if (-not $uninstalled) {
  # Clean up portable install directory if present
  $installDir = Join-Path $(Get-ToolsLocation) 'newscid'
  if (Test-Path $installDir) {
    Remove-Item -Recurse -Force $installDir
  }
  Clear-BinFile -Name 'newscid'
}
