# Ragnarok Installer v1.1.0
# AI Governance & Memory Layer Ecosystem
# Usage: irm https://raw.githubusercontent.com/andragon31/Ragnarok/v1.1.0/install.ps1 | iex

param(
    [string]$ProjectName = "ragnarok",
    [string]$InstallDir = "$env:LOCALAPPDATA\Ragnarok",
    [switch]$AddToPath,
    [switch]$SkipDependencies,
    [switch]$Unattended
)

$ErrorActionPreference = "Stop"
$VERSION = "1.1.0"
$REPO_URL = "https://github.com/andragon31/Ragnarok"
$BINARIES_URL = "https://github.com/andragon31/Ragnarok/releases/download/v$VERSION"

function Write-Step($message) {
    $colors = @{ForegroundColor = "Cyan"; BackgroundColor = "Black"}
    Write-Host "`n[STEP] $message" @colors
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

function Get-Downloader {
    if (Test-Command "curl") { return { curl -L -o $args[1] $args[0] } }
    if (Test-Command "Invoke-WebRequest") { return { Invoke-WebRequest -Uri $args[0] -OutFile $args[1] } }
    throw "No downloader found. Install curl or use PowerShell 5+"
}

function New-Directory($path) {
    if (!(Test-Path $path)) {
        New-Item -ItemType Directory -Path $path -Force | Out-Null
    }
}

function Expand-Zip($zipPath, $dest) {
    Expand-Archive -Path $zipPath -DestinationPath $dest -Force
}

Write-Host @"
                                    `
     _  _   _ ___ ___    _ _____ _  _ ___ _  _ ___    _   _ _____
    | \| | | | __| _ \  / \_   _| || | __| \| |   \  | | | |_   _|
    | .` | |_| _||   / / _ \| | | __ | _|| .` | |) | | |_| | | |
    |_|\_|____|___|_|_\___/|_| |_||_|___|_|\_|___/   \___/  |_|
                                                                            v$VERSION
    ──────────────────────────────────────────────────────────────────────────────
     AI Governance & Memory Layer Ecosystem
     https://github.com/andragon31/Ragnarok
    ──────────────────────────────────────────────────────────────────────────────
"@ -ForegroundColor Cyan

Write-Host "`nInstalling Ragnarok v$VERSION..." -ForegroundColor White

# Detect OS
$IS_WINDOWS = $env:OS -eq "Windows_NT"
$IS_LINUX = $env:OSTYPE -eq "linux-gnu"
$IS_MACOS = $env:OSTYPE -match "darwin"

if (!$IS_WINDOWS -and !$IS_LINUX -and !$IS_MACOS) {
    Write-Err "Unsupported operating system"
    exit 1
}

Write-Step "1. Checking prerequisites"

# Check for Go (required to build)
if (!$SkipDependencies) {
    if (Test-Command "go") {
        $goVersion = (go version) -match 'go([0-9]+\.[0-9]+)'
        if ($goVersion) {
            Write-Success "Go installed: $($Matches[1])"
        }
    } else {
        Write-Warn "Go not found. Installing Ragnarok from pre-built binaries instead."
    }
}

# Check for Git
if (Test-Command "git") {
    Write-Success "Git installed"
} else {
    Write-Warn "Git not found. Some features may not work."
}

Write-Step "2. Creating installation directory"
New-Directory $InstallDir
Write-Success "Installation directory: $InstallDir"

# Define paths
$BIN_DIR = Join-Path $InstallDir "bin"
$DATA_DIR = Join-Path $InstallDir "data"
$FENRIR_DIR = Join-Path $DATA_DIR ".fenrir"
$HATI_DIR = Join-Path $DATA_DIR ".hati"
$SKOLL_DIR = Join-Path $DATA_DIR ".skoll"
$TYR_DIR = Join-Path $DATA_DIR ".tyr"

New-Directory $BIN_DIR
New-Directory $FENRIR_DIR
New-Directory $HATI_DIR
New-Directory $SKOLL_DIR
New-Directory $TYR_DIR

Write-Success "Data directories created"

Write-Step "3. Downloading binaries"

$PLUGINS = @(
    @{Name="fenrir"; Port=7437},
    @{Name="hati"; Port=7439},
    @{Name="skoll"; Port=7438},
    @{Name="tyr"; Port=7440},
    @{Name="rag"; Port=0}
)

$OS_SUFFIX = ""
if ($IS_LINUX) { $OS_SUFFIX = "-linux-amd64" }
if ($IS_MACOS) { $OS_SUFFIX = "-darwin-amd64" }
if ($IS_WINDOWS) { $OS_SUFFIX = "-windows-amd64.zip" }

$download = Get-Downloader

foreach ($plugin in $PLUGINS) {
    $binName = $plugin.Name
    if ($IS_WINDOWS) {
        $remoteUrl = "$BINARIES_URL/${binName}$OS_SUFFIX"
        $localZip = Join-Path $InstallDir "${binName}.zip"
        $localExe = Join-Path $BIN_DIR "${binName}.exe"
        
        Write-Host "  Downloading $binName..." -NoNewline
        try {
            & $download $remoteUrl $localZip
            if (Test-Path $localZip) {
                Expand-Zip $localZip $BIN_DIR
                # Move exe from nested folder if needed
                $nestedExe = Get-ChildItem -Path $BIN_DIR -Filter "${binName}.exe" -Recurse | Select-Object -First 1
                if ($nestedExe) {
                    Move-Item -Path $nestedExe.FullName -Destination $localExe -Force
                    Remove-Item -Path $nestedExe.DirectoryName -Recurse -Force -ErrorAction SilentlyContinue
                }
                Remove-Item $localZip -Force -ErrorAction SilentlyContinue
                Write-Success $binName
            } else {
                Write-Warn "$binName (using build from source)"
            }
        } catch {
            Write-Warn "$binName (download failed, will build from source)"
        }
    } else {
        # For Linux/macOS, download tar.gz
        $remoteUrl = "$BINARIES_URL/${binName}$OS_SUFFIX.tar.gz"
        Write-Host "  Downloading $binName..." -NoNewline
        try {
            & $download $remoteUrl (Join-Path $InstallDir "${binName}.tar.gz")
            Write-Success $binName
        } catch {
            Write-Warn "$binName (download failed)"
        }
    }
}

# If no pre-built binaries, clone and build from source
$hasBinaries = Get-ChildItem -Path $BIN_DIR -Filter "*.exe" -ErrorAction SilentlyContinue | Measure-Object
if ($hasBinaries.Count -eq 0) {
    Write-Step "3b. Building from source"
    
    Write-Host "  Cloning repository..." -NoNewline
    $TEMP_DIR = Join-Path $env:TEMP "ragnarok_build"
    Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue
    
    try {
        git clone --depth 1 --branch "v$VERSION" $REPO_URL $TEMP_DIR 2>$null
        Write-Success "Repository cloned"
    } catch {
        Write-Warn "Clone failed. Trying main branch..."
        git clone --depth 1 $REPO_URL $TEMP_DIR 2>$null
    }
    
    if (Test-Path (Join-Path $TEMP_DIR "go.mod")) {
        Write-Host "  Building binaries..." -NoNewline
        
        $buildCmds = @(
            "go build -ldflags='-s -w' -o '$BIN_DIR/fenrir.exe' ./fenrir/cmd/fenrir",
            "go build -ldflags='-s -w' -o '$BIN_DIR/hati.exe' ./hati/cmd/hati",
            "go build -ldflags='-s -w' -o '$BIN_DIR/skoll.exe' ./skoll/cmd/skoll",
            "go build -ldflags='-s -w' -o '$BIN_DIR/tyr.exe' ./tyr/cmd/tyr",
            "go build -ldflags='-s -w' -o '$BIN_DIR/rag.exe' ./installer/cmd/rag"
        )
        
        foreach ($cmd in $buildCmds) {
            $parts = $cmd -split ' -o '
            $outPath = $parts[1] -replace "'", ""
            $buildCmd = ($cmd -split ' -o ')[0]
            
            Push-Location $TEMP_DIR
            try {
                Invoke-Expression $buildCmd 2>$null
                if (Test-Path $outPath) {
                    Write-Host "." -NoNewline
                }
            } catch {
                Write-Host "!" -NoNewline
            }
            Pop-Location
        }
        
        Write-Success "Build complete"
        Remove-Item -Path $TEMP_DIR -Recurse -Force -ErrorAction SilentlyContinue
    } else {
        Write-Err "Could not build from source. Go may not be installed."
        Write-Host "Please install Go 1.22+ and run installer again."
    }
}

Write-Step "4. Detecting OpenCode configuration directory"

# Try common OpenCode config locations
$OPENCODE_CONFIG_DIRS = @(
    "$env:APPDATA\opencode",
    "$env:LOCALAPPDATA\opencode",
    "$env:USERPROFILE\.opencode",
    "$env:HOME\.opencode"
)

$opencodeConfigDir = $null
foreach ($dir in $OPENCODE_CONFIG_DIRS) {
    if (Test-Path $dir) {
        $opencodeConfigDir = $dir
        break
    }
}

# Also check for .mcp.json in common locations
$MCP_CONFIG_LOCATIONS = @(
    "$env:APPDATA\opencode\.mcp.json",
    "$env:LOCALAPPDATA\opencode\.mcp.json",
    "$env:USERPROFILE\.mcp.json",
    "$env:HOME\.opencode\.mcp.json"
)

$existingMcp = $null
foreach ($loc in $MCP_CONFIG_LOCATIONS) {
    if (Test-Path $loc) {
        $existingMcp = $loc
        break
    }
}

if ($opencodeConfigDir) {
    Write-Success "OpenCode config: $opencodeConfigDir"
} else {
    Write-Warn "OpenCode not found. Will create config in user's home."
    $opencodeConfigDir = "$env:USERPROFILE\.opencode"
    New-Directory $opencodeConfigDir
}

Write-Step "5. Creating MCP configuration (.mcp.json)"

# Generate .mcp.json content
$PLUGIN_PORTS = @{
    "fenrir" = 7437
    "hati" = 7439
    "skoll" = 7438
    "tyr" = 7440
}

$mcpServers = @{}
foreach ($plugin in $PLUGINS) {
    if ($plugin.Port -eq 0) { continue }  # Skip 'rag'
    
    $mcpServers[$plugin.Name] = @{
        command = Join-Path $BIN_DIR "$($plugin.Name).exe"
        args = @("serve", "--port", $PLUGIN_PORTS[$plugin.Name].ToString())
        env = @{
            "MCP_TRANSPORT" = "tcp"
            "RAGNAROK_DATA" = $DATA_DIR
        }
    }
}

$mcpConfig = @{
    mcpServers = $mcpServers
}

$mcpJsonPath = Join-Path $opencodeConfigDir ".mcp.json"
$mcpJsonContent = $mcpConfig | ConvertTo-Json -Depth 10
$mcpJsonContent | Set-Content -Path $mcpJsonPath -Encoding UTF8

Write-Success "MCP config: $mcpJsonPath"

# Backup existing .mcp.json if exists
if ($existingMcp -and $existingMcp -ne $mcpJsonPath) {
    Write-Host "  Backed up existing config to $($existingMcp).bak"
    Copy-Item $existingMcp "$existingMcp.bak" -Force
}

Write-Step "6. Creating plugin data directories"

# Create data directory structure
$DATA_STRUCTURE = @{
    ".fenrir" = @("memory", "graphs", "config.json")
    ".hati" = @("plans", "checkpoints", "config.json")
    ".skoll" = @("skills", "rules", "config.json")
    ".tyr" = @("standards", "findings", "config.json")
}

foreach ($dir in @($FENRIR_DIR, $HATI_DIR, $SKOLL_DIR, $TYR_DIR)) {
    New-Directory $dir
}

Write-Success "Data directories initialized"

Write-Step "7. Creating rag wrapper scripts"

# Create rag.cmd for easy access
$ragCmdContent = "@echo off
`"$BIN_DIR\rag.exe`" %*
"
$ragCmdPath = Join-Path $BIN_DIR "rag.cmd"
$ragCmdContent | Set-Content -Path $ragCmdPath -Encoding ASCII

Write-Success "Created rag wrapper"

Write-Step "8. Setting up PATH"

$PATH_SETUP = @"
To add Ragnarok to your PATH, run:

    \$userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    [Environment]::SetEnvironmentVariable('PATH', '\$userPath;$BIN_DIR', 'User')

Or add this line to your PowerShell profile (~/.config/powershell/Microsoft.PowerShell_profile.ps1):

    \$env:PATH += ';$BIN_DIR'
"@

if ($AddToPath) {
    $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    $newPath = "$userPath;$BIN_DIR"
    [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
    $env:PATH = $newPath
    Write-Success "Added to PATH: $BIN_DIR"
} else {
    Write-Host $PATH_SETUP -ForegroundColor Yellow
}

Write-Step "9. Verifying installation"

$allGood = $true
foreach ($plugin in $PLUGINS) {
    $exePath = Join-Path $BIN_DIR "$($plugin.Name).exe"
    if (Test-Path $exePath) {
        $version = & $exePath version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "$($plugin.Name): $version" -ForegroundColor Green
        } else {
            Write-Warn "$($plugin.Name): installed but failed to run"
        }
    } else {
        Write-Err "$($plugin.Name): not found"
        $allGood = $false
    }
}

Write-Host "`n═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  INSTALLATION COMPLETE!" -ForegroundColor Green
Write-Host "═══════════════════════════════════════════════════════════════`n" -ForegroundColor Cyan

Write-Host "Next steps:`n" -ForegroundColor White
Write-Host "  1. Start the ecosystem:" -ForegroundColor White
Write-Host "     .\$BIN_DIR\rag.exe serve" -ForegroundColor Yellow
Write-Host ""
Write-Host "  2. Initialize a project:" -ForegroundColor White
Write-Host "     .\$BIN_DIR\rag.exe init --project my-project" -ForegroundColor Yellow
Write-Host ""
Write-Host "  3. Scan and bootstrap:" -ForegroundColor White
Write-Host "     .\$BIN_DIR\rag.exe scan --path .\my-project" -ForegroundColor Yellow
Write-Host ""
Write-Host "  4. Check ecosystem health:" -ForegroundColor White
Write-Host "     .\$BIN_DIR\rag.exe stats --ecosystem" -ForegroundColor Yellow
Write-Host ""

if (!$AddToPath) {
    Write-Host "  5. Add to PATH permanently:" -ForegroundColor White
    Write-Host "     \$env:PATH += ';$BIN_DIR'" -ForegroundColor Yellow
    Write-Host "     (Add this to your PowerShell profile)" -ForegroundColor Gray
}

Write-Host "`nDocumentation: https://github.com/ragnarok-ecosystem/ragnarok" -ForegroundColor Gray

if ($Unattended) {
    exit 0
}

Write-Host "`nPress any key to exit..." -ForegroundColor Gray
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
