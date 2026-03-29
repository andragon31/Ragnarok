# Ragnarok — Agent Guidelines

> v2.2.4 | Actualizar con cada cambio arquitectónico significativo

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

## Workflows MCP Principales

| Workflow | Descripción |
|----------|-------------|
| `workflow_stack_based_init` | Detecta stack y crea plan completo |
| `workflow_plan_develop_v2` | Desarrollo multi-agente con delegation |
| `workflow_checkpoint_create` | Validación de calidad con human review |

## Testing

Tests de comportamiento en `internal/*/database/`:
```bash
go test -race ./...
```

Cobertura mínima objetivo: 60-70% en rutas críticas.
