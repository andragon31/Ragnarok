# Ragnarok Installer Verification Script
# Run this after installation to verify everything works

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Ragnarok",
    [switch]$Verbose
)

$ErrorActionPreference = "Continue"
$testsPassed = 0
$testsFailed = 0

function Test-Item($name, $condition, $errorMsg) {
    if ($condition) {
        Write-Host "[PASS] $name" -ForegroundColor Green
        return $true
    } else {
        Write-Host "[FAIL] $name - $errorMsg" -ForegroundColor Red
        return $false
    }
}

Write-Host @"
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   Ragnarok v1.1.0 - Installation Verification                  ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
"@ -ForegroundColor Cyan

Write-Host "`nChecking installation in: $InstallDir`n" -ForegroundColor White

# Test 1: Binary directory exists
$binDir = Join-Path $InstallDir "bin"
if (Test-Item "Binary directory exists" (Test-Path $binDir) "Run installer first") {
    $testsPassed++
    
    # Test binaries
    $binaries = @("fenrir.exe", "hati.exe", "skoll.exe", "tyr.exe", "rag.exe")
    foreach ($bin in $binaries) {
        $binPath = Join-Path $binDir $bin
        $exists = Test-Path $binPath
        if (Test-Item "$bin exists" $exists "Missing") {
            $testsPassed++
        } else {
            $testsFailed++
        }
        
        # Test version command
        if ($exists) {
            try {
                $version = & $binPath version 2>$null
                if ($LASTEXITCODE -eq 0 -and $version) {
                    if (Test-Item "$bin runs" $true "Failed to run") {
                        $testsPassed++
                        if ($Verbose) { Write-Host "       $version" -ForegroundColor Gray }
                    }
                } else {
                    $testsFailed++
                }
            } catch {
                if (Test-Item "$bin runs" $false "Error: $_") { }
                $testsFailed++
            }
        } else {
            $testsFailed++
        }
    }
} else {
    $testsFailed++
}

# Test MCP config
Write-Host "`nChecking MCP configuration..." -ForegroundColor White
$mcpPaths = @(
    "$env:APPDATA\opencode\.mcp.json",
    "$env:LOCALAPPDATA\opencode\.mcp.json",
    "$env:USERPROFILE\.opencode\.mcp.json"
)

$mcpFound = $false
foreach ($path in $mcpPaths) {
    if (Test-Path $path) {
        $mcpFound = $true
        if (Test-Item ".mcp.json found" $true "at $path") {
            $testsPassed++
            
            try {
                $content = Get-Content $path -Raw | ConvertFrom-Json
                $servers = $content.mcpServers.PSObject.Properties.Name
                
                foreach ($server in @("fenrir", "hati", "skoll", "tyr")) {
                    if ($servers -contains $server) {
                        if (Test-Item "  $server configured" $true "") {
                            $testsPassed++
                        }
                    } else {
                        if (Test-Item "  $server configured" $false "Missing from .mcp.json") {
                            $testsFailed++
                        }
                    }
                }
            } catch {
                if (Test-Item ".mcp.json valid JSON" $false "Parse error: $_") {
                    $testsFailed++
                }
            }
        }
        break
    }
}

if (!$mcpFound) {
    if (Test-Item ".mcp.json found" $false "Not in expected locations") {
        $testsFailed++
    }
}

# Test data directories
Write-Host "`nChecking data directories..." -ForegroundColor White
$dataDir = Join-Path $InstallDir "data"
if (Test-Item "Data directory exists" (Test-Path $dataDir) "") {
    $testsPassed++
    
    $plugins = @("fenrir", "hati", "skoll", "tyr")
    foreach ($plugin in $plugins) {
        $pluginDir = Join-Path $dataDir ".$plugin"
        if (Test-Item "  .$plugin directory" (Test-Path $pluginDir) "") {
            $testsPassed++
        } else {
            $testsFailed++
        }
    }
} else {
    $testsFailed++
}

# Summary
Write-Host "`n═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  VERIFICATION SUMMARY" -ForegroundColor White
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Tests passed: $testsPassed" -ForegroundColor Green
Write-Host "  Tests failed: $testsFailed" -ForegroundColor $(if ($testsFailed -eq 0) { "Green" } else { "Red" })

if ($testsFailed -eq 0) {
    Write-Host "`n  ✓ Installation verified successfully!" -ForegroundColor Green
    Write-Host "`n  Next: Run 'rag serve' to start the ecosystem" -ForegroundColor White
    exit 0
} else {
    Write-Host "`n  ✗ Some tests failed. Re-run installer or check configuration." -ForegroundColor Yellow
    exit 1
}
