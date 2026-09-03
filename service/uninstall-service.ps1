#Requires -RunAsAdministrator

[CmdletBinding()]
param(
    [string]$TargetDirectory = "C:\Program Files\Chronos"
)

$ErrorActionPreference = "Stop"
$serviceName = "ChronosTimeService"
$firewallRuleName = "Chronos NTP (UDP 123)"
$wrapper = Join-Path $TargetDirectory "chronos-service.exe"
$service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue

if (-not $service) {
    Write-Host "The $serviceName service is not installed."
    return
}

if (-not (Test-Path -LiteralPath $wrapper)) {
    throw "Service wrapper is missing: $wrapper"
}

if ($service.Status -ne "Stopped") {
    & $wrapper stop
}

& $wrapper uninstall
if ($LASTEXITCODE -ne 0) {
    throw "Service removal failed (exit code $LASTEXITCODE)."
}

Get-NetFirewallRule -DisplayName $firewallRuleName -ErrorAction SilentlyContinue |
    Remove-NetFirewallRule
