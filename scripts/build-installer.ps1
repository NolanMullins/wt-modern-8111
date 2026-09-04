param(
  [string]$Version = '0.0.0',
  [switch]$SkipPortableBuild
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$output = Join-Path $root 'dist'
$portable = Join-Path $output 'wt-modern-windows-amd64.exe'
if (!$SkipPortableBuild) {
  & (Join-Path $PSScriptRoot 'build-windows.ps1') -Version $Version
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
  Copy-Item -Force (Join-Path $output 'wt-modern.exe') $portable
} elseif (!(Test-Path $portable)) {
  throw "Portable executable not found: $portable"
}

$compilerPaths = @(
  (Join-Path ${env:ProgramFiles(x86)} 'Inno Setup 6\ISCC.exe'),
  (Join-Path $env:LOCALAPPDATA 'Programs\Inno Setup 6\ISCC.exe')
)
$compiler = $compilerPaths | Where-Object { Test-Path $_ } | Select-Object -First 1
if (!$compiler) {
  throw 'Inno Setup 6 is required to build the installer.'
}

& $compiler "/DAppVersion=$Version" (Join-Path $root 'installer\wt-modern.iss')
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& (Join-Path $PSScriptRoot 'write-checksums.ps1')
