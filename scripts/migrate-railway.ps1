param(
    [ValidateSet('version', 'up-one')]
    [string]$Action = 'version'
)

$ErrorActionPreference = 'Stop'
if (-not $env:DATABASE_PUBLIC_URL) { throw 'DATABASE_PUBLIC_URL is unavailable' }
$migrate = Join-Path $PSScriptRoot '..\.tools\gobin\migrate.exe'
if (-not (Test-Path -LiteralPath $migrate)) { throw 'migrate.exe is unavailable' }

Push-Location (Join-Path $PSScriptRoot '..')
try {
    switch ($Action) {
        'version' { & $migrate -path 'api/migrations' -database $env:DATABASE_PUBLIC_URL version }
        'up-one' { & $migrate -path 'api/migrations' -database $env:DATABASE_PUBLIC_URL up 1 }
    }
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Pop-Location
}
