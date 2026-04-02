# Ragnarok v3.1.0 - Installation Verification Script
# Tests the unified binary architecture (single rag.exe)
# Usage:
#   .\verify_install.ps1
#   .\verify_install.ps1 -Verbose
#   .\verify_install.ps1 -InstallDir "C:\MyCustomPath\Ragnarok"

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Ragnarok",
    [switch]$Verbose
)

$ErrorActionPreference = "Continue"
$passed = 0
$failed = 0

function Pass($name) {
    Write-Host "[PASS] $name" -ForegroundColor Green
    $script:passed++
}

function Fail($name, $reason) {
    Write-Host "[FAIL] $name — $reason" -ForegroundColor Red
    $script:failed++
}

function Skip($name, $reason) {
    if ($Verbose) {
        Write-Host "[SKIP] $name — $reason" -ForegroundColor Gray
    }
}

Write-Host @"
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   Ragnarok v3.1.0 - Installation Verification                 ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
"@ -ForegroundColor Cyan

Write-Host "`nChecking installation in: $InstallDir`n" -ForegroundColor White

# ─── 1. Unified binary ───────────────────────────────────────────────────────
Write-Host "── Binary ────────────────────────────────────────" -ForegroundColor DarkCyan

$ragBin = Join-Path $InstallDir "rag.exe"
if (Test-Path $ragBin) {
    Pass "rag.exe exists ($ragBin)"

    # version command
    try {
        $ver = & $ragBin version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Pass "rag version: $ver"
        } else {
            Fail "rag version" "exit code $LASTEXITCODE"
        }
    } catch {
        Fail "rag version" "exception: $_"
    }

    # doctor command
    try {
        $doc = & $ragBin doctor 2>&1
        if ($LASTEXITCODE -eq 0) {
            Pass "rag doctor passed"
            if ($Verbose) { $doc | ForEach-Object { Write-Host "       $_" -ForegroundColor Gray } }
        } else {
            Fail "rag doctor" ($doc | Select-Object -First 3 | Out-String).Trim()
        }
    } catch {
        Fail "rag doctor" "exception: $_"
    }

    # MCP stdio responds
    try {
        $initMsg = '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"verify","version":"1"}}}'
        $mcp = $initMsg | & $ragBin mcp 2>$null | Select-Object -First 1
        if ($mcp -match '"result"') {
            Pass "MCP stdio responds"
        } else {
            Fail "MCP stdio" "no valid response (got: $($mcp | Select-Object -First 1))"
        }
    } catch {
        Fail "MCP stdio" "exception: $_"
    }

} else {
    Fail "rag.exe exists" "not found at $ragBin — run installer first"
}

# ─── 2. PATH ─────────────────────────────────────────────────────────────────
Write-Host "`n── PATH ──────────────────────────────────────────" -ForegroundColor DarkCyan

$inPath = $env:PATH -split ";" | Where-Object { $_ -like "*Ragnarok*" }
if ($inPath) {
    Pass "PATH contains Ragnarok dir ($($inPath | Select-Object -First 1))"
} else {
    Fail "PATH" "Ragnarok dir not in PATH — restart terminal or re-run installer"
}

# Verify rag is resolvable from PATH
$ragInPath = Get-Command "rag" -ErrorAction SilentlyContinue
if ($ragInPath) {
    Pass "rag resolves from PATH ($($ragInPath.Source))"
} else {
    Fail "rag resolves from PATH" "not resolvable — may need to restart terminal"
}

# ─── 3. MCP config for detected IDEs ─────────────────────────────────────────
Write-Host "`n── MCP Configuration ────────────────────────────" -ForegroundColor DarkCyan

$ideConfigs = @{
    "OpenCode"         = "$env:USERPROFILE\.config\opencode\opencode.json"
    "Cursor"           = "$env:USERPROFILE\.cursor\mcp.json"
    "Windsurf"         = "$env:USERPROFILE\.windsurf\mcp.json"
    "Claude Code"      = "$env:USERPROFILE\.claude\settings.json"
    "Gemini CLI"       = "$env:USERPROFILE\.gemini\settings.json"
}

$anyIdeFound = $false
foreach ($ide in $ideConfigs.Keys) {
    $path = $ideConfigs[$ide]
    if (Test-Path $path) {
        $anyIdeFound = $true
        try {
            $json = Get-Content $path -Raw | ConvertFrom-Json
            # Check both mcpServers (standard) and mcp (OpenCode legacy)
            $ragFound = $false
            if ($json.mcpServers -and $json.mcpServers.ragnarok) { $ragFound = $true }
            if ($json.mcp -and $json.mcp.ragnarok)               { $ragFound = $true }
            if ($ragFound) {
                Pass "MCP config: $ide ($path)"
            } else {
                Fail "MCP config: $ide" "ragnarok server not found — run 'rag setup $(($ide -replace ' ','').ToLower())'"
            }
        } catch {
            Fail "MCP config: $ide" "JSON parse error — $_"
        }
    } else {
        Skip "MCP config: $ide" "not installed (config not found at $path)"
    }
}

if (-not $anyIdeFound) {
    Write-Host "[INFO] No supported IDEs detected. Run 'rag setup <ide>' after installing an IDE." -ForegroundColor Yellow
}

# ─── 4. Data directory ────────────────────────────────────────────────────────
Write-Host "`n── Data Directories ─────────────────────────────" -ForegroundColor DarkCyan

$dataDir = "$env:USERPROFILE\.ragnarok"
if (Test-Path $dataDir) {
    Pass "Data dir exists (~/.ragnarok)"
    foreach ($db in @("fenrir.db", "hati.db", "skoll.db", "tyr.db")) {
        $dbPath = Join-Path $dataDir $db
        if (Test-Path $dbPath) {
            if ($Verbose) { Pass "  $db" }
        } else {
            if ($Verbose) { Write-Host "[INFO]   $db not yet created (normal — created on first use)" -ForegroundColor Gray }
        }
    }
} else {
    Write-Host "[INFO] ~/.ragnarok not yet initialized — run 'rag doctor' to initialize" -ForegroundColor Yellow
}

# ─── Summary ──────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  VERIFICATION SUMMARY" -ForegroundColor White
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Tests passed: $passed" -ForegroundColor Green
Write-Host "  Tests failed: $failed" -ForegroundColor $(if ($failed -eq 0) { "Green" } else { "Red" })

if ($failed -eq 0) {
    Write-Host "`n  ✓ Installation verified successfully!" -ForegroundColor Green
    Write-Host "`n  Next: Run 'rag --help' to see all commands" -ForegroundColor White
    exit 0
} else {
    Write-Host "`n  ✗ Some checks failed. Re-run 'rag doctor' or the installer." -ForegroundColor Yellow
    exit $failed
}
