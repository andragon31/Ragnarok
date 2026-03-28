# PRD — Hati v2.0.0
## Product Requirements Document — Multi-Agent Task Planning

**Producto:** Hati  
**Versión:** 2.0.0  
**Tipo:** MCP Plugin — Task Planning & Human-in-the-Loop Layer  
**Lenguaje:** Go 1.22+  
**Licencia:** MIT  
**Fecha:** Marzo 2026

---

## Historial de versiones

| Versión | Cambios principales |
|---|---|
| v1.0.0 | Ciclo PRE→execute→POST, Approval Record, granularidad dinámica, Quality Snapshot |
| v2.0.0 | **Multi-Agent Tasks**, task_agents table, Hati-Skoll orchestration directa |

---

## Qué cambia en v2.0.0

| Mejora | Origen | Impacto |
|---|---|---|
| Multi-Agent Tasks | Estándar Claude | Tasks pueden asignarse a múltiples agentes |
| task_agents table | v2.0.0 | Tracking granular de agentes por tarea |
| Hati-Skoll Direct | v2.0.0 | Skoll recibe tareas directamente, no workflows |
| Agent Roles | v2.0.0 | Roles específicos para agentes en tareas |

---

## Tabla de contenidos

1. [Visión del producto](#1-visión-del-producto)
2. [Conceptos principales](#2-conceptos-principales)
3. [Multi-Agent Tasks](#3-multi-agent-tasks)
4. [Task Agents](#4-task-agents)
5. [MCP Tools v2.0.0](#5-mcp-tools-v200)
6. [CLI v2.0.0](#6-cli-v200)
7. [Integración con Skoll v2.0](#7-integración-con-skoll-v20)
8. [Modelo de datos v2.0.0](#8-modelo-de-datos-v200)

---

## 1. Visión del producto

Hati v2.0.0 soporta **Multi-Agent Tasks** donde una tarea puede ser ejecutada por múltiples agentes simultáneamente. Cada agente tiene un rol específico (worker, reviewer, qa) y puede reportar su estado independientemente. La integración con Skoll v2.0 permite orquestación directa agente-a-agente sin la capa de workflows.

> **Misión v2.0.0:** Que las tareas se ejecuten con el agente correcto (o agentes correctos), con tracking granular del progreso.

---

## 2. Conceptos principales

- **Multi-Agent Tasks** — una tarea puede tener múltiples agentes asignados
- **Task Agents** — tabla para tracking de agente por tarea con roles
- **Agent Roles** — worker, reviewer, qa, coordinator
- **Skoll Direct Integration** — Skoll recibe tareas directamente, no workflows
- **Task Execution Sync** — sincronización de estado con Skoll task_executions

---

## 3. Multi-Agent Tasks

### Estructura de Task v2.0.0

```go
type Task struct {
    ID                string      `json:"id"`
    PhaseID           string      `json:"phase_id"`
    Title             string      `json:"title"`
    Description       string      `json:"description,omitempty"`
    Status            string      `json:"status"`
    Priority          int         `json:"priority"`
    AssignedAgentIDs   []string    `json:"assigned_agent_ids,omitempty"`  // NUEVO: array
    AssignedAgentType string      `json:"assigned_agent_type,omitempty"`  // deprecated
    EstimatedHours    float64     `json:"estimated_hours,omitempty"`
    ActualHours       float64     `json:"actual_hours,omitempty"`
    Notes             string      `json:"notes,omitempty"`
    Blocker           string      `json:"blocker,omitempty"`
    Milestone         bool        `json:"milestone"`
    Subtasks          []string    `json:"subtasks,omitempty"`
    CompletedAt       *time.Time  `json:"completed_at,omitempty"`
    CreatedAt         time.Time   `json:"created_at"`
    UpdatedAt         time.Time   `json:"updated_at"`
}
```

### Diferencia con v1.0.0

| Campo | v1.0.0 | v2.0.0 |
|---|---|---|
| `assigned_agent_id` | `string` (singular) | Eliminado |
| `assigned_agent_ids` | No existía | `[]string` (array) |
| `assigned_agent_type` | `string` | Deprecated |

### Flujo de Creación

```
1. Developer/Agent llama task_create con agent_ids:
   {
     "phase_id": "phase-1",
     "title": "Implementar API de payments",
     "assigned_agent_ids": ["agent-backend-1", "agent-qa-1"]
   }

2. Hati crea task + task_agents:
   - task: id, phase_id, title, status=pending
   - task_agents: 
     - agent-backend-1, role=worker, status=pending
     - agent-qa-1, role=worker, status=pending

3. Hati llama a Skoll: task_delegate
   Skoll crea task_executions para cada agente
```

---

## 4. Task Agents

### Tabla `task_agents`

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
    error TEXT,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);
```

### Roles de Agente

| Rol | Descripción |
|---|---|
| `worker` | Ejecuta la tarea principal |
| `reviewer` | Revisa el trabajo del worker |
| `qa` | Ejecuta testing/QA |
| `coordinator` | Coordina múltiples workers |

### Estados de TaskAgent

| Estado | Descripción |
|---|---|
| `pending` | Agent asignado, esperando |
| `assigned` | Notificado para ejecutar |
| `in_progress` | Ejecutando |
| `completed` | Completado exitosamente |
| `failed` | Falló |
| `blocked` | Bloqueado por dependencia |

### Task Status Calculation

El status de la Task se calcula desde los TaskAgents:

```
SI algún task_agent.status = 'failed' → task.status = 'blocked'
SI todos task_agent.status = 'completed' → task.status = 'completed'
SI algún task_agent.status = 'in_progress' → task.status = 'in_progress'
SI todos task_agent.status = 'pending' → task.status = 'pending'
```

---

## 5. MCP Tools v2.0.0

### Tools de Tasks (MODIFICADOS)

| Tool | Cambios |
|---|---|
| `task_create` | `assigned_agent_id` → `assigned_agent_ids` (array) |
| `task_get` | Retorna `task_agents` array |
| `task_update` | Ya no actualiza assigned_agent (usar task_assign_agents) |
| `task_list` | `agent_id` → `agent_ids` en respuesta |

### Tools de Tasks Agents (NUEVOS)

| Tool | Descripción |
|---|---|
| `task_assign_agents` | Asigna agentes adicionales a una tarea |
| `task_agent_update` | Actualiza estado de un task_agent |
| `task_agent_list` | Lista agentes de una tarea |

### Especificaciones de Tools Nuevos

#### `task_assign_agents`

```json
{
  "name": "task_assign_agents",
  "description": "Assign multiple agents to a task",
  "inputSchema": {
    "type": "object",
    "required": ["task_id", "agent_ids"],
    "properties": {
      "task_id": { "type": "string" },
      "agent_ids": { "type": "array", "items": { "type": "string" } },
      "role": { "type": "string", "default": "worker" }
    }
  }
}
```

**Respuesta:**
```json
{
  "task_id": "task-xxx",
  "agents_assigned": [
    { "task_agent_id": "ta_xxx", "agent_id": "agent-1", "role": "worker", "status": "assigned" }
  ],
  "updated_at": "2026-03-27T10:00:00Z"
}
```

#### `task_agent_update`

```json
{
  "name": "task_agent_update",
  "description": "Update a task agent status",
  "inputSchema": {
    "type": "object",
    "required": ["task_agent_id"],
    "properties": {
      "task_agent_id": { "type": "string" },
      "status": { "type": "string", "enum": ["in_progress", "completed", "failed", "blocked"] },
      "result": { "type": "string" },
      "error": { "type": "string" }
    }
  }
}
```

---

## 6. CLI v2.0.0

```bash
# ─── TASKS ──────────────────────────────────────────────────────
hati task create <phase_id> <title> [--agents <agent_ids>] [--priority <n>]
hati task get <task_id>
hati task list [--phase <phase_id>] [--plan <plan_id>] [--status <status>]
hati task update <task_id> [--status <status>] [--notes <notes>]
hati task assign <task_id> --agents <agent_ids> [--role <role>]

# ─── TASK AGENTS ─────────────────────────────────────────────────
hati task-agent update <task_agent_id> --status <status> [--result <result>]
hati task-agent list <task_id>

# ─── PLANS ───────────────────────────────────────────────────────
hati plan status [--verbose]
hati plan show <plan_id> [--agents] [--hints]
```

---

## 7. Integración con Skoll v2.0

### Flujo Completo Hati → Skoll

```
┌─────────────────────────────────────────────────────────────────┐
│                     HATI (Planning)                              │
│                                                                  │
│  plan_create → phases → tasks (multi-agent) → checkpoints      │
│       │                                                      │
│       └─ task_create                                         │
│              │                                                │
│              ▼                                                │
│       ┌──────────────────┐                                    │
│       │ task_assign_agents │ (opcional, múltiples agentes)    │
│       └──────────────────┘                                    │
│              │                                                │
│              ▼                                                │
│       ┌─────────────────────────────────────────┐             │
│       │ Skoll: task_delegate                    │             │
│       │ {                                       │             │
│       │   task_id: "task-xxx",                  │             │
│       │   hati_task_id: "hati-task-xxx",        │             │
│       │   agent_ids: ["agent-1", "agent-2"]     │             │
│       │ }                                       │             │
│       └─────────────────────────────────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

### Sincronización de Estados

```
1. Skoll: task_execute → in_progress
   → Skoll actualiza task_agents en Hati: status=in_progress

2. Agent reporta heartbeat
   → Skoll: task_heartbeat
   → Hati: task_agent_update (status=in_progress)

3. Agent completa tarea
   → Skoll: task_complete
   → Hati: task_agent_update (status=completed)

4. Hati calcula task status desde task_agents
```

---

## 8. Modelo de datos v2.0.0

### Cambios en Tabla `tasks`

```sql
-- ANTES (v1.0.0)
assigned_agent_id TEXT,
assigned_agent_type TEXT,

-- AHORA (v2.0.0)
assigned_agent_ids TEXT,
assigned_agent_type TEXT,  -- deprecated
```

### Nueva Tabla `task_agents`

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
    error TEXT,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE INDEX idx_task_agents_task ON task_agents(task_id);
CREATE INDEX idx_task_agents_agent ON task_agents(agent_id);
```

### Índices Actualizados

```sql
-- Eliminado
CREATE INDEX idx_tasks_assigned ON tasks(assigned_agent_id);

-- Agregado (nuevo, para buscar por cualquier agente en task_agents)
CREATE INDEX idx_task_agents_task ON task_agents(task_id);
CREATE INDEX idx_task_agents_agent ON task_agents(agent_id);
```

---

## 9. Roadmap v2.0.0

| Fase | Semanas | Deliverable |
|---|---|---|
| 1 — Multi-Agent Tasks | 1-2 | Schema update, task_create/list/get con agent_ids |
| 2 — Task Agents | 3-4 | Tabla task_agents, task_assign_agents, task_agent_update |
| 3 — Status Calculation | 5 | Auto-calculate task status desde task_agents |
| 4 — Skoll Integration | 6 | Sincronización con Skoll task_executions |
| 5 — CLI Update | 7 | Actualizar CLI con nuevos comandos |
| 6 — Release v2.0.0 | 8 | Testing, release |

---

## 10. Mejoras Planificadas v2.1.0

| Mejora | Descripción | Prioridad |
|---|---|---|
| **Task Dependencies** | Una task puede depender de otra | 🟡 MEDIA |
| **Parallel Execution** | Múltiples agents ejecutando en paralelo | 🟡 MEDIA |
| **Agent Availability** | Routing inteligente según disponibilidad | 🟡 MEDIA |

---

*Hati PRD v2.0.0 — Marzo 2026*
*~30 MCP tools · Go 1.22+ · MIT*
*Multi-Agent Tasks · Task Agents · Direct Skoll Integration*
*3 Pilares: Velocidad ⚡ · Eficiencia de Tokens 💎 · Eficacia 🎯*