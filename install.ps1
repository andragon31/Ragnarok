# Ragnarok Installer
# AI Governance & Memory Layer Ecosystem
# Usage:
#   irm https://raw.githubusercontent.com/andragon31/Ragnarok/vX.X.X/install.ps1 | iex
#   Or: powershell -File install.ps1 -Version 2.2.4

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Ragnarok",
    [string]$Version = ""
)

$REPO = "andragon31/Ragnarok"

if ($Version -eq "") {
    try {
        $release = Invoke-RestMethod "https://api.github.com/repos/$REPO/releases/latest"
        $VERSION = $release.tag_name.TrimStart("v")
        Write-Host "Latest version: $VERSION" -ForegroundColor Cyan
    } catch {
        Write-Warn "No se pudo detectar la ultima version. Usando fallback."
        $VERSION = "2.2.4"
    }
} else {
    $VERSION = $Version
}

$ARCH = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$ASSET = "ragnarok_${VERSION}_windows_${ARCH}.zip"
$DOWNLOAD_URL = "https://github.com/$REPO/releases/download/v$VERSION/$ASSET"
$CHECKSUM_URL = "https://github.com/$REPO/releases/download/v$VERSION/checksums.txt"

function Write-Step($message) {
    Write-Host "`n[STEP] $message" -ForegroundColor Cyan
}

function Write-Success($message) {
    Write-Host "[OK] $message" -ForegroundColor Green
}

function Write-Warn($message) {
    Write-Host "[WARN] $message" -ForegroundColor Yellow
}

function Write-Err($message) {
    Write-Host "[ERROR] $message" -ForegroundColor Red
}

Write-Host @"

  +++  +++++  +++++  +++++  +     +++++  +++++  +++++
  +  + +    + +     +     + +       +     +       +
  +++  +++++  +++++  +++++  + ++    +     +++++   ++++
  +  + +    + +     +     + +  +    +     +         +
  +++  +++++  +++++  +++++  +++++  +++++  +++++  +++++

     v$VERSION - AI Governance & Memory Layer Ecosystem
     https://github.com/$REPO

"@ -ForegroundColor Cyan

Write-Host "`nInstalling Ragnarok v$VERSION..." -ForegroundColor White

$IS_WINDOWS = $env:OS -eq "Windows_NT"

if (!$IS_WINDOWS) {
    Write-Err "This installer is for Windows only"
    throw "Unsupported OS"
}

Write-Step "1. Downloading binary"

$zipPath = Join-Path $env:TEMP $ASSET
Write-Host "  Downloading $ASSET..." -NoNewline
try {
    Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile $zipPath -UseBasicParsing
    Write-Success "Downloaded"
} catch {
    Write-Err "Failed to download binary"
    Write-Host "Ensure release v$VERSION exists at https://github.com/$REPO/releases" -ForegroundColor Yellow
    throw "Download failed"
}

Write-Step "2. Verifying checksum"

$webClient = New-Object System.Net.WebClient
$checksums = $webClient.DownloadString($CHECKSUM_URL)
$webClient.Dispose()
$assetEscaped = [regex]::Escape($ASSET)
$checksumLine = $checksums.Split([char]0x0A) | Where-Object { $_.Trim([char]0x0D) -match "^\s*([a-fA-F0-9]+)\s+${assetEscaped}\s*$" }
if (-not $checksumLine) {
    Remove-Item $zipPath -ErrorAction SilentlyContinue
    Write-Err "Could not find checksum for $ASSET in checksums.txt"
    throw "Checksum verification failed"
}
$expected = $matches[1].ToLower()
$actual = (Get-FileHash $zipPath -Algorithm SHA256).Hash.ToLower()

if ($actual -ne $expected) {
    Remove-Item $zipPath -ErrorAction SilentlyContinue
    Write-Err "Checksum mismatch. Expected: $expected, Got: $actual"
    throw "Checksum verification failed"
}
Write-Success "Checksum verified"

Write-Step "3. Installing"

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Expand-Archive -Path $zipPath -DestinationPath $InstallDir -Force
Remove-Item $zipPath

$outFile = Join-Path $InstallDir "rag.exe"
if (Test-Path $outFile) {
    $size = [math]::Round((Get-Item $outFile).Length / 1MB, 1)
    Write-Success "Installed to $InstallDir ($size MB)"
} else {
    Write-Err "Installation failed - binary not found after extraction"
    throw "Install failed"
}

Write-Step "4. Adding to PATH"

$userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
if ($userPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable('PATH', "$userPath;$InstallDir", 'User')
    $env:PATH = "$env:PATH;$InstallDir"
    Write-Success "Added to PATH: $InstallDir"
} else {
    Write-Success "Already in PATH"
}

Write-Step "5. Verifying installation"

$ragVersion = & $outFile version 2>$null
if ($LASTEXITCODE -eq 0 -and $ragVersion) {
    Write-Success $ragVersion
} else {
    Write-Err "Verification failed"
}

Write-Step "6. Configuring IDEs"

$ragSetupPath = Join-Path $InstallDir "rag.exe"
try {
    & $ragSetupPath setup all 2>$null
    Write-Success "IDE MCP configuration updated"
} catch {
    Write-Warn "Could not auto-configure IDEs. Run 'rag setup all' manually."
}

Write-Host "`n---------------------------------------------------------------" -ForegroundColor Cyan
Write-Host "  INSTALLATION COMPLETE!" -ForegroundColor Green
Write-Host "---------------------------------------------------------------`n" -ForegroundColor Cyan

Write-Host "Usage:" -ForegroundColor White
Write-Host "  rag new --project NAME --path ./dir   Create new project" -ForegroundColor Yellow
Write-Host "  rag continue --plan ID                Resume development" -ForegroundColor Yellow
Write-Host "  rag setup all                         Re-configure all IDEs" -ForegroundColor Yellow
Write-Host "  rag doctor                            Health check" -ForegroundColor Yellow
Write-Host "  rag --help                            Show all commands" -ForegroundColor Yellow
Write-Host ""
Write-Host "Documentation: https://github.com/$REPO" -ForegroundColor Gray
Write-Host ""
