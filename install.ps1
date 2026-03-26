# Ragnarok Installer v1.1.0
# AI Governance & Memory Layer Ecosystem
# Usage: 
#   irm https://raw.githubusercontent.com/andragon31/Ragnarok/v1.1.0/install.ps1 | iex
#   Or download and run manually

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Ragnarok",
    [switch]$AddToPath,
    [switch]$Unattended
)

$VERSION = "1.1.0"
$REPO_URL = "https://github.com/andragon31/Ragnarok"

# Save script to temp if running from remote (irm | iex)
if ($MyInvocation.InvocationName -eq "iex") {
    $scriptPath = Join-Path $env:TEMP "ragnarok_install_$([guid]::NewGuid().ToString('N').Substring(0,8)).ps1"
    $content = Get-Content $PSCommandPath -Raw
    $content | Set-Content $scriptPath -Encoding UTF8
    Write-Host "Script saved to: $scriptPath" -ForegroundColor Yellow
    Write-Host "Running locally...`n" -ForegroundColor Yellow
    & $scriptPath -InstallDir $InstallDir -AddToPath:$AddToPath -Unattended:$Unattended
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
$IS_LINUX = $env:OSTYPE -eq "linux-gnu"
$IS_MACOS = $env:OSTYPE -match "darwin"

if (!$IS_WINDOWS -and !$IS_LINUX -and !$IS_MACOS) {
    Write-Err "Unsupported operating system"
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
$DATA_DIR = Join-Path $InstallDir "data"
$FENRIR_DIR = Join-Path $DATA_DIR ".fenrir"
$HATI_DIR = Join-Path $DATA_DIR ".hati"
$SKOLL_DIR = Join-Path $DATA_DIR ".skoll"
$TYR_DIR = Join-Path $DATA_DIR ".tyr"

New-Directory $BIN_DIR
New-Directory $DATA_DIR
New-Directory $FENRIR_DIR
New-Directory $HATI_DIR
New-Directory $SKOLL_DIR
New-Directory $TYR_DIR

Write-Success "Data directories created"

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
        Write-Host "Run these commands manually to see the error:" -ForegroundColor Yellow
        Write-Host "  git clone --depth 1 --branch v$VERSION $REPO_URL" -ForegroundColor Gray
        throw "Git clone failed"
    }
}

Write-Success "Repository cloned"

Write-Step "4. Building binaries"

$PLUGINS = @(
    @{Name="fenrir"; Dir="fenrir"; Package="./cmd/fenrir"},
    @{Name="hati"; Dir="hati"; Package="./cmd/hati"},
    @{Name="skoll"; Dir="skoll"; Package="./cmd/skoll"},
    @{Name="tyr"; Dir="tyr"; Package="./cmd/tyr"},
    @{Name="rag"; Dir="installer"; Package="./cmd/rag"}
)

$buildSuccess = $true

foreach ($plugin in $PLUGINS) {
    $outFile = Join-Path $BIN_DIR "$($plugin.Name).exe"
    Write-Host "  Building $($plugin.Name)..." -NoNewline
    
    Push-Location $TEMP_DIR
    $buildArgs = @("build", "-C", $plugin.Dir, "-ldflags=-s -w", "-o", $outFile, $plugin.Package)
    $buildOutput = & go @buildArgs 2>&1
    Pop-Location
    
    if ($LASTEXITCODE -eq 0 -and (Test-Path $outFile)) {
        $size = [math]::Round((Get-Item $outFile).Length / 1MB, 1)
        Write-Success "$($plugin.Name) ($size MB)"
    } else {
        Write-Err "$($plugin.Name) failed"
        if ($buildOutput) { Write-Host $buildOutput -ForegroundColor Gray }
        $buildSuccess = $false
    }
}

Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue

if (-not $buildSuccess) {
    Write-Err "Some binaries failed to build"
    throw "Build failed"
}

Write-Step "5. Creating MCP configuration"

$OPENCODE_CONFIG_DIRS = @(
    "$env:APPDATA\opencode",
    "$env:LOCALAPPDATA\opencode",
    "$env:USERPROFILE\.opencode"
)

$opencodeConfigDir = $null
foreach ($dir in $OPENCODE_CONFIG_DIRS) {
    if (Test-Path $dir) {
        $opencodeConfigDir = $dir
        break
    }
}

if (-not $opencodeConfigDir) {
    $opencodeConfigDir = "$env:USERPROFILE\.opencode"
    New-Directory $opencodeConfigDir
    Write-Warn "Created OpenCode config directory"
} else {
    Write-Success "OpenCode config: $opencodeConfigDir"
}

$PLUGIN_PORTS = @{
    "fenrir" = 7437
    "hati" = 7439
    "skoll" = 7438
    "tyr" = 7440
}

$mcpServers = @{}
foreach ($entry in $PLUGIN_PORTS.GetEnumerator()) {
    $mcpServers[$entry.Key] = @{
        command = Join-Path $BIN_DIR "$($entry.Key).exe"
        args = @("serve", "--port", $entry.Value.ToString())
        env = @{
            "MCP_TRANSPORT" = "tcp"
            "RAGNAROK_DATA" = $DATA_DIR
        }
    }
}

$mcpConfig = @{ mcpServers = $mcpServers }
$mcpJsonPath = Join-Path $opencodeConfigDir ".mcp.json"
$mcpJsonContent = $mcpConfig | ConvertTo-Json -Depth 10
$mcpJsonContent | Set-Content $mcpJsonPath -Encoding UTF8

Write-Success "MCP config: $mcpJsonPath"

Write-Step "6. Verifying installation"

foreach ($plugin in $PLUGINS) {
    $exePath = Join-Path $BIN_DIR "$($plugin.Name).exe"
    if (Test-Path $exePath) {
        $version = & $exePath version 2>$null
        if ($LASTEXITCODE -eq 0 -and $version) {
            Write-Success "$($plugin.Name): $version"
        } else {
            Write-Warn "$($plugin.Name): installed but failed to respond"
        }
    } else {
        Write-Err "$($plugin.Name): not found"
    }
}

if ($AddToPath) {
    Write-Step "7. Adding to PATH"
    $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    $newPath = "$userPath;$BIN_DIR"
    [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
    $env:PATH = $newPath
    Write-Success "Added to PATH: $BIN_DIR"
}

Write-Host "`n---------------------------------------------------------------" -ForegroundColor Cyan
Write-Host "  INSTALLATION COMPLETE!" -ForegroundColor Green
Write-Host "---------------------------------------------------------------`n" -ForegroundColor Cyan

Write-Host "Next steps:`n" -ForegroundColor White
Write-Host "  1. Start the ecosystem:" -ForegroundColor White
Write-Host "     $BIN_DIR\rag.exe serve" -ForegroundColor Yellow
Write-Host ""
Write-Host "  2. Check ecosystem health:" -ForegroundColor White
Write-Host "     $BIN_DIR\rag.exe stats --ecosystem" -ForegroundColor Yellow
Write-Host ""

if (-not $AddToPath) {
    Write-Host "  To add to PATH permanently, run:" -ForegroundColor White
    Write-Host "     [Environment]::SetEnvironmentVariable('PATH', [Environment]::GetEnvironmentVariable('PATH', 'User') + ';$BIN_DIR', 'User')" -ForegroundColor Yellow
}

Write-Host "`nDocumentation: https://github.com/andragon31/Ragnarok" -ForegroundColor Gray
Write-Host ""
