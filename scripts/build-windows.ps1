$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$output = Join-Path $root 'dist'

npm --prefix (Join-Path $root 'web') ci
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
npm --prefix (Join-Path $root 'web') run build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
New-Item -ItemType Directory -Force $output | Out-Null
go -C $root build -trimpath -ldflags '-s -w -H windowsgui' `
  -o (Join-Path $output 'wt-modern.exe') `
  ./cmd/wt-modern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
