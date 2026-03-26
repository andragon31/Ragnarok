# Ragnarok Quick Installer
# One-line install: iwr https://tinyurl.com/ragnarok-install | iex
# Or:              iwr https://bit.ly/ragnarok-install | iex

$script = "$env:TEMP\ragnarok_install.ps1"
$url = "https://raw.githubusercontent.com/ragnarok-ecosystem/ragnarok/main/install.ps1"

try {
    Write-Host "Downloading Ragnarok installer..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $url -OutFile $script -UseBasicParsing
    & $script @args
} catch {
    Write-Host "Download failed: $_" -ForegroundColor Red
    Write-Host "Alternative: Clone and run manually:" -ForegroundColor Yellow
    Write-Host "  git clone https://github.com/ragnarok-ecosystem/ragnarok" -ForegroundColor Yellow
    Write-Host "  cd ragnarok" -ForegroundColor Yellow
    Write-Host "  .\install.ps1" -ForegroundColor Yellow
}
