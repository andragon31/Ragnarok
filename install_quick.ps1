# Ragnarok Quick Installer
# One-line install: iwr https://tinyurl.com/ragnarok-install | iex
# Or:              iwr https://bit.ly/ragnarok-install | iex

param([string]$Version = "")

$REPO_OWNER = "andragon31"
$REPO_NAME  = "Ragnarok"
$BASE_URL   = "https://raw.githubusercontent.com/$REPO_OWNER/$REPO_NAME"

if ($Version -eq "") {
    $release  = Invoke-RestMethod "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"
    $tag      = $release.tag_name
} else {
    $tag = "v$Version"
}

$scriptUrl = "$BASE_URL/$tag/install.ps1"
$tmpScript = Join-Path $env:TEMP "ragnarok_install_$(Get-Random).ps1"

try {
    Write-Host "Descargando instalador de Ragnarok $tag..." -ForegroundColor Cyan
    Invoke-WebRequest -Uri $scriptUrl -OutFile $tmpScript -UseBasicParsing
    & $tmpScript -Version ($tag.TrimStart("v")) @args
} finally {
    Remove-Item $tmpScript -ErrorAction SilentlyContinue
}
