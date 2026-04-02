# Ragnarok — Agent Guidelines

> v3.1.0 | Actualizar con cada cambio arquitectónico significativo

## Propósito del Proyecto

Ragnarok es un ecosistema MCP de 4 módulos para orquestar agentes IA en desarrollo de software.
Un único binario (`rag`) expone todos los módulos via MCP stdio.

## Estructura de Módulos

```
cmd/rag/           # CLI + unified MCP entrypoint
internal/
  hati/            # Planning: plans, phases, tasks, human-in-the-loop
  fenrir/          # Memory: FTS5, sessions, specs, graph context
  skoll/           # Orchestration: agents, skills, rules, teams
  tyr/             # Quality: SAST, pkg vetting, standards, pre-commit
  mcp/             # Unified MCP server (dispatches to the 4 modules)
```

## Base de Datos

Cada módulo tiene su propia base SQLite en `~/.ragnarok/`:
- `hati.db`   — Planning (plans, phases, tasks, checkpoints)
- `fenrir.db`  — Memory (observations, sessions, specs, graph)
- `skoll.db`   — Orchestration (agents, skills, rules, teams)
- `tyr.db`     — Quality (sast_findings, pkg_cache, standards)

## Convenciones de Código

### Handlers MCP
Todos los handlers MCP siguen la firma:
```go
func (h *Handler) HandleXxx(ctx context.Context, args map[string]any) (map[string]any, error)
```

### IDs Thread-Safe
IDs se generan con `generateID()` — thread-safe via `sync.Mutex`.

### Database Setup
Todas las DBs inicializan con:
```go
PRAGMA foreign_keys = ON
PRAGMA journal_mode = WAL
```

### Timeout
Handlers tienen timeout de 30s via context.

## Flujo de Trabajo Principal

```
PRD.md → rag new --project NAME --path ./path --stack=go → plan_id
plan_id → rag continue → ciclo de desarrollo
ciclo → rag review → checkpoint + validación humana
```

## Comandos Principales

```bash
rag new --project NAME --path ./path --stack=STACK   # Crear proyecto desde stack
rag continue --plan ID                              # Reanudar desarrollo
rag feature --name NAME --plan ID                   # Nueva feature
rag review --plan ID                               # Checkpoint de calidad
rag status --plan ID                                # Estado del plan
rag scan --path ./project [--bootstrap]              # Escanear proyecto
rag serve / rag mcp                                 # Iniciar servidor MCP
```

## Comandos de Diagnóstico

```bash
rag doctor          # Health check completo
rag version         # Versión de todos los módulos
rag setup opencode  # Configurar OpenCode MCP
rag setup cursor    # Configurar Cursor MCP
rag setup windsurf  # Configurar Windsurf MCP
rag setup claude    # Configurar Claude Code MCP
rag setup gemini    # Configurar Gemini CLI MCP
```

## Configuración de IDEs

### OpenCode
```json
{ "mcp": { "ragnarok": { "type": "local", "command": ["rag", "mcp"], "enabled": true } } }
```

### Claude Code, Cursor, Windsurf
```json
{ "mcpServers": { "ragnarok": { "command": "rag", "args": ["mcp"] } } }
```

## Workflow de Inicio de Proyecto (PRD-Driven)

Para inicializar un proyecto desde cero con un PRD, los agentes DEBEN utilizar el workflow integrado. Esto configura automáticamente los 4 módulos.

**Herramienta:** `workflow_project_lifecycle`
**Parámetros:**
- `project_path`: Ruta raíz del proyecto.
- `prd_file`: Ruta al archivo PRD.
- `title`: (Opcional) Título del proyecto.

**Efectos:**
1. **Analiza** el stack técnico y arquitectura (Fenrir).
2. **Parsea** requerimientos del PRD (Hati).
3. **Crea** agentes especialistas en Skoll y forma un equipo.
4. **Genera** un plan de desarrollo con fases y tareas (Hati).
5. **Asigna** automáticamente los agentes a las tareas según su rol.
6. **Ejecuta** un escaneo de seguridad inicial (Tyr).

| Workflow | Descripción |
|----------|-------------|
| `workflow_project_lifecycle` | Inicialización completa (Recomendado para nuevos proyectos) |
| `workflow_prd_analyze` | Análisis de PRD con creación de plan y agentes (Recomendado) |
| `workflow_team_setup_from_prd` | Solo creación de agentes y equipo desde PRD |
| `workflow_stack_based_init` | Creación de plan basado en stack (sin PRD) |
| `workflow_plan_develop_v2` | Desarrollo multi-agente con delegation |
| `workflow_checkpoint_create` | Validación de calidad con human review |

## Reglas de Comunicación con el Usuario

**CRÍTICO: Después de ejecutar CUALQUIER función o workflow, el agente DEBE mostrar al usuario:**

### Después de `workflow_*`:
- **plan_id**: Mostrar el ID y título del plan creado
- **phases**: Número de fases creadas
- **tasks**: Número total de tareas
- **agents_created**: Nombres y roles de los agentes creados
- **team_id**: ID del equipo creado
- **stack_detected**: Stack y arquitectura detectada

### Después de `plan_create` o `plan_create_from_prd`:
- Mostrar: `plan_id`, `title`, `description`, `risk_level`, `phase_count`, `task_count`

### Después de `task_create` o `phase_create`:
- Mostrar: ID creado, título, y número de elementos creados

### Después de `agent_create`:
- Mostrar: `agent_id`, `name`, `role`, `agent_type`, `skills`

### Después de `team_create`:
- Mostrar: `team_id`, `name`, `member_count`, `agents`

### Después de `human_review_create`:
- Mostrar: `review_id`, `question`, `review_type`

### Formato de presentación:
```
📋 **Resumen de [workflow/función]:**
- **Plan ID**: xxx | **Título**: xxx
- **Fases**: X | **Tareas**: X
- **Agentes Creados**: X (backend-agent, qa-agent, etc.)
- **Equipo**: xxx Team (ID: xxx)
- **Stack**: Python/FastAPI con arquitectura monolítica
- **Próximo paso**: Revisar y aprobar el plan para continuar
```

**NUNCA omitas esta información. El usuario necesita estos datos para tomar decisiones.**

## Testing

Tests de comportamiento en `internal/*/database/`:
```bash
go test -race ./...
```

Cobertura mínima objetivo: 60-70% en rutas críticas.
