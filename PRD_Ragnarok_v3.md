# PRD — Ragnarok Ecosystem v3.0
> Product Requirements Document | Abril 2026

---

## 1. Visión del Producto

Ragnarok es un ecosistema MCP (Model Context Protocol) de 4 módulos que actúa como **capa de gobernanza para agentes de IA** en proyectos de software. Permite que herramientas como OpenCode, Windsurf, Cursor, Claude Code y Antigravity trabajen de forma estructurada, con memoria persistente, planificación, reglas de calidad y contexto de orquestación.

### Objetivo Principal
Que cualquier agente conectado a `rag mcp` pueda:
1. Entender qué hacer en menos de 60 segundos
2. Trabajar en proyectos nuevos (desde un PRD) y existentes (onboarding)
3. Completar tareas con calidad verificable
4. Coordinar sin conflictos con otros agentes en el mismo proyecto

### Principios de Diseño
- **Funciona solo o junto**: cada módulo es útil por sí mismo
- **LLM-first**: el modelo no simula procesos — registra contexto stateless
- **Una fuente de verdad**: Hati para tareas, Fenrir para memoria, Tyr para calidad
- **Cero fricción**: menos de 3 llamadas MCP para arrancar a trabajar

---

## 2. Arquitectura: Monorepo Federado

### Estructura Final

```
Ragnarok/
├── cmd/
│   ├── fenrir/main.go     → binario: fenrir mcp (memoria sola)
│   ├── hati/main.go       → binario: hati mcp (planning solo)
│   ├── skoll/main.go      → binario: skoll mcp (skills+rules solo)
│   ├── tyr/main.go        → binario: tyr mcp (quality solo)
│   └── rag/main.go        → binario: rag mcp (los 4 juntos)
├── internal/
│   ├── fenrir/            → Memory & Context
│   ├── hati/              → Planning & Tasks
│   ├── skoll/             → Skills, Rules & Context Registration
│   ├── tyr/               → Quality Gates
│   └── mcp/unified/       → Servidor unificado (solo usado por rag)
└── go.mod
```

### Modos de Uso

```bash
# Solo memoria (proyecto simple)
{ "mcpServers": { "fenrir": { "command": "fenrir", "args": ["mcp"] } } }

# Solo quality gates (CI/CD pipeline)
{ "mcpServers": { "tyr": { "command": "tyr", "args": ["mcp"] } } }

# Ecosistema completo
{ "mcpServers": { "ragnarok": { "command": "rag", "args": ["mcp"] } } }
```

---

## 3. Casos de Uso Primarios

### UC-1: Proyecto Nuevo desde PRD

```
Actor: Agente de IA + Usuario
Precondición: Existe un archivo PRD.md con requisitos

Flujo:
1. ragnarok_help()                           → entiende el sistema
2. workflow_project_lifecycle({              → workflow completo:
     project_path, prd_file })                 escanea → plan → agentes → quality init
3. human_review_pending()                    → espera aprobación del usuario
4. human_review_decide({ decision:"approved" })
5. [Loop de desarrollo]:
   task_get_next({ plan_id })               → tarea con contexto completo
   context_start({ role })                 → registra quién trabaja
   task_claim({ task_id })                 → marca como "en progreso"
   [LLM trabaja]
   mem_save({ type, title, content })      → guarda lo aprendido
   task_update({ task_id, status:"completed" })
6. Para cada fase completa:
   quality_gate({ path })                  → verifica calidad
   workflow_checkpoint_create({ plan_id }) → genera human review
7. plan_complete({ plan_id })              → cierra el proyecto
```

### UC-2: Proyecto Existente (Onboarding)

```
Actor: Agente de IA
Precondición: Proyecto con código existente, sin plan previo en Ragnarok

Flujo:
1. ragnarok_help()
2. project_scan({ path })                   → analiza stack, arquitectura
3. spec_save({ type:"architecture", ... })  → documenta lo encontrado
4. tyr_bootstrap({ path })                  → genera standards según stack detectado
5. skill_generate({ project_path })         → genera skills para el proyecto
6. rules_generate({ project_path })         → genera reglas basadas en el stack
7. plan_create({ title, description })      → crea plan de trabajo
8. phase_create({ plan_id, name, ... })     → crea fases
9. task_create({ phase_id, ... })           → crea tareas
10. [Loop de desarrollo normal - igual que UC-1 desde paso 5]
```

### UC-3: Retomar Trabajo (Continuación)

```
Actor: Agente de IA recién conectado
Precondición: Trabajo previo existente en las DBs

Flujo:
1. session_context_full()    → estado completo en 1 llamada:
                               plan activo, tarea siguiente,
                               reviews pendientes, memoria reciente
2. Si hay task pendiente:
   task_claim + continuar trabajando
3. Si hay human_review pendiente:
   Informar al usuario y esperar decisión
```

---

## 4. Módulo: Fenrir (Memory & Context)

### Responsabilidad
Persistencia de conocimiento entre sesiones. Responde: "¿Qué sé sobre este proyecto?"

### Herramientas Finales (14)

| Herramienta | Acción | Parámetros Clave | Estado |
|-------------|--------|-----------------|--------|
| `mem_save` | Guardar observación | type, title, content, project_path | [x] |
| `mem_find` | Buscar en memoria (FTS5) | query, project_path, limit | [x] |
| `mem_context` | Contexto reciente del proyecto | project_path, limit | [x] |
| `mem_timeline` | Timeline de actividad | project_path, days | [x] |
| `mem_stats` | Estadísticas de memoria | project_path | [x] |
| `mem_session_start` | Iniciar sesión de trabajo | goal, project_path | [x] |
| `mem_session_end` | Cerrar sesión | session_id, summary | [x] |
| `mem_session_checkpoint` | Punto de control en sesión | session_id, note | [x] |
| `mem_get_observation` | Obtener observación por ID | observation_id | [x] |
| `mem_project_summary` | **[NUEVO]** Resumen semántico del proyecto | project_path, days | [ ] |
| `spec_save` | Guardar especificación técnica | type, title, content | [x] |
| `spec_list` | Listar specs | project_path, type | [x] |
| `spec_check` | Verificar spec vigente | spec_id | [x] |
| `project_scan` | Analizar stack y arquitectura | path | [x] |

### Cambios Requeridos

**ELIMINAR:**
- [x] `handleProjectBootstrap` → mover al workflow `workflow_onboard_existing`
- [x] `handleSkillGenerate`, `handleRulesGenerate`, `handleStandardsGenerate` → mover a módulos correctos (Skoll/Tyr)
- [x] Tablas `nodes`, `edges` del schema (grafo nunca usado, ~400 LOC)
- [x] `internal/fenrir/graph/` directorio completo

**AGREGAR:**
```go
// mem_project_summary
// Consulta las últimas N observaciones, agrupa por tipo,
// retorna resumen de actividad reciente y temas frecuentes
func handleMemProjectSummary(ctx, req) → {
    recent_activity: [...],
    frequent_topics: [...],
    active_specs: [...],
    last_session: {...}
}
```

**MEJORAR:**
- Todas las descripciones MCP >80 chars con formato: [CUÁNDO usar] → [QUÉ hace] → [RETORNA]
- Índice: `CREATE INDEX idx_observations_project ON observations(project_path, created_at DESC)`

### Schema SQLite Final (Fenrir)
```sql
-- CONSERVAR: observations, sessions, session_checkpoints, specs, prompts
-- ELIMINAR: nodes, edges
-- AGREGAR: índice por project_path en observations
```

---

## 5. Módulo: Hati (Planning & Tasks)

### Responsabilidad
Fuente de verdad para planes, fases y tareas. Responde: "¿Qué hay que hacer y en qué orden?"

### Herramientas Finales (22)

| Herramienta | Acción | Es nueva | Estado |
|-------------|--------|----------|--------|
| `plan_create` | Crear plan | No | [x] |
| `plan_get` | Obtener plan | No | [x] |
| `plan_get_active` | **[NUEVO]** Plan activo actual | Sí | [x] |
| `plan_list` | Listar planes | No | [x] |
| `plan_activate` | Activar plan | No | [x] |
| `plan_complete` | Completar plan | No | [x] |
| `plan_abandon` | Abandonar plan | No | [x] |
| `plan_dashboard` | Vista general del plan | No | [x] |
| `plan_blockers` | Bloqueadores activos | No | [x] |
| `plan_progress` | Progreso del plan | No | [x] |
| `plan_create_from_prd` | Crear plan desde PRD | No | [x] |
| `phase_create` | Crear fase | No | [x] |
| `phase_start` | Iniciar fase | No | [x] |
| `phase_update` | Actualizar fase | No | [x] |
| `phase_report` | Completar fase con reporte | No | [x] |
| `task_create` | Crear tarea | No | [x] |
| `task_get` | Obtener tarea | No | [x] |
| `task_get_next` | **[MEJORADO]** Siguiente tarea + contexto completo | Mejora | [x] |
| `task_list` | Listar tareas | No | [x] |
| `task_update` | **[MEJORADO]** Actualizar estado (absorbe task_complete de Skoll) | Mejora | [x] |
| `task_claim` | **[NUEVO]** Marcar tarea como "en progreso por mí" | Sí | [x] |
| `task_release` | **[NUEVO]** Soltar tarea sin completar | Sí | [x] |
| `human_review_create` | Crear revisión humana | No | [x] |
| `human_review_pending` | Revisiones pendientes | No | [x] |
| `human_review_decide` | Decidir revisión | No | [x] |

### Cambios Requeridos

**ELIMINAR del schema (tablas muertas):**
```sql
**ELIMINAR del schema (tablas muertas):**
- [x] `DROP TABLE plan_quality_scores;`  -- siempre devuelve 0.0, Tyr calcula esto
- [x] `DROP TABLE plan_recovery;`        -- ningún handler lo usa
- [x] `DROP TABLE agent_locks;`          -- ningún handler escribe aquí
- [x] `DROP TABLE checkpoint_sla;`       -- no hay handler de escalación
```

**AGREGAR columnas a `task_agents`:**
```sql
ALTER TABLE task_agents ADD COLUMN context_id TEXT;
ALTER TABLE task_agents ADD COLUMN claimed_at DATETIME;
ALTER TABLE task_agents ADD COLUMN released_at DATETIME;
```

**AGREGAR handlers:**
```go
// plan_get_active: retorna el plan más reciente en estado active/in_progress
func handlePlanGetActive(ctx, req) → { plan_id, title, phase_count, task_pending }

// task_claim: marca tarea en task_agents, previene conflictos entre agentes
func handleTaskClaim(ctx, req) → { task_id, claimed_at, task_detail, relevant_specs }

// task_release: libera tarea sin completar, vuelve a status pending
func handleTaskRelease(ctx, req) → { task_id, released_at, reason }
```

**MEJORAR `task_get_next`:**
```go
// Actual: devuelve solo la tarea
// Nuevo: devuelve tarea + contexto completo en 1 llamada
func handleTaskGetNext(ctx, req) → {
    task: { id, title, description, phase, priority },
    context: {
        relevant_specs: [...],     // specs relacionadas a la tarea
        recent_memory: [...],      // últimas 5 observaciones del proyecto
        applicable_rules: [...],   // reglas de Skoll que aplican
    },
    next_steps: "Llama task_claim para tomar esta tarea"
}
```

**CORREGIR el conflicto `quality_snapshot` / `spec_impact`:**
- `quality_snapshot` en Hati → **eliminar** (solo debe estar en Tyr como `tyr_snapshot`)
- `spec_impact` en Hati → **eliminar** (solo debe estar en Fenrir)

**Schema SQLite Final (Hati):**
```sql
-- CONSERVAR: plans, phases, checkpoints, feedback, approval_record,
--            plan_revisions, execution_blockers, notifications,
--            plan_dependencies, prds, prd_requirements, tasks, task_agents,
--            human_reviews
-- ELIMINAR: plan_quality_scores, plan_recovery, agent_locks, checkpoint_sla
```

---

## 6. Módulo: Skoll (Skills, Rules & Context)

### Responsabilidad
Caja de herramientas del agente. Responde: "¿Qué sé hacer, qué está permitido y quién soy en este contexto?"

### El Rediseño Central: Context-Based Model

**Modelo ANTERIOR (ficticio):**
```
agent_create → agent_activate → task_execute → task_heartbeat → task_complete
```
Este modelo simula procesos persistentes. Los LLMs son stateless — no hay proceso que envíe heartbeats.

**Modelo NUEVO (real):**
```
context_start(role) → task_claim(task_id) → [LLM trabaja] → task_update(done)
```

### Herramientas Finales (14)

| Herramienta | Acción | Estado |
|-------------|--------|--------|
| `context_start` | **[NUEVO]** Registrar contexto de trabajo actual | [x] |
| `context_end` | **[NUEVO]** Cerrar contexto de trabajo | [x] |
| `skill_list` | Listar skills disponibles | [x] |
| `skill_load` | Cargar skill completa (instrucciones) | [x] |
| `skill_search` | Buscar skill por descripción | [x] |
| `skill_verify` | Verificar compatibilidad de skill | [x] |
| `rule_list` | Listar reglas activas | [x] |
| `rule_check` | Verificar si acción cumple reglas | [x] |
| `rule_get` | Obtener regla detallada | [x] |
| `rule_create_or_reuse` | Crear o reusar regla existente | [x] |
| `team_create` | Crear equipo de contextos | [x] |
| `team_get` | Obtener equipo y miembros | [x] |
| `agent_list` | Listar contextos registrados | [x] |
| `agent_get` | Obtener metadata de contexto | [x] |

### Cambios Requeridos

**ELIMINAR handlers:**
```
handleAgentHeartbeat      → ficticio, LLM no envía heartbeats
handleAgentActivate       → status idle/working es inútil sin proceso real
handleAgentAssignTask     → duplica Hati task_claim
handleAgentCompleteTask   → duplica Hati task_update
handleAgentHandoff        → nunca implementado correctamente
handleAgentContext        → reemplazado por context_start
handleTaskExecute         → duplica Hati (con tracking falso)
handleTaskDelegate        → duplica Hati task_agents
handleTaskHeartbeat       → nadie lo llama
handleTaskComplete (Skoll) → duplica Hati task_update
handleTaskCancel (Skoll)   → duplica Hati task_update
handleWorkflowDeprecate   → obsoleto
handleBootstrapImport     → renombrar a skoll_skills_import
handleSkollValidate       → sin implementación útil
handleSkillReadFile       → mover instrucción al skill_load
```

**AGREGAR handlers:**
```go
// context_start: registra que alguien está trabajando (sin proceso persistente)
func handleContextStart(ctx, req) → {
    context_id,          // ID temporal de esta sesión de trabajo
    role,                // backend, frontend, qa, etc.
    applicable_rules,    // reglas que aplican a este rol
    suggested_skills,    // skills relevantes para el rol
    note: "Usa task_claim para tomar una tarea"
}

// context_end: cierra la sesión de trabajo
func handleContextEnd(ctx, req) → { context_id, duration, tasks_done }
```

**SIMPLIIFCAR `AgentTypes`:**
```go
// ANTES: hardcoded con skills/tools del IDE mezclados
var AgentTypes = map[string]map[string]interface{}{
    "backend": { "skills": ["go", "python"...], "allowed_tools": ["Bash", "Read"...] }
}

// DESPUÉS: solo metadata de etiqueta
var ContextRoles = map[string]string{
    "backend":  "Implementa APIs, bases de datos y servicios",
    "frontend": "Construye interfaces, componentes y experiencia de usuario",
    "qa":       "Diseña y ejecuta pruebas de calidad",
    "devops":   "Gestiona infraestructura, CI/CD y deployment",
    "security": "Audita seguridad y conformidad",
    "docs":     "Crea y mantiene documentación técnica",
    "custom":   "Rol personalizado para este proyecto",
}
// Los tools del IDE los da el IDE, no Skoll
```

**ELIMINAR tablas del schema:**
```sql
DROP TABLE task_executions;  -- movido a hati.task_agents
DROP TABLE agent_tasks;      -- movido a hati.task_agents
DROP TABLE workflows;        -- deprecated, no se usa activamente
```

**CONSERVAR tablas:**
```sql
-- skills, rules, agents (como labels de contexto), teams, team_members
-- pending_rules, team_context
```

---

## 7. Módulo: Tyr (Quality Gates)

### Responsabilidad
Verificación de calidad del código. Responde: "¿Este código es seguro y cumple los estándares?"

### Herramientas Finales (14)

| Herramienta | Acción | Estado |
|-------------|--------|--------|
| `pkg_check` | Verificar seguridad de paquete | [x] |
| `pkg_license` | Verificar licencia de paquete | [x] |
| `pkg_audit` | Auditar dependencias del proyecto | [x] |
| `pkg_audit_snapshot` | Snapshot de auditoría | [x] |
| `sast_run` | Análisis estático de seguridad | [x] |
| `sast_findings` | Obtener hallazgos de SAST | [x] |
| `sast_resolve` | Marcar hallazgo como resuelto | [x] |
| `standard_run` | Ejecutar un estándar específico | [x] |
| `standard_run_all` | Ejecutar todos los estándares | [x] |
| `standard_list` | Listar estándares configurados | [x] |
| `tyr_snapshot` | **[RENOMBRADO]** Snapshot de calidad completo | [x] |
| `tyr_bootstrap` | **[RENOMBRADO]** Importar estándares base | [x] |
| `precommit_validate` | Validar antes de commit | [x] |
| `quality_gate` | **[NUEVO]** Gate unificado (sast+standards+precommit) | [x] |

### Cambios Requeridos

**RENOMBRAR (resolver conflictos):**
```
quality_snapshot → tyr_snapshot   (conflicto con Hati)
bootstrap_import → tyr_bootstrap  (conflicto con Skoll)
```

**ELIMINAR de Hati:**
- `quality_snapshot` en `hati/mcp/server.go` → debe existir SOLO en Tyr

**AGREGAR:**
```go
// quality_gate: ejecuta verificación completa en una llamada
func handleQualityGate(ctx, req) → {
    path: string,
    gate_level: "basic" | "full",
    // basic: sast_run + precommit_validate
    // full: basic + standard_run_all + pkg_audit
    result: "pass" | "fail",
    details: { sast: {...}, standards: {...}, packages: {...} },
    blocking_issues: [...],
    warnings: [...]
}
```

---

## 8. Servidor Unificado (`rag mcp`)

### Herramientas Expuestas (Total: 62)

| Módulo | Count | Herramientas |
|--------|-------|-------------|
| Fenrir | 14 | mem_save, mem_find, mem_context, mem_timeline, mem_stats, mem_session_start, mem_session_end, mem_session_checkpoint, mem_get_observation, mem_project_summary, spec_save, spec_list, spec_check, project_scan |
| Hati | 25 | plan_create, plan_get, plan_get_active, plan_list, plan_activate, plan_complete, plan_abandon, plan_dashboard, plan_blockers, plan_progress, plan_create_from_prd, phase_create, phase_start, phase_update, phase_report, task_create, task_get, task_get_next, task_list, task_update, task_claim, task_release, human_review_create, human_review_pending, human_review_decide |
| Skoll | 14 | context_start, context_end, skill_list, skill_load, skill_search, skill_verify, rule_list, rule_check, rule_get, rule_create_or_reuse, team_create, team_get, agent_list, agent_get |
| Tyr | 14 | pkg_check, pkg_license, pkg_audit, pkg_audit_snapshot, sast_run, sast_findings, sast_resolve, standard_run, standard_run_all, standard_list, tyr_snapshot, tyr_bootstrap, precommit_validate, quality_gate |
| Meta | 3 | ragnarok_help, ragnarok_status, session_context_full |
| Workflows | 6 | workflow_project_lifecycle, workflow_onboard_existing, workflow_prd_analyze, workflow_checkpoint_create, workflow_plan_develop_v2, workflow_team_setup |
| **Total** | **76** | |

### Correcciones de Bugs Críticos

```go
// BUG-1: Crash en handleWorkflowSessionStart (nil pointer)
// Archivo: internal/mcp/unified/workflow_handlers.go ~L396
// Error: err.Error() llamado sobre err == nil cuando step tiene éxito
// Fix:
if err != nil {
    stepError = err.Error()  // solo si err != nil
}

// BUG-2: serverVersion hardcoded incorrecto
// Archivo: internal/mcp/unified/server.go ~L85
serverVersion = "2.4.11"  // era "1.4.0"

// BUG-3: notifications/initialized no manejado
// Agregar en el switch de HandleNotification:
case "notifications/initialized":
    return nil  // silenciosamente ignorar, es válido en MCP
```

### Descripciones MCP Mejoradas

Formato obligatorio para todas las herramientas:
```
"[CUÁNDO usar]. [QUÉ hace exactamente]. [QUÉ retorna]. [Parámetros obligatorios]: X, Y."
```

Ejemplo (antes / después):
```
ANTES: "mem_save: Save an observation to memory"
DESPUÉS: "mem_save: Usa después de completar trabajo significativo (bugfixes, decisiones, 
          refactors). Guarda qué pasó, por qué y qué se aprendió para recuperación futura. 
          Retorna observation_id. Requeridos: type, title, content."
```

### Instrucciones MCP en `initialize`
```go
serverInstructions = `INICIO OBLIGATORIO: Llama ragnarok_help() para entender el sistema.

PROYECTO NUEVO: workflow_project_lifecycle({ project_path, prd_file })
PROYECTO EXISTENTE: workflow_onboard_existing({ project_path })
RETOMAR TRABAJO: session_context_full() → revela estado actual completo

CICLO DE TRABAJO:
  task_get_next({plan_id}) → context_start({role}) → task_claim({task_id})
  → [trabajar] → mem_save({...}) → task_update({status:"completed"})

CALIDAD: quality_gate({path}) antes de cada human_review_create
HUMAN REVIEW: Revisar human_review_pending() regularmente`
```

---

## 9. Workflows

### WF-1: `workflow_project_lifecycle` (Proyecto Nuevo)
```
Parámetros: project_path, prd_file, title?
Pasos:
  1. project_scan(project_path)          → detecta stack existente si hay código
  2. prd_parse(prd_file)                 → extrae requisitos
  3. plan_create_from_prd(...)           → crea plan con fases y tareas
  4. skill_generate(project_path)        → genera skills para el stack detectado
  5. rules_generate(project_path)        → genera reglas de gobernanza
  6. tyr_bootstrap(project_path)         → configura estándares de calidad
  7. human_review_create(plan_id)        → solicita aprobación del usuario
  8. sast_run(project_path)              → baseline de seguridad inicial
Retorna: { plan_id, phases, tasks, human_review_id, next_step }
```

### WF-2: `workflow_onboard_existing` (Proyecto Existente) — NUEVO
```
Parámetros: project_path, goal?
Pasos:
  1. project_scan(project_path)          → analiza stack, estructura, patterns
  2. mem_project_summary(project_path)   → busca si hay memoria previa del proyecto
  3. spec_save(architecture findings)    → documenta la arquitectura encontrada
  4. skill_generate(project_path)        → genera skills para el stack
  5. rules_generate(project_path)        → reglas según best practices del stack
  6. tyr_bootstrap(project_path)         → estándares según el stack detectado
  7. sast_run(project_path)              → auditoría de seguridad inicial
  8. [Sin plan automático — el agente crea el plan según el objetivo]
Retorna: { scan_result, specs_created, skills_created, rules_created, next_step }
```

### WF-3: `workflow_checkpoint_create` (Checkpoint de Calidad)
```
Parámetros: plan_id, phase_id?
Pasos:
  1. quality_gate(project_path, "full")  → verifica calidad completa
  2. mem_project_summary(project_path)   → resumen de lo hecho
  3. human_review_create({              → solicita revisión humana
       plan_id, question, context })
Retorna: { quality_result, review_id, quality_passed }
```

### WF-4: `workflow_prd_analyze` (Solo Análisis de PRD)
```
Parámetros: prd_file, project_path, plan_title?
Pasos:
  1. prd_parse(prd_file)
  2. prd_requirements_extract(prd_id)
  3. plan_create_from_prd(...)
  4. human_review_create(plan_id)
Retorna: { plan_id, requirements_count, phase_count, task_count, review_id }
```

---

## 10. Herramientas Meta

### `ragnarok_help` — Punto de Entrada Obligatorio

```json
{
  "version": "3.0.0",
  "description": "Ecosistema MCP para gobernanza de agentes de IA",
  "quick_start": {
    "new_project": "workflow_project_lifecycle({ project_path, prd_file })",
    "existing_project": "workflow_onboard_existing({ project_path })",
    "resume_work": "session_context_full()"
  },
  "modules": {
    "fenrir": "Memoria persistente — mem_save/mem_find",
    "hati": "Planning y tareas — task_get_next/task_update",
    "skoll": "Skills y reglas — skill_load/rule_check",
    "tyr": "Calidad — quality_gate/sast_run"
  },
  "work_cycle": [
    "task_get_next({plan_id})",
    "context_start({role})",
    "task_claim({task_id})",
    "[ejecutar la tarea]",
    "mem_save({type, title, content})",
    "task_update({task_id, status:'completed'})",
    "→ repetir hasta plan_complete"
  ]
}
```

### `session_context_full` — Estado Completo en 1 Llamada

```json
{
  "active_plan": { "id", "title", "phase_current", "progress_pct" },
  "next_task": { "id", "title", "description", "phase", "priority" },
  "pending_reviews": [{ "review_id", "question", "created_at" }],
  "recent_memory": [{ "title", "type", "created_at" }],
  "context_registered": [{ "context_id", "role", "started_at" }],
  "ecosystem_health": {
    "fenrir": "healthy",
    "hati": "healthy",
    "skoll": "healthy",
    "tyr": "healthy"
  },
  "recommended_action": "Llama task_claim({task_id}) para tomar la siguiente tarea"
}
```

### `ragnarok_status` — Health Check

```json
{
  "version": "3.0.0",
  "tools_registered": 76,
  "tools_with_schema": 76,
  "tools_with_description": 76,
  "databases": {
    "fenrir": { "healthy": true, "observations": 142 },
    "hati": { "healthy": true, "active_plans": 1, "pending_tasks": 8 },
    "skoll": { "healthy": true, "skills": 12, "rules": 7 },
    "tyr": { "healthy": true, "standards": 5, "sast_findings": 2 }
  },
  "issues": []
}
```

---

## 11. Calidad del Código

### Tests de Integración Requeridos

```go
// internal/mcp/unified/integration_test.go

func TestMCPHandshake(t *testing.T)           // initialize → tools/list sin error
func TestAllToolsHaveSchemas(t *testing.T)    // JSON schema válido para cada tool
func TestAllToolsHaveDescriptions(t *testing.T) // descripción >80 chars
func TestNoGhostHandlers(t *testing.T)        // cada tool ejecutable sin panic
func TestRagnarokHelp(t *testing.T)           // ragnarok_help retorna estructura válida
func TestSessionContextFull(t *testing.T)     // session_context_full no crashea
func TestWorkflowOnboardExisting(t *testing.T) // onboarding con proyecto de prueba
func TestTaskClaimRelease(t *testing.T)       // claim → release → claim de otra sesión
func TestQualityGate(t *testing.T)            // quality_gate basic en proyecto test
func TestContextStartEnd(t *testing.T)        // context lifecycle completo
```

### Comando `rag doctor` Mejorado

```bash
$ rag doctor
✅ Fenrir DB: healthy (142 observaciones)
✅ Hati DB: healthy (1 plan activo, 8 tareas pendientes)
✅ Skoll DB: healthy (12 skills, 7 reglas)
✅ Tyr DB: healthy (5 estándares)
✅ Tools registradas: 76/76
✅ Tools con schema: 76/76
✅ Tools con desc >80 chars: 76/76
✅ Sin ghost handlers detectados
✅ Tests de integración: 10/10 passing
⚠️  WARNINGS: 0
❌ ERRORS: 0

Estado: HEALTHY ✅
```

### Makefile Final

```makefile
build:        go build -o rag ./cmd/rag
build-all:    compila fenrir, hati, skoll, tyr, rag
test:         go test -race ./...
doctor:       rag doctor
clean:        elimina *.exe del root + dist/
release:      goreleaser release
lint:         golangci-lint run
```

### CI/CD (GitHub Actions)

```yaml
# .github/workflows/ci.yml — en cada PR
- go test -race ./...
- go vet ./...
- rag doctor
- golangci-lint run

# .github/workflows/release.yml — en tags v*
- goreleaser release (publica: fenrir, hati, skoll, tyr, rag)
```

---

## 12. Configuración de IDEs

### OpenCode
```json
{ "mcp": { "ragnarok": { "type": "local", "command": ["rag", "mcp"], "enabled": true } } }
```

### Claude Code / Cursor / Windsurf / Antigravity
```json
{ "mcpServers": { "ragnarok": { "command": "rag", "args": ["mcp"] } } }
```

### Solo módulos específicos
```json
{
  "mcpServers": {
    "fenrir": { "command": "fenrir", "args": ["mcp"] },
    "tyr": { "command": "tyr", "args": ["mcp"] }
  }
}
```

---

## 13. Plan de Ejecución

### Fase 0 — Preparación (2-3 días) [COMPLETA ✅]
- [x] Eliminar 25+ `.exe` del root del repo
- [x] Actualizar `.gitignore`
- [x] Verificar que `go test -race ./...` pasa como baseline

### Fase 1 — Estabilizar (1 semana) [COMPLETA ✅]
- [x] Corregir BUG-1 (crash workflow_session_start)
- [x] Corregir BUG-2 (serverVersion → Centralizado en `internal/version`)
- [x] Corregir BUG-3 (notifications/initialized)
- [x] Renombrar `quality_snapshot → tyr_snapshot`, `bootstrap_import → tyr_bootstrap`
- [x] Eliminar `quality_snapshot` y `spec_impact` de Hati server
- [x] Mejorar descripciones de las 62 tools core
- [x] Implementar `ragnarok_help` con estructura completa
- [x] Tests: `TestMCPHandshake`, `TestAllToolsHaveDescriptions`

### Fase 2 — Rediseñar Skoll (1-2 semanas) [COMPLETA ✅]
- [x] Agregar `context_start` / `context_end`
- [x] Eliminar 15+ handlers ficticios
- [x] Simplificar `AgentTypes` → `ContextRoles`
- [x] Eliminar `DefaultToolsByType`
- [x] Eliminar tablas `task_executions`, `agent_tasks`, `workflows` de Skoll DB
- [x] Actualizar servidor unificado
- [x] Tests: `TestContextStartEnd`

### Fase 3 — Mejorar Hati y Fenrir (1 semana) [EN PROGRESO 🚧]
- [x] Eliminar 4 tablas muertas de Hati
- [x] Agregar columnas a `task_agents`
- [x] Implementar `plan_get_active`, `task_claim`, `task_release`
- [x] Mejorar `task_get_next` con contexto completo
- [x] Eliminar `internal/fenrir/graph/`
- [ ] Implementar `mem_project_summary`
- [x] Implementar `quality_gate` en Tyr
- [x] Tests: `TestTaskClaimRelease`, `TestQualityGate`

### Fase 4 — Workflows + Standalones + CI/CD (1 semana) [PENDIENTE ⏳]
- [ ] Implementar `workflow_onboard_existing`
- [x] Verificar standalones: `fenrir mcp`, `tyr mcp`, `hati mcp`
- [x] Mejorar `session_context_full`
- [x] Mejorar `ragnarok_status`
- [/] Mejorar `rag doctor` (En progreso conforme se añaden checks)
- [ ] GitHub Actions CI/CD
- [/] goreleaser para los 5 binarios (en proceso de configuración)
- [ ] Tests: `TestWorkflowOnboardExisting`, `TestSessionContextFull`

---

## 14. Criterios de Aceptación

| Criterio | Métrica |
|----------|---------|
| 0 crashes conocidos | `go test -race ./...` pasa |
| 0 ghost handlers | `TestNoGhostHandlers` pasa |
| Tools 100% documentadas | Descripción >80 chars en cada tool |
| Flujo nuevo proyecto <5 min | `workflow_project_lifecycle` completo |
| Flujo onboarding <3 min | `workflow_onboard_existing` completo |
| Agente nuevo lista en <60s | `ragnarok_help` + `session_context_full` suficientes |
| Standalones funcionales | `fenrir mcp`, `tyr mcp` arrancan solos |
| CI verde | GitHub Actions pasa en cada PR |
| `rag doctor` healthy | 0 warnings, 0 errors |
| Tests integración | ≥10 tests pasando |

---

## Apéndice: Herramientas por Módulo (Inventario Completo)

### Eliminadas definitivamente
```
# De Skoll:
agent_heartbeat, agent_activate, agent_assign_task, agent_complete_task
agent_handoff, agent_context, task_execute, task_delegate, task_heartbeat
task_complete (Skoll), task_cancel (Skoll), workflow_deprecate
skoll_validate, agent_specialized_list (merge con agent_list)

# De Hati:
quality_snapshot (mover a Tyr como tyr_snapshot)
spec_impact (mover a Fenrir)

# De Fenrir:
project_bootstrap, skill_generate, rules_generate, standards_generate
(mover a workflows)

# Workflows deprecated:
workflow_project_bootstrap, workflow_agentic_init, workflow_plan_develop
```

### Nuevas herramientas
```
mem_project_summary     (Fenrir)
plan_get_active         (Hati)
task_claim              (Hati)
task_release            (Hati)
context_start           (Skoll)
context_end             (Skoll)
quality_gate            (Tyr)
workflow_onboard_existing (Unified)
```
