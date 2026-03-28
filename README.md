# Ragnarok Ecosystem v2.0.0

**AI Governance & Autonomous Development Ecosystem**

Sistema agentico de 4 plugins MCP diseñados para orchestrar agentes AI en proyectos de desarrollo, con **Agent-Based Orchestration** y validación humana en puntos clave.

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
    end
    
    HATI --> |"Tareas Multi-Agente"| SKOLL
    
    subgraph SKOLL["⚙️ SKOLL - Orchestration"]
        Agents[Agentes]
        TaskExec[Task Executions]
        Skills[Skills]
        Rules[Rules]
    end
    
    SKOLL --> |"Memoria"| FENRIR
    FENRIR --> |"Contexto"| SKOLL
    
    subgraph FENRIR["🧠 FENRIR - Memory"]
        Context[Contexto]
        Decisions[Decisiones]
        Timeline[Línea de Tiempo]
    end
    
    FENRIR --> TYR
    
    subgraph TYR["🛡️ TYR - Quality"]
        Standards[Standards]
        SAST[SAST]
        Precommit[Pre-commit]
    end
    
    TYR --> |"Validación"| HumanReview
    HumanReview --> |"Approval"| HATI
```

---

## Workflows de Alto Nivel

En lugar de múltiples llamadas MCP, Ragnarok ofrece **workflows** que executan todo internamente:

### 1. `workflow_stack_based_init` ⭐ RECOMENDADO
Inicializa proyecto detectando stack automáticamente y creando fases/tareas apropiadas:

```bash
workflow_stack_based_init --project_path "./mi-proyecto" --title "MiApp"
```

**Ejecuta internamente:**
- `project_scan` → Detecta stack (Go, Node, Python, etc.), arquitectura, CI/CD
- `plan_create` → Crea plan basado en el stack detectado
- `phase_create` → Crea fases según stack (Setup, Backend, Frontend, Database, Testing, DevOps, Docs)
- `task_create` → Crea tareas específicas del stack
- `task_assign_agents` → Asigna agentes según tipo de tarea
- `human_review_create` → Solicita approval humano

**Fases generadas según stack:**
| Stack | Fases |
|-------|-------|
| Go | Setup, Backend, API, Database, Testing, DevOps, Documentation |
| Node/React | Setup, Frontend, API, Database, Testing, DevOps, Documentation |
| Python | Setup, Backend, API, Database, Testing, DevOps, Documentation |
| Multi-stack | Setup, Backend, Frontend, Database, Testing, DevOps, Documentation |

---

### 2. `workflow_plan_develop_v2` ⭐ RECOMENDADO
Ejecuta el desarrollo con delegación multi-agente:

```bash
workflow_plan_develop_v2 --plan_id "plan_xxx" --auto_continue true
```

**Flujo autónomo:**
```
while (tareas_pendientes) {
    task = task_get_next(plan_id, agent_id)
    if (task.tiene_agentes) {
        for each agente in task.agentes {
            task_execute(task_id, agente.id)  // Delega a Skoll
        }
    } else {
        task_update(status: "in_progress")
    }
    
    if (is_milestone) {
        checkpoint_create
        human_review_create  // Approval antes de continuar
    }
}
```

---

### 3. `workflow_prd_analyze` [DEPRECATED]
Analiza un PRD y crea el plan de desarrollo:

```bash
workflow_prd_analyze --prd_file "./PRD.md" --project_path "./mi-proyecto"
```

**Nota:** Ahora incluye `project_scan` para detección de stack.

---

### 4. `workflow_agentic_init` [DEPRECATED]
Crea la estructura agentica completa:

```bash
workflow_agentic_init --title "MiApp" --phases "Backend,Frontend,Testing,Deploy"
```

**Recomendación:** Usar `workflow_stack_based_init` en su lugar.

---

### 5. `workflow_plan_develop` [DEPRECATED]
Usar `workflow_plan_develop_v2` para soporte multi-agente.

---

### 6. `workflow_checkpoint_create`
Crea checkpoint de calidad:

```bash
workflow_checkpoint_create --plan_id "plan_xxx" --description "Milestone 1"
```

**Ejecuta internamente:**
- `checkpoint_open`
- `standard_run_all`
- `sast_run`
- `precommit_validate`
- `human_review_create` → Decision humana

---

## Human-in-the-Loop

Puntos donde se requiere validación humana:

| Punto | Tipo | Descripción |
|-------|------|-------------|
| Post PRD | `prd_approval` | "¿Aprobar este plan?" |
| Team Setup | `team_approval` | "¿Asignar agentes a fases?" |
| Post Phase | `phase_approval` | "¿Avanzar a siguiente fase?" |
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

### PRD → Plan → Phase → Task

```mermaid
graph TD
    PRD[📄 PRD] --> Plan[📋 Plan]
    Plan --> Phase1[Phase: Backend]
    Plan --> Phase2[Phase: Frontend]
    Plan --> Phase3[Phase: Testing]
    
    Phase1 --> Task1[Tarea: API Users]
    Phase1 --> Task2[Tarea: Auth]
    Phase2 --> Task3[Tarea: Login UI]
    Phase2 --> Task4[Tarea: Dashboard]
    
    Task1 --> |"Skoll Agent"| Result1[✅ Completado]
    Task2 --> |"Skoll Agent"| Blocker[🚧 Blocker]
```

---

## Instalación

```powershell
irm https://raw.githubusercontent.com/andragon31/Ragnarok/v2.0.1/install.ps1 | iex
```

## Uso Rápido

```bash
# 1. Inicializar proyecto con detección automática de stack (RECOMENDADO)
workflow_stack_based_init --project_path "./mi-proyecto" --title "MiApp"

# 2. Analizar PRD y crear plan (con detección de stack)
workflow_prd_analyze --prd_file "./PRD.md" --project_path "./mi-proyecto"

# 3. Ejecutar desarrollo con multi-agente (RECOMENDADO)
workflow_plan_develop_v2 --plan_id "plan_xxx" --auto_continue true
```

### Novedades v2.0.1

- **Tests**: Cobertura de tests para task handlers y workflow handlers
- **Workflow PRD Analyze Actualizado**: Ahora incluye detección de stack
- **Workflows Deprecated**: Marcas deprecación en workflows antiguos
- **Gitignore**: Archivos de release ignorados

### Novedades v2.0.0

- **Multi-Agent Tasks**: Las tareas pueden asignarse a múltiples agentes simultáneamente
- **Agent-Based Orchestration**: Skoll delega directamente a agentes sin workflows
- **Task Executions**: Tracking granular de cada ejecución de tarea por agente
- **Workflows Deprecated**: Los workflows se reemplazan por task_* commands

---

## Herramientas Base (para uso granular)

| Plugin | Herramientas |
|--------|-------------|
| **Fenrir** | `mem_stats`, `mem_timeline`, `mem_context`, `mem_find`, `mem_save` |
| **Hati** | `plan_get`, `plan_list`, `task_create`, `task_get_next`, `task_update`, `task_assign_agents`, `task_agent_update` |
| **Skoll** | `skill_list`, `agent_list`, `agent_create`, `team_register`, `task_execute`, `task_delegate`, `task_status` |
| **Tyr** | `standard_run_all`, `sast_run`, `pkg_check`, `precommit_validate` |

---

## Ejemplo Completo

```bash
# 1. Crear PRD.md con requisitos...

# 2. Inicializar proyecto con detección automática de stack
workflow_stack_based_init --project_path "./mi-proyecto" --title "MiApp"

# 3. El agente recibe notification de approval
#    Usuario aprueba via: human_review_decide --review_id "xxx" --decision "approved"

# 4. Ejecutar desarrollo con multi-agente
workflow_plan_develop_v2 --plan_id "plan_xxx" --auto_continue true

# 5. En cada checkpoint, el agente notifica al humano
#    Usuario approves via: human_review_decide --review_id "yyy" --decision "approved"

# 6. Al final, usuario approves deploy
```

---

**v2.0.1** - Agent-Based Orchestration con Multi-Agent Tasks
