param(
  [string]$Version = 'dev'
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$output = Join-Path $root 'dist'
$revision = (& git -C $root rev-parse --short=12 HEAD 2>$null)
if ($LASTEXITCODE -ne 0 -or !$revision) { $revision = 'unknown' }

npm --prefix (Join-Path $root 'web') ci
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
npm --prefix (Join-Path $root 'web') run build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
New-Item -ItemType Directory -Force $output | Out-Null
$linkerFlags = "-s -w -H windowsgui -X github.com/NolanMullins/wt-modern-8111/internal/buildinfo.Version=$Version -X github.com/NolanMullins/wt-modern-8111/internal/buildinfo.Revision=$revision"
go -C $root build -trimpath -ldflags $linkerFlags `
  -o (Join-Path $output 'wt-modern.exe') `
  ./cmd/wt-modern
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
