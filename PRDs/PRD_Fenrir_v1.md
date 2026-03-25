# PRD — Fenrir v1.0.0
## Product Requirements Document — Versión 1.0.0

**Producto:** Fenrir  
**Versión:** 1.0.0  
**Tipo:** MCP Plugin — Memory, Knowledge & Institutional Intelligence Layer  
**Lenguaje:** Go 1.22+  
**Licencia:** MIT  
**Fecha:** Marzo 2026

---

## Resumen de cambios por versión

| Versión | Cambios principales |
|---|---|
| v1.0.0 | **Versión inicial unificada.** Memory, Knowledge Graph, Authority Levels, Lifecycle, Incidents, Conflicts, Bootstrap, Onboarding, Velocity, Reliability, Progressive Disclosure, mem_save_prompt, Auto-inject, Compaction Checkpoint, AGENTS.md canónico, Specs Module, Skills built-in, Semantic Search, Cache Inteligente, Proactive Notifications, Compresión automática de nodos. Seguridad (Validator/Shield/SAST/ScopeEnforcer) migrada a Tyr. |

---

## Qué cambia en v1.0.0

### Principios de diseño

Todas las mejoras de v1.0.0 están guiadas por tres criterios:

| Criterio | Descripción | Aplicación |
|----------|-------------|------------|
| **VELOCIDAD** | Respuesta < 2s para consultas simples | Semantic search < 5ms, cache instantáneo |
| **EFICIENCIA DE TOKENS** | Progressive disclosure, resúmenes automáticos | 80% reducción en sesiones típicas |
| **EFICACIA** | El agente siempre tiene contexto relevante | Notificaciones proactivas, búsqueda por intención |

### Mejoras en v1.0.0

| Mejora | Open Source | Velocidad | Tokens | Eficacia |
|--------|-------------|------------|--------|----------|
| Semantic Search con embeddings locales | SQLite-vss / ChromaDB (local) | ⚡⚡⚡ | ⚡⚡⚡ | ⚡⚡⚡ |
| Cache Inteligente multi-nivel | freecache + badger | ⚡⚡⚡ | ⚡⚡ | ⚡⚡ |
| Proactive Notifications | MCP Notifications | ⚡⚡⚡ | ⚡ | ⚡⚡⚡ |
| Compresión automática de nodos | Usa el LLM de la herramienta | ⚡ | ⚡⚡⚡ | ⚡ |
| GraphQL API (opcional) | gqlgen | ⚡ | ⚡ | ⚡⚡ |

---

---

## Tabla de contenidos

1. [Visión del producto](#1-visión-del-producto)
2. [Propuesta de valor](#2-propuesta-de-valor)
3. [Conceptos principales](#3-conceptos-principales)
4. [Requisitos funcionales](#4-requisitos-funcionales)
5. [Progressive Disclosure](#5-progressive-disclosure)
6. [Módulo Specs](#6-módulo-specs)
7. [AGENTS.md como archivo canónico](#7-agentsmd-como-archivo-canónico)
8. [Auto-inject y Compaction Checkpoint](#8-auto-inject-y-compaction-checkpoint)
9. [Skills built-in](#9-skills-built-in)
10. [Modelo de datos v1.0.0](#10-modelo-de-datos-v100)
11. [MCP Tools — Catálogo completo v1.0.0](#11-mcp-tools--catálogo-completo-v100)
12. [CLI completo v1.0.0](#12-cli-completo-v100)
13. [Configuración v1.0.0](#13-configuración-v100)
14. [Integración con Tyr, Skoll y Hati](#14-integración-con-tyr-skoll-y-hati)
15. [Roadmap v1.0.0](#15-roadmap-v100)

---

## 1. Visión del producto

Fenrir es el **cerebro institucional** del proyecto. Responde una sola pregunta:

> **"¿Qué sabemos sobre este proyecto, cómo llegamos aquí, y qué debe hacer el sistema?"**

Fenrir proporciona la capa de memoria y conocimiento institucional, incluyendo captura de intención original del developer (`mem_save_prompt`), entrega eficiente de contexto (Progressive Disclosure), requisitos vivos del sistema (Módulo Specs), y compatibilidad con el estándar AGENTS.md adoptado por 60k+ proyectos.

> **Misión v1.0.0:** Que el conocimiento del proyecto sea siempre accesible, confiable, eficiente en tokens, y compatible con cualquier herramienta del ecosistema.

---

## 2. Propuesta de valor

```
Fenrir responde:  "¿Qué sabemos y qué debe hacer el sistema?"
Tyr responde:     "¿Es seguro lo que el agente está haciendo?"
Skoll responde:   "¿Quién hace qué y cómo?"
Hati responde:    "¿Cuál es el plan y quién lo aprueba?"
```

Fenrir sin Tyr: memoria y conocimiento sin seguridad activa (válido para proyectos internos).
Tyr sin Fenrir: seguridad sin memoria (válido para CI/CD gates).
Juntos: el stack completo.

---

## 3. Conceptos principales

- **Knowledge Graph** — nodos, edges, FTS5, relaciones causales
- **Authority Levels** — exploratory / confirmed / authoritative
- **Knowledge Lifecycle** — stale, decay, expire, archive
- **Session DNA** — fingerprint de cada sesión con métricas de calidad
- **Drift Detection** — score de deriva arquitectónica por módulo
- **Predictive Enforcement** — alertas basadas en historial por módulo
- **Bootstrap & Onboarding** — entrevista guiada + documento para developers nuevos
- **Incidents** — bugs de producción que alimentan el grafo
- **Conflict Resolution** — detección y resolución de conocimiento contradictorio
- **Suggest Rules a Skoll** — reglas derivadas de patrones detectados
- **Project Scan** — análisis automático de 5 capas al inicializar
- **Velocity Metrics** — tendencias de calidad a lo largo del tiempo
- **Agent Reliability Score** — confiabilidad del agente por módulo
- **PR Summary Export** — evidencia de gobernanza para PRs
- **Git Sync** — sincronización de chunks entre developers del equipo
- **Progressive Disclosure** — entrega de contexto en capas (compact, timeline, full)
- **mem_save_prompt** — captura la intención original del developer
- **Auto-inject de contexto** — mem_session_start inyecta contexto automáticamente
- **Compaction Checkpoint** — recuperación de sesión después de compaction
- **Specs Module** — requisitos vivos en formato GIVEN/WHEN/THEN
- **Semantic Search** — búsqueda por intención usando embeddings locales
- **Cache Inteligente** — cache multi-nivel para respuestas rápidas
- **Proactive Notifications** — alertas cuando hay situaciones importantes
- **Compresión automática de nodos** — resúmenes de nodos antiguos

---

## 4. Conceptos nuevos v1.0.0

### 4.1 Progressive Disclosure

En lugar de devolver todo el contenido de una observación en `mem_find`, Fenrir entrega el conocimiento en tres capas:

```
Capa 1 — Compact (~100 tokens por resultado)
  mem_find "auth middleware"
  → lista de resultados con ID, título, tipo, fecha, confidence
  → el agente decide qué necesita ver completo

Capa 2 — Timeline (~300 tokens)
  mem_timeline <observation_id>
  → contexto cronológico: qué pasó antes y después de esa observación
  → sesiones relacionadas, decisiones vinculadas

Capa 3 — Full (contenido completo)
  mem_get_observation <observation_id>
  → contenido completo sin truncar, incluyendo relaciones del grafo
```

Esto reduce el consumo de tokens en hasta 80% en sesiones con grafos grandes.

### 4.2 mem_save_prompt

Un tipo especial de observación que captura el **prompt original del developer** antes de cualquier interpretación del agente. Diferente a `mem_save`: no documenta lo que hizo el agente — documenta lo que quiso el humano.

```
Tipo: prompt
Guarda: texto exacto del developer + timestamp + contexto de sesión
Usos: entender la intención original, detectar cuando el agente malinterpretó, onboarding
```

### 4.3 Auto-inject de contexto

En v3.0, `mem_session_start` requería que el agente llamara `mem_context` explícitamente para recuperar contexto. En v4.0, el contexto se inyecta **automáticamente** en la respuesta de `mem_session_start` — el agente no puede "olvidar" recuperarlo.

### 4.4 Compaction Auto-Checkpoint

En v3.0 esto era documentación en AGENTS.md. En v4.0 es código: el plugin de cliente (OpenCode, Claude Code) detecta el evento de compaction y automáticamente:
1. Llama `mem_session_checkpoint` para guardar el estado actual
2. Inyecta en el prompt de compaction: "Después de reanudar, el contexto fue guardado"

### 4.5 Módulo Specs

Una capa de conocimiento nueva y diferente a las decisiones (`arch_save`). Los specs capturan **qué debe hacer el sistema** en formato GIVEN/WHEN/THEN. Son los requisitos vivos del proyecto.

```
Decisions → "Decidimos usar Repository Pattern"        (pasado)
Specs     → "El sistema DEBE separar dominio de datos" (presente permanente)
```

Compatible con OpenSpec: importa desde `openspec/specs/` si el proyecto ya lo usa.

### 4.6 AGENTS.md como archivo canónico

El ecosistema adopta AGENTS.md (estándar de la Linux Foundation, 60k+ proyectos) como el archivo de instrucciones del agente en lugar de FENRIR.md. Compatibilidad automática con Codex, Jules, Factory, Devin, Aider y 25+ herramientas más.

### 4.7 Skills built-in

Fenrir distribuye un conjunto de skills esenciales dentro del binario, disponibles inmediatamente sin que el usuario los cree. Skoll los consume automáticamente.

---

## 4. Requisitos funcionales

### RF-01 — Progressive Disclosure

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N01-01 | `mem_find` debe retornar por defecto resultados compactos (ID, título, tipo, fecha, confidence) sin el contenido completo | MUST |
| RF-N01-02 | El campo `include_content` en `mem_find` permite obtener contenido completo cuando se necesita | MUST |
| RF-N01-03 | `mem_timeline` debe retornar el contexto cronológico de una observación específica | MUST |
| RF-N01-04 | `mem_get_observation` debe retornar el contenido completo sin truncar de un nodo | MUST |
| RF-N01-05 | Los resultados compactos de `mem_find` deben ser de máximo 150 tokens por resultado | MUST |
| RF-N01-06 | El grafo de relaciones de un nodo se incluye en `mem_get_observation`, no en `mem_find` | MUST |

### RF-02 — mem_save_prompt

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N02-01 | `mem_save_prompt` debe guardar el prompt original del developer como nodo tipo `prompt` | MUST |
| RF-N02-02 | El nodo prompt debe incluir: texto exacto, session_id, timestamp, módulo de contexto | MUST |
| RF-N02-03 | Los prompts guardados deben ser consultables con `mem_find` filtrando por tipo:prompt | MUST |
| RF-N02-04 | `mem_session_start` debe retornar los prompts de sesiones previas en el mismo módulo como parte del auto-inject | SHOULD |

### RF-03 — Auto-inject en session_start

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N03-01 | `mem_session_start` debe retornar automáticamente contexto de sesiones previas relevantes sin necesidad de llamar `mem_context` | MUST |
| RF-N03-02 | El auto-inject debe usar el módulo declarado en `mem_session_start` para filtrar contexto relevante | MUST |
| RF-N03-03 | Si no se declara módulo, el auto-inject retorna las últimas N observaciones del proyecto | MUST |
| RF-N03-04 | El auto-inject debe incluir: últimas 5 observaciones, predicciones activas, incidents abiertos, nodos `stale` relevantes | MUST |
| RF-N03-05 | El contenido del auto-inject debe seguir Progressive Disclosure — compacto por defecto | MUST |

### RF-04 — Compaction Auto-Checkpoint

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N04-01 | `mem_session_checkpoint` debe guardar un snapshot del estado actual de la sesión | MUST |
| RF-N04-02 | El plugin de cliente OpenCode debe llamar `mem_session_checkpoint` automáticamente al detectar compaction | MUST |
| RF-N04-03 | El plugin debe inyectar en el prompt post-compaction: instrucción de llamar `mem_session_start` | MUST |
| RF-N04-04 | Los checkpoints deben ser distinguibles de los summaries en el Approval Record | MUST |

### RF-05 — Módulo Specs

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N05-01 | `spec_save` debe guardar un requisito en formato GIVEN/WHEN/THEN como nodo tipo `requirement` | MUST |
| RF-N05-02 | Cada spec debe tener: capability, title, requirement text, scenarios GIVEN/WHEN/THEN, status | MUST |
| RF-N05-03 | `spec_list` debe listar todos los specs activos organizados por capability | MUST |
| RF-N05-04 | `spec_check` debe verificar si un cambio propuesto afecta algún spec activo | MUST |
| RF-N05-05 | `spec_delta` debe generar el diff de specs afectados por un plan de Hati al completarse | MUST |
| RF-N05-06 | `fenrir spec import` debe importar specs desde `openspec/specs/` si el proyecto usa OpenSpec | MUST |
| RF-N05-07 | Los specs deben aparecer en `fenrir onboard` como primera sección, antes de las decisiones | MUST |
| RF-N05-08 | Los specs con status `violated` deben aparecer en `predict` como alertas de alta prioridad | MUST |
| RF-N05-09 | `spec_update` debe actualizar un spec existente y crear un edge `evolved_from` en el grafo | SHOULD |

### RF-06 — AGENTS.md canónico

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N06-01 | `fenrir init` debe detectar si existe AGENTS.md en la raíz y trabajar con él en lugar de crear FENRIR.md | MUST |
| RF-N06-02 | Si AGENTS.md existe: agregar sección `## Fenrir Protocol` dentro del archivo existente | MUST |
| RF-N06-03 | Si AGENTS.md no existe: crear AGENTS.md con estructura compatible con el estándar | MUST |
| RF-N06-04 | El AGENTS.md generado/actualizado debe incluir secciones de Fenrir, Skoll, Hati y Tyr si están instalados | MUST |
| RF-N06-05 | `fenrir scan` debe leer AGENTS.md y detectar comandos de testing para sugerir como Standards a Tyr | MUST |
| RF-N06-06 | En monorepos, respetar la jerarquía: el AGENTS.md más cercano al archivo editado tiene precedencia | MUST |
| RF-N06-07 | Si existe FENRIR.md de versiones anteriores, `fenrir init` debe migrar su contenido a AGENTS.md | SHOULD |

### RF-07 — Skills built-in

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N07-01 | El binario de Fenrir debe incluir un conjunto de skills esenciales como assets embebidos | MUST |
| RF-N07-02 | `fenrir skills list` debe mostrar los skills built-in disponibles | MUST |
| RF-N07-03 | `fenrir skills install <nombre>` debe copiar un skill built-in al directorio `.skoll/skills/` del proyecto | MUST |
| RF-N07-04 | Los skills built-in deben estar en formato SKILL.md oficial (con frontmatter YAML) | MUST |
| RF-N07-05 | Los skills built-in deben actualizarse con cada nueva versión del binario | MUST |

---

## 5. Progressive Disclosure

### Ejemplo de flujo optimizado

```go
// ANTES (v3.0) — carga todo el contexto
result := fenrir.MemFind("auth middleware")
// → retorna 5 observaciones × 500 tokens = 2,500 tokens consumidos

// DESPUÉS (v4.0) — progressive disclosure
compact := fenrir.MemFind("auth middleware")
// → retorna 5 observaciones × 100 tokens = 500 tokens (IDs + títulos)

// El agente decide cuál necesita completa
full := fenrir.MemGetObservation(compact.Results[0].ID)
// → retorna 1 observación × 500 tokens = 500 tokens
// Total: 1,000 tokens en lugar de 2,500
```

### Formato de resultado compacto

```json
{
  "results": [
    {
      "id": "obs-089",
      "type": "decision",
      "title": "Usar jose para OAuth — no jsonwebtoken",
      "date": "2026-03-15",
      "confidence": 0.95,
      "authority": "authoritative",
      "module": "src/modules/auth/",
      "tags": ["oauth", "security", "jose"]
    }
  ],
  "total": 12,
  "query": "auth middleware",
  "hint": "Use mem_timeline or mem_get_observation for full content"
}
```

---

## 6. Módulo Specs

### Formato de spec en el knowledge graph

```markdown
<!-- Almacenado como nodo tipo 'requirement' -->

capability: auth-session
title: Session Expiration

## Requirement
The system SHALL expire sessions after a configured duration.

## Scenarios

### GIVEN a user has authenticated
### WHEN 24 hours pass without activity
### THEN invalidate the session token
### AND require re-authentication

### GIVEN user checks "Remember me" at login
### WHEN 30 days have passed
### THEN invalidate the session token
### AND clear the persistent cookie
```

### `spec_check` — verificación antes de implementar

```json
// El agente llama spec_check antes de modificar /auth/
{
  "name": "spec_check",
  "input": {
    "proposed_change": "Añadir OAuth con Google al módulo /auth/",
    "module": "src/modules/auth/"
  }
}

// Respuesta
{
  "affected_specs": [
    {
      "id": "spec-001",
      "capability": "auth-session",
      "title": "Session Expiration",
      "impact": "extends",
      "detail": "El nuevo proveedor OAuth debe respetar las duraciones de sesión configuradas"
    },
    {
      "id": "spec-002",
      "capability": "auth-login",
      "title": "OAuth Provider Support",
      "impact": "implements",
      "detail": "Este cambio implementa el requisito de soporte multi-proveedor"
    }
  ],
  "new_specs_needed": [
    "Considerar spec para el callback URL de Google OAuth"
  ]
}
```

### `spec_delta` — delta al completar un plan de Hati

```markdown
## Spec Delta — Plan: "Implementar OAuth con Google"

### Specs implementados ✅
- auth-login/spec.md: "OAuth Provider Support" → marcado como implementado

### Specs extendidos 🔄
- auth-session/spec.md: "Session Expiration" → escenario "Extended session with OAuth"
  + GIVEN usuario autenticado via OAuth con Google
  + WHEN token de Google expira
  + THEN forzar re-autenticación con Google

### Specs nuevos creados 📝
- auth-oauth-callback/spec.md: "OAuth Callback Security" → nuevo
```

### Compatibilidad con OpenSpec

```bash
# Si el proyecto ya usa OpenSpec
fenrir spec import
# → Lee openspec/specs/**/*.md
# → Importa como nodos tipo 'requirement' en el knowledge graph
# → Crea edge 'imported_from' para mantener la referencia

# Exportar specs de Fenrir al formato OpenSpec
fenrir spec export --format openspec
# → Genera openspec/specs/{capability}/spec.md por cada spec
```

---

## 7. AGENTS.md como archivo canónico

### Lógica de `fenrir init`

```go
// internal/adapters/agents_md.go

func (a *AgentsMDAdapter) Init(plugins []string) error {
    existing := fileExists("AGENTS.md")

    if existing {
        // Agregar sección del ecosistema al AGENTS.md existente
        return a.appendEcosystemSection("AGENTS.md", plugins)
    }

    // Crear AGENTS.md nuevo con estructura completa
    return a.createAgentsMD(plugins)
}

func (a *AgentsMDAdapter) generateContent(plugins []string) string {
    var b strings.Builder

    // Secciones base del proyecto (detectadas por fenrir scan)
    b.WriteString("# AGENTS.md\n\n")
    b.WriteString("## Setup\n")
    b.WriteString(a.scanSetupCommands())
    b.WriteString("\n## Testing\n")
    b.WriteString(a.scanTestingCommands())
    b.WriteString("\n## Code Style\n")
    b.WriteString(a.scanCodeStyle())

    // Sección del ecosistema
    b.WriteString("\n---\n\n")
    b.WriteString("## Agent Protocol — Fenrir Ecosystem\n\n")

    for _, plugin := range plugins {
        b.WriteString(a.pluginSection(plugin))
    }

    return b.String()
}
```

### Estructura del AGENTS.md generado

```markdown
# AGENTS.md

## Setup
- Install: `npm install`
- Build: `npm run build`
- Dev: `npm run dev`

## Testing
- Unit: `npm test`
- E2E: `npx playwright test`
- Coverage: `npm test -- --coverage`

## Code Style
- TypeScript strict mode
- ESLint + Prettier configured
- No `any` types

---

## Agent Protocol — Fenrir Ecosystem

### Fenrir (Memory & Knowledge)
You have access to Fenrir via MCP. Fenrir gives you institutional memory.

**Mandatory:**
- START: `mem_session_start` (auto-injects previous context)
- SAVE prompts: `mem_save_prompt` before interpreting any request
- SAVE work: `mem_save` after bugfix/decision/discovery
- END: `mem_session_end` — non-negotiable
- After compaction: `mem_session_start` recovers state automatically

### Tyr (Security & Validation)
- BEFORE installing any package: `pkg_check`
- BEFORE writing code: check active SAST findings with `sast_findings`
- REPORT security findings: `sast_resolve` when fixed

### Skoll (Roles & Workflows)
- START: `skoll_status` to verify RSAW index
- ACTIVATE role: `agent_activate` before working
- CHECK rules: `rule_check` before acting
- ON handoff: `agent_handoff` with explicit contract

### Hati (Planning & Approval)
- For non-trivial requests: `plan_create` first
- NEVER proceed after `checkpoint_open` without developer response
- NEVER skip PRE or POST checkpoints
```

### Lectura de testing commands → Standards (vía Tyr)

```go
// internal/modules/scan/agents_reader.go

func (r *AgentsReader) ExtractTestingCommands(agentsMD string) []StandardCandidate {
    var candidates []StandardCandidate

    // Patrones comunes en AGENTS.md
    patterns := []struct {
        regex   *regexp.Regexp
        id      string
        failure string
    }{
        {regexp.MustCompile("`(npm test[^`]*)`"), "test-pass", "block"},
        {regexp.MustCompile("`(pnpm test[^`]*)`"), "test-pass", "block"},
        {regexp.MustCompile("`(go test[^`]*)`"), "test-pass", "block"},
        {regexp.MustCompile("`(pytest[^`]*)`"), "test-pass", "block"},
        {regexp.MustCompile("`(npm run lint[^`]*)`"), "lint-clean", "warn"},
        {regexp.MustCompile("`(playwright test[^`]*)`"), "e2e-critical", "block"},
    }

    for _, p := range patterns {
        if match := p.regex.FindStringSubmatch(agentsMD); match != nil {
            candidates = append(candidates, StandardCandidate{
                ID:        p.id,
                Command:   match[1],
                OnFailure: p.failure,
            })
        }
    }
    return candidates
}
```

---

## 8. Auto-inject y Compaction Checkpoint

### `mem_session_start` v4.0 — respuesta completa

```json
{
  "session_id": "ses-uuid",
  "auto_injected_context": {
    "recent_observations": [
      { "id": "obs-089", "title": "Usar jose para OAuth", "type": "decision", "date": "2026-03-15" },
      { "id": "obs-112", "title": "Bug en /payments/ - timeout en Stripe", "type": "incident" }
    ],
    "recent_prompts": [
      { "id": "prm-034", "text": "Implementar OAuth con Google", "date": "2026-03-20" }
    ],
    "active_predictions": [
      "⚠️ /auth/ drift_score 0.41 — 3 violaciones en últimas 5 sesiones"
    ],
    "open_incidents": [
      { "id": "inc-003", "description": "OAuth falla con emails Unicode", "severity": "high" }
    ],
    "affected_specs": [
      { "id": "spec-001", "title": "Session Expiration", "status": "active" }
    ]
  },
  "compaction_checkpoint_available": true,
  "last_checkpoint": "2026-03-22T09:15:00Z"
}
```

### Plugin de cliente OpenCode v4.0 — compaction hook

```typescript
// plugin/opencode/fenrir-ecosystem.ts

export default {
  name: "fenrir-ecosystem",

  async onCompaction(summary: string): Promise<string> {
    // 1. Guardar checkpoint automáticamente
    await fetch("http://localhost:7438/api/session/checkpoint", {
      method: "POST",
      body: JSON.stringify({ summary, trigger: "compaction" }),
    }).catch(() => {});

    // 2. Inyectar instrucción en el prompt post-compaction
    return `${summary}

---
CONTEXT SAVED: Fenrir checkpoint created automatically.
On resume: call mem_session_start to restore full context.
The context will be auto-injected — no need to call mem_context separately.`;
  },
} satisfies Plugin;
```

---

## 9. Skills built-in

### Skills incluidos en el binario de Fenrir

```
fenrir/internal/skills/ (embedded con go:embed)
├── api-validation/
│   └── SKILL.md
├── git-workflow/
│   └── SKILL.md
├── clean-architecture/
│   └── SKILL.md
├── error-handling/
│   └── SKILL.md
├── testing/
│   └── SKILL.md
└── api-design/
    └── SKILL.md
```

Todos en formato SKILL.md oficial con frontmatter YAML (estándar AgentSkills.io).

### `fenrir skills install`

```bash
fenrir skills list
# → api-validation, git-workflow, clean-architecture,
#   error-handling, testing, api-design

fenrir skills install clean-architecture
# → Copia a .skoll/skills/clean-architecture/SKILL.md
# ✅ Instalado: clean-architecture v1.2 (built-in)

fenrir skills install --all
# → Instala todos los skills built-in en .skoll/skills/
```

---

## 10. Modelo de datos v1.0.0

### Tablas existentes
`nodes`, `edges`, `nodes_fts`, `sessions`, `drift_scores`, `incidents`, `conflicts`, `pending_rules`, `scan_runs`, `velocity_metrics`, `pr_summaries`, `commit_registry`, `specs`, `spec_deltas`, `session_checkpoints`, `prompts`

### Tablas nuevas en v4.0

```sql
-- Specs / Requirements Layer
CREATE TABLE specs (
    id           TEXT PRIMARY KEY,
    capability   TEXT NOT NULL,
    title        TEXT NOT NULL,
    requirement  TEXT NOT NULL,
    scenarios    TEXT NOT NULL,    -- JSON array de GIVEN/WHEN/THEN
    status       TEXT DEFAULT 'active', -- active | implemented | violated | deprecated
    node_id      TEXT REFERENCES nodes(id),
    imported_from TEXT,            -- 'openspec' si fue importado
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Spec deltas por plan de Hati
CREATE TABLE spec_deltas (
    id          TEXT PRIMARY KEY,
    plan_id     TEXT NOT NULL,     -- ID del plan de Hati
    spec_id     TEXT REFERENCES specs(id),
    delta_type  TEXT NOT NULL,     -- implemented | extended | created | violated
    description TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Session checkpoints (compaction recovery)
CREATE TABLE session_checkpoints (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id),
    trigger     TEXT NOT NULL,     -- compaction | manual | auto
    summary     TEXT,
    snapshot    TEXT NOT NULL,     -- JSON del estado de la sesión
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Prompts guardados por el developer
CREATE TABLE prompts (
    id          TEXT PRIMARY KEY,
    session_id  TEXT REFERENCES sessions(id),
    text        TEXT NOT NULL,
    module      TEXT,
    node_id     TEXT REFERENCES nodes(id),
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 11. MCP Tools — Catálogo completo v1.0.0

### Tools de v1.0-v3.0 que permanecen en Fenrir (sin módulos de seguridad)

**Memory Core (7):**
`mem_save`, `mem_find` *(modificado: progressive disclosure)*, `mem_context`, `mem_timeline`, `mem_session_start` *(modificado: auto-inject)*, `mem_session_end`, `mem_dna`

**Memory Extended (3):**
`mem_get_observation` *(nuevo en v4.0 — capa 3 del progressive disclosure)*, `mem_save_prompt` *(nuevo v4.0)*, `mem_session_checkpoint` *(nuevo v4.0)*

**Knowledge Lifecycle (4):**
`graph_review`, `graph_expire`, `node_authorize`, `confidence_update`

**Enforcer/Arch (4):**
`arch_save`, `arch_verify`, `arch_drift`, `policy_check`

**Predictive (2):**
`predict`, `insights`

**Incidents (3):**
`incident_log`, `incident_resolve`, `incident_list`

**Conflicts (1):**
`conflict_list`

**Suggest Rules (2):**
`suggest_rule`, `pending_rules`

**Scan (2):**
`scan_run`, `scan_status`

**Velocity & Reliability (3):**
`velocity_trends`, `reliability_score`, `bootstrap_status`

**PR Export (1):**
`export_pr_summary`

**Sistema (2):**
`fenrir_stats`, `trace`

### Tools nuevas en v4.0 (8 adicionales)

| Tool | Módulo | Descripción |
|---|---|---|
| `mem_get_observation` | Memory | Capa 3: contenido completo de una observación |
| `mem_save_prompt` | Memory | Guardar prompt original del developer |
| `mem_session_checkpoint` | Memory | Guardar checkpoint (compaction recovery) |
| `spec_save` | Specs | Guardar un requisito GIVEN/WHEN/THEN |
| `spec_list` | Specs | Listar specs activos por capability |
| `spec_check` | Specs | Verificar si un cambio afecta specs activos |
| `spec_delta` | Specs | Generar delta de specs de un plan de Hati |
| `spec_update` | Specs | Actualizar spec con edge evolved_from |

**Total v1.0.0: ~42 MCP tools**

---

## 12. CLI completo v1.0.0

```bash
# ─── INICIALIZACIÓN ──────────────────────────────────
fenrir init [--dry-run]
fenrir scan [--delta] [--layer <stack|arch|config|modules|patterns>]

# ─── MEMORIA ─────────────────────────────────────────
fenrir search <query>                  # Progressive: resultados compactos
fenrir show <observation_id>           # Progressive: contenido completo
fenrir context [--module <path>]
fenrir timeline <node_id>
fenrir trace <target>

# ─── SESIONES ────────────────────────────────────────
fenrir session list
fenrir session show <id>
fenrir session checkpoint              # Manual checkpoint

# ─── SPECS ───────────────────────────────────────────
fenrir spec list [--capability <cap>]
fenrir spec show <id>
fenrir spec add --capability <cap>
fenrir spec import [--from openspec]
fenrir spec export [--format openspec]
fenrir spec check "<cambio propuesto>"

# ─── KNOWLEDGE LIFECYCLE ─────────────────────────────
fenrir review [--stale] [--expiring] [--conflicts]
fenrir authorize <node_id>
fenrir expire <node_id> [--replaced-by <id>]
fenrir archive [--older-than <days>] [--dry-run]

# ─── ARQUITECTURA ────────────────────────────────────
fenrir arch list
fenrir arch add
fenrir arch deprecate <id>
fenrir drift [--module <path>]

# ─── INCIDENTS ───────────────────────────────────────
fenrir incident [--severity high] [--files <path>] [--plan <id>]
fenrir incidents [--status open|resolved]
fenrir incident-patterns

# ─── CONFLICTOS ──────────────────────────────────────
fenrir conflicts
fenrir conflict resolve <id> [--keep <node_id>] [--auto]

# ─── REGLAS SUGERIDAS ────────────────────────────────
fenrir pending-rules
fenrir approve-rule <id>
fenrir reject-rule <id>

# ─── STATS ───────────────────────────────────────────
fenrir stats
fenrir stats --velocity [--sessions 20]
fenrir stats --reliability [--module <path>]

# ─── BOOTSTRAP & ONBOARDING ──────────────────────────
fenrir bootstrap [--legacy]
fenrir onboard [--role <rol>] [--output <path>]

# ─── SKILLS BUILT-IN ─────────────────────────────────
fenrir skills list
fenrir skills install <nombre>
fenrir skills install --all

# ─── PR EXPORT ───────────────────────────────────────
fenrir export-pr-summary [--plan <id>] [--output <path>]

# ─── GIT SYNC ────────────────────────────────────────
fenrir sync [--import] [--status]

# ─── SISTEMA ─────────────────────────────────────────
fenrir serve [--port 7438]
fenrir mcp
fenrir tui
fenrir version
```

---

## 13. Configuración v1.0.0

### `.fenrir/config.json`

```json
{
  "project": "mi-proyecto",
  "version": "1.0.0",

  "agents_md": {
    "canonical": true,
    "auto_import_testing_commands": true,
    "suggest_standards_from_agents_md": true
  },

  "knowledge_lifecycle": {
    "default_stale_after_days": 90,
    "exploratory_decay_rate": 0.05,
    "exploratory_decay_after_days": 30,
    "archive_after_days": 365
  },

  "authority": {
    "auto_promote_to_confirmed": false,
    "require_tech_lead_for_authoritative": true
  },

  "scan": {
    "auto_on_empty_graph": true,
    "read_agents_md": true,
    "layers": ["stack", "arch", "config", "modules", "patterns"]
  },

  "memory": {
    "progressive_disclosure": true,
    "auto_inject_on_session_start": true,
    "auto_inject_limit": 5,
    "save_prompts": true
  },

  "specs": {
    "enabled": true,
    "import_from_openspec": false,
    "openspec_dir": "openspec/specs/",
    "generate_delta_on_plan_complete": true
  },

  "velocity": {
    "degradation_alert_threshold": 0.15,
    "degradation_window_sessions": 5
  },

  "incidents": {
    "auto_link_to_knowledge_graph": true,
    "risk_modifier_per_incident": 0.3
  },

  "conflicts": {
    "block_propagation": true,
    "auto_resolve_exploratory_vs_confirmed": true
  },

  "tyr": {
    "enabled": true,
    "expose_standards_results": true,
    "expose_reliability_score": true
  },

  "skoll": {
    "enabled": true,
    "suggest_rules": true
  },

  "hati": {
    "enabled": true,
    "generate_spec_delta_on_complete": true
  }
}
```

---

## 14. Integración con Tyr, Skoll y Hati

### Con Tyr

| Evento Fenrir | Acción Tyr |
|---|---|
| `fenrir scan` detecta comandos de testing en AGENTS.md | Sugiere agregarlos como Standards en `.fenrir/standards.json` (gestionado por Tyr) |
| `export_pr_summary` | Incluye resultados de Standards de Tyr en el summary |
| `reliability_score` calculado | Incorpora SAST findings de Tyr como factor del score |
| `predict` | Incluye alertas de CVEs activos de Tyr |

### Con Skoll

| Evento Fenrir | Acción Skoll |
|---|---|
| `approve-rule <id>` | Escribe en `.skoll/rules/` |
| `incident_log` recurrente | Crea `pending_rule` |
| `fenrir skills install` | Copia skills a `.skoll/skills/` |
| `spec_save` nuevo spec | Puede generar un skill de referencia si el spec es de dominio técnico |

### Con Hati

| Evento Fenrir | Acción Hati |
|---|---|
| `mem_session_start` (auto-inject) | Provee contexto inicial para `plan_create` |
| `spec_check` en módulo objetivo | Hati incluye specs afectados en checkpoint PLAN |
| `spec_delta` al completar plan | Hati lo incluye en el Approval Record final |
| `reliability_score` por módulo | Hati ajusta granularidad de fases |
| `incident_list` en módulo | Hati puede bloquear plan hasta resolución |

---

## 15. Roadmap v1.0.0

| Fase | Semanas | Deliverable |
|---|---|---|
| 1 — Core Memory & Graph | 1–2 | Knowledge Graph, nodes, edges, FTS5, sessions |
| 2 — Authority & Lifecycle | 3 | Authority levels, stale/decay/expire/archive |
| 3 — Incidents & Conflicts | 4 | Incident logging, conflict resolution |
| 4 — Bootstrap & Onboarding | 5 | Interview, onboarding docs, velocity metrics |
| 5 — Progressive Disclosure | 6 | mem_find compacto, mem_get_observation, mem_timeline |
| 6 — mem_save_prompt + Auto-inject | 7 | Nuevo tool, session_start con contexto automático |
| 7 — Compaction Checkpoint | 8 | mem_session_checkpoint, plugin hooks |
| 8 — AGENTS.md canónico | 9 | Init detecta AGENTS.md, lectura de testing commands |
| 9 — Módulo Specs | 10–11 | spec_save, spec_list, spec_check, spec_delta, import OpenSpec |
| 10 — Skills built-in | 12 | go:embed, fenrir skills list/install |
| 11 — Semantic Search | 13 | Embeddings locales, búsqueda por intención |
| 12 — Cache Inteligente | 14 | freecache + badger, invalidation |
| 13 — Proactive Notifications | 15 | MCP notifications, proactive alerts |
| 14 — Compresión de nodos | 16 | Background compression, LLM summarization |
| 15 — Release v1.0.0 | 17 | Docs, testing, release |

---

## 16. Mejoras Planificadas v1.1.0

Ver documento `PRD_Ragnarok_v1.1_Improvements.md` para especificaciones completas.

| Mejora | Descripción | Prioridad |
|---|---|---|
| **Intent Verifier** | `intent_save`, `intent_verify` — valida que código resuelve intención original | 🔴 ALTA |
| **Bias Detector** | Detecta recency, authority, confirmation y survivorship bias | 🟡 MEDIA |
| **CLI Stats** | `ecosystem_stats` — stats unificados del ecosistema | 🟡 MEDIA |

### Nuevos tools en v1.1.0

| Tool | Descripción |
|---|---|
| `intent_save` | Guardar intención original con embedding vectorial |
| `intent_verify` | Comparar intención vs código implementado |
| `bias_report` | Generar reporte de sesgos detectados en módulo |

---

*Fenrir PRD v1.0.0 — Marzo 2026*
*~45 MCP tools (v1.1.0) · SQLite + FTS5 + Vector Search + Knowledge Graph + Specs · Go 1.22+ · MIT*
*Semantic Search · Cache Inteligente · Proactive Notifications · Compresión Automática · Intent Verifier*
*3 Pilares: Velocidad ⚡ · Eficiencia de Tokens 💎 · Eficacia 🎯*
