# PRD — Skoll v2.0.0
## Product Requirements Document — Agent-Based Orchestration

**Producto:** Skoll  
**Versión:** 2.0.0  
**Tipo:** MCP Plugin — RSAW Orchestration Layer  
**Lenguaje:** Go 1.22+  
**Licencia:** MIT  
**Fecha:** Marzo 2026

---

## Historial de versiones

| Versión | Cambios principales |
|---|---|
| v1.0.0 | Sistema RSAW base, SKILL.md oficial, Progressive Disclosure, SkillsMP |
| v2.0.0 | **Agent-Based Orchestration**, eliminación de workflows, task_executions, deprecated workflow_* |

---

## Qué cambia en v2.0.0

| Mejora | Origen | Impacto |
|---|---|---|
| Task Executions | Estándar Claude | Tracking granular de ejecuciones por agente |
| Agent-Based Orchestration | Estándar Claude | Skoll delega directamente a agentes, no a workflows |
| Workflow Deprecation | v2.0.0 | workflows_* marcados como deprecated, usar task_* |
| Multi-Agent Tasks | Hati v2.0 | Soporte para múltiples agentes por tarea |

---

## Tabla de contenidos

1. [Visión del producto](#1-visión-del-producto)
2. [Conceptos principales](#2-conceptos-principales)
3. [Agent-Based Orchestration](#3-agent-based-orchestration)
4. [Task Executions](#4-task-executions)
5. [MCP Tools v2.0.0](#5-mcp-tools-v200)
6. [Workflow Deprecation](#6-workflow-deprecation)
7. [CLI v2.0.0](#7-cli-v200)
8. [Integración con Hati v2.0](#8-integración-con-hati-v20)

---

## 1. Visión del producto

Skoll v2.0.0 adopta el **Agent-Based Orchestration** del estándar Claude. En lugar de workflows con pasos predefinidos, Skoll ahora recibe tareas de Hati y las delega directamente a agentes específicos. Cada tarea puede ser ejecutada por uno o múltiples agentes, con tracking granular del estado de ejecución.

> **Misión v2.0.0:** Que las tareas de Hati se ejecuten directamente a través de agentes específicos, sin la capa de workflows.

---

## 2. Conceptos principales

- **Agent-Based Orchestration** — Skoll delega tareas directamente a agentes, no a workflows
- **Task Executions** — tabla `task_executions` para tracking granular
- **Multi-Agent Tasks** — una tarea puede tener múltiples agentes asignados
- **Workflow Deprecation** — workflows_* marcados como deprecated en v2.0.0
- **Hati Integration** — Skoll recibe tareas de Hati via `hati_task_id`

---

## 3. Agent-Based Orchestration

### Arquitectura v2.0.0

```
┌─────────────────────────────────────────────────────────────────┐
│                     HATI (Planning)                             │
│  plan_create → phases → tasks (multi-agent) → checkpoints     │
└────────────────────────┬────────────────────────────────────────┘
                         │ task_delegate / task_execute
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     SKOLL (Orchestration)                       │
│  agents → skills → rules → task_executions → teams              │
│  ↑                                                                 │
│  │ agent_activate retorna skills + allowed_tools                 │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     AGENTS (Execution)                          │
│  backend, frontend, qa, devops, security, docs                  │
└─────────────────────────────────────────────────────────────────┘
```

### Flujo de Ejecución

```
Hati crea tarea con múltiples agentes:
{
  "title": "Implementar API de payments",
  "assigned_agent_ids": ["agent-backend-1", "agent-qa-1"]
}

Skoll recibe task_delegate:
{
  "task_id": "task-payments-api",
  "hati_task_id": "hati-task-123",
  "agent_ids": ["agent-backend-1", "agent-qa-1"]
}

Skoll crea task_executions:
- texec_1: agent-backend-1, status=in_progress
- texec_2: agent-qa-1, status=pending

Agentes reportan via task_heartbeat y task_complete
```

---

## 4. Task Executions

### Tabla `task_executions`

```sql
CREATE TABLE task_executions (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    hati_task_id TEXT,
    agent_id TEXT NOT NULL,
    phase_id TEXT,
    status TEXT DEFAULT 'pending',
    result TEXT,
    error TEXT,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    heartbeat_at DATETIME,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

### Estados de TaskExecution

| Estado | Descripción |
|---|---|
| `pending` | Agent asignado, esperando para ejecutar |
| `in_progress` | Agent ejecutando la tarea |
| `completed` | Tarea completada exitosamente |
| `failed` | Tarea falló con error |
| `cancelled` | Tarea cancelada |

### Estados de Agente Asociados

| Estado | Descripción |
|---|---|
| `idle` | Agent sin tarea asignada |
| `working` | Agent ejecutando una tarea |

---

## 5. MCP Tools v2.0.0

### Tools de Tasks (NUEVOS en v2.0)

| Tool | Descripción |
|---|---|
| `task_execute` | Ejecuta una tarea con un agente específico |
| `task_delegate` | Delega una tarea a múltiples agentes |
| `task_status` | Consulta estado de ejecuciones de tarea |
| `task_heartbeat` | Actualiza heartbeat de una ejecución |
| `task_complete` | Marca una ejecución como completada/fallida |
| `task_cancel` | Cancela una ejecución de tarea |

### Tools de Workflows (DEPRECATED en v2.0)

| Tool | Status | Alternativa |
|---|---|---|
| `workflow_start` | deprecated | `task_execute` |
| `workflow_step` | deprecated | `task_heartbeat` |
| `workflow_status` | deprecated | `task_status` |
| `workflow_complete` | deprecated | `task_complete` |
| `workflow_deprecate` | **NEW** | — |

### Especificaciones de Tools Nuevos

#### `task_execute`

```json
{
  "name": "task_execute",
  "description": "Execute a task with a specific agent",
  "inputSchema": {
    "type": "object",
    "required": ["task_id", "agent_id"],
    "properties": {
      "task_id": { "type": "string" },
      "hati_task_id": { "type": "string" },
      "agent_id": { "type": "string" },
      "phase_id": { "type": "string" }
    }
  }
}
```

**Respuesta:**
```json
{
  "execution_id": "texec_xxx",
  "task_id": "task-xxx",
  "hati_task_id": "hati-task-123",
  "agent_id": "agent-backend-1",
  "status": "in_progress",
  "started_at": "2026-03-27T10:00:00Z"
}
```

#### `task_delegate`

```json
{
  "name": "task_delegate",
  "description": "Delegate a task to multiple agents",
  "inputSchema": {
    "type": "object",
    "required": ["task_id", "agent_ids"],
    "properties": {
      "task_id": { "type": "string" },
      "hati_task_id": { "type": "string" },
      "agent_ids": { "type": "array", "items": { "type": "string" } },
      "phase_id": { "type": "string" }
    }
  }
}
```

**Respuesta:**
```json
{
  "task_id": "task-xxx",
  "hati_task_id": "hati-task-123",
  "delegated_to": [
    { "execution_id": "texec_1", "agent_id": "agent-backend-1", "status": "pending" },
    { "execution_id": "texec_2", "agent_id": "agent-qa-1", "status": "pending" }
  ],
  "total_agents": 2,
  "created_at": "2026-03-27T10:00:00Z"
}
```

#### `task_status`

```json
{
  "name": "task_status",
  "description": "Get status of task executions",
  "inputSchema": {
    "type": "object",
    "properties": {
      "execution_id": { "type": "string" },
      "task_id": { "type": "string" },
      "agent_id": { "type": "string" }
    }
  }
}
```

#### `task_complete`

```json
{
  "name": "task_complete",
  "description": "Mark a task execution as completed or failed",
  "inputSchema": {
    "type": "object",
    "required": ["execution_id"],
    "properties": {
      "execution_id": { "type": "string" },
      "status": { "type": "string", "enum": ["completed", "failed"] },
      "result": { "type": "string" },
      "error": { "type": "string" }
    }
  }
}
```

---

## 6. Workflow Deprecation

### Marcado de Deprecation

```go
func (s *Server) handleWorkflowDeprecate(ctx context.Context, req *Request) (*Response, error) {
    // UPDATE workflows SET deprecated = 1 WHERE id = ?
    // Retorna: { workflow_id, deprecated: true, note: "Use task_execute/task_delegate instead" }
}
```

### Tabla `workflows` con deprecated flag

```sql
ALTER TABLE workflows ADD COLUMN deprecated INTEGER DEFAULT 0;
```

### Migración de Workflows a Tasks

| Workflow | Task Equivalente |
|---|---|
| `workflow_start` | `task_execute` o `task_delegate` |
| `workflow_step` | `task_heartbeat` |
| `workflow_status` | `task_status` |
| `workflow_complete` | `task_complete` |

---

## 7. CLI v2.0.0

```bash
# ─── TASKS (NUEVO) ──────────────────────────────────────────────
skoll task execute <task_id> --agent <agent_id> [--hati-task <hati_task_id>]
skoll task delegate <task_id> --agents <agent_ids> [--hati-task <hati_task_id>]
skoll task status [--execution <id>] [--task <task_id>] [--agent <agent_id>]
skoll task heartbeat <execution_id>
skoll task complete <execution_id> [--status completed|failed] [--result <result>]
skoll task cancel <execution_id> [--reason <reason>]

# ─── WORKFLOWS (DEPRECATED) ─────────────────────────────────────
skoll workflows list [--include-deprecated]
skoll workflow deprecate <workflow_id>

# ─── AGENTS ─────────────────────────────────────────────────────
skoll agents list
skoll agents show <agent_id>
skoll agents create <name> --type <backend|frontend|qa|devops|security|docs>
skoll agents activate <agent_id> [--context-path <path>]

# ─── TASKS LEGACY ───────────────────────────────────────────────
skoll task get <task_id>     # Alias para task_status
skoll task list [--status pending|in_progress|completed|failed]
```

---

## 8. Integración con Hati v2.0

### Flujo Hati → Skoll

```
1. Hati: task_create con assigned_agent_ids
   → Crea tarea con múltiples agentes

2. Hati: task_assign_agents (nuevo)
   → Asocia agentes adicionales a tarea existente

3. Hati llama a Skoll: task_delegate
   {
     "task_id": "task-payments-api",
     "hati_task_id": "hati-task-123",
     "agent_ids": ["agent-backend-1", "agent-qa-1"]
   }

4. Skoll crea task_executions para cada agente

5. Agentes reportan via Skoll: task_heartbeat, task_complete

6. Skoll actualiza estado en Hati via callback o Hati polling
```

### Integración con Hati Task Agents

Hati v2.0 tiene tabla `task_agents` para tracking de agentes por tarea:

```sql
CREATE TABLE task_agents (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    role TEXT DEFAULT 'worker',
    status TEXT DEFAULT 'pending',
    started_at DATETIME,
    completed_at DATETIME,
    result TEXT,
    error TEXT
);
```

Skoll sincroniza estados con Hati via `hati.task_agent_update`.

---

## 9. Modelo de datos v2.0.0

### Tablas en Skoll

| Tabla | Descripción |
|---|---|
| `skills` | Skill definitions |
| `rules` | Rule definitions |
| `agents` | Specialized agents |
| `teams` | Team coordination |
| `team_members` | Team memberships |
| `agent_tasks` | Task execution tracking |
| `task_executions` | **NUEVA**: Granular task execution tracking |
| `workflows` | Workflows (deprecated) |
| `pending_rules` | Inbound rule proposals |
| `team_context` | Active scope per module |

### Tablas en Hati (actualizadas)

| Tabla | Cambios |
|---|---|
| `tasks` | `assigned_agent_id` → `assigned_agent_ids` (JSON array) |
| `task_agents` | **NUEVA**: Agent-task relationship tracking |

---

## 10. Configuración v2.0.0

```json
{
  "project": "mi-proyecto",
  "version": "2.0.0",

  "orchestration": {
    "mode": "agent_based",
    "delegate_to_hati": false
  },

  "tasks": {
    "heartbeat_timeout_minutes": 5,
    "auto_cancel_on_agent_offline": true
  },

  "agents": {
    "coordination_enabled": true,
    "role_ttl_hours": 4
  },

  "workflows": {
    "deprecated": true,
    "migrate_to_tasks": true
  },

  "integrations": {
    "hati": {
      "enabled": true,
      "sync_task_agents": true,
      "callback_url": ""
    }
  }
}
```

---

## 11. Roadmap v2.0.0

| Fase | Semanas | Deliverable |
|---|---|---|
| 1 — Task Executions | 1-2 | Tabla task_executions, handlers task_* |
| 2 — Agent Delegation | 3 | task_delegate multi-agent |
| 3 — Hati Integration | 4 | Sincronización con task_agents de Hati |
| 4 — Workflow Deprecation | 5 | Marca workflows como deprecated |
| 5 — Documentation | 6 | Actualizar docs y examples |
| 6 — Release v2.0.0 | 7 | Testing, release |

---

## 12. Mejoras Planificadas v2.1.0

| Mejora | Descripción | Prioridad |
|---|---|---|
| **Task Dependencies** | Una task puede depender de otra antes de ejecutarse | 🟡 MEDIA |
| **Agent Pooling** | Múltiples agentes del mismo tipo para parallelización | 🟡 MEDIA |
| **Task Priorities** | Priorities dinámicas basadas en plan phase | 🟡 MEDIA |

---

*Skoll PRD v2.0.0 — Marzo 2026*
*~35 MCP tools · Go 1.22+ · MIT*
*Agent-Based Orchestration · Task Executions · Multi-Agent Tasks*
*3 Pilares: Velocidad ⚡ · Eficiencia de Tokens 💎 · Eficacia 🎯*