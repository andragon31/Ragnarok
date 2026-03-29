# Ragnarok — Auditoría Técnica del Instalador y Ecosistema

**Versión analizada:** v2.2.4  
**Fecha:** 2026-03-28  
**Referencia comparativa:** [engram](https://github.com/Gentleman-Programming/engram)

---

## Índice

1. [Resumen Ejecutivo](#1-resumen-ejecutivo)
2. [Problema Central: El Paradigma de Instalación Está Invertido](#2-problema-central-el-paradigma-de-instalación-está-invertido)
3. [Bug Crítico: install_quick.ps1 apunta al repo equivocado](#3-bug-crítico-install_quickps1-apunta-al-repo-equivocado)
4. [Bug Crítico: Detección irm-iex está fundamentalmente rota](#4-bug-crítico-detección-irm--iex-está-fundamentalmente-rota)
5. [Ausencia Total de CI/CD y Release Pipeline](#5-ausencia-total-de-cicd-y-release-pipeline)
6. [Versión Hardcodeada en el Instalador](#6-versión-hardcodeada-en-el-instalador)
7. [Makefile Inconsistente y Desactualizado](#7-makefile-inconsistente-y-desactualizado)
8. [go.work Sobrante y Convención de Módulo](#8-gowork-sobrante-y-convención-de-módulo)
9. [Directorios de Runtime Comprometidos en el Repo](#9-directorios-de-runtime-comprometidos-en-el-repo)
10. [Sin Soporte Linux/macOS](#10-sin-soporte-linuxmacos)
11. [Plan de Acción por Prioridad](#11-plan-de-acción-por-prioridad)

---

## 1. Resumen Ejecutivo

Ragnarok tiene una arquitectura de ecosistema sólida (HATI → SKOLL → FENRIR → TYR) y un conjunto de funcionalidades bien diseñadas. El problema no está en la lógica de negocio — está en la **capa de distribución y empaquetado**. Un desarrollador nuevo que sigue las instrucciones del README no puede instalar el proyecto de forma confiable, porque:

- El one-liner rápido descarga de un repositorio que no existe.
- El instalador principal tiene una detección bugueada de `irm | iex`.
- No existen binarios precompilados — el usuario necesita tener Go instalado.
- No hay CI/CD configurado, por lo que no existen binarios para ninguna plataforma.

La comparación con engram es directa: engram se instala con `brew install` o descargando un binario. Ragnarok requiere que el usuario instale Go, clone el repo, y compile desde fuente. Esta diferencia de experiencia es la que separa un tool de producción de un proyecto personal.

---

## 2. Problema Central: El Paradigma de Instalación Está Invertido

### El problema

El `install.ps1` funciona así:

```
usuario ejecuta install.ps1
    → verifica que Go esté instalado  ← BARRERA
    → verifica que Git esté instalado ← BARRERA
    → git clone --depth 1 (todo el repo)
    → go build ./cmd/rag
    → copia rag.exe al PATH
    → elimina el clone temporal
```

Esto convierte al instalador en un **wrapper de `go install`** con pasos extra. El usuario necesita Go para instalar una herramienta hecha en Go — exactamente lo que el paradigma de "binary distribution" busca eliminar.

### La causa raíz

No existen binarios precompilados en los GitHub Releases. Hay 17 releases publicados pero ninguno contiene un `.zip` o `.tar.gz` con el binario compilado para cada plataforma. Esto obliga al instalador a compilar desde fuente.

### Cómo debe resolverse

**Paso 1:** Crear `.goreleaser.yaml` para compilar binarios cross-platform automáticamente:

```yaml
# .goreleaser.yaml
version: 2

project_name: ragnarok

before:
  hooks:
    - go mod tidy

builds:
  - id: rag
    main: ./cmd/rag
    binary: rag
    ldflags:
      - -s -w -X main.version={{.Version}}
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - id: rag
    builds:
      - rag
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

release:
  github:
    owner: andragon31
    name: Ragnarok
  name_template: "Ragnarok v{{.Version}}"
```

**Paso 2:** Reescribir `install.ps1` para descargar el binario precompilado:

```powershell
# install.ps1 — paradigma correcto (sin requerir Go)
param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Ragnarok",
    [string]$Version = ""
)

$REPO = "andragon31/Ragnarok"

# Auto-detect latest version
if ($Version -eq "") {
    $release = Invoke-RestMethod "https://api.github.com/repos/$REPO/releases/latest"
    $VERSION = $release.tag_name.TrimStart("v")
} else {
    $VERSION = $Version
}

$ARCH = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$ASSET = "ragnarok_${VERSION}_windows_${ARCH}.zip"
$DOWNLOAD_URL = "https://github.com/$REPO/releases/download/v$VERSION/$ASSET"
$CHECKSUM_URL = "https://github.com/$REPO/releases/download/v$VERSION/checksums.txt"

# Descargar binario precompilado
$zipPath = Join-Path $env:TEMP $ASSET
Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile $zipPath -UseBasicParsing

# Verificar checksum
$checksums = (Invoke-WebRequest -Uri $CHECKSUM_URL -UseBasicParsing).Content
$expected = ($checksums -split "`n" | Where-Object { $_ -match $ASSET }) -split "\s+" | Select-Object -First 1
$actual = (Get-FileHash $zipPath -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected) {
    throw "Checksum mismatch. Expected: $expected, Got: $actual"
}

# Extraer e instalar
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Expand-Archive -Path $zipPath -DestinationPath $InstallDir -Force
Remove-Item $zipPath

# Agregar al PATH
$userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
if ($userPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable('PATH', "$userPath;$InstallDir", 'User')
    $env:PATH = "$env:PATH;$InstallDir"
}
```

---

## 3. Bug Crítico: install_quick.ps1 apunta al repo equivocado

### El problema

```powershell
# ESTADO ACTUAL en install_quick.ps1 — línea 6:
$url = "https://raw.githubusercontent.com/ragnarok-ecosystem/ragnarok/main/install.ps1"
#                                          ^^^^^^^^^^^^^^^^^^^^^^^^^^
#                                          Esta organización NO EXISTE
```

La organización `ragnarok-ecosystem` no existe en GitHub. Cualquier usuario que ejecute el one-liner recomendado (`iwr https://... | iex`) recibe un error 404 inmediato. Es el primer punto de contacto de un nuevo usuario y está completamente roto.

### Cómo debe resolverse

```powershell
# install_quick.ps1 — corregido
param([string]$Version = "")

$REPO_OWNER = "andragon31"
$REPO_NAME  = "Ragnarok"
$BASE_URL   = "https://raw.githubusercontent.com/$REPO_OWNER/$REPO_NAME"

# Obtener la versión más reciente si no se especificó
if ($Version -eq "") {
    $release  = Invoke-RestMethod "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"
    $tag      = $release.tag_name   # e.g. "v2.2.4"
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
```

---

## 4. Bug Crítico: Detección irm | iex está fundamentalmente rota

### El problema

```powershell
# install.ps1 — bloque de re-ejecución (líneas 19-28):
if ($MyInvocation.InvocationName -eq "iex") {
    $scriptPath = Join-Path $env:TEMP "ragnarok_install_..."
    $content = Get-Content $PSCommandPath -Raw    # ← PSCommandPath es NULL en irm|iex
    $content | Set-Content $scriptPath -Encoding UTF8
    & $scriptPath -InstallDir $InstallDir -Version $Version
    Remove-Item $scriptPath -ErrorAction SilentlyContinue
    exit
}
```

Cuando PowerShell recibe un script via pipe (`irm ... | iex`), el script **no tiene ruta en disco**. Las variables `$PSCommandPath` y `$PSScriptRoot` son `$null` o string vacío. El `Get-Content $PSCommandPath` falla porque no hay nada que leer. El bloque completo es código muerto que además puede lanzar excepciones.

### Por qué existe este bloque

El objetivo era guardar el script a disco antes de ejecutarlo (para que los parámetros funcionen con `-File`). Sin embargo, con el paradigma correcto de "descargar binario precompilado", este problema desaparece — el script ya no necesita pasarse a sí mismo como archivo porque ya no hace un `go build` que requiere un directorio de trabajo.

### Cómo debe resolverse

Eliminar el bloque de detección completamente. El script reestructurado del punto 2 no lo necesita. Si se quiere mantener compatibilidad con el modo `irm | iex`, el patrón correcto es no depender de `$PSCommandPath`:

```powershell
# Patrón correcto si se necesita guardar a disco:
$scriptContent = $MyInvocation.MyCommand.ScriptContents  # PowerShell 5+
# O simplemente: diseñar el script para que funcione en memoria sin
# necesitar escribirse a sí mismo en disco.
```

---

## 5. Ausencia Total de CI/CD y Release Pipeline

### El problema

La pestaña de Actions del repositorio no tiene ningún workflow configurado. Los 17 releases existentes fueron creados manualmente. Esto implica:

- No hay verificación automática de que el código compila y los tests pasan en cada PR.
- No hay binarios multiplataforma en los releases (solo el código fuente).
- Cada release requiere trabajo manual del maintainer, lo que introduce riesgo de error.
- No es posible implementar el paradigma de binary distribution sin resolver esto primero.

### Cómo debe resolverse

**Archivo 1: `.github/workflows/ci.yml`** — Validación en cada push/PR:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build-and-test:
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true

      - name: Build
        run: go build -v ./cmd/rag

      - name: Test
        run: go test ./...

      - name: Vet
        run: go vet ./...
```

**Archivo 2: `.github/workflows/release.yml`** — Release automático al crear un tag:

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Con estos dos archivos y el `.goreleaser.yaml` del punto 2, cada `git tag v2.3.0 && git push --tags` genera automáticamente binarios para Windows/Linux/macOS en amd64 y arm64, los sube a GitHub Releases, y calcula checksums.

---

## 6. Versión Hardcodeada en el Instalador

### El problema

```powershell
# install.ps1 — línea 12:
$VERSION = "2.2.4"
```

Cada vez que se publica una versión nueva, este valor queda obsoleto hasta que alguien lo actualiza manualmente. Un usuario que sigue el README de `main` instala siempre la versión hardcodeada, independientemente de si hay versiones más recientes disponibles.

### Cómo debe resolverse

```powershell
# Detectar latest automáticamente, con fallback a versión específica:
param([string]$Version = "")

if ($Version -eq "") {
    try {
        $rel     = Invoke-RestMethod "https://api.github.com/repos/andragon31/Ragnarok/releases/latest"
        $VERSION = $rel.tag_name.TrimStart("v")
        Write-Host "Latest version: $VERSION" -ForegroundColor Cyan
    } catch {
        Write-Warn "No se pudo detectar la última versión. Usando fallback."
        $VERSION = "2.2.4"  # fallback solo si la API falla
    }
} else {
    $VERSION = $Version
}
```

---

## 7. Makefile Inconsistente y Desactualizado

### El problema

**Versión drift:** El target `help` muestra `v1.4.0` mientras el proyecto está en `v2.2.4`.

```makefile
# Makefile línea 47 — versión desactualizada:
help:
    @echo "Ragnarok v1.4.0 - AI Governance & Memory Layer Ecosystem"
```

**Extensión `.exe` hardcodeada con path Unix:**

```makefile
RAG_BIN := $(BIN_DIR)/rag.exe    # Extensión Windows
install:
    cp $(RAG_BIN) ~/.local/bin/  # Path Unix
```

**Ausencia de cross-compilation y release targets.**

### Cómo debe resolverse

```makefile
# Makefile — versión corregida
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags="-s -w -X main.version=$(VERSION)"
BIN_DIR  := bin

# Detectar OS para nombre del binario
ifeq ($(OS),Windows_NT)
    BINARY := $(BIN_DIR)/rag.exe
    INSTALL_DIR := $(LOCALAPPDATA)\Ragnarok
else
    BINARY := $(BIN_DIR)/rag
    INSTALL_DIR := $(HOME)/.local/bin
endif

.PHONY: all build test clean install lint release help

all: build

build:
	@echo "Building Ragnarok $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BINARY) ./cmd/rag
	@echo "✓ Built: $(BINARY)"

build-all:
	@echo "Cross-compiling for all platforms..."
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_linux_amd64   ./cmd/rag
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_linux_arm64   ./cmd/rag
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_darwin_amd64  ./cmd/rag
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_darwin_arm64  ./cmd/rag
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o $(BIN_DIR)/rag_windows_amd64.exe ./cmd/rag

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)

install: build
	@echo "Installing to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/
	@echo "✓ Installed"

release:
	@echo "Creating release $(VERSION)..."
	goreleaser release --clean

help:
	@echo "Ragnarok $(VERSION) — AI Governance & Autonomous Development Ecosystem"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build for current platform"
	@echo "  build-all   Cross-compile for all platforms"
	@echo "  test        Run all tests"
	@echo "  lint        Run go vet"
	@echo "  clean       Remove build artifacts"
	@echo "  install     Build and install locally"
	@echo "  release     Create GoReleaser release"
```

---

## 8. go.work Sobrante y Convención de Módulo

### El problema — go.work

El archivo `go.work` en la raíz indica una intención de Go workspace multi-módulo (probablemente un vestigio de cuando cada plugin era un módulo separado). Con un solo `go.mod` activo, el `go.work` puede interferir con `go build`, `go test`, y herramientas como `gopls` y `golangci-lint`.

```
# Verificar si go.work causa conflictos:
go work sync   # si falla → el go.work está inconsistente
```

### El problema — nombre del módulo

```go
// go.mod:
module github.com/andragon31/Ragnarok  // Capital R — no convencional
```

La convención de Go establece paths de módulo en lowercase. Si bien el compilador lo acepta, herramientas como `pkg.go.dev`, `golangci-lint`, y algunos IDEs pueden comportarse de forma inconsistente.

### Cómo debe resolverse

Si el proyecto es un único módulo (lo que parece ser el caso):

```bash
# Eliminar go.work
rm go.work go.work.sum

# Si se planea tener sub-módulos en el futuro, recrearlo cuando corresponda:
# go work init ./cmd/rag ./internal/...
```

Para el nombre del módulo, evaluar si vale el costo de migración. Si el proyecto no tiene dependientes externos aún, renombrarlo ahora es barato:

```go
// go.mod — convención estándar:
module github.com/andragon31/ragnarok
```

---

## 9. Directorios de Runtime Comprometidos en el Repo

### El problema

```
Ragnarok/
├── .ragnarok/       ← datos de runtime del ecosistema
├── .skoll/
│   └── skills/
│       └── go-testing/  ← skills de Skoll (datos de ejecución)
└── .tyr/            ← datos de Tyr
```

Estos directorios son **datos de ejecución** del ecosistema — se generan cuando Ragnarok corre sobre un proyecto. Estar en la raíz del repo fuente crea ambigüedad: ¿son parte del código o son datos de una instalación de desarrollo? Si un usuario clona el repo para contribuir, su entorno de desarrollo queda mezclado con estos datos.

### Cómo debe resolverse

**Opción A (recomendada):** Moverlos a `.gitignore` si son datos generados en runtime:

```gitignore
# .gitignore — agregar:
.ragnarok/
.tyr/
.skoll/

# Excepción: si .skoll/skills/ contiene skills de ejemplo para tests
!.skoll/skills/
.skoll/skills/*/cache/
```

**Opción B:** Si `.skoll/skills/go-testing/` es un skill de ejemplo que debe distribuirse, moverlo a `examples/skills/go-testing/` o `testdata/skills/go-testing/` para que sea explícitamente dato de prueba, no dato de runtime.

---

## 10. Sin Soporte Linux/macOS

### El problema

```powershell
# install.ps1 — líneas 56-59:
if (!$IS_WINDOWS) {
    Write-Err "This installer is for Windows only"
    throw "Unsupported OS"
}
```

Ragnarok está diseñado para trabajar con Claude Code, Cursor, y Windsurf — tres tools con adoption masiva en Linux y macOS. La ausencia de un `install.sh` excluye a una porción significativa del público objetivo.

### Cómo debe resolverse

Crear `install.sh` siguiendo el mismo paradigma de binary download:

```bash
#!/usr/bin/env bash
set -euo pipefail

REPO="andragon31/Ragnarok"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detectar OS y arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo "Arquitectura no soportada: $ARCH"; exit 1 ;;
esac

# Obtener versión latest
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/')

echo "Instalando Ragnarok v$VERSION para $OS/$ARCH..."

ASSET="ragnarok_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/$ASSET"
TMP_DIR=$(mktemp -d)

# Descargar y verificar
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET"
curl -fsSL "https://github.com/$REPO/releases/download/v${VERSION}/checksums.txt" \
    | grep "$ASSET" | sha256sum --check --status

# Instalar
mkdir -p "$INSTALL_DIR"
tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR"
mv "$TMP_DIR/rag" "$INSTALL_DIR/rag"
chmod +x "$INSTALL_DIR/rag"
rm -rf "$TMP_DIR"

# Verificar PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "Agregar al PATH (añadir a ~/.bashrc o ~/.zshrc):"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "✓ Ragnarok v$VERSION instalado en $INSTALL_DIR/rag"
```

---

## 11. Plan de Acción por Prioridad

### Prioridad 1 — Bugs bloqueantes (resolver antes del próximo release)

| # | Archivo | Acción |
|---|---------|--------|
| 1 | `install_quick.ps1` | Corregir URL: `ragnarok-ecosystem` → `andragon31/Ragnarok` |
| 2 | `install.ps1` | Eliminar bloque de detección `irm\|iex` (líneas 19–28) |
| 3 | `install.ps1` | Cambiar paradigma: de `go build` a descarga de binario precompilado |

### Prioridad 2 — Release pipeline (habilita el paradigma correcto)

| # | Archivo nuevo | Acción |
|---|---------------|--------|
| 4 | `.goreleaser.yaml` | Crear con targets win/linux/darwin × amd64/arm64 |
| 5 | `.github/workflows/release.yml` | Workflow de GoReleaser en push de tag `v*` |
| 6 | `.github/workflows/ci.yml` | Workflow de build+test en push/PR |
| 7 | `install.ps1` | Integrar auto-detect de latest version via GitHub API |
| 8 | `install.sh` | Crear instalador para Linux/macOS |

### Prioridad 3 — Deuda técnica (cleanup)

| # | Archivo | Acción |
|---|---------|--------|
| 9  | `go.work` / `go.work.sum` | Evaluar y eliminar si no hay workspace multi-módulo real |
| 10 | `go.mod` | Evaluar renombrar módulo a lowercase (`ragnarok`) |
| 11 | `Makefile` | Actualizar versión en `help`, agregar `build-all`, corregir paths cross-platform |
| 12 | `.gitignore` | Agregar `.ragnarok/`, `.skoll/`, `.tyr/` |

---

## Comparativa Final: Estado Actual vs Estado Objetivo

| Aspecto | Estado Actual | Estado Objetivo |
|---------|---------------|-----------------|
| **Instalación Windows** | Requiere Go + Git, compila desde fuente | Descarga binario precompilado, cero dependencias |
| **Instalación Linux/macOS** | No soportado | `curl \| bash` con binario precompilado |
| **One-liner rápido** | URL rota (404) | URL correcta con auto-detect de versión |
| **CI/CD** | Sin workflows | build+test en PR, release en tag |
| **Binarios en Releases** | Solo código fuente | `.zip`/`.tar.gz` para 5 plataformas + checksums |
| **Versión en instalador** | Hardcodeada (`2.2.4`) | Autodetectada via GitHub API |
| **Makefile** | Versión stale (`v1.4.0`), paths mixtos | Versionado dinámico, cross-platform |

---

*Documento generado a partir del análisis del repositorio [andragon31/Ragnarok](https://github.com/andragon31/Ragnarok) comparado contra [Gentleman-Programming/engram](https://github.com/Gentleman-Programming/engram) como referencia de buenas prácticas en distribución de herramientas Go.*
