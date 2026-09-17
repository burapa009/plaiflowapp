param(
    [Parameter(Mandatory)] [string] $ApiUrl,
    [Parameter(Mandatory)] [string] $WebUrl,
    [Parameter(Mandatory)] [string] $LineChannelSecret,
    [Parameter(Mandatory)] [string] $DashboardToken,
    [string] $VercelDeployment
)

$ErrorActionPreference = 'Stop'
function Get-LineSignature([byte[]] $Value) {
    $hmac = [Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($LineChannelSecret))
    try { return [Convert]::ToBase64String($hmac.ComputeHash($Value)) } finally { $hmac.Dispose() }
}

function Invoke-WebSmokeRequest([string] $Path) {
    if (-not $VercelDeployment) { return Invoke-WebRequest "$WebUrl$Path" -UseBasicParsing }
    $tempPath = Join-Path ([IO.Path]::GetTempPath()) ("plaiflow-smoke-" + [guid]::NewGuid().ToString('N') + '.html')
    try {
        & npx --yes vercel curl $Path --deployment $VercelDeployment -- --silent --output $tempPath
        if ($LASTEXITCODE -ne 0) { throw "Authenticated Web request failed: $Path" }
        return [pscustomobject]@{ StatusCode = 200; Content = Get-Content -LiteralPath $tempPath -Raw }
    } finally {
        Remove-Item -LiteralPath $tempPath -Force -ErrorAction SilentlyContinue
    }
}

$body = '{"events":[{"webhookEventId":"synthetic-' + [guid]::NewGuid().ToString('N') + '","type":"message","timestamp":' + [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() + '}]}';
$bytes = [Text.Encoding]::UTF8.GetBytes($body)
$signature = Get-LineSignature $bytes

if ((Invoke-WebRequest "$ApiUrl/healthz" -UseBasicParsing).StatusCode -ne 200) { throw 'API liveness failed' }
if ((Invoke-WebRequest "$ApiUrl/readyz" -UseBasicParsing).StatusCode -ne 200) { throw 'API readiness failed' }
$catalogTimer = [Diagnostics.Stopwatch]::StartNew()
$catalog = Invoke-RestMethod "$ApiUrl/v1/plans"
$catalogTimer.Stop()
$starter = $catalog.plans | Where-Object key -eq 'Starter'
$business = $catalog.plans | Where-Object key -eq 'Business'
if ($catalog.billing_enabled -ne $false -or $starter.prices.six_months.total_satang -ne 109848 -or $business.entitlements.'business_contacts.export.drive' -ne $true) { throw 'Authoritative Plan catalog failed' }
if ($catalogTimer.ElapsedMilliseconds -gt 5000) { throw 'Plan catalog response exceeded 5 seconds' }
$requestID = 'smoke-' + [guid]::NewGuid().ToString('N')
$baselineResponse = Invoke-WebRequest "$ApiUrl/v1/dashboard" -UseBasicParsing -Headers @{ Authorization = "Bearer $DashboardToken"; 'X-Request-ID' = $requestID }
if ($baselineResponse.Headers['X-Request-ID'] -ne $requestID) { throw 'Request ID was not propagated' }
$baseline = $baselineResponse.Content | ConvertFrom-Json
$emptyBytes = [Text.Encoding]::UTF8.GetBytes('{"events":[]}')
if ((Invoke-WebRequest "$ApiUrl/webhooks/line" -UseBasicParsing -Method Post -Headers @{ 'x-line-signature' = (Get-LineSignature $emptyBytes) } -Body $emptyBytes -ContentType 'application/json').StatusCode -ne 200) { throw 'LINE verification delivery failed' }
try {
    Invoke-WebRequest "$ApiUrl/webhooks/line" -UseBasicParsing -Method Post -Headers @{ 'x-line-signature' = 'forged' } -Body $bytes -ContentType 'application/json'
    throw 'Forged signature was accepted'
} catch {
    if ($_.Exception.Response.StatusCode.value__ -ne 401) { throw }
}
$timer = [Diagnostics.Stopwatch]::StartNew()
if ((Invoke-WebRequest "$ApiUrl/webhooks/line" -UseBasicParsing -Method Post -Headers @{ 'x-line-signature' = $signature } -Body $bytes -ContentType 'application/json').StatusCode -ne 200) { throw 'Signed webhook failed' }
if ((Invoke-WebRequest "$ApiUrl/webhooks/line" -UseBasicParsing -Method Post -Headers @{ 'x-line-signature' = $signature } -Body $bytes -ContentType 'application/json').StatusCode -ne 200) { throw 'Duplicate webhook acknowledgement failed' }
$timer.Stop()
for ($attempt = 0; $attempt -lt 15; $attempt++) {
    Start-Sleep -Seconds 1
    $snapshot = Invoke-RestMethod "$ApiUrl/v1/dashboard" -Headers @{ Authorization = "Bearer $DashboardToken" }
    if ($snapshot.counts.processed -eq ($baseline.counts.processed + 1)) { break }
}
if ($snapshot.counts.processed -ne ($baseline.counts.processed + 1) -or $snapshot.worker -ne 'ok') { throw 'Idempotency, worker, or dashboard state failed' }
$web = Invoke-WebSmokeRequest '/'
if ($web.StatusCode -ne 200 -or $web.Content -notmatch 'PlaiFlow') { throw 'Web shell failed' }
$pricingTimer = [Diagnostics.Stopwatch]::StartNew()
$pricing = Invoke-WebSmokeRequest '/pricing?interval=six_months'
$pricingTimer.Stop()
if ($pricing.StatusCode -ne 200 -or $pricing.Content -notmatch 'Starter' -or $pricing.Content -notmatch '1,098.48') { throw 'Pricing presentation failed' }
if ($pricingTimer.ElapsedMilliseconds -gt 10000) { throw 'Pricing response exceeded 10 seconds' }
[pscustomobject]@{ webhook_ms = $timer.ElapsedMilliseconds; catalog_ms = $catalogTimer.ElapsedMilliseconds; pricing_ms = $pricingTimer.ElapsedMilliseconds; api = $snapshot.api; database = $snapshot.database; worker = $snapshot.worker }
