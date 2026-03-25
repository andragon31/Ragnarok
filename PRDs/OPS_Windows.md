# OPS — Ragnarok Operations Guide
## Operations Guide — Versión 1.1.0

**Producto:** Ecosistema RAGNAROK  
**Versión:** 1.1.0  
**Lenguaje:** Go 1.22+  
**Licencia:** MIT  
**Fecha:** Marzo 2026

---

## Tabla de contenidos

1. [Contexto y Alcance](#1-contexto-y-alcance)
2. [Arquitectura de Instalación](#2-arquitectura-de-instalación)
3. [Configuración en Windows](#3-configuración-en-windows)
4. [Integración con Agentes (OpenCode, Cursor, Windsurf)](#4-integración-con-agentes)
5. [Backup y Disaster Recovery](#5-backup-y-disaster-recovery)
6. [Seguridad](#6-seguridad)
7. [Performance y Límites](#7-performance-y-límites)
8. [Troubleshooting](#8-troubleshooting)
9. [Mantenimiento](#9-mantenimiento)
10. [Project Scanner y Bootstrap](#10-project-scanner-y-bootstrap)

---

## 1. Contexto y Alcance

### 1.1 Perfil de Usuario

```
┌─────────────────────────────────────────────────────────────────┐
│                    USUARIO TÍPICO RAGNAROK                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  OS:        Windows 10/11 (principal)                           │
│  Agentes:   OpenCode, Cursor, Windsurf, Claude Code, Gemini CLI │
│  Contexto:  Desarrollo individual o equipo pequeño (≤5 devs)    │
│  Proyecto:  1 proyecto activo a la vez                          │
│  Red:       Conexión internet estándar                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Lo que NO es RAGNAROK

| No es | Por qué |
|---|---|
| Sistema enterprise multi-tenant | Enfoque single-user |
| Solución cloud-hosted | Todo local en Windows |
| Plataforma de deployment | Solo desarrollo local |
| Integración con CI/CD externa | Uso del agente en desarrollo |

### 1.3 Flujo de trabajo típico

```
┌─────────────────────────────────────────────────────────────────┐
│                    FLUJO RAGNAROK EN WINDOWS                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Developer abre terminal en proyecto Windows                  │
│                    │                                              │
│                    ▼                                              │
│  2. Inicializa RAGNAROK:                                         │
│     fenrir init                                                    │
│     hati init                                                      │
│     skoll init                                                     │
│     tyr init                                                       │
│                    │                                              │
│                    ▼                                              │
│  3. Agent (OpenCode/Cursor) se conecta via MCP                   │
│                    │                                              │
│                    ▼                                              │
│  4. Agent trabaja con contexto de RAGNAROK:                       │
│     - mem_session_start → contexto del proyecto                  │
│     - hati checkpoint → approval antes de fases                   │
│     - tyr validate → seguridad/calidad                           │
│     - skoll skills → guía de implementación                      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Arquitectura de Instalación

### 2.1 Estructura de directorios

```
%USERPROFILE%/
├── .fenrir/                    # Fenrir (Memory & Knowledge)
│   ├── fenrir.db              # SQLite knowledge graph
│   ├── fenrir.db-wal         # WAL log
│   ├── fenrir.db-shm         # Shared memory
│   ├── config.json           # Configuración
│   ├── standards.json        # Standards del proyecto
│   └── skills/               # Skills instalados
│
├── .hati/                     # Hati (Planning & Approvals)
│   ├── hati.db               # SQLite plans y approvals
│   ├── hati.db-wal
│   ├── hati.db-shm
│   └── config.json
│
├── .skoll/                    # Skoll (RSAW & Skills)
│   ├── skoll.db              # SQLite skills y rules
│   ├── skoll.db-wal
│   ├── skoll.db-shm
│   ├── config.json
│   ├── skills/               # Skills descargados
│   └── rules/                # Rules activas
│
├── .tyr/                      # Tyr (Security & Validation)
│   ├── tyr.db                # SQLite cache y findings
│   ├── tyr.db-wal
│   ├── tyr.db-shm
│   ├── config.json
│   ├── policies.json         # Policies de seguridad
│   ├── semgrep-rules/        # Rules SAST custom
│   └── cache/                # Cache de paquetes
│
└── .rag/
    └── rag.db                # SQLite unificado (futuro)
```

### 2.2 Puertos MCP

| Plugin | Puerto default | Protocolo |
|---|---|---|
| Fenrir | 7438 | stdio / TCP |
| Hati | 7439 | stdio / TCP |
| Skoll | 7441 | stdio / TCP |
| Tyr | 7440 | stdio / TCP |

### 2.3 Configuración MCP en cliente Windows

```json
// %USERPROFILE%/.claude-agents/mcp_config.json
{
  "mcpServers": {
    "fenrir": {
      "command": "C:\\Users\\<user>\\.local\\bin\\fenrir.exe",
      "args": ["serve", "--port", "7438"],
      "env": {
        "FENRIR_DIR": "C:\\Users\\<user>\\.fenrir"
      }
    },
    "hati": {
      "command": "C:\\Users\\<user>\\.local\\bin\\hati.exe",
      "args": ["serve", "--port", "7439"],
      "env": {
        "HATI_DIR": "C:\\Users\\<user>\\.hati"
      }
    },
    "skoll": {
      "command": "C:\\Users\\<user>\\.local\\bin\\skoll.exe",
      "args": ["serve", "--port", "7441"],
      "env": {
        "SKOLL_DIR": "C:\\Users\\<user>\\.skoll"
      }
    },
    "tyr": {
      "command": "C:\\Users\\<user>\\.local\\bin\\tyr.exe",
      "args": ["serve", "--port", "7440"],
      "env": {
        "TYR_DIR": "C:\\Users\\<user>\\.tyr"
      }
    }
  }
}
```

---

## 3. Configuración en Windows

### 3.1 Requisitos del sistema

| Recurso | Mínimo | Recomendado |
|---|---|---|
| RAM | 8 GB | 16 GB |
| CPU | 4 cores | 8 cores |
| Disco | 1 GB libre | 5 GB libre |
| Windows | 10 64-bit | 11 64-bit |
| Git | 2.30+ | Latest |

### 3.2 Instalación con scoop (Windows package manager)

```powershell
# Agregar buckets
scoop bucket add raguen https://github.com/ragnarok-ecosystem/scoop-bucket

# Instalar plugins
scoop install fenrir hati skoll tyr

# Verificar instalación
fenrir version
hati version
skoll version
tyr version
```

### 3.3 Instalación manual

```powershell
# Descargar releases desde GitHub
$env:USERPROFILE = $env:USERPROFILE
Invoke-WebRequest -Uri "https://github.com/ragnarok-ecosystem/fenrir/releases/latest/download/fenrir_windows_amd64.zip" -OutFile "$env:TEMP\fenrir.zip"
Expand-Archive -Path "$env:TEMP\fenrir.zip" -DestinationPath "$env:USERPROFILE\.local\bin"

# Agregar al PATH
[Environment]::SetEnvironmentVariable("PATH", "$env:PATH;$env:USERPROFILE\.local\bin", "User")
```

### 3.4 Inicialización del proyecto

```powershell
# En el directorio del proyecto
cd C:\Projects\MiProyecto

# Inicializar cada plugin
fenrir init --project "MiProyecto"
hati init --project "MiProyecto"
skoll init --project "MiProyecto"
tyr init --project "MiProyecto"

# Verificar estado
fenrir status
hati status
skoll status
tyr status
```

### 3.5 Variables de entorno

```powershell
# User-level (permanente)
[Environment]::SetEnvironmentVariable("FENRIR_DIR", "$env:USERPROFILE\.fenrir", "User")
[Environment]::SetEnvironmentVariable("HATI_DIR", "$env:USERPROFILE\.hati", "User")
[Environment]::SetEnvironmentVariable("SKOLL_DIR", "$env:USERPROFILE\.skoll", "User")
[Environment]::SetEnvironmentVariable("TYR_DIR", "$env:USERPROFILE\.tyr", "User")

# Project-level (en .env del proyecto)
FENRIR_PROJECT=MiProyecto
HATI_AUTO_CHECKPOINT=true
SKOLL_ALLOWED_TOOLS=read,edit,write,bash,grep,glob
TYR_STRICT_MODE=false
```

---

## 4. Integración con Agentes

### 4.1 OpenCode

```json
// En OpenCode config (opencode.json)
{
  "mcp": {
    "servers": {
      "fenrir": {
        "command": "fenrir",
        "args": ["mcp"],
        "env": {
          "FENRIR_PROJECT": "${workspace.name}"
        }
      },
      "hati": {
        "command": "hati",
        "args": ["mcp"]
      },
      "skoll": {
        "command": "skoll",
        "args": ["mcp"]
      },
      "tyr": {
        "command": "tyr",
        "args": ["mcp"]
      }
    }
  },
  "rules": {
    "pre_agent_run": ["tyr.inject_guard"],
    "post_agent_run": ["tyr.sanitize"],
    "on_plan_approve": ["hati.checkpoint_open"]
  }
}
```

### 4.2 Cursor

```json
// .cursor/mcp.json
{
  "mcpServers": {
    "fenrir": {
      "command": "fenrir",
      "args": ["serve", "--stdio"]
    },
    "hati": {
      "command": "hati", 
      "args": ["serve", "--stdio"]
    },
    "skoll": {
      "command": "skoll",
      "args": ["serve", "--stdio"]
    },
    "tyr": {
      "command": "tyr",
      "args": ["serve", "--stdio"]
    }
  }
}
```

### 4.3 Windsurf

```yaml
# .windsurf/mcp.yaml
mcp_servers:
  fenrir:
    command: fenrir
    args: [serve, --stdio]
    
  hati:
    command: hati
    args: [serve, --stdio]
    
  skoll:
    command: skoll
    args: [serve, --stdio]
    
  tyr:
    command: tyr
    args: [serve, --stdio]
```

### 4.4 Claude Code

```json
// claude_desktop_config.json (en %APPDATA%)
{
  "mcpServers": {
    "fenrir": {
      "command": "fenrir",
      "args": ["serve", "--port", "7438"]
    },
    "hati": {
      "command": "hati",
      "args": ["serve", "--port", "7439"]
    },
    "skoll": {
      "command": "skoll", 
      "args": ["serve", "--port", "7441"]
    },
    "tyr": {
      "command": "tyr",
      "args": ["serve", "--port", "7440"]
    }
  }
}
```

### 4.5 Gemini CLI

```yaml
# gemini.yaml
mcp:
  servers:
    - name: fenrir
      command: fenrir
      args: [mcp]
    - name: hati
      command: hati
      args: [mcp]
    - name: skoll
      command: skoll
      args: [mcp]
    - name: tyr
      command: tyr
      args: [mcp]
```

---

## 5. Backup y Disaster Recovery

### 5.1 Estrategia de Backup

```
┌─────────────────────────────────────────────────────────────────┐
│                    BACKUP STRATEGY                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  LOCALLY (siempre):                                              │
│  ├── Git commit automático del knowledge graph                   │
│  ├── SQLite WAL para recoverability                              │
│  └── Cache en disco                                              │
│                                                                  │
│  PERIODIC (recomendado):                                         │
│  ├── Daily: Export a JSON                                         │
│  ├── Weekly: Backup a OneDrive/Google Drive                     │
│  └── Monthly: Backup a disco externo                             │
│                                                                  │
│  CRITICAL (projectos importantes):                               │
│  └── Real-time sync a GitHub private repo                        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Backup automático al cerrar sesión

```go
// internal/backup/auto_backup.go

type AutoBackup struct {
    backupDir string
    gitRepo   *git.Repository
    config    *BackupConfig
}

func (b *AutoBackup) OnSessionEnd(session *Session) error {
    // 1. Exportar estado a JSON
    stateFile := filepath.Join(b.backupDir, fmt.Sprintf("backup_%s.json", session.ID))
    if err := b.exportState(session, stateFile); err != nil {
        return err
    }
    
    // 2. Commit a git si hay cambios
    if b.gitRepo.HasChanges() {
        _, err := b.gitRepo.Commit(&git.CommitOptions{
            Message: fmt.Sprintf("Session backup: %s", session.ID),
        })
        if err != nil {
            log.Warn("Git commit failed: %v", err)
        }
    }
    
    return nil
}
```

### 5.3 Script de backup para Windows (Tarea Programada)

```powershell
# backup_ragnarok.ps1
# Crear tarea programada: schtasks /create /sc daily /tn "Ragnarok Backup" /tr "powershell -File backup_ragnarok.ps1"

$BackupDir = "$env:USERPROFILE\OneDrive\RagnarokBackups"
$Date = Get-Date -Format "yyyy-MM-dd_HHmmss"

# Crear directorio si no existe
if (!(Test-Path $BackupDir)) {
    New-Item -ItemType Directory -Path $BackupDir | Out-Null
}

# Backup de cada plugin
$Plugins = @("fenrir", "hati", "skoll", "tyr")
foreach ($plugin in $Plugins) {
    $pluginDir = "$env:USERPROFILE\.$plugin"
    $backupFile = "$BackupDir\$plugin`_$Date.zip"
    
    if (Test-Path $pluginDir) {
        Compress-Archive -Path $pluginDir -DestinationPath $backupFile -Force
        Write-Host "Backed up $plugin to $backupFile"
    }
}

# Limpiar backups antiguos (> 30 días)
Get-ChildItem $BackupDir -Filter "*.zip" | 
    Where-Object { $_.LastWriteTime -lt (Get-Date).AddDays(-30) } |
    Remove-Item -Force

Write-Host "Backup completed at $(Get-Date)"
```

### 5.4 Restauración

```powershell
# restore_ragnarok.ps1

param(
    [Parameter(Mandatory=$true)]
    [string]$BackupFile,
    
    [Parameter(Mandatory=$true)]
    [ValidateSet("fenrir", "hati", "skoll", "tyr")]
    [string]$Plugin
)

$pluginDir = "$env:USERPROFILE\.$Plugin"

# Backup actual antes de restaurar
$tempBackup = "$env:TEMP\$Plugin`_restore_backup_$(Get-Date -Format 'yyyyMMdd_HHmmss').zip"
if (Test-Path $pluginDir) {
    Compress-Archive -Path $pluginDir -DestinationPath $tempBackup -Force
    Write-Host "Current state backed up to $tempBackup"
}

# Restaurar
Expand-Archive -Path $BackupFile -DestinationPath $pluginDir -Force
Write-Host "Restored $Plugin from $BackupFile"
Write-Host "Please restart the MCP server for changes to take effect."
```

### 5.5 Recovery desde Git

```powershell
# Si el knowledge graph se corrompe
cd $env:USERPROFILE\.fenrir

# Ver historial
git log --oneline -10

# Restaurar versión anterior
git checkout abc1234 -- fenrir.db

# O exportar desde commit específico
git show abc1234:fenrir.db > fenrir_restored.db
```

---

## 6. Seguridad

### 6.1 Modelo de amenazas

| Amenaza | Probabilidad | Impacto | Mitigación |
|---|---|---|---|
| Prompt injection via archivos | 🟡 MEDIA | 🔴 ALTO | `tyr.inject_guard` + `tyr.proactive_scan` |
| Secrets en código | 🟡 MEDIA | 🔴 ALTO | `tyr.sanitize` + `tyr.secret_scan` |
| Skills maliciosos | 🟢 BAJA | 🔴 ALTO | `skoll` security scan antes de import |
| Acceso no autorizado a datos | 🟢 BAJA | 🔴 ALTO | RBAC básico + permisos NTFS |
| Corruption de SQLite | 🟢 BAJA | 🟡 MEDIO | WAL mode + backup automático |

### 6.2 Permisos NTFS (Windows)

```powershell
# Solo el usuario actual puede acceder a sus datos
icacls "$env:USERPROFILE\.fenrir" /inheritance:r /grant:r "%USERNAME%:(OI)(CI)F"
icacls "$env:USERPROFILE\.hati" /inheritance:r /grant:r "%USERNAME%:(OI)(CI)F"
icacls "$env:USERPROFILE\.skoll" /inheritance:r /grant:r "%USERNAME%:(OI)(CI)F"
icacls "$env:USERPROFILE\.tyr" /inheritance:r /grant:r "%USERNAME%:(OI)(CI)F"

# Verificar
icacls "$env:USERPROFILE\.fenrir"
```

### 6.3 Plugins de seguridad

```powershell
# Verificar que los plugins están usando las protecciones
tyr stats

# Debería mostrar:
# - inject_guard: enabled
# - secret_scan: enabled  
# - proactive_scan: enabled

# Escanear un skill antes de instalar
skoll skills scan C:\path\to\skill
```

---

## 7. Performance y Límites

### 7.1 Benchmarks típicos (Windows)

| Operación | Latencia típica | Latencia P99 |
|---|---|---|
| `mem_find` (FTS) | < 50ms | < 200ms |
| `mem_session_start` | < 500ms | < 2s |
| `tyr.pkg_check` (cached) | < 5ms | < 50ms |
| `tyr.pkg_check` (network) | < 500ms | < 3s |
| `hati.checkpoint_open` | < 100ms | < 500ms |
| `skoll.skill_load` | < 100ms | < 500ms |

### 7.2 Límites recomendados

| Recurso | Límite soft | Límite hard |
|---|---|---|
| Sessions concurrentes | 5 | 10 |
| Tamaño del knowledge graph | 100k nodos | 500k nodos |
| Tamaño de archivo a escanear | 10 MB | 100 MB |
| Skills instalados | 50 | 200 |
| Memória por plugin | 512 MB | 1 GB |

### 7.3 Monitoreo de performance

```powershell
# Stats de cada plugin
fenrir stats
hati stats
skoll stats
tyr stats

# Debería mostrar:
# - Nodes/edges count
# - Query latency
# - Cache hit rate
# - Memory usage
```

---

## 8. Troubleshooting

### 8.1 Problemas comunes

| Problema | Causa | Solución |
|---|---|---|
| MCP connection failed | Puerto en uso | `netstat -ano | findstr 7438` → matar proceso |
| SQLite locked | WAL en uso | Esperar o reiniciar servidor |
| Memory exceeded | Graph muy grande | `fenrir compact` para comprimir |
| Cache llena | Muchos pkg_check | `tyr cache clear` |
| Skill no carga | Falta dependencias | `skoll skills install-dependencies` |

### 8.2 Logs

```powershell
# Ver logs en tiempo real
fenrir serve --log-level debug 2>&1 | Out-File fenrir.log

# Logs en directorio del plugin
Get-ChildItem "$env:USERPROFILE\.fenrir\logs"
Get-ChildItem "$env:USERPROFILE\.tyr\logs"
```

### 8.3 Reset

```powershell
# Reset de un plugin (mantiene datos)
fenrir reset --soft

# Reset completo (borra datos -谨慎!)
fenrir reset --hard

# Re-inicializar proyecto
Remove-Item "$env:USERPROFILE\.fenrir\fenrir.db"
fenrir init
```

### 8.4 Verificar integridad

```powershell
# Verificar databases
sqlite3 "$env:USERPROFILE\.fenrir\fenrir.db" "PRAGMA integrity_check;"
sqlite3 "$env:USERPROFILE\.hati\hati.db" "PRAGMA integrity_check;"
sqlite3 "$env:USERPROFILE\.skoll\skoll.db" "PRAGMA integrity_check;"
sqlite3 "$env:USERPROFILE\.tyr\tyr.db" "PRAGMA integrity_check;"
```

---

## 9. Mantenimiento

### 9.1 Tareas periódicas

| Frecuencia | Tarea | Comando |
|---|---|---|
| Diaria | Backup automático | Tarea programada (ver 5.3) |
| Semanal | Limpiar cache | `tyr cache clear` |
| Mensual | Compactar DB | `fenrir db compact` |
| Mensual | Verificar integridad | `sqlite3 fenrir.db "PRAGMA integrity_check;"` |
| Trimestral | Update plugins | `scoop update fenrir hati skoll tyr` |

### 9.2 Upgrade

```powershell
# Actualizar con scoop
scoop update fenrir hati skoll tyr

# O manual
Invoke-WebRequest -Uri "https://github.com/ragnarok-ecosystem/fenrir/releases/latest/download/fenrir_windows_amd64.zip" -OutFile "$env:TEMP\fenrir.zip"
Stop-Process -Name fenrir -Force -ErrorAction SilentlyContinue
Expand-Archive -Path "$env:TEMP\fenrir.zip" -DestinationPath "$env:USERPROFILE\.local\bin" -Force
Start-Process fenrir
```

### 9.3 Desinstalación

```powershell
# Con scoop
scoop uninstall fenrir hati skoll tyr

# Con scoop buckets extras
scoop bucket rm raguen

# Manual - eliminar datos (opcional)
Remove-Item -Recurse -Force "$env:USERPROFILE\.fenrir"
Remove-Item -Recurse -Force "$env:USERPROFILE\.hati"
Remove-Item -Recurse -Force "$env:USERPROFILE\.skoll"
Remove-Item -Recurse -Force "$env:USERPROFILE\.tyr"
Remove-Item -Force "$env:USERPROFILE\.local\bin\fenrir.exe"
Remove-Item -Force "$env:USERPROFILE\.local\bin\hati.exe"
Remove-Item -Force "$env:USERPROFILE\.local\bin\skoll.exe"
Remove-Item -Force "$env:USERPROFILE\.local\bin\tyr.exe"
```

---

## 10. Project Scanner y Bootstrap

### 10.1 Overview

Ragnarok v1.1.0 incluye un **Project Scanner** que analiza la estructura de un proyecto y detecta:
- Lenguaje de programación y framework
- Gestor de paquetes
- Arquitectura (monolith, modular, monorepo)
- Módulos y dependencias
- Patrones (testing, CI/CD, Docker)
- Archivos de configuración relevantes

El **Bootstrap** genera automáticamente archivos de configuración para agentes AI:
- `.ragnarok/skills.json` - Skills sugeridos para el stack detectado
- `.ragnarok/rules.json` - Reglas de proyecto específicas
- `.ragnarok/standards.json` - Estándares de calidad (test, lint)

### 10.2 Comandos

```powershell
# Escanear un proyecto (análisis completo)
fenrir scan --path ./mi-proyecto

# Escanear solo el stack tecnológico
fenrir scan --path ./mi-proyecto --layer stack

# Escanear solo la arquitectura
fenrir scan --path ./mi-proyecto --layer arch

# Escanear patrones (testing, CI/CD, Docker)
fenrir scan --path ./mi-proyecto --layer patterns

# Generar estructura agentic (.ragnarok/*.json)
fenrir bootstrap --path ./mi-proyecto

# Generar AGENTS.md con guidelines del proyecto
fenrir init --project "Mi Proyecto"

# Ver integración de datos bootstrap
rag integrate --path ./mi-proyecto
```

### 10.3 Capas de Análisis

| Capa | Detecta |
|------|---------|
| `stack` | Language, Framework, Package Manager, Runtime |
| `arch` | Architecture Type, Modules, API, Frontend |
| `config` | Config Files (package.json, go.mod, etc.) |
| `modules` | Project modules y dependencias |
| `patterns` | Testing, CI/CD, Docker, TypeScript |

### 10.4 Integración con Plugins

Después de ejecutar `fenrir bootstrap`, los datos se pueden integrar en los plugins:

```powershell
# Ver resumen de datos bootstrap
rag integrate --path ./mi-proyecto

# Importar a Skoll (skills y rules) via MCP
# Usa: skoll.bootstrap_import con project_path

# Importar a Tyr (standards) via MCP
# Usa: tyr.bootstrap_import con project_path
```

### 10.5 AGENTS.md

`fenrir init` genera un `AGENTS.md` con:
- Stack del proyecto (language, framework)
- Módulos detectados
- Reglas del proyecto
- Estándares de calidad
- Skills sugeridos
- Comandos comunes (test, lint, build)

### 10.6 Estructura .ragnarok

```
mi-proyecto/
├── .ragnarok/
│   ├── skills.json      # Skills sugeridos (para Skoll)
│   ├── rules.json       # Reglas del proyecto (para Skoll)
│   └── standards.json   # Estándares de calidad (para Tyr)
└── AGENTS.md            # Guidelines para agentes AI
```

### 10.7 Tecnologías Detectadas

| Lenguaje | Frameworks | Package Manager |
|----------|------------|----------------|
| Go | - | go |
| JavaScript/TypeScript | Next.js, Nuxt, React, Vue, Express | npm |
| Python | - | pip |
| Rust | - | cargo |
| Java | - | maven, gradle |

---

*OPS Guide v1.0.0 — Marzo 2026*
*Enfoque: Windows single-user con agentes (OpenCode, Cursor, Windsurf, Claude Code)*
