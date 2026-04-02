# Ragnarok Ecosystem v3.1.0

**AI Governance & Autonomous Development Ecosystem**

Sistema agentico de 4 módulos MCP diseñados para orquestar agentes AI en proyectos de desarrollo, con **Agent-Based Orchestration** y validación humana en puntos clave.

---

## Quick Start (Simplified Commands)

```bash
# 1. Crear nuevo proyecto (RECOMENDADO)
rag new --project myapi --path ./myapi --stack=go

# 2. Continuar proyecto existente
rag continue --plan <plan_id>

# 3. Nueva feature en proyecto
rag feature --name user-auth --plan <plan_id>

# 4. Revisión de calidad
rag review --plan <plan_id>

# 5. Ver estado
rag status --plan <plan_id>
```

---

## Arquitectura

### Flujo Principal: HATI → SKOLL → FENRIR → TYR

```mermaid
graph LR
    PRD([📄 PRD]) --> HATI
    
    subgraph HATI["📋 HATI - Planning"]
        Plan[Plan de Desarrollo]
        Phases[Fases]
        Tasks[Tareas]
        MultiAgent[Multi-Agent Tasks]
        HumanReview[👤 Human-in-the-Loop]
        Checkpoints[Checkpoints]
        Notifications[Notificaciones]
    end
    
    HATI --> |"Tareas Multi-Agente"| SKOLL
    
    subgraph SKOLL["⚙️ SKOLL - Orchestration"]
        Agents[Agentes]
        TaskExec[Task Executions]
        Skills[Skills]
        Rules[Rules]
        Teams[Teams]
    end
    
    SKOLL --> |"Memoria"| FENRIR
    FENRIR --> |"Contexto"| SKOLL
    
    subgraph FENRIR["🧠 FENRIR - Memory"]
        Context[Contexto]
        Decisions[Decisiones]
        Timeline[Línea de Tiempo]
        Specs[Specs]
    end
    
    FENRIR --> TYR
    
    subgraph TYR["🛡️ TYR - Quality"]
        Standards[Standards]
        SAST[SAST]
        Precommit[Pre-commit]
        PkgVet[Package Vetting]
    end
    
    TYR --> |"Validación"| HumanReview
    HumanReview --> |"Approval"| HATI
```

---

## Módulos

### 📋 HATI - Planning Layer
Gestión de planes de desarrollo, fases, tareas y validaciones humanas.

**Funcionalidades:**
- Creación y seguimiento de planes de desarrollo
- Fases con estados (pending, in_progress, completed, blocked)
- Tareas con soporte multi-agente (múltiples agentes por tarea)
- Checkpoints con approval humano
- Human-in-the-loop con múltiples tipos de approval
- Notificaciones push/pull

**Tablas principales:** `plans`, `phases`, `tasks`, `task_agents`, `checkpoints`, `human_reviews`, `notifications`

### Funciones: `plan_create`, `plan_get`, `plan_list`, `plan_complete`, `plan_abandon`, `plan_resume`, `plan_revise`, `plan_blockers`, `plan_dependencies`, `phase_create`, `phase_update`, `task_create`, `task_get`, `task_get_next`, `task_update`, `task_list`, `checkpoint_open`, `checkpoint_approve`, `human_review_create`, `human_review_decide`, `human_review_pending`, `notification_send`, `notification_list`, `spec_impact`, `quality_snapshot`, `prd_parse`, `prd_requirements_extract`

---

### ⚙️ SKOLL - Orchestration Layer
Orquestación de agentes, skills y ejecución de tarea.

**Funcionalidades:**
- Registro y tracking de agentes
- Skills basados en filesystem con metadata en SQLite
- Rules engine para validación
- Agent heartbeat y tracking
- Team management
- Skill matching automático por agent type

**Tablas principales:** `agents`, `skills`, `rules`, `task_executions`, `teams`

### Funciones: `skill_list`, `skill_load`, `skill_search`, `skill_verify`, `skill_version_check`, `skill_read_file`, `skills_import`, `skills_update`, `agent_list`, `agent_create`, `agent_get`, `agent_activate`, `agent_context`, `agent_handoff`, `agent_specialized_list`, `agent_assign_task`, `agent_complete_task`, `agent_heartbeat`, `agent_skills_get`, `team_create`, `team_get`, `rule_list`, `rule_check`, `rule_get`, `skoll_status`, `skoll_validate`, `bootstrap_import`

---

### 🧠 FENRIR - Memory Layer
Memoria institucional y contexto para agentes.

**Funcionalidades:**
- Observations con FTS5 full-text search
- Graph-based context search
- Sessions con tracking de actividad
- Specs con delta history
- Project scanning y bootstrap
- Memory deduplication y TTL

**Tablas principales:** `observations`, `sessions`, `specs`, `nodes`, `edges`

### Funciones: `mem_save`, `mem_find`, `mem_context`, `mem_timeline`, `mem_stats`, `mem_session_start`, `mem_session_end`, `mem_save_prompt`, `mem_session_checkpoint`, `mem_get_observation`, `spec_save`, `spec_list`, `spec_delta`, `spec_impact`, `spec_check`, `project_scan`, `project_bootstrap`, `skill_generate`, `rules_generate`, `standards_generate`, `prompt_analyze`, `agents_md_get`

---

### 🛡️ TYR - Quality Layer
Validación de código, seguridad y estándares.

**Funcionalidades:**
- SAST scanner con rules engine
- Package vetting (npm, pypi, go, cargo, nuget, maven, rubygems, packagist)
- CVE/GitHub Advisories integration
- Standards execution con pass rate tracking
- Pre-commit validation

**Tablas principales:** `sast_findings`, `pkg_cache`, `standards`, `standards_results`

### Funciones: `pkg_check`, `pkg_license`, `pkg_audit`, `pkg_audit_snapshot`, `pkg_audit_continuous`, `sast_run`, `sast_findings`, `sast_resolve`, `standard_list`, `standard_run`, `standard_run_all`, `precommit_validate`, `precommit_autofix`, `bootstrap_import`, `quality_snapshot`

---

## Workflows de Alto Nivel

En lugar de múltiples llamadas MCP, Ragnarok ofrece **workflows** que ejecutan todo internamente:

### 1. `rag new` (CLI) → `workflow_stack_based_init` ⭐ RECOMENDADO
Inicializa proyecto detectando stack automáticamente y creando fases/tareas apropiadas.

```bash
rag new --project myapi --path ./myapi --stack=go
```

**Ejecuta internamente:**
- `project_scan` → Detecta stack (Go, Node, Python, etc.), arquitectura, CI/CD
- `plan_create` → Crea plan basado en el stack detectado
- `phase_create` → Crea fases según stack
- `task_create` → Crea tareas específicas del stack
- `human_review_create` → Solicita approval humano

### 2. `rag continue` (CLI) → `workflow_plan_develop_v2` ⭐ RECOMENDADO
Ejecuta el desarrollo con delegación multi-agente.

```bash
rag continue --plan <plan_id>
```

**Flujo autónomo:**
```
while (tareas_pendientes) {
    task = task_get_next(plan_id, agent_id)
    if (task.tiene_agentes) {
        task_execute(task_id, agente.id)
    } else {
        task_update(status: "in_progress")
    }
    
    if (is_milestone) {
        checkpoint_create
        human_review_create
    }
}
```

### 3. `rag review` (CLI) → `workflow_checkpoint_create`
Crea checkpoint de calidad con validaciones.

```bash
rag review --plan <plan_id>
```

**Ejecuta:**
- `checkpoint_open`
- `standard_run_all`
- `sast_run`
- `precommit_validate`
- `human_review_create`

### 4. `rag status` (CLI)
Muestra estado del ecosistema y plan.

```bash
rag status --plan <plan_id>
```

### 5. `rag feature` (CLI)
Crea nueva feature en un plan existente.

```bash
rag feature --name user-auth --plan <plan_id>
```

---

## Human-in-the-Loop

Puntos donde se requiere validación humana:

| Punto | Tipo | Descripción |
|-------|------|-------------|
| Post PRD | `prd_approval` | "¿Aprobar este plan?" |
| Post Milestone | `checkpoint_approval` | "¿Aprobar checkpoint?" |
| On Blocker | `blocker_resolution` | "¿Cómo resolver este blocker?" |
| Pre Deploy | `deploy_approval` | "¿Desplegar a producción?" |

---

## Agentes Especializados (SKOLL)

| Agente | Tipo | Skills | Ejecuta |
|--------|------|--------|---------|
| `backend-agent` | backend | go, python, api, db | endpoints, database |
| `frontend-agent` | frontend | react, vue, typescript | UI, components |
| `qa-agent` | qa | testing, jest, cypress | tests, e2e |
| `devops-agent` | devops | docker, k8s, ci/cd | deploy, infra |
| `security-agent` | security | sast, audit | security checks |
| `docs-agent` | docs | markdown, api-docs | documentation |

---

## Estructura de Datos

### PRD → Plan → Phase → Task → TaskAgent

```mermaid
graph TD
    PRD[📄 PRD] --> Plan[📋 Plan]
    Plan --> Phase1[Phase: Backend]
    Plan --> Phase2[Phase: Frontend]
    Plan --> Phase3[Phase: Testing]
    
    Phase1 --> Task1[Task: API Users]
    Phase1 --> Task2[Task: Auth]
    Phase2 --> Task3[Task: Login UI]
    
    Task1 --> TA1[TaskAgent: backend-agent]
    Task1 --> TA2[TaskAgent: qa-agent]
    Task2 --> TA3[TaskAgent: backend-agent]
    
    Task2 -.-> Blocker[🚧 Blocker]
```

---

## Instalación

### Windows

```powershell
# Instalación rápida (detecta la última versión automáticamente)
irm https://raw.githubusercontent.com/andragon31/Ragnarok/main/install_quick.ps1 | iex
```

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/andragon31/Ragnarok/main/install.sh | bash
```

---

## IDE Setup

Ragnarok funciona con cualquier cliente MCP. Después de instalar, ejecuta:

```bash
rag setup all   # Detecta y configura todos los IDEs instalados
```

O configura manualmente:

### OpenCode
Agrega a `~/.config/opencode/opencode.json`:
```json
{ "mcp": { "ragnarok": { "type": "local", "command": ["rag", "mcp"], "enabled": true } } }
```

### Claude Code
Agrega a `~/.claude/settings.json`:
```json
{ "mcpServers": { "ragnarok": { "command": "rag", "args": ["mcp"] } } }
```

### Cursor
Agrega a `.cursor/mcp.json` en tu proyecto:
```json
{ "mcpServers": { "ragnarok": { "command": "rag", "args": ["mcp"] } } }
```

### Windsurf
Agrega a `~/.windsurf/mcp.json`:
```json
{ "mcpServers": { "ragnarok": { "command": "rag", "args": ["mcp"] } } }
```

### Gemini CLI
Agrega a `~/.gemini/settings.json`:
```json
{ "mcpServers": { "ragnarok": { "command": "rag", "args": ["mcp"] } } }
```

---

## Uso Rápido

```bash
# 1. Crear nuevo proyecto (RECOMENDADO)
rag new --project myapi --path ./myapi --stack=go

# 2. Continuar proyecto existente
rag continue --plan <plan_id>

# 3. Nueva feature en proyecto
rag feature --name user-auth --plan <plan_id>

# 4. Revisión de calidad
rag review --plan <plan_id>

# 5. Ver estado
rag status --plan <plan_id>

# 6. Inicializar plugins (primera vez)
rag init --project mi-proyecto

# 7. Escanear proyecto existente
rag scan --path ./mi-proyecto

# 8. Iniciar servidor MCP
rag serve
```

---

## Changelog

### v3.1.0 (Latest)
- CI/CD pipeline con GitHub Actions (`ci.yml`, `release.yml`)
- Release automation con GoReleaser (cross-platform binaries)
- Instalador Linux/macOS (`install.sh`)
- `verify_install.ps1` reescrito para arquitectura unificada
- `rag setup claude`, `rag setup cursor`, `rag setup windsurf`, `rag setup gemini`
- `CHANGELOG.md` y `CONTRIBUTING.md` agregados
- `go.work` eliminado (proyecto single-module)

### v2.2.x
- Arquitectura unificada: único binario `rag` expone todos los módulos via MCP
- `rag setup opencode/cursor/windsurf/claude/gemini`
- Corrección URL `install_quick.ps1` (apuntaba a org inexistente)
- `opencode.json` usando `rag` desde PATH (sin path hardcodeado)
- Runtime dirs eliminados del tracking git (`.ragnarok/`, `.skoll/`, `.tyr/`)

### v2.1.0
- Nuevo CLI unificado: `rag new`, `rag continue`, `rag feature`, `rag review`, `rag status`
- Métodos `ExecuteWorkflow` y `CallTool` en unified server
- 44 funciones MCP consolidadas en sistema de workflows

### v2.0.x
- Arquitectura multi-módulo (4 módulos SQLite independientes)
- Fix multi-digit phase numbers bug
- Thread-safety en generateID
- `standard_run_all` implementado (era stub)
- Schema validation tests

[Ver CHANGELOG completo](./CHANGELOG.md)
