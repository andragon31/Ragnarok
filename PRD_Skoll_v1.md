# PRD — Skoll v1.0.0
## Product Requirements Document — Versión 1.0.0

**Producto:** Skoll  
**Versión:** 1.0.0  
**Tipo:** MCP Plugin — RSAW Orchestration Layer  
**Lenguaje:** Go 1.22+  
**Licencia:** MIT  
**Fecha:** Marzo 2026

---

## Historial de versiones

| Versión | Cambios principales |
|---|---|
| v1.0.0 | **Versión inicial unificada.** Sistema RSAW base, Skill Versioning, Verifiable DoD, Inbound Rule Channel, Team Coordination, API Validation, Copyright Rule, Security Guide, Pedagogical Links, SKILL.md oficial (AgentSkills estándar), SkillsMP import/publish, Progressive Disclosure nativo, allowed-tools, AGENTS.md anidados en monorepos, Lazy Loading, Compiled Rules, AGENTS.md Auto-Detect, Workflow State Machine. |

---

## Qué cambia en v1.0.0

| Mejora | Origen | Impacto |
|---|---|---|
| Migrar formato de skills a SKILL.md oficial con frontmatter YAML | AgentSkills.io | Skills compatibles con 67k+ del marketplace |
| Estructura de directorios scripts/references/assets por skill | AgentSkills.io | Modularidad y carga eficiente |
| Progressive Disclosure nativo del estándar | AgentSkills.io | Reducción de tokens en startup |
| `skoll skills import` desde SkillsMP (67k+ skills) | SkillsMP | Acceso al ecosistema completo |
| `skoll skills publish` | SkillsMP | Compartir skills del equipo |
| `allowed-tools` pre-aprueba tools del ecosistema | AgentSkills.io | Menos fricción en ejecución |
| `skoll validate` compatible con `skills-ref validate` | AgentSkills.io | Validación estándar |
| Lectura de AGENTS.md anidados en monorepos | AGENTS.md estándar | Instrucciones por subdirectorio |
| Security scanning de skills importadas vía Tyr | SkillsMP research | El 5.2% de skills pueden ser maliciosas |

---

## Tabla de contenidos

1. [Visión del producto](#1-visión-del-producto)
2. [Conceptos principales](#2-conceptos-principales)
3. [Requisitos funcionales](#3-requisitos-funcionales)
4. [Formato SKILL.md oficial v1.0.0](#4-formato-skillmd-oficial-v100)
5. [Estructura de directorios por skill](#5-estructura-de-directorios-por-skill)
6. [Progressive Disclosure nativo](#6-progressive-disclosure-nativo)
7. [SkillsMP — Import y Publish](#7-skillsmp--import-y-publish)
8. [allowed-tools del ecosistema](#8-allowed-tools-del-ecosistema)
9. [AGENTS.md anidados en monorepos](#9-agentsmd-anidados-en-monorepos)
10. [Formato RSAW v1.0.0 completo](#10-formato-rsaw-v100-completo)
11. [MCP Tools — Catálogo completo v1.0.0](#11-mcp-tools--catálogo-completo-v100)
12. [CLI completo v1.0.0](#12-cli-completo-v100)
13. [Configuración v1.0.0](#13-configuración-v100)
14. [Integración con Fenrir, Tyr y Hati](#14-integración-con-fenrir-tyr-y-hati)
15. [Roadmap v1.0.0](#15-roadmap-v100)

---

## 1. Visión del producto

Skoll define quién hace qué, cómo se hace y en qué orden. La v1.0.0 adopta el estándar oficial de Anthropic para skills (AgentSkills.io), abriendo el ecosistema a 67k+ skills del marketplace y garantizando compatibilidad con cualquier herramienta que adopte el estándar — Claude Code, Codex CLI, y cualquier otro.

> **Misión v1.0.0:** Que los skills sean compatibles con el ecosistema global, que el conocimiento del equipo sea publicable y reutilizable, y que el agente tenga acceso eficiente al conocimiento correcto en el momento correcto.

---

## 2. Conceptos principales

*(Todos los conceptos unificados en v1.0.0)*

- **Sistema RSAW** — Rules, Skills, Agents, Workflows como jerarquía de cuatro capas
- **Regla de oro RSAW** — clasificación por tipo de pregunta que responde
- **Skill Versioning** — campos framework/min_version/max_version/last_verified
- **Verifiable DoD** — workflows con field `standards` verificado automáticamente
- **Inbound Rule Channel** — reglas propuestas por Fenrir en `_proposed/`
- **Team Role Coordination** — `.skoll/team.json` con scope activo por developer
- **API Validation** — skill `api-validation.md` + tool `api_docs_check`
- **Copyright Rule** — `copyright.md` activa por defecto
- **Security Config Guide** — `SECURITY_CONFIG.md` generado por init
- **Pedagogical Links** — `skill_load` retorna ejemplos reales desde Fenrir
- **SKILL.md oficial** — Formato AgentSkills.io con frontmatter YAML
- **Estructura de directorios** — scripts/, references/, assets/ por skill
- **Progressive Disclosure nativo** — Carga en capas (startup, activation, on-demand)
- **allowed-tools** — Pre-aprobación de tools del ecosistema
- **SkillsMP Integration** — Import/publish desde marketplace
- **AGENTS.md anidados** — Instrucciones por subdirectorio en monorepos
- **Lazy Loading** — Skills cargados solo cuando se necesitan
- **Compiled Rules** — Rules como funciones Go compiladas
- **Workflow State Machine** — Transiciones explícitas y verificables

El estándar AgentSkills define tres capas de carga que Skoll implementa:
- **Startup**: solo `name` y `description` (~100 tokens) para TODOS los skills
- **Activation**: el cuerpo completo de `SKILL.md` cuando el agente activa el skill
- **On-demand**: archivos en `scripts/`, `references/`, `assets/` solo cuando se necesitan

### 3.4 allowed-tools en SKILL.md

El campo `allowed-tools` del frontmatter pre-aprueba tools del ecosistema cuando el skill está activo. Un skill de clean-architecture puede pre-aprobar `fenrir.arch_verify` y `fenrir.mem_save` sin que el agente tenga que solicitarlos.

### 3.5 SkillsMP Integration

Import y publish de skills desde/hacia el marketplace de 67k+ skills. Con security scanning automático vía Tyr antes de instalar cualquier skill externa.

### 3.6 AGENTS.md anidados

En monorepos, el AGENTS.md del subdirectorio más cercano al archivo que se edita tiene precedencia. Skoll lee estos AGENTS.md anidados cuando `agent_activate` se llama con contexto de un subdirectorio específico.

---

## 3. Requisitos funcionales

### RF-01 — Formato SKILL.md oficial

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N01-01 | Todos los skills nuevos creados por `skoll skills add` deben usar el formato SKILL.md con frontmatter YAML | MUST |
| RF-N01-02 | Los campos obligatorios son: `name` y `description` | MUST |
| RF-N01-03 | `name` debe ser lowercase, máximo 64 caracteres, solo letras/números/hyphens, debe coincidir con el nombre del directorio | MUST |
| RF-N01-04 | `description` debe incluir qué hace el skill Y cuándo usarlo, máximo 1024 caracteres | MUST |
| RF-N01-05 | Los campos opcionales soportados: `license`, `compatibility`, `metadata`, `allowed-tools` | MUST |
| RF-N01-06 | `skoll validate` debe verificar el frontmatter de todos los skills contra el estándar | MUST |
| RF-N01-07 | Skills existentes en formato v3.0 deben poder migrarse con `skoll skills migrate` | MUST |
| RF-N01-08 | La validación debe ser compatible con `skills-ref validate` del estándar oficial | SHOULD |

### RF-02 — Estructura de directorios

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N02-01 | `skoll skills add <nombre>` debe crear un directorio con SKILL.md, scripts/, references/, assets/ | MUST |
| RF-N02-02 | `skill_load` debe retornar el contenido de SKILL.md y listar archivos disponibles en los subdirectorios | MUST |
| RF-N02-03 | `skill_read_file` debe retornar el contenido de un archivo específico en scripts/, references/ o assets/ | MUST |
| RF-N02-04 | Los archivos en scripts/ deben ser ejecutables por el agente si el cliente AI lo soporta | SHOULD |
| RF-N02-05 | El body de SKILL.md debe mantenerse bajo 500 líneas; el contenido extenso va en references/ | SHOULD |

### RF-03 — Progressive Disclosure

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N03-01 | Al startup del servidor MCP, Skoll debe cargar solo `name` y `description` de todos los skills | MUST |
| RF-N03-02 | El cuerpo completo de SKILL.md se carga solo cuando `skill_load` es llamado | MUST |
| RF-N03-03 | Los archivos en scripts/references/assets se cargan solo cuando `skill_read_file` es llamado | MUST |
| RF-N03-04 | `skill_list` debe retornar solo name, description y trigger (~100 tokens por skill) | MUST |
| RF-N03-05 | El índice de startup debe completarse en < 100ms para hasta 100 skills | MUST |

### RF-04 — SkillsMP Integration

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N04-01 | `skoll skills import --from skillsmp --query <query>` debe buscar y listar skills relevantes | MUST |
| RF-N04-02 | `skoll skills import --from skillsmp --url <url>` debe importar una skill específica por URL de GitHub | MUST |
| RF-N04-03 | Antes de instalar cualquier skill externa, Tyr debe escanearla con `inject_guard` | MUST |
| RF-N04-04 | El scan de seguridad debe ser automático y no bypasseable | MUST |
| RF-N04-05 | `skoll skills publish` debe preparar el skill para publicación en SkillsMP | SHOULD |
| RF-N04-06 | Las skills importadas deben marcarse con `metadata.source: skillsmp` para trazabilidad | MUST |
| RF-N04-07 | `skoll skills update` debe actualizar skills importadas desde su fuente original | SHOULD |

### RF-05 — allowed-tools

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N05-01 | `skill_load` debe retornar el campo `allowed-tools` del frontmatter | MUST |
| RF-N05-02 | Los tools listados en `allowed-tools` deben considerarse pre-aprobados cuando el skill está activo | MUST |
| RF-N05-03 | El formato es `Plugin.tool_name` (ej: `fenrir.arch_verify`, `tyr.pkg_check`, `hati.plan_create`) | MUST |
| RF-N05-04 | `agent_activate` debe incluir la lista de `allowed-tools` de todos los skills del agente en su respuesta | MUST |

### RF-06 — AGENTS.md anidados

| ID | Requisito | Prioridad |
|---|---|---|
| RF-N06-01 | `agent_activate` con un `context_path` debe leer el AGENTS.md más cercano a esa ruta | MUST |
| RF-N06-02 | Si existe `/payments/AGENTS.md`, usarlo en lugar del AGENTS.md raíz cuando el agente trabaje en `/payments/` | MUST |
| RF-N06-03 | El contenido del AGENTS.md local debe agregarse al contexto del agente en `agent_activate` | MUST |
| RF-N06-04 | Las instrucciones del AGENTS.md local tienen precedencia sobre las del raíz para ese contexto | MUST |
| RF-N06-05 | Notificar a Hati si el AGENTS.md local tiene instrucciones de testing especiales (para ajustar la fase) | SHOULD |

---

## 4. Formato SKILL.md oficial v1.0.0

### Template completo

```markdown
---
name: clean-architecture
description: |
  Apply clean architecture layering (domain, application, infrastructure).
  Use when creating new modules, designing service layers, or separating
  business logic from infrastructure concerns.
license: MIT
compatibility: Framework-agnostic. Examples use NestJS/TypeScript.
metadata:
  author: mi-equipo
  version: "2.0"
  framework: ""
  min_version: ""
  max_version: ""
  last_verified: "2026-03-22"
allowed-tools: fenrir.arch_verify fenrir.mem_save fenrir.spec_check tyr.standard_run
---

## Cuándo aplicar
Al crear módulos nuevos, diseñar capas de servicios, o cuando
necesitas separar lógica de negocio de la infraestructura.

## Proceso

### Paso 1 — Identificar el dominio
[instrucciones concretas...]

### Paso 2 — Definir las capas
[instrucciones concretas...]

## Checklist
- [ ] Las entidades del dominio no importan infraestructura
- [ ] Los servicios de aplicación no importan el ORM directamente
- [ ] Los repositorios implementan interfaces del dominio

## Anti-patrones
- Acceder a Prisma directamente desde un servicio de aplicación
- Poner lógica de negocio en los controllers

## Ver también
- [Referencia completa](references/REFERENCE.md)
- [Ejemplos de implementación](references/examples.md)
```

### Skills built-in (distribuidos por Fenrir, instalables en Skoll)

```
api-validation/     → Verificar APIs externas antes de implementar
clean-architecture/ → Separación de capas
copyright/          → Protocolo de revisión de código generado
error-handling/     → Manejo consistente de errores
git-workflow/       → Commits y PRs
testing/            → Estructura de tests
api-design/         → Diseño de endpoints REST
```

### Regla de oro del description

El `description` debe responder DOS preguntas:
1. **¿Qué hace?** → "Apply clean architecture layering"
2. **¿Cuándo usarlo?** → "Use when creating modules or separating business logic"

Sin las dos, el agente no puede decidir si debe cargar el skill.

---

## 5. Estructura de directorios por skill

```
.skoll/skills/
└── clean-architecture/
    ├── SKILL.md               ← REQUERIDO: frontmatter + instrucciones principales
    ├── scripts/
    │   └── verify-layers.sh   ← verifica que los imports respetan las capas
    ├── references/
    │   ├── REFERENCE.md       ← referencia técnica detallada
    │   └── examples.md        ← ejemplos de implementación real
    └── assets/
        ├── diagram.md         ← diagrama de capas en ASCII
        └── module-template/   ← template de módulo con clean arch
            ├── domain/
            ├── application/
            └── infrastructure/
```

### Carga on-demand de archivos

```go
// internal/rsaw/skill_loader.go

func (l *SkillLoader) LoadFile(skillName, filePath string) (string, error) {
    // Validar que el path es relativo al directorio del skill
    skillDir := filepath.Join(l.skillsDir, skillName)
    targetPath := filepath.Join(skillDir, filePath)

    // Prevenir path traversal
    if !strings.HasPrefix(targetPath, skillDir) {
        return "", fmt.Errorf("invalid file path: %s", filePath)
    }

    content, err := os.ReadFile(targetPath)
    if err != nil {
        return "", fmt.Errorf("file not found in skill %s: %s", skillName, filePath)
    }
    return string(content), nil
}
```

---

## 6. Progressive Disclosure nativo

### Índice en startup (todos los skills, solo metadata)

```json
// Cargado al iniciar el servidor MCP
{
  "skills_index": [
    {
      "name": "clean-architecture",
      "description": "Apply clean architecture layering. Use when creating modules...",
      "has_scripts": true,
      "has_references": true,
      "has_assets": true,
      "version_status": "current",
      "allowed_tools_count": 4
    },
    {
      "name": "api-validation",
      "description": "Verify external API endpoints before writing code. Use when calling APIs...",
      "has_scripts": false,
      "has_references": true,
      "has_assets": false,
      "version_status": "current",
      "allowed_tools_count": 2
    }
  ]
}
```

### Carga completa en `skill_load`

```json
// Solo cuando el agente llama skill_load('clean-architecture')
{
  "name": "clean-architecture",
  "content": "## Cuándo aplicar\n...",  // SKILL.md body completo
  "allowed_tools": ["fenrir.arch_verify", "fenrir.mem_save", "fenrir.spec_check", "tyr.standard_run"],
  "available_files": {
    "scripts": ["verify-layers.sh"],
    "references": ["REFERENCE.md", "examples.md"],
    "assets": ["diagram.md"]
  },
  "version_status": "current",
  "in_practice": { ... }  // pedagogical links de Fenrir
}
```

---

## 7. SkillsMP — Import y Publish

### `skoll skills import` — flujo completo

```bash
$ skoll skills import --from skillsmp --query "nestjs testing patterns"

🐺 Buscando en SkillsMP...

Resultados (filtrado por compatibilidad con tu stack):
  1. nestjs-testing          ⭐ 4.2k  MIT  by nestjs-team
     "NestJS unit and e2e testing with Jest. Use when writing tests for NestJS modules."
     
  2. jest-best-practices     ⭐ 2.1k  MIT  by testing-experts
     "Jest setup, mocking patterns, async tests. Use for any Jest-based testing."
     
  3. supertest-api           ⭐ 890   MIT  by community
     "API endpoint testing with Supertest. Use when testing REST endpoints."

¿Instalar? [1/2/3/all/none]: 1

🔒 Tyr escaneando nestjs-testing por seguridad...
   ✅ Sin patrones de injection detectados
   ✅ Sin secrets hardcodeados
   ✅ Scripts verificados

✅ Instalado: .skoll/skills/nestjs-testing/
   metadata.source: skillsmp
   metadata.version: "3.1"
```

### `skoll skills publish` — preparación para SkillsMP

```bash
$ skoll skills publish clean-architecture

🐺 Preparando clean-architecture para publicación...

Validando formato SKILL.md...
  ✅ Frontmatter válido
  ✅ name: clean-architecture
  ✅ description: 187 chars (good)
  ✅ license: MIT
  ✅ Sin patrones de injection
  ⚠️  SKILL.md tiene 620 líneas (> 500 recomendado)
     Considera mover parte del contenido a references/

Checklist antes de publicar:
  ✅ SKILL.md tiene cuándo aplicar
  ✅ Tiene checklist
  ✅ Tiene anti-patrones
  ⚠️  Sin scripts/ — considera agregar uno de verificación

¿Publicar de todas formas? [y/N]: y

Generado: clean-architecture-publish/
  ├── SKILL.md (listo para PR)
  └── README.md (instrucciones de instalación)

Para publicar: crea un PR en github.com/agentskills/agentskills
```

### Security scanning de skills importadas

```go
// internal/modules/skillsmp/scanner.go

func (s *Scanner) ScanBeforeInstall(skillPath string) (*ScanResult, error) {
    skillMD := filepath.Join(skillPath, "SKILL.md")

    // 1. Leer SKILL.md
    content, err := os.ReadFile(skillMD)
    if err != nil { return nil, err }

    // 2. Escanear con Tyr si está disponible
    if s.tyrClient != nil && s.tyrClient.Available() {
        result, err := s.tyrClient.InjectGuard(string(content))
        if err == nil && result.HasFindings {
            return &ScanResult{
                Safe: false,
                Findings: result.Findings,
                Reason: "Prompt injection patterns detected in SKILL.md",
            }, nil
        }
    }

    // 3. Fallback: scan local con patrones básicos
    return s.localScan(content)
}
```

---

## 8. allowed-tools del ecosistema

### Formato del campo

```yaml
allowed-tools: fenrir.arch_verify fenrir.mem_save tyr.pkg_check hati.plan_create
```

El formato es `{plugin}.{tool_name}` — siempre prefijado con el plugin que expone el tool.

### Cómo el agente los usa

```json
// Respuesta de agent_activate con allowed-tools

{
  "agent": "Backend Engineer",
  "skills_loaded": ["clean-architecture", "api-validation"],
  "allowed_tools": [
    "fenrir.arch_verify",
    "fenrir.mem_save",
    "fenrir.spec_check",
    "tyr.pkg_check",
    "tyr.standard_run",
    "skoll.agent_handoff",
    "skoll.rule_check"
  ],
  "scope": { ... }
}
```

El agente no necesita pedir permiso para estos tools — están pre-aprobados mientras el skill esté activo.

### Skills del ecosistema con allowed-tools preconfigurados

| Skill | allowed-tools pre-aprobados |
|---|---|
| `api-validation` | `tyr.pkg_check skoll.api_docs_check` |
| `clean-architecture` | `fenrir.arch_verify fenrir.mem_save fenrir.spec_check` |
| `testing` | `tyr.standard_run fenrir.mem_save` |
| `git-workflow` | `fenrir.export_pr_summary hati.hati_commit_info` |

---

## 9. AGENTS.md anidados en monorepos

### Estructura de ejemplo

```
mi-monorepo/
├── AGENTS.md                  ← instrucciones globales
├── packages/
│   ├── frontend/
│   │   └── AGENTS.md          ← instrucciones específicas del frontend
│   ├── backend/
│   │   └── AGENTS.md          ← instrucciones específicas del backend
│   └── payments/
│       └── AGENTS.md          ← instrucciones críticas de pagos
└── infra/
    └── AGENTS.md              ← instrucciones de infraestructura
```

### `agent_activate` con contexto de subdirectorio

```json
// El agente llama agent_activate con contexto del subdirectorio
{
  "name": "agent_activate",
  "input": {
    "agent_id": "backend",
    "context_path": "packages/payments/src/"
  }
}

// Respuesta — incluye AGENTS.md del subdirectorio
{
  "agent": "Backend Engineer",
  "scope": { ... },
  "local_agents_md": {
    "found_at": "packages/payments/AGENTS.md",
    "content": "## Payments Module Notes\n\n- Always run integration tests before changes\n- Stripe API version is pinned to 2024-11-20\n- PCI compliance: never log card numbers\n",
    "precedence": "local_overrides_root"
  },
  "additional_rules": [
    "Always run integration tests before changes (from packages/payments/AGENTS.md)",
    "PCI compliance: never log card numbers (from packages/payments/AGENTS.md)"
  ]
}
```

### Notificación a Hati si hay instrucciones de testing especiales

```go
// internal/engine/agents_reader.go

func (r *AgentsMDReader) NotifyHati(localAgentsMD string, modulePath string) {
    if r.hatiClient == nil || !r.hatiClient.Available() { return }

    // Detectar instrucciones de testing
    if hasTestingInstructions(localAgentsMD) {
        r.hatiClient.ModuleHint(ModuleHint{
            Module:  modulePath,
            Message: "Este módulo tiene instrucciones de testing en su AGENTS.md local",
            Action:  "consider_e2e_standard",
        })
    }

    // Detectar instrucciones de seguridad (pagos, auth, etc.)
    if hasSecurityInstructions(localAgentsMD) {
        r.hatiClient.ModuleHint(ModuleHint{
            Module:  modulePath,
            Message: "Este módulo tiene notas de seguridad/compliance",
            Action:  "upgrade_risk",
        })
    }
}
```

---

## 10. Formato RSAW v1.0.0 completo

### Rules — sin cambios de formato

El formato de Rules no cambia en v4.0. La novedad es que todos los proyectos incluyen cuatro rules base:

```
.skoll/rules/
├── global.md           ← restricciones universales
├── security.md         ← seguridad
├── copyright.md        ← código generado por IA
└── api-safety.md       ← verificación de APIs externas
```

### Skills — formato v4.0 con SKILL.md

```
.skoll/skills/
├── api-validation/
│   └── SKILL.md
├── clean-architecture/
│   ├── SKILL.md
│   └── references/
│       └── REFERENCE.md
├── testing/
│   └── SKILL.md
└── [nombre]/
    ├── SKILL.md          ← REQUERIDO
    ├── scripts/          ← opcional
    ├── references/       ← opcional
    └── assets/           ← opcional
```

### Agents — sin cambios de formato

El formato de Agents v3.0 se mantiene. La única novedad es que `allowed-tools` de los skills se propaga a la respuesta de `agent_activate`.

### Workflows — sin cambios de formato

El formato de Workflows v3.0 se mantiene. Los workflows pueden referenciar standards de Tyr (no de Fenrir) en el DoD.

---

## 11. MCP Tools — Catálogo completo v1.0.0

### Tools de v1.0-v3.0 (modificados)

`rule_list`, `rule_check`, `rule_get`

`skill_list` *(modificado: retorna índice compacto con metadata only)*
`skill_load` *(modificado: body + available_files + allowed_tools + in_practice)*
`skill_search` *(modificado: busca en nombre, description y metadata)*
`skill_version_check`, `skill_verify`

`agent_list`, `agent_activate` *(modificado: lee AGENTS.md anidado + retorna allowed_tools)*
`agent_context`, `agent_handoff`

`workflow_start`, `workflow_step`, `workflow_status`, `workflow_complete`

`skoll_status`, `skoll_validate` *(modificado: verifica frontmatter SKILL.md)*

`rule_pending`, `rule_promote`
`team_status`, `team_register`
`dod_check`

### Tools nuevas en v4.0 (4 adicionales)

| Tool | Descripción |
|---|---|
| `skill_read_file` | Leer un archivo específico de scripts/, references/ o assets/ de un skill |
| `skills_import` | Importar skill desde SkillsMP o URL de GitHub con security scan automático |
| `skills_update` | Actualizar skill importada desde su fuente original |
| `api_docs_check` | Verificar existencia de endpoint en OpenAPI spec o docs oficial |

**Total v4.0: 29 MCP tools**

### Especificaciones de tools nuevas

#### `skill_read_file`

```json
{
  "name": "skill_read_file",
  "description": "Load a specific file from a skill's scripts/, references/ or assets/ directory",
  "inputSchema": {
    "type": "object",
    "required": ["skill_name", "file_path"],
    "properties": {
      "skill_name": { "type": "string" },
      "file_path":  { "type": "string", "description": "Relative path within skill directory, e.g. 'references/REFERENCE.md'" }
    }
  }
}
```

#### `skills_import`

```json
{
  "name": "skills_import",
  "description": "Import a skill from SkillsMP or GitHub URL with automatic security scanning",
  "inputSchema": {
    "type": "object",
    "required": ["source"],
    "properties": {
      "source": { "type": "string", "enum": ["skillsmp", "github", "local"] },
      "query":  { "type": "string", "description": "Search query for skillsmp" },
      "url":    { "type": "string", "description": "GitHub URL for github source" },
      "skip_scan": { "type": "boolean", "default": false, "description": "NOT recommended" }
    }
  }
}
```

---

## 12. CLI completo v1.0.0

```bash
# ─── INICIALIZACIÓN ──────────────────────────────────
skoll init [--dry-run] [--rules-only] [--from <url>]
    # Genera: RSAW base + skills built-in (desde Fenrir) + SECURITY_CONFIG.md
    # Detecta AGENTS.md existente (no crea SKOLL.md separado)

# ─── SISTEMA ─────────────────────────────────────────
skoll mcp
skoll tui
skoll status
skoll validate [--ci] [--strict]
skoll version

# ─── RULES ───────────────────────────────────────────
skoll rules list [--category <cat>]
skoll rules show <nombre>
skoll rules check "<acción>"
skoll rules pending
skoll rules promote <nombre>
skoll rules reject <nombre>

# ─── SKILLS ──────────────────────────────────────────
skoll skills list [--compact]             # Progressive: solo metadata
skoll skills show <nombre>                # Progressive: body completo + archivos
skoll skills read <nombre> <archivo>      # Progressive: archivo específico

skoll skills add <nombre>                 # Crea directorio con SKILL.md template
skoll skills migrate <nombre>            # Migra skill v3 a formato SKILL.md v4
skoll skills migrate --all               # Migra todos los skills existentes

skoll skills import --from skillsmp --query "<query>"
skoll skills import --from github --url <url>
skoll skills update [<nombre>]           # Actualiza skills importadas
skoll skills publish <nombre>            # Prepara para SkillsMP

skoll skills check [--fix]               # Verifica versiones de frameworks
skoll skills verify <nombre>             # Actualiza last_verified

# ─── API VALIDATION ──────────────────────────────────
skoll api check <servicio> <endpoint> [--method GET|POST|...]
skoll api cache list
skoll api cache clear

# ─── AGENTS ──────────────────────────────────────────
skoll agents list
skoll agents show <nombre>
skoll agents add <nombre>

# ─── WORKFLOWS ───────────────────────────────────────
skoll workflows list
skoll workflows show <nombre>
skoll workflows add <nombre>
skoll workflows run <nombre>
skoll dod check [--workflow <nombre>]

# ─── TEAM ────────────────────────────────────────────
skoll team
skoll team register [--agent <nombre>]
skoll team clear
```

---

## 13. Configuración v1.0.0

### `skoll.json`

```json
{
  "project": "mi-proyecto",
  "version": "1.0.0",

  "skills": {
    "format": "agentskills-v1",
    "directory": ".skoll/skills/",
    "auto_install_builtins": true,
    "progressive_disclosure": true,
    "security_scan_on_import": true
  },

  "skill_versioning": {
    "enabled": true,
    "warn_on_outdated": true,
    "stale_after_days": 60
  },

  "dod": {
    "require_standards_pass": true,
    "tyr_timeout_seconds": 10,
    "fallback_on_tyr_unavailable": "manual"
  },

  "team": {
    "coordination_enabled": true,
    "role_ttl_hours": 4,
    "warn_on_scope_overlap": true
  },

  "inbound_rules": {
    "accept_from_fenrir": true,
    "proposed_dir": ".skoll/rules/_proposed",
    "auto_promote": false
  },

  "api_validation": {
    "enabled": true,
    "cache_ttl_hours": 24,
    "known_specs": {
      "stripe": "https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.json",
      "github": "https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json"
    }
  },

  "copyright": {
    "enabled": true,
    "min_lines_for_review": 10
  },

  "agents_md": {
    "read_nested": true,
    "notify_hati_on_test_instructions": true,
    "notify_hati_on_security_notes": true
  },

  "skillsmp": {
    "enabled": true,
    "registry_url": "https://api.skillsmp.com/v1"
  },

  "fenrir": { "enabled": true },
  "tyr":    { "enabled": true, "scan_imported_skills": true },
  "hati":   { "enabled": true }
}
```

---

## 14. Integración con Fenrir, Tyr y Hati

### Con Fenrir

| Evento Skoll | Interacción Fenrir |
|---|---|
| `skill_load` | Consulta `mem_find` para ejemplos reales (pedagogical links) |
| `api_docs_check` detecta endpoint inventado | `mem_save type:failed_attempt` |
| `workflow_complete` | `standard_run_all` (via Tyr) para DoD verifiable |
| `workflow_start` | `mem_context` + `predict` |
| `agent_handoff` | `mem_save` con contrato |
| `rule_pending` | Lee `_proposed/` escritos por Fenrir |

### Con Tyr

| Evento Skoll | Interacción Tyr |
|---|---|
| `skills_import` cualquier skill externa | `inject_guard` antes de instalar |
| `workflow_complete` con DoD standards | `standard_run_all` para verificar DoD |
| `agent_activate` | Tyr leerá scope activo para scope_enforcer |
| AGENTS.md local con instrucciones de testing | Tyr puede agregar standard correspondiente |

### Con Hati

| Evento Skoll | Interacción Hati |
|---|---|
| `agent_activate` | Hati incluye scope y skills en checkpoints PRE |
| `agent_activate` con AGENTS.md local que tiene testing instructions | Notifica a Hati para ajustar fase |
| `workflow_complete` con DoD | Hati recibe quality_score |
| `agent_handoff` | Hati registra en Approval Record |

---

## 15. Roadmap v1.0.0

| Fase | Semanas | Deliverable |
|---|---|---|
| 1 — Core RSAW | 1–2 | Sistema RSAW base, Rules, Skills, Agents, Workflows |
| 2 — SKILL.md oficial | 3 | Frontmatter YAML, validación AgentSkills.io |
| 3 — Estructura de directorios | 4 | scripts/, references/, assets/ por skill |
| 4 — Progressive Disclosure | 5 | Índice en startup, skill_list compacto, skill_load completo |
| 5 — SkillsMP Import | 6 | Import desde marketplace, security scan vía Tyr |
| 6 — SkillsMP Publish | 7 | Publish hacia marketplace |
| 7 — allowed-tools | 8 | Pre-aprobación de tools del ecosistema |
| 8 — AGENTS.md anidados | 9 | Lectura por context_path, notificación a Hati |
| 9 — Lazy Loading | 10 | Carga on-demand de skills |
| 10 — Compiled Rules | 11 | Rules como funciones Go compiladas |
| 11 — Workflow State Machine | 12 | Transiciones explícitas y verificables |
| 12 — Release v1.0.0 | 13 | Docs, testing, release |

---

## 16. Mejoras Planificadas v1.1.0

Ver documento `PRD_Ragnarok_v1.1_Improvements.md` para especificaciones completas.

| Mejora | Descripción | Prioridad |
|---|---|---|
| **CLI Stats** | `ecosystem_stats` — stats unificados del ecosistema | 🟡 MEDIA |
| **Backup Automation** | Scripts de backup/restore para Windows | 🟡 MEDIA |

*Nota: Distributed Sync (CRDT) está en backlog para v1.2+ ya que en contexto single-user Windows no hay equipos distribuidos.*

### Nuevos tools en v1.1.0

| Tool | Descripción |
|---|---|
| `ecosystem_stats` | Stats unificados del ecosistema (incluye Skoll) |

---

*Skoll PRD v1.0.0 — Marzo 2026*
*~30 MCP tools (v1.1.0) · Go 1.22+ · MIT*
*SKILL.md oficial · SkillsMP · Progressive Disclosure · allowed-tools · CLI Stats*
*3 Pilares: Velocidad ⚡ · Eficiencia de Tokens 💎 · Eficacia 🎯*
