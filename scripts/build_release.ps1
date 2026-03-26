# Build and Release Script for Ragnarok
# Usage: .\scripts\build_release.ps1 [-SkipBuild] [-SkipUpload]

param(
    [switch]$SkipBuild,
    [switch]$SkipUpload,
    [string]$Version = "1.1.0"
)

$ErrorActionPreference = "Stop"
$REPO_DIR = Split-Path -Parent $PSScriptRoot
$RELEASE_DIR = Join-Path $REPO_DIR "release"
$BIN_DIR = Join-Path $RELEASE_DIR "bin"

function New-Directory($path) {
    if (!(Test-Path $path)) {
        New-Item -ItemType Directory -Path $path -Force | Out-Null
    }
}

Write-Host @"
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   Ragnarok v$Version - Build & Release Script                    ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
"@ -ForegroundColor Cyan

# Step 1: Clean previous release
Write-Host "`n[1/4] Cleaning previous build..." -ForegroundColor White
if (Test-Path $RELEASE_DIR) {
    Remove-Item -Path $RELEASE_DIR -Recurse -Force
}
New-Directory $RELEASE_DIR
New-Directory $BIN_DIR
Write-Host "  Done." -ForegroundColor Green

# Step 2: Build binaries
if (!$SkipBuild) {
    Write-Host "`n[2/4] Building binaries..." -ForegroundColor White
    
    Push-Location $REPO_DIR
    
    $builds = @(
        @{Name="fenrir"; Path=".\fenrir\cmd\fenrir"},
        @{Name="hati"; Path=".\hati\cmd\hati"},
        @{Name="skoll"; Path=".\skoll\cmd\skoll"},
        @{Name="tyr"; Path=".\tyr\cmd\tyr"},
        @{Name="rag"; Path=".\installer\cmd\rag"}
    )
    
    foreach ($build in $builds) {
        Write-Host "  Building $($build.Name)..." -NoNewline
        $outFile = Join-Path $BIN_DIR "$($build.Name).exe"
        
        try {
            go build -ldflags="-s -w" -o $outFile $build.Path
            if (Test-Path $outFile) {
                $size = (Get-Item $outFile).Length / 1MB
                Write-Host " [$($size.ToString('F1')) MB]" -ForegroundColor Green
            } else {
                Write-Host " [FAILED]" -ForegroundColor Red
            }
        } catch {
            Write-Host " [ERROR: $_]" -ForegroundColor Red
        }
    }
    
    Pop-Location
} else {
    Write-Host "`n[2/4] Skipping build (using existing binaries)" -ForegroundColor Yellow
}

# Step 3: Create ZIP packages
Write-Host "`n[3/4] Creating ZIP packages..." -ForegroundColor White

$zipPrefix = "ragnarok-$Version-windows-amd64"

foreach ($exe in Get-ChildItem -Path $BIN_DIR -Filter "*.exe") {
    Write-Host "  Packaging $($exe.Name)..." -NoNewline
    
    $zipName = "$zipPrefix-$($exe.BaseName).zip"
    $zipPath = Join-Path $RELEASE_DIR $zipName
    
    try {
        Compress-Archive -Path $exe.FullName -DestinationPath $zipPath -Force
        Write-Host " [$zipName]" -ForegroundColor Green
    } catch {
        Write-Host " [FAILED]" -ForegroundColor Red
    }
}

# Also create a bundle with all binaries
Write-Host "  Creating bundle..." -NoNewline
$bundleZip = Join-Path $RELEASE_DIR "ragnarok-$Version-windows-amd64.zip"
try {
    Compress-Archive -Path "$BIN_DIR\*.exe" -DestinationPath $bundleZip -Force
    Write-Host " [ragnarok-$Version-windows-amd64.zip]" -ForegroundColor Green
} catch {
    Write-Host " [FAILED]" -ForegroundColor Red
}

# Step 4: Upload to GitHub
Write-Host "`n[4/4] GitHub Release..." -ForegroundColor White

if (!$SkipUpload) {
    if (Test-Command "gh") {
        Write-Host "  Creating GitHub release..." -NoNewline
        
        $tag = "v$Version"
        
        # Check if tag exists
        $existingTag = git tag -l $tag 2>$null
        if ($existingTag -eq $tag) {
            Write-Host " [Tag $tag exists, skipping]" -ForegroundColor Yellow
        } else {
            git tag $tag
            git push origin $tag
            Write-Host " [Tag $tag created and pushed]" -ForegroundColor Green
        }
        
        # Create release
        $releaseNotes = @"
# Ragnarok v$Version

## AI Governance & Memory Layer Ecosystem

### Installation
\`\`\`powershell
irm https://tinyurl.com/ragnarok-install | iex
\`\`\`

### Binaries
- fenrir.exe - Memory & Knowledge
- hati.exe - Planning & Approvals
- skoll.exe - Skills & Rules
- tyr.exe - Security & Validation
- rag.exe - Ragnarok Installer

### What's New
- TCP transport support for all plugins
- Human-in-the-loop corrections
- Multi-agent coordination with locks
- SLA-based escalation

### Downloads
See the Assets section for binary packages.
"@
        
        try {
            gh release create $tag --title "Ragnarok v$Version" --notes "$releaseNotes" 2>$null
            Write-Host " [Release created]" -ForegroundColor Green
        } catch {
            Write-Host " [Manual upload required: https://github.com/ragnarok-ecosystem/ragnarok/releases/new]" -ForegroundColor Yellow
        }
        
        # Upload assets
        foreach ($zip in Get-ChildItem -Path $RELEASE_DIR -Filter "*.zip") {
            Write-Host "  Uploading $($zip.Name)..." -NoNewline
            try {
                gh release upload $tag $zip.FullName --clobber 2>$null
                Write-Host " [Uploaded]" -ForegroundColor Green
            } catch {
                Write-Host " [Skipped - manual upload required]" -ForegroundColor Yellow
            }
        }
    } else {
        Write-Host "  GitHub CLI not found. Manually create release at:" -ForegroundColor Yellow
        Write-Host "  https://github.com/ragnarok-ecosystem/ragnarok/releases/new?tag=v$Version" -ForegroundColor Yellow
    }
} else {
    Write-Host "  Skipped upload" -ForegroundColor Yellow
}

Write-Host "`n═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  BUILD COMPLETE!" -ForegroundColor Green
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "`nRelease files in: $RELEASE_DIR" -ForegroundColor White
Write-Host "`nNext steps:" -ForegroundColor White
Write-Host "  1. Test the release: .\release\bin\rag.exe version" -ForegroundColor Yellow
Write-Host "  2. Upload to GitHub: .\scripts\build_release.ps1" -ForegroundColor Yellow
Write-Host "  3. Create tinyurl: https://tinyurl.com/create.php" -ForegroundColor Yellow
