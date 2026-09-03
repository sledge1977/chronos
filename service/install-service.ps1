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
    throw "chronos.exe fehlt. Zuerst das Go-Programm kompilieren."
}

if (-not (Test-Path -LiteralPath $sourceWrapper)) {
    Write-Host "WinSW v2.12.0 wird von GitHub geladen ..."
    Invoke-WebRequest -Uri $wrapperDownloadUrl -OutFile $sourceWrapper
}

$downloadedHash = (Get-FileHash -LiteralPath $sourceWrapper -Algorithm SHA256).Hash
if ($downloadedHash -ne $wrapperSHA256) {
    throw "Die SHA-256-Prüfsumme des WinSW-Wrappers stimmt nicht. Erwartet: $wrapperSHA256, erhalten: $downloadedHash"
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
    throw "Installation des Dienstes ist fehlgeschlagen (Exitcode $LASTEXITCODE)."
}

& $wrapper start
if ($LASTEXITCODE -ne 0) {
    throw "Start des Dienstes ist fehlgeschlagen (Exitcode $LASTEXITCODE)."
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
