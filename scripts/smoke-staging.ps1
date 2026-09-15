param(
    [Parameter(Mandatory)] [string] $ApiUrl,
    [Parameter(Mandatory)] [string] $WebUrl,
    [Parameter(Mandatory)] [string] $LineChannelSecret,
    [Parameter(Mandatory)] [string] $DashboardToken
)

$ErrorActionPreference = 'Stop'
$body = '{"events":[{"webhookEventId":"synthetic-' + [guid]::NewGuid().ToString('N') + '","type":"message","timestamp":' + [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() + '}]}';
$bytes = [Text.Encoding]::UTF8.GetBytes($body)
$hmac = [Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($LineChannelSecret))
$signature = [Convert]::ToBase64String($hmac.ComputeHash($bytes))

if ((Invoke-WebRequest "$ApiUrl/readyz" -UseBasicParsing).StatusCode -ne 200) { throw 'API readiness failed' }
$baseline = Invoke-RestMethod "$ApiUrl/v1/dashboard" -Headers @{ Authorization = "Bearer $DashboardToken" }
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
if ((Invoke-WebRequest $WebUrl -UseBasicParsing).StatusCode -ne 200) { throw 'Web URL failed' }
[pscustomobject]@{ webhook_ms = $timer.ElapsedMilliseconds; api = $snapshot.api; database = $snapshot.database; worker = $snapshot.worker }
