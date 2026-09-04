$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$output = Join-Path $root 'dist'
$assets = @(
  (Join-Path $output 'wt-modern-windows-amd64.exe'),
  (Join-Path $output 'wt-modern-setup.exe')
)

foreach ($asset in $assets) {
  if (!(Test-Path $asset)) {
    throw "Release asset not found: $asset"
  }
}

$lines = foreach ($asset in $assets) {
  $hash = (Get-FileHash $asset -Algorithm SHA256).Hash.ToLowerInvariant()
  "$hash  $(Split-Path $asset -Leaf)"
}
Set-Content -Path (Join-Path $output 'checksums.txt') -Value $lines -Encoding ascii
