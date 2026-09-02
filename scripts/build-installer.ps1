param(
  [string]$Version = '0.0.0'
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$output = Join-Path $root 'dist'
& (Join-Path $PSScriptRoot 'build-windows.ps1') -Version $Version
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$portable = Join-Path $output 'wt-modern-windows-amd64.exe'
Copy-Item -Force (Join-Path $output 'wt-modern.exe') $portable

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

$assets = @(
  $portable,
  (Join-Path $output 'wt-modern-setup.exe')
)
$lines = foreach ($asset in $assets) {
  $hash = (Get-FileHash $asset -Algorithm SHA256).Hash.ToLowerInvariant()
  "$hash  $(Split-Path $asset -Leaf)"
}
Set-Content -Path (Join-Path $output 'checksums.txt') -Value $lines -Encoding ascii
