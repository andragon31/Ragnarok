# Ragnarok Installer v1.2.0
# AI Governance & Memory Layer Ecosystem
# Usage: 
#   irm https://raw.githubusercontent.com/andragon31/Ragnarok/v1.2.0/install.ps1 | iex
#   Or download and run manually

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Ragnarok",
    [switch]$AddToPath,
    [switch]$SkipOpenCodeSetup
)

$VERSION = "1.2.0"
$REPO_URL = "https://github.com/andragon31/Ragnarok"

# Save script to temp if running from remote (irm | iex)
if ($MyInvocation.InvocationName -eq "iex") {
    $scriptPath = Join-Path $env:TEMP "ragnarok_install_$([guid]::NewGuid().ToString('N').Substring(0,8)).ps1"
    $content = Get-Content $PSCommandPath -Raw
    $content | Set-Content $scriptPath -Encoding UTF8
    Write-Host "Script saved to: $scriptPath" -ForegroundColor Yellow
    Write-Host "Running locally...`n" -ForegroundColor Yellow
    & $scriptPath -InstallDir $InstallDir -AddToPath:$AddToPath -SkipOpenCodeSetup:$SkipOpenCodeSetup
    Remove-Item $scriptPath -ErrorAction SilentlyContinue
    exit
}

$ErrorActionPreference = "Continue"

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

function Test-Command($cmd) {
    try { Get-Command $cmd -ErrorAction Stop | Out-Null; return $true } catch { return $false }
}

function New-Directory($path) {
    if (!(Test-Path $path)) {
        New-Item -ItemType Directory -Path $path -Force | Out-Null
    }
}

Write-Host @"

  +++  +++++  +++++  +++++  +     +++++  +++++  +++++
  +  + +    + +     +     + +       +     +       +
  +++  +++++  +++++  +++++  + ++    +     +++++   ++++
  +  + +    + +     +     + +  +    +     +         +
  +++  +++++  +++++  +++++  +++++  +++++  +++++  +++++

     v$VERSION - AI Governance & Memory Layer Ecosystem
     https://github.com/andragon31/Ragnarok

"@ -ForegroundColor Cyan

Write-Host "`nInstalling Ragnarok v$VERSION..." -ForegroundColor White

$IS_WINDOWS = $env:OS -eq "Windows_NT"

if (!$IS_WINDOWS) {
    Write-Err "This installer is for Windows only"
    throw "Unsupported OS"
}

Write-Step "1. Checking prerequisites"

if (Test-Command "go") {
    $goVersion = (go version) -match 'go([0-9]+\.[0-9]+)'
    if ($goVersion) {
        Write-Success "Go installed: $($Matches[1])"
    }
} else {
    Write-Err "Go not found. Please install Go 1.22+ from https://go.dev/dl/"
    throw "Go not installed"
}

if (Test-Command "git") {
    Write-Success "Git installed"
} else {
    Write-Err "Git not found. Please install Git from https://git-scm.com/"
    throw "Git not installed"
}

Write-Step "2. Creating installation directory"
New-Directory $InstallDir
Write-Success "Installation directory: $InstallDir"

$BIN_DIR = Join-Path $InstallDir "bin"
New-Directory $BIN_DIR

Write-Step "3. Cloning repository"

$TEMP_DIR = Join-Path $env:TEMP "ragnarok_build_$([guid]::NewGuid().ToString('N').Substring(0,8))"
Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue

Write-Host "  Cloning $REPO_URL (tag v$VERSION)..." -NoNewline

$gitArgs = @("clone", "--depth", "1", "--branch", "v$VERSION", $REPO_URL, $TEMP_DIR)
$gitOutput = & git @gitArgs 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Warn "Clone failed. Trying main branch..."
    $gitArgs = @("clone", "--depth", "1", $REPO_URL, $TEMP_DIR)
    $gitOutput = & git @gitArgs 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Failed to clone repository"
        throw "Git clone failed"
    }
}

Write-Success "Repository cloned"

Write-Step "4. Building rag.exe"

$outFile = Join-Path $BIN_DIR "rag.exe"
Write-Host "  Building rag.exe..." -NoNewline

Push-Location $TEMP_DIR
$buildArgs = @("build", "-ldflags=-s -w", "-o", $outFile, "./cmd/rag")
$buildOutput = & go @buildArgs 2>&1
Pop-Location

if ($LASTEXITCODE -eq 0 -and (Test-Path $outFile)) {
    $size = [math]::Round((Get-Item $outFile).Length / 1MB, 1)
    Write-Success "rag.exe built ($size MB)"
} else {
    Write-Err "Build failed"
    if ($buildOutput) { Write-Host $buildOutput -ForegroundColor Gray }
    Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue
    throw "Build failed"
}

Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue

Write-Step "5. Adding to PATH"

$ragExe = Join-Path $BIN_DIR "rag.exe"
$userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
if ($userPath -notlike "*$BIN_DIR*") {
    $newPath = "$userPath;$BIN_DIR"
    [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
    $env:PATH = $newPath
    Write-Success "Added to PATH: $BIN_DIR"
} else {
    Write-Success "Already in PATH"
}

if (-not $SkipOpenCodeSetup) {
    Write-Step "6. Setting up OpenCode MCP integration"
    
    try {
        $setupOutput = & $ragExe setup opencode 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "OpenCode MCP configured"
        } else {
            Write-Warn "OpenCode setup skipped (may not be installed)"
        }
    } catch {
        Write-Warn "OpenCode not detected, skipping MCP setup"
    }
}

Write-Step "7. Verifying installation"

$version = & $ragExe version 2>$null
if ($LASTEXITCODE -eq 0 -and $version) {
    Write-Success $version
} else {
    Write-Err "rag.exe verification failed"
}

Write-Host "`n---------------------------------------------------------------" -ForegroundColor Cyan
Write-Host "  INSTALLATION COMPLETE!" -ForegroundColor Green
Write-Host "---------------------------------------------------------------`n" -ForegroundColor Cyan

Write-Host "That's it! OpenCode will automatically use Ragnarok MCP.`n" -ForegroundColor White

Write-Host "Usage:" -ForegroundColor White
Write-Host "  rag init --project NAME    Initialize plugins for a project" -ForegroundColor Yellow
Write-Host "  rag scan --path ./project   Scan and bootstrap a project" -ForegroundColor Yellow
Write-Host "  rag setup opencode         Re-configure OpenCode MCP" -ForegroundColor Yellow
Write-Host "  rag --help                 Show all commands" -ForegroundColor Yellow
Write-Host ""

Write-Host "Documentation: https://github.com/andragon31/Ragnarok" -ForegroundColor Gray
Write-Host ""
