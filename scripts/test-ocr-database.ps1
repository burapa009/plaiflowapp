$ErrorActionPreference = 'Stop'
if (-not $env:DATABASE_PUBLIC_URL) { throw 'DATABASE_PUBLIC_URL is unavailable' }
$env:OCR_TEST_ADMIN_URL = $env:DATABASE_PUBLIC_URL
$env:GOTOOLCHAIN = 'local'
$env:GOCACHE = Join-Path $PSScriptRoot '../.tools/gocache'
$env:GOMODCACHE = Join-Path $PSScriptRoot '../.tools/gomodcache'
Push-Location (Join-Path $PSScriptRoot '../api')
try {
    & ../.tools/go127/go/bin/go.exe test ./internal/postgres -run TestOCRIsolatedDatabaseLifecycle -count=1 -v
    exit $LASTEXITCODE
} finally { Pop-Location; Remove-Item Env:OCR_TEST_ADMIN_URL -ErrorAction SilentlyContinue }
