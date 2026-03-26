# Ragnarok Backup Script for Windows
# Requiere: PowerShell 5.0+
# Uso: .\backup_ragnarok.ps1 [-Plugin <name>] [-BackupDir <path>]

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("fenrir", "hati", "skoll", "tyr", "all")]
    [string]$Plugin = "all",

    [Parameter(Mandatory=$false)]
    [string]$BackupDir = "$env:USERPROFILE\OneDrive\RagnarokBackups",

    [Parameter(Mandatory=$false)]
    [int]$RetentionDays = 30
)

$ErrorActionPreference = "Stop"

$Plugins = @{
    "fenrir" = @{ Dir = "$env:USERPROFILE\.fenrir"; Port = 7437 }
    "hati"   = @{ Dir = "$env:USERPROFILE\.hati"; Port = 7439 }
    "skoll"  = @{ Dir = "$env:USERPROFILE\.skoll"; Port = 7438 }
    "tyr"    = @{ Dir = "$env:USERPROFILE\.tyr"; Port = 7440 }
}

$Date = Get-Date -Format "yyyy-MM-dd_HHmmss"
$Timestamp = Get-Date -Format "yyyy-MM-dd"

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] $Message"
}

function Test-PluginOnline {
    param([string]$PluginName, [int]$Port)
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:$Port/stats" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        return $response.StatusCode -eq 200
    } catch {
        return $false
    }
}

function Backup-Plugin {
    param(
        [string]$PluginName,
        [string]$SourceDir,
        [string]$BackupPath
    )

    Write-Log "Backing up $PluginName..."

    if (-not (Test-Path $SourceDir)) {
        Write-Log "  Source directory not found: $SourceDir - skipping"
        return $false
    }

    $backupFile = Join-Path $BackupPath "$PluginName`_$Timestamp.zip"

    try {
        Compress-Archive -Path "$SourceDir\*" -DestinationPath $backupFile -Force -ErrorAction Stop
        $size = (Get-Item $backupFile).Length / 1KB
        Write-Log "  Saved: $backupFile ($([math]::Round($size, 2)) KB)"

        $isOnline = Test-PluginOnline $PluginName $Plugins[$PluginName].Port
        if ($isOnline) {
            Write-Log "  Plugin $PluginName is online"
        } else {
            Write-Log "  Plugin $PluginName is offline"
        }

        return $true
    } catch {
        Write-Log "  ERROR: Failed to backup $PluginName - $_"
        return $false
    }
}

function Remove-OldBackups {
    param(
        [string]$BackupPath,
        [int]$Days
    )

    $cutoffDate = (Get-Date).AddDays(-$Days)
    $oldBackups = Get-ChildItem -Path $BackupPath -Filter "*.zip" -ErrorAction SilentlyContinue |
                  Where-Object { $_.LastWriteTime -lt $cutoffDate }

    if ($oldBackups) {
        Write-Log "Removing $($oldBackups.Count) old backup(s)..."
        foreach ($backup in $oldBackups) {
            Remove-Item $backup.FullName -Force
            Write-Log "  Removed: $($backup.Name)"
        }
    }
}

Write-Log "Ragnarok Backup Starting..."
Write-Log "Backup directory: $BackupDir"
Write-Log "Retention: $RetentionDays days"

if (-not (Test-Path $BackupDir)) {
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null
    Write-Log "Created backup directory"
}

$successCount = 0
$totalCount = 0

if ($Plugin -eq "all") {
    foreach ($pluginName in $Plugins.Keys) {
        $totalCount++
        if (Backup-Plugin -PluginName $pluginName -SourceDir $Plugins[$pluginName].Dir -BackupPath $BackupDir) {
            $successCount++
        }
    }
} else {
    $totalCount = 1
    if (Backup-Plugin -PluginName $Plugin -SourceDir $Plugins[$Plugin].Dir -BackupPath $BackupDir) {
        $successCount++
    }
}

Remove-OldBackups -BackupPath $BackupDir -Days $RetentionDays

Write-Log "Backup Complete: $successCount/$totalCount plugins backed up"

$summaryFile = Join-Path $BackupDir "backup_summary.txt"
@"
Ragnarok Backup Summary
======================
Date: $Date
Plugins backed up: $successCount/$totalCount
Backup directory: $BackupDir
Retention: $RetentionDays days
"@ | Out-File -FilePath $summaryFile -Encoding UTF8

Write-Log "Summary saved to: $summaryFile"