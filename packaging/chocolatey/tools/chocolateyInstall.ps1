$ErrorActionPreference = 'Stop'
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$url64      = 'https://github.com/christopherpatrickfair/newscid/releases/download/v1.0.0/newscid-windows-amd64-installer.exe'
$checksum64 = ''

# Check for local installer first (offline installation or local packaging)
$localInstaller = Join-Path $toolsDir 'newscid-installer.exe'
$localZip       = Join-Path $toolsDir 'newscid-windows-amd64.zip'

$packageArgs = @{
  packageName    = 'newscid'
  fileType       = 'exe'
  silentArgs     = '/S'
  validExitCodes = @(0)
  softwareName   = 'NewScid*'
}

if (Test-Path $localInstaller) {
  $packageArgs['file'] = $localInstaller
  Install-ChocolateyInstallPackage @packageArgs
} elseif (Test-Path $localZip) {
  $installDir = Join-Path $(Get-ToolsLocation) 'newscid'
  Install-ChocolateyZipPackage -PackageName 'newscid' -ZipFileName $localZip -UnzipLocation $installDir
  Install-BinFile -Name 'newscid' -Path (Join-Path $installDir 'newscid.exe')
} else {
  $packageArgs['url64bit'] = $url64
  if ($checksum64 -ne '') {
    $packageArgs['checksum64'] = $checksum64
    $packageArgs['checksumType64'] = 'sha256'
  }
  Install-ChocolateyPackage @packageArgs
}
