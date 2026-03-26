# Ragnarok Restore Script for Windows
# Requiere: PowerShell 5.0+
# Uso: .\restore_ragnarok.ps1 -BackupFile <path> -Plugin <name>

param(
    [Parameter(Mandatory=$true)]
    [string]$BackupFile,

    [Parameter(Mandatory=$true)]
    [ValidateSet("fenrir", "hati", "skoll", "tyr")]
    [string]$Plugin,

    [Parameter(Mandatory=$false)]
    [switch]$Force
)

$ErrorActionPreference = "Stop"

$Plugins = @{
    "fenrir" = @{ Dir = "$env:USERPROFILE\.fenrir" }
    "hati"   = @{ Dir = "$env:USERPROFILE\.hati" }
    "skoll"  = @{ Dir = "$env:USERPROFILE\.skoll" }
    "tyr"    = @{ Dir = "$env:USERPROFILE\.tyr" }
}

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] $Message"
}

Write-Log "Ragnarok Restore Starting..."
Write-Log "Plugin: $Plugin"
Write-Log "Backup file: $BackupFile"

if (-not (Test-Path $BackupFile)) {
    Write-Log "ERROR: Backup file not found: $BackupFile"
    exit 1
}

$targetDir = $Plugins[$Plugin].Dir

if ((Test-Path $targetDir) -and -not $Force) {
    Write-Log "WARNING: Target directory exists: $targetDir"
    Write-Log "Use -Force to overwrite current data"
    $confirm = Read-Host "Continue? (y/N)"
    if ($confirm -ne "y" -and $confirm -ne "Y") {
        Write-Log "Restore cancelled by user"
        exit 0
    }
}

$tempDir = "$env:TEMP\ragnarok_restore_$PID"
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

Write-Log "Extracting backup..."
try {
    Expand-Archive -Path $BackupFile -DestinationPath $tempDir -Force -ErrorAction Stop
} catch {
    Write-Log "ERROR: Failed to extract backup - $_"
    Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    exit 1
}

if (Test-Path $targetDir) {
    Write-Log "Backing up current data..."
    $currentBackup = "$targetDir`_backup_$(Get-Date -Format 'yyyyMMdd_HHmmss').zip"
    Compress-Archive -Path "$targetDir\*" -DestinationPath $currentBackup -Force
    Write-Log "Current data backed up to: $currentBackup"

    Write-Log "Removing current data..."
    Remove-Item "$targetDir\*" -Recurse -Force
}

Write-Log "Restoring data..."
$extractedContent = Get-ChildItem -Path $tempDir -Recurse -File
foreach ($file in $extractedContent) {
    $relativePath = $file.FullName.Substring($tempDir.Length)
    $targetPath = Join-Path $targetDir $relativePath

    $targetParent = Split-Path $targetPath -Parent
    if (-not (Test-Path $targetParent)) {
        New-Item -ItemType Directory -Path $targetParent -Force | Out-Null
    }

    Copy-Item $file.FullName -Destination $targetPath -Force
}

Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue

Write-Log "Restore complete for $Plugin"
Write-Log "Target directory: $targetDir"
Write-Log ""
Write-Log "IMPORTANT: Restart the MCP server for changes to take effect:"
Write-Log "  $Plugin serve"