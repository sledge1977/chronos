#Requires -RunAsAdministrator

[CmdletBinding()]
param(
    [string]$TargetDirectory = "C:\Program Files\Chronos"
)

$ErrorActionPreference = "Stop"
$projectDirectory = Split-Path -Parent $PSScriptRoot
$serviceName = "ChronosTimeService"
$firewallRuleName = "Chronos NTP (UDP 123)"
$wrapper = Join-Path $TargetDirectory "chronos-service.exe"
$sourceWrapper = Join-Path $PSScriptRoot "chronos-service.exe"
$wrapperDownloadUrl = "https://github.com/winsw/winsw/releases/download/v2.12.0/WinSW-x64.exe"
$wrapperSHA256 = "05B82D46AD331CC16BDC00DE5C6332C1EF818DF8CEEFCD49C726553209B3A0DA"

if (-not (Test-Path -LiteralPath (Join-Path $projectDirectory "chronos.exe"))) {
    throw "chronos.exe is missing. Build the Go program first."
}

if (-not (Test-Path -LiteralPath $sourceWrapper)) {
    Write-Host "Downloading WinSW v2.12.0 from GitHub ..."
    Invoke-WebRequest -Uri $wrapperDownloadUrl -OutFile $sourceWrapper
}

$downloadedHash = (Get-FileHash -LiteralPath $sourceWrapper -Algorithm SHA256).Hash
if ($downloadedHash -ne $wrapperSHA256) {
    throw "The WinSW wrapper SHA-256 checksum does not match. Expected: $wrapperSHA256, received: $downloadedHash"
}

$existingService = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($existingService) {
    if ($existingService.Status -ne "Stopped") {
        & $wrapper stop
    }
    & $wrapper uninstall
}

New-Item -ItemType Directory -Path $TargetDirectory -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $TargetDirectory "logs") -Force | Out-Null

Copy-Item -LiteralPath (Join-Path $projectDirectory "chronos.exe") -Destination $TargetDirectory -Force
Copy-Item -LiteralPath $sourceWrapper -Destination $TargetDirectory -Force
Copy-Item -LiteralPath (Join-Path $PSScriptRoot "chronos-service.xml") -Destination $TargetDirectory -Force

& $wrapper install
if ($LASTEXITCODE -ne 0) {
    throw "Service installation failed (exit code $LASTEXITCODE)."
}

& $wrapper start
if ($LASTEXITCODE -ne 0) {
    throw "Service startup failed (exit code $LASTEXITCODE)."
}

if (-not (Get-NetFirewallRule -DisplayName $firewallRuleName -ErrorAction SilentlyContinue)) {
    New-NetFirewallRule `
        -DisplayName $firewallRuleName `
        -Direction Inbound `
        -Action Allow `
        -Protocol UDP `
        -LocalPort 123 `
        -Profile Any | Out-Null
}

Get-Service -Name $serviceName
