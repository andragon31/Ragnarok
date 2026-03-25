# Ragnarok Backup Scheduler for Windows
# Requiere: PowerShell 5.0+
# Uso: .\schedule_backup.ps1 [-Time "02:00"] [-Daily]

param(
    [Parameter(Mandatory=$false)]
    [string]$Time = "02:00",

    [Parameter(Mandatory=$false)]
    [ValidateSet("Daily", "Weekly")]
    [string]$Frequency = "Daily",

    [Parameter(Mandatory=$false)]
    [int]$RetentionDays = 30
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BackupScript = Join-Path $ScriptDir "backup_ragnarok.ps1"
$TaskName = "RagnarokBackup"

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] $Message"
}

if (-not (Test-Path $BackupScript)) {
    Write-Log "ERROR: Backup script not found: $BackupScript"
    Write-Log "Make sure backup_ragnarok.ps1 is in the same directory"
    exit 1
}

$existingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Write-Log "Removing existing scheduled task..."
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
}

$action = New-ScheduledTaskAction -Execute "PowerShell.exe" -Argument "-NoProfile -ExecutionPolicy Bypass -File `"$BackupScript`" -RetentionDays $RetentionDays"

$trigger = New-ScheduledTaskTrigger -Daily -At $Time
if ($Frequency -eq "Weekly") {
    $trigger = New-ScheduledTaskTrigger -Weekly -DaysOfWeek Sunday -At $Time
}

$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Limited

$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable

Write-Log "Creating scheduled task..."
Write-Log "  Task name: $TaskName"
Write-Log "  Schedule: $Frequency at $Time"
Write-Log "  Retention: $RetentionDays days"

Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Description "Ragnarok ecosystem backup" | Out-Null

Write-Log "Scheduled task created successfully"
Write-Log ""
Write-Log "To run manually:"
Write-Log "  .\backup_ragnarok.ps1"
Write-Log ""
Write-Log "To view scheduled tasks:"
Write-Log "  Get-ScheduledTask | Where-Object TaskName -like 'Ragnarok*'"
Write-Log ""
Write-Log "To remove:"
Write-Log "  Unregister-ScheduledTask -TaskName $TaskName -Confirm:`$false"