# PRD — Hati v1.0.0
## Product Requirements Document — Versión 1.0.0

**Producto:** Hati  
**Versión:** 1.0.0  
**Tipo:** MCP Plugin — Task Planning & Human-in-the-Loop Layer  
**Lenguaje:** Go 1.22+  
**Licencia:** MIT  
**Fecha:** Marzo 2026

---

## Historial de versiones

| Versión | Cambios principales |
|---|---|
| v1.0.0 | **Versión inicial unificada.** Ciclo PRE→execute→POST, Rejection Loop, Approval Record, granularidad dinámica, Quality Snapshot, MEP, Plan Completeness, Explainability Report, Plan Quality Score, E2E Standards, Agent Reliability, Learning Mode, Plan ID en commits, AGENTS.md anidados, Spec Delta, Module Hints, Async Checkpoints, Quality Express, Plan Templates, Async Handoff. |

---

## Qué cambia en v1.0.0

| Mejora | Origen | Impacto |
|---|---|---|
| Detección de AGENTS.md anidados para ajustar granularidad | AGENTS.md estándar | Fases más precisas en monorepos |
| Spec Delta en checkpoint PLAN | OpenSpec + módulo Fenrir Specs | Developer entiende qué requisitos cambia el plan |
| Quality Snapshot ahora viene de Tyr (no de Fenrir) | Separación Fenrir/Tyr | Arquitectura más limpia |
| Module hints desde Skoll (AGENTS.md con testing/security notes) | AGENTS.md + Skoll v1.0.0 | Riesgo ajustado automáticamente por contexto del módulo |

---

## Tabla de contenidos

1. [Visión del producto](#1-visión-del-producto)
2. [Conceptos principales](#2-conceptos-principales)
3. [Requisitos funcionales](#3-requisitos-funcionales)
4. [AGENTS.md anidados → ajuste de granularidad](#4-agentsmd-anidados--ajuste-de-granularidad)
5. [Spec Delta en checkpoint PLAN](#5-spec-delta-en-checkpoint-plan)
6. [Quality Snapshot desde Tyr](#6-quality-snapshot-desde-tyr)
7. [Module Hints desde Skoll](#7-module-hints-desde-skoll)
8. [Modelo de datos v1.0.0](#8-modelo-de-datos-v100)
9. [MCP Tools — Catálogo completo v1.0.0](#9-mcp-tools--catálogo-completo-v100)
10. [CLI completo v1.0.0](#10-cli-completo-v100)
11. [Configuración v1.0.0](#11-configuración-v100)
12. [Integración con Fenrir, Tyr y Skoll](#12-integración-con-fenrir-tyr-y-skoll)
13. [El flujo completo v1.0.0](#13-el-flujo-completo-v100)
14. [Roadmap v1.0.0](#14-roadmap-v100)

---

## 1. Visión del producto

Hati garantiza que el developer aprueba cada fase con información completa. La v1.0.0 proporciona granularidad del plan ajustada automáticamente al contexto real del módulo (incluyendo instrucciones del AGENTS.md local), y el checkpoint PLAN muestra exactamente qué requisitos del sistema cambian — no solo qué código se va a escribir.

> **Misión v1.0.0:** Que el plan refleje el contexto real del módulo, que las aprobaciones sean informadas por requisitos concretos, y que la ejecución sea trazable al primer principio.

Hati integra con Tyr para Quality Snapshots, Fenrir para contexto y specs, y Skoll para module hints y workflow coordination.

---

## 2. Conceptos principales

- **Ciclo PRE→execute→POST** — aprobación antes y después de cada fase
- **Rejection Loop** — protocolo de rechazo con feedback estructurado
- **Approval Record** — audit trail completo de todas las decisiones humanas
- **Granularidad dinámica por riesgo** — el tamaño de las fases depende del nivel de riesgo
- **Quality Snapshot** — resultados de Standards antes de cada checkpoint POST
- **MEP (Minimum Engagement Protocol)** — tiempo mínimo de engagement por nivel de riesgo
- **Plan Completeness Score** — verificación de que el plan es completo
- **Explainability Report** — `why_this_approach` obligatorio en `phase_report`
- **Plan Quality Score** — score consolidado al cerrar el plan
- **E2E Standards** — campo `run_on` para tests selectivos por checkpoint
- **Agent Reliability Integration** — granularidad ajustada por reliability score de Fenrir
- **Learning Mode** — preguntas de comprensión antes de aprobar
- **Plan ID en commits** — trazabilidad entre código y Approval Record
- **AGENTS.md Context-Aware Planning** — granularidad ajustada por AGENTS.md local
- **Spec Delta** — información de requisitos afectados en checkpoint PLAN
- **Module Hints** — ajustes de riesgo desde Skoll
- **Async Checkpoints** — checkpoints no bloqueantes
- **Quality Snapshot Express** — output compacto de Quality Snapshot
- **Plan Templates** — generación de planes sin LLM externo
- **Async Handoff** — transferencia no bloqueante entre agentes
- **AGENTS.md Context-Aware Planning** — granularidad ajustada por AGENTS.md local
- **Spec Delta** — información de requisitos afectados en checkpoint PLAN
- **Module Hints** — ajustes de riesgo desde Skoll

---

## 3. Requisitos funcionales

### RF-01 — AGENTS.md Context-Aware Planning

| ID | Requisito | Prioridad |
|---|---|---|
| RF-01-01 | `plan_create` debe consultar Skoll sobre AGENTS.md anidados en los módulos del plan | MUST |
| RF-01-02 | Si un módulo tiene AGENTS.md con instrucciones de testing, agregar un standard E2E al checkpoint POST de esa fase | MUST |
| RF-01-03 | Si un módulo tiene AGENTS.md con notas de seguridad/compliance, subir el risk level de esa fase | MUST |
| RF-01-04 | Los ajustes por AGENTS.md local deben ser visibles en el checkpoint PLAN con su justificación | MUST |
| RF-01-05 | El developer debe poder sobreescribir los ajustes automáticos antes de aprobar el plan | MUST |
| RF-01-06 | Si Skoll no está disponible, Hati puede leer AGENTS.md directamente si tiene acceso al filesystem | SHOULD |

### RF-02 — Spec Delta en checkpoint PLAN

| ID | Requisito | Prioridad |
|---|---|---|
| RF-02-01 | `plan_create` debe consultar `fenrir.spec_check` con el cambio propuesto antes de generar el plan | MUST |
| RF-02-02 | Los specs afectados deben incluirse en el checkpoint PLAN si Fenrir está disponible | MUST |
| RF-02-03 | Para cada spec afectado, mostrar: título, tipo de impacto (implements/extends/modifies), detalle | MUST |
| RF-02-04 | Los specs de tipo `violated` deben mostrarse como alertas críticas en el checkpoint PLAN | MUST |
| RF-02-05 | Al completar el plan, llamar `fenrir.spec_delta` para registrar el delta de specs | MUST |
| RF-02-06 | El Approval Record debe incluir los spec deltas del plan completado | SHOULD |
| RF-02-07 | Si Fenrir no está disponible, el checkpoint PLAN funciona sin spec delta (degradación graceful) | MUST |

### RF-03 — Quality Snapshot desde Tyr

| ID | Requisito | Prioridad |
|---|---|---|
| RF-03-01 | `checkpoint_open type:post` debe llamar a Tyr en lugar de Fenrir para el Quality Snapshot | MUST |
| RF-03-02 | Tyr expone `standard_run_all` con el contexto del checkpoint (tipo, risk_level) | MUST |
| RF-03-03 | Si Tyr no está disponible pero Fenrir sí, usar el legacy standard_run_all de Fenrir como fallback | SHOULD |
| RF-03-04 | Si ninguno está disponible, abrir el checkpoint sin Quality Snapshot con nota | MUST |

### RF-04 — Module Hints desde Skoll

| ID | Requisito | Prioridad |
|---|---|---|
| RF-04-01 | Hati debe aceptar `module_hint` de Skoll como input a `plan_create` | MUST |
| RF-04-02 | Un hint de tipo `upgrade_risk` debe subir el risk level de la fase afectada | MUST |
| RF-04-03 | Un hint de tipo `consider_e2e_standard` debe agregar un standard E2E al checkpoint POST de la fase | MUST |
| RF-04-04 | Un hint de tipo `compliance_note` debe aparecer como advertencia en el checkpoint PRE | SHOULD |
| RF-04-05 | Los module hints deben registrarse en el Approval Record para trazabilidad | SHOULD |

---

## 4. AGENTS.md anidados → ajuste de granularidad

### El flujo de `plan_create` v1.0.0

```go
// internal/engine/planner.go v1.0.0

func (p *Planner) CreatePlan(request string, ctx *PlanContext) (*Plan, error) {
    // 1. Análisis base (igual que v3.0)
    phases := p.decompose(request, ctx)

    // 2. Consultar reliability score de Fenrir (v3.0)
    if p.fenrirClient.Available() {
        phases = p.adjustByReliability(phases)
    }

    // 3. NUEVO v4.0: Consultar module hints de Skoll
    if p.skollClient.Available() {
        hints := p.skollClient.GetModuleHints(p.extractModules(phases))
        phases = p.adjustByModuleHints(phases, hints)
    }

    // 4. NUEVO v4.0: Consultar specs afectados de Fenrir
    specImpact := &SpecImpact{}
    if p.fenrirClient.Available() {
        specImpact, _ = p.fenrirClient.SpecCheck(request)
    }

    return &Plan{
        Phases:      phases,
        SpecImpact:  specImpact,
        ModuleHints: hints,
    }, nil
}

func (p *Planner) adjustByModuleHints(phases []Phase, hints []ModuleHint) []Phase {
    for i, phase := range phases {
        for _, hint := range hints {
            if pathMatchesPhase(hint.Module, phase.PrimaryModule) {
                switch hint.Action {
                case "upgrade_risk":
                    phases[i].RiskLevel = upgradeRisk(phase.RiskLevel)
                    phases[i].Adjustments = append(phases[i].Adjustments, PlanAdjustment{
                        Source:  "agents_md_local",
                        Reason:  hint.Message,
                        Applied: fmt.Sprintf("Risk upgraded to %s", phases[i].RiskLevel),
                    })
                case "consider_e2e_standard":
                    phases[i].SuggestedStandards = append(
                        phases[i].SuggestedStandards,
                        "e2e-integration",
                    )
                }
            }
        }
    }
    return phases
}
```

### Ejemplo de checkpoint PLAN con ajustes por AGENTS.md

```
✋ Checkpoint PLAN — "Implementar procesamiento de pagos con Stripe"

📋 PLAN (4 fases)

  Fase 1  Análisis de dominio         [low]
  Fase 2  Implementación del core     [CRITICAL] ← subido de HIGH
           📝 Razón: packages/payments/AGENTS.md indica que este módulo
              requiere PCI compliance y pruebas de integración obligatorias
  Fase 3  Contrato de API             [high]
  Fase 4  Tests + commit              [medium] ← subido de LOW
           📝 Razón: packages/payments/AGENTS.md requiere e2e tests
              antes de cualquier cambio en pagos

📊 PLAN COMPLETENESS: 0.92 / 1.0
  ✅ Fase de testing incluida
  ✅ Rollback para fase critical
  ✅ Handoffs cubiertos

📋 SPECS AFECTADOS (de Fenrir)
  ✅ payments/spec.md → "Payment Processing" [implements]
     "Este plan implementa el requisito de procesamiento con Stripe"
  ⚠️  payments/spec.md → "PCI Compliance" [extends]
     "Considera agregar escenario para tokenización de tarjetas"

📊 AGENT RELIABILITY en /payments/: 0.71 ⚠️
  Basado en 6 sesiones — fases critical ajustadas

¿Apruebas este plan?
```

---

## 5. Spec Delta en checkpoint PLAN

### `spec_check` response de Fenrir

```json
{
  "affected_specs": [
    {
      "id": "spec-payments-001",
      "capability": "payments",
      "title": "Payment Processing",
      "impact": "implements",
      "detail": "Este plan implementa el requisito de procesamiento con Stripe como provider",
      "status": "active"
    },
    {
      "id": "spec-payments-002",
      "capability": "payments",
      "title": "PCI Compliance",
      "impact": "extends",
      "detail": "Stripe tokenización es compatible con PCI DSS Level 1",
      "status": "active"
    }
  ],
  "violated_specs": [],
  "new_specs_suggested": [
    "Considera documentar el flujo de webhook de Stripe como spec"
  ]
}
```

### `spec_delta` al completar el plan

```go
// Llamado automáticamente en checkpoint_open type:final (approved)

func (h *Hati) onPlanComplete(plan *Plan) {
    if !h.fenrirClient.Available() { return }

    delta := &SpecDeltaInput{
        PlanID:      plan.ID,
        PlanTitle:   plan.Title,
        Implemented: plan.SpecImpact.Implemented(),
        Extended:    plan.SpecImpact.Extended(),
        NewSpecs:    plan.SpecImpact.Suggested(),
    }

    result, err := h.fenrirClient.SpecDelta(delta)
    if err != nil { return }

    // Agregar al Approval Record
    h.store.AddToApprovalRecord(plan.ID, ApprovalEntry{
        Type:    "spec_delta",
        Content: result.Summary,
    })
}
```

---

## 6. Quality Snapshot desde Tyr

### Actualización del cliente de Quality Snapshot

```go
// internal/clients/quality.go v4.0

type QualityClient interface {
    RunAll(checkpointType, riskLevel string) (*QualitySnapshot, error)
}

// Implementación con Tyr (preferido)
type TyrQualityClient struct {
    tyrClient *TyrMCPClient
}

func (c *TyrQualityClient) RunAll(checkpointType, riskLevel string) (*QualitySnapshot, error) {
    return c.tyrClient.StandardRunAll(checkpointType, riskLevel)
}

// Fallback a Fenrir si Tyr no está disponible
type FenrirQualityFallback struct {
    fenrirClient *FenrirMCPClient
}

func (c *FenrirQualityFallback) RunAll(checkpointType, riskLevel string) (*QualitySnapshot, error) {
    return c.fenrirClient.StandardRunAll(checkpointType, riskLevel)
}

// Selector automático en Hati
func (h *Hati) getQualityClient() QualityClient {
    if h.tyrClient != nil && h.tyrClient.Available() {
        return &TyrQualityClient{h.tyrClient}
    }
    if h.fenrirClient != nil && h.fenrirClient.Available() {
        return &FenrirQualityFallback{h.fenrirClient}
    }
    return nil // checkpoint se abre sin Quality Snapshot
}
```

---

## 7. Module Hints desde Skoll

### Estructura del ModuleHint

```go
// internal/clients/skoll_hints.go

type ModuleHint struct {
    Module  string   // path del módulo
    Source  string   // "agents_md_local" | "team_policy" | "fenrir_incident"
    Action  string   // "upgrade_risk" | "consider_e2e_standard" | "compliance_note"
    Message string   // descripción para mostrar al developer
    Data    map[string]string // datos adicionales
}
```

### Fuentes de ModuleHints

```
AGENTS.md local detectado por Skoll:
  → Instrucciones de testing  → hint: consider_e2e_standard
  → Notas de seguridad/PCI    → hint: upgrade_risk + compliance_note
  → Instrucciones de deploy   → hint: compliance_note

Incidents activos de Fenrir:
  → incident en /payments/    → hint: upgrade_risk (ya existía en v3.0)

Team policies de Tyr:
  → policy severity:critical en /auth/ → hint: upgrade_risk
```

---

## 8. Modelo de datos v1.0.0

### Tablas heredadas sin cambios
`plans`, `phases`, `checkpoints`, `feedback`, `approval_record`, `plan_quality_scores`, `commit_registry`

### Campos nuevos en tabla `phases` (v1.0.0)

```sql
ALTER TABLE phases ADD COLUMN agents_md_hints    TEXT;  -- JSON array de ModuleHints aplicados
ALTER TABLE phases ADD COLUMN spec_ids_affected  TEXT;  -- JSON array de spec IDs
```

### Campos nuevos en tabla `plans` (v1.0.0)

```sql
ALTER TABLE plans ADD COLUMN spec_impact       TEXT;   -- JSON: affected_specs, violated_specs
ALTER TABLE plans ADD COLUMN module_hints_used TEXT;   -- JSON array de hints aplicados
ALTER TABLE plans ADD COLUMN quality_source    TEXT DEFAULT 'tyr';  -- tyr | fenrir | none
```

### Tabla nueva: spec_deltas

```sql
CREATE TABLE spec_deltas (
    id          TEXT PRIMARY KEY,
    plan_id     TEXT NOT NULL REFERENCES plans(id),
    spec_id     TEXT NOT NULL,
    delta_type  TEXT NOT NULL,   -- implemented | extended | created | violated
    description TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 9. MCP Tools — Catálogo completo v1.0.0

### Tools de v1.0-v3.0 (con cambios internos en v4.0)

`plan_get`, `plan_revise`, `plan_abandon`

`plan_create` *(v4.0: consulta module hints de Skoll + spec_check de Fenrir)*
`plan_completeness`, `plan_quality`

`checkpoint_open` *(v4.0: Quality Snapshot desde Tyr, spec delta en tipo:plan)*
`checkpoint_decide` *(v4.0: verifica MEP + learning mode)*
`checkpoint_status`

`phase_start` *(v4.0: recibe module hints en el contexto PRE)*
`phase_report`

`feedback_request`, `feedback_receive`, `feedback_escalate`

`record_list`, `record_get`, `record_export`

`hati_status`, `hati_stats`

`hati_commit_info`, `hati_register_commit`

`quality_snapshot` *(v4.0: delega a Tyr primero, fallback a Fenrir)*

`learning_answer`

### Tools nuevas en v4.0 (2 adicionales)

| Tool | Descripción |
|---|---|
| `module_hints` | Consultar module hints activos para los módulos del plan actual |
| `spec_impact` | Ver los specs afectados por el plan activo |

**Total v1.0.0: 26 MCP tools**

### Especificaciones de tools nuevas

#### `module_hints`

```json
{
  "name": "module_hints",
  "description": "Get active module hints from Skoll AGENTS.md and Fenrir incidents for planning",
  "inputSchema": {
    "type": "object",
    "properties": {
      "modules": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Module paths to check for hints"
      }
    }
  }
}
```

**Retorna:**
```json
{
  "hints": [
    {
      "module": "packages/payments/",
      "source": "agents_md_local",
      "action": "upgrade_risk",
      "message": "packages/payments/AGENTS.md: always run integration tests, PCI compliance required",
      "applied_to_phases": ["Implementación del core"]
    },
    {
      "module": "src/modules/auth/",
      "source": "fenrir_incident",
      "action": "upgrade_risk",
      "message": "INC-003 activo: OAuth falla con emails Unicode",
      "applied_to_phases": ["Implementación OAuth"]
    }
  ]
}
```

#### `spec_impact`

```json
{
  "name": "spec_impact",
  "description": "Get specs affected by the active or specified plan",
  "inputSchema": {
    "type": "object",
    "properties": {
      "plan_id": { "type": "string", "description": "Specific plan, or omit for active" }
    }
  }
}
```

---

## 10. CLI completo v1.0.0

```bash
# ─── PLAN ────────────────────────────────────────────
hati plan status [--verbose]
hati plan show <plan_id> [--specs] [--hints]
hati plan abandon <plan_id> [--reason "<razón>"]
hati plan quality <plan_id>
hati plan completeness <plan_id>
hati plan specs <plan_id>           # Ver spec impact del plan
hati plan hints <plan_id>           # Ver module hints aplicados

# ─── HISTORIAL ───────────────────────────────────────
hati record list [--status all|completed|abandoned] [--quality-below 0.7]
hati record show <plan_id> [--explainability] [--learning] [--specs]
hati record export <plan_id> [--format markdown|json]

# ─── COMMITS ─────────────────────────────────────────
hati commit-info [--plan <id>]
hati commit-register <plan_id> <commit_hash>

# ─── ESTADÍSTICAS ────────────────────────────────────
hati stats [--days 30]
hati stats --metric fast_approvals|quality|rejections|learning|reliability|specs

# ─── SISTEMA ─────────────────────────────────────────
hati init [--dry-run]
hati mcp
hati serve [--port 7439]
hati tui
hati version
```

---

## 11. Configuración v1.0.0

### `.hati/config.json`

```json
{
  "project": "mi-proyecto",
  "version": "1.0.0",
  "max_attempts_per_phase": 3,

  "completeness": {
    "min_score_to_warn": 0.7,
    "min_score_to_block": 0.4,
    "require_rollback_for_critical": true,
    "penalize_missing_e2e": true
  },

  "minimum_engagement": {
    "enabled": true,
    "by_risk_level": {
      "low": 0,
      "medium": 10,
      "high": 30,
      "critical": 60
    },
    "fast_approval_threshold_seconds": 5
  },

  "explainability": {
    "min_length": 50,
    "require_alternatives_for_risk": ["high", "critical"]
  },

  "learning_mode": {
    "enabled": false,
    "apply_to_risk_levels": ["high", "critical"],
    "question_min_length": 20
  },

  "reliability": {
    "upgrade_risk_threshold": 0.7,
    "notify_fenrir_on_complete": true
  },

  "e2e": {
    "show_separately_in_snapshot": true,
    "treat_flaky_as_warning": true
  },

  "traceability": {
    "include_plan_id_in_commit": true
  },

  "agents_md": {
    "read_nested": true,
    "auto_adjust_risk_on_security_notes": true,
    "auto_add_e2e_on_testing_instructions": true
  },

  "specs": {
    "check_on_plan_create": true,
    "show_in_plan_checkpoint": true,
    "generate_delta_on_complete": true
  },

  "quality_source": {
    "primary": "tyr",
    "fallback": "fenrir",
    "fail_gracefully": true
  },

  "integrations": {
    "fenrir": {
      "enabled": true,
      "get_context_on_plan_create": true,
      "spec_check_on_plan_create": true,
      "spec_delta_on_complete": true,
      "session_end_on_plan_complete": true
    },
    "tyr": {
      "enabled": true,
      "quality_snapshot_on_post": true,
      "pkg_check_risk_adjustment": true
    },
    "skoll": {
      "enabled": true,
      "activate_agent_on_phase_start": true,
      "get_module_hints_on_plan_create": true,
      "check_agents_md_nested": true,
      "workflow_complete_on_plan_complete": true
    }
  }
}
```

---

## 12. Integración con Fenrir, Tyr y Skoll

### Con Fenrir

| Evento Hati | Interacción Fenrir |
|---|---|
| `plan_create` | `mem_context` + `predict` + `incident_list` + `reliability_score` + `spec_check` ← NEW |
| `phase_start` | `mem_context` del módulo específico |
| `checkpoint_decide (approved)` | `mem_save type:decision` con Explainability |
| `feedback_receive` | `mem_save type:failed_attempt` |
| `plan_complete` | `mem_session_end` + `spec_delta` ← NEW |

### Con Tyr

| Evento Hati | Interacción Tyr |
|---|---|
| `checkpoint_open (post)` | `standard_run_all` con checkpoint_type y risk_level ← FUENTE PRINCIPAL |
| `plan_create` | `pkg_check` si hay paquetes en el plan → risk adjustment |
| Resultado de SAST findings | Subir riesgo de fases que toquen archivos afectados |

### Con Skoll

| Evento Hati | Interacción Skoll |
|---|---|
| `plan_create` | `GetModuleHints` para AGENTS.md anidados ← NEW |
| `phase_start` | `agent_activate` + `rule_list` + `skill_version_check` |
| `checkpoint_open (pre)` | `agent_context` para scope del agente |
| `plan_complete` | `workflow_complete` |

---

## 13. El flujo completo v1.0.0

```
Developer: "Implementar procesamiento de pagos con Stripe"
                      │
                      ▼
          ┌─────────────────────────┐
          │     HATI: plan_create   │
          │                         │
          │ → Fenrir: mem_context   │
          │ → Fenrir: predict       │
          │ → Fenrir: incidents     │
          │ → Fenrir: reliability   │
          │ → Fenrir: spec_check    │ ← NEW v4.0
          │   "Afecta spec:         │
          │    payments/PCI         │
          │    payments/Processing" │
          │                         │
          │ → Skoll: ModuleHints    │ ← NEW v4.0
          │   "payments/AGENTS.md:  │
          │    PCI compliance,      │
          │    e2e tests required"  │
          │                         │
          │ RESULTADO:              │
          │ 4 fases                 │
          │ Fase 2: CRITICAL        │ ← subido por AGENTS.md
          │ Fase 4: MEDIUM          │ ← subido por AGENTS.md
          │ completeness: 0.92      │
          └──────────┬──────────────┘
                     │
          ✋ checkpoint_open type:plan
             (con spec impact + module hints)
                     │
          Developer aprueba
                     │
          ┌──────────▼──────────────┐
          │       POR FASE          │
          │                         │
          │ checkpoint_open PRE     │
          │  + reliability context  │
          │  + module hints activos │ ← NEW v4.0
          │                         │
          │ phase_start             │
          │  → Skoll: activate      │
          │  → Fenrir: context      │
          │                         │
          │ Agente trabaja          │
          │                         │
          │ phase_report            │
          │  + why_this_approach    │
          │  → Fenrir: mem_save     │
          │                         │
          │ checkpoint_open POST    │
          │  + Quality Snapshot     │
          │    (desde Tyr ← NEW)    │ ← NEW v4.0
          │    unit + E2E + SAST    │
          │  + Reliability          │
          │  + Learning question    │
          │                         │
          └──────────┬──────────────┘
                     │
          ✋ checkpoint_open type:final
                     │
          Developer aprueba
                     │
          ┌──────────▼──────────────┐
          │     plan_complete       │
          │                         │
          │ → Fenrir: session_end   │
          │ → Fenrir: spec_delta    │ ← NEW v4.0
          │ → Fenrir: reliability   │
          │ → Skoll: workflow_end   │
          │                         │
          │ hati_commit_info →      │
          │ "hati-plan: pln-uuid    │
          │  quality: 0.89          │
          │  specs-implemented: 1"  │ ← NEW v4.0
          └─────────────────────────┘
```

---

## 14. Roadmap v1.0.0

| Fase | Semanas | Deliverable | Gap cerrado |
|---|---|---|---|
| 1 — Core Planning | 1–2 | Ciclo PRE→execute→POST, Approval Record, granularidad dinámica | Planificación base |
| 2 — Quality Snapshot | 3 | Integration con Tyr para Quality Snapshot | Calidad objetiva |
| 3 — Spec Delta | 4–5 | spec_check en plan_create, spec impact en checkpoint PLAN | Requisitos vivos |
| 4 — Module Hints | 6 | GetModuleHints desde Skoll, adjustByModuleHints | Contexto de AGENTS.md |
| 5 — AGENTS.md Anidados | 7 | Detección de AGENTS.md local, ajuste de granularidad | Monorepos |
| 6 — Async Checkpoints | 8 | Checkpoints no bloqueantes con polling | Agente nunca bloqueado |
| 7 — Quality Express | 9 | Quality Snapshot < 100 tokens | Eficiencia de tokens |
| 8 — Plan Templates | 10 | Generación de planes sin LLM externo | Velocidad |
| 9 — Async Handoff | 11 | Transferencia no bloqueante entre agentes | Workflows parallel |
| 10 — Spec Delta por fase | 12 | Spec impact calculado EN CADA FASE | Trazabilidad completa |
| 11 — Timeline Mermaid | 13 | Diagramas estáticos sin LLM | Visualización |
| 12 — Release v1.0.0 | 14 | Docs, testing, release | — |

---

## 15. Mejoras Planificadas v1.1.0

Ver documento `PRD_Ragnarok_v1.1_Improvements.md` para especificaciones completas.

| Mejora | Descripción | Prioridad |
|---|---|---|
| **CLI Stats** | `ecosystem_stats` — stats unificados del ecosistema | 🟡 MEDIA |

*Nota: Async Checkpoints está en backlog para v1.2+ ya que en contexto single-user el usuario está disponible para approve/reject en tiempo real.*

### Nuevos tools en v1.1.0

| Tool | Descripción |
|---|---|
| `ecosystem_stats` | Stats unificados del ecosistema (incluye Hati) |

---

*Hati PRD v1.0.0 — Marzo 2026*
*~27 MCP tools (v1.1.0) · SQLite · Go 1.22+ · MIT*
*Plan Templates · Quality Express · Async Handoff · CLI Stats*
*3 Pilares: Velocidad ⚡ · Eficiencia de Tokens 💎 · Eficacia 🎯*
