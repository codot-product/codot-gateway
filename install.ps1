# install.ps1 - PowerShell Installer for Codot Gateway
# Usage: irm https://raw.githubusercontent.com/codot-product/codot-gateway/main/install.ps1 | iex
$ErrorActionPreference = "Stop"

$repoOwner  = "codot-product"
$repoName   = "codot-gateway"
$binaryName = "codot-gateway.exe"

# Deploying directly to WindowsApps directory maps it globally to the user Path without requiring admin rights
$installDir = "$env:USERPROFILE\AppData\Local\Microsoft\WindowsApps"

Write-Host "🤖 Initializing Codot Gateway Windows Installer..." -ForegroundColor Cyan

$downloadUrl = "https://github.com/$repoOwner/$repoName/releases/latest/download/codot-gateway-windows-amd64.exe"
$targetPath  = Join-Path $installDir $binaryName

Write-Host "📥 Downloading compiled asset for Windows x64 architecture..." -ForegroundColor Yellow
Write-Host "🔗 Source: $downloadUrl"

try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $targetPath -UseBasicParsing
} catch {
    Write-Host "❌ Download failed! Please ensure the binary is published to public GitHub Releases." -ForegroundColor Red
    exit 1
}

Write-Output ""
Write-Host "==================================================================" -ForegroundColor Green
Write-Host "🎉 CODOT GATEWAY SUCCESSFULLY INSTALLED ON WINDOWS!" -ForegroundColor Green
Write-Host "👉 Open a fresh terminal and run 'codot-gateway' to begin." -ForegroundColor Green
Write-Host "==================================================================" -ForegroundColor Green
Write-Output ""
