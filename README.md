# Ragnarok Ecosystem v1.1.0

**Memory, Knowledge & Institutional Intelligence for AI Agents**

Ecosistema de 4 plugins MCP diseñados para orquestar agentes AI (OpenCode, Cursor, Windsurf, Claude Code) en proyectos de desarrollo Windows.

---

## Tabla de Contenidos

1. [Arquitectura General](#1-arquitectura-general)
2. [Flujo de Integración](#2-flujo-de-integración)
3. [Plugins Detallados](#3-plugins-detallados)
   - [Fenrir (Memory & Knowledge)](#31-fenrir-memory--knowledge)
   - [Hati (Planning & Approvals)](#32-hari-planning--approvals)
   - [Skoll (Skills & Rules)](#33-skoll-skills--rules)
   - [Tyr (Security & Validation)](#34-tyr-security--validation)
4. [Project Scanner & Bootstrap](#4-project-scanner--bootstrap)
5. [Guía de Usuario](#5-guía-de-usuario)
   - [5.1 Primeros Pasos](#51--primeros-pasos--flujo-completo)
   - [5.2 Comandos Ragnarok](#52--comandos-ragnarok-rag)
   - [5.3 Comandos por Plugin](#53--comandos-por-plugin)
   - [5.4 Ejemplos Prácticos](#54--ejemplos-prácticos)
   - [5.5 Configuración de Agentes](#55--configuración-de-agentes-ai)
   - [5.6 Estructura de Archivos](#56--📁-estructura-de-archivos-generados)
   - [5.7 Solución de Problemas](#57--⚠️-solución-de-problemas)
   - [5.8 Flujo Completo Visual](#58--📊-flujo-completo-visual)
6. [Referencia de Comandos](#6-referencia-de-comandos)
   - [6.1 Comandos Macro](#61-comandos-macro-rag)
   - [6.2 Comandos por Plugin](#62-comandos-por-plugin)
   - [6.3 Puertos Default](#63-puertos-default)
7. [MCP Tools Catalog](#7-mcp-tools-catalog)
8. [Configuración](#8-configuración)
7. [MCP Tools Catalog](#7-mcp-tools-catalog)
8. [Configuración](#8-configuración)

---

## 1. Arquitectura General

### 1.1 Vista del Sistema

```mermaid
flowchart TB
    subgraph User["👤 USUARIO"]
        Prompt["💬 Prompt del Developer"]
    end

    subgraph Agent["🤖 AI AGENT"]
        OpenCode["OpenCode / Cursor / Windsurf / Claude Code"]
    end

    subgraph Ragnarok["📦 RAGNAROK ECOSYSTEM"]
        subgraph ProjectLayer["📁 PROJECT LAYER"]
            AGENTS["AGENTS.md"]
            RagnarokDir[".ragnarok/"]
            Config["package.json, go.mod, etc."]
        end

        subgraph Plugins["🔌 MCP PLUGINS"]
            FEN["🧠 FENRIR<br/>Port 7437"]
            HA["📋 HATI<br/>Port 7439"]
            SK["⚙️ SKOLL<br/>Port 7438"]
            TY["🛡️ TYR<br/>Port 7440"]
        end

        subgraph SharedData["📊 SHARED DATA"]
            Specs["Specs"]
            Skills["Skills"]
            Rules["Rules"]
            Standards["Standards"]
        end
    end

    Prompt -->|"Escribe prompt"| OpenCode
    OpenCode -->|"MCP JSON-RPC"| FEN
    OpenCode -->|"MCP JSON-RPC"| HA
    OpenCode -->|"MCP JSON-RPC"| SK
    OpenCode -->|"MCP JSON-RPC"| TY

    FEN <-->|"specs, context"| SharedData
    SK <-->|"skills, rules"| SharedData
    TY <-->|"standards"| SharedData

    AGENTS -.->|"Lee contexto"| SK
    RagnarokDir -.->|"Bootstrap data"| SK
    RagnarokDir -.->|"Bootstrap data"| TY

    OpenCode -->|"6. Código aprobado"| Result["✅ Código Aplicado"]
```

### 1.2 Responsabilidades de Cada Plugin

```mermaid
quadrantChart
    title Posicionamiento de Plugins
    x-axis Baja Frecuencia --> Alta Frecuencia
    y-axis Baja Criticidad --> Alta Criticidad
    Fenrir: [0.7, 0.8]
    Hati: [0.3, 0.9]
    Skoll: [0.6, 0.6]
    Tyr: [0.8, 0.7]
```

| Plugin | Pregunta que responde | Frecuencia | Criticidad |
|--------|---------------------|------------|------------|
| **Fenrir** | ¿Qué sabemos sobre el proyecto? | Alta | Alta |
| **Hati** | ¿Cuál es el plan y quién lo aprueba? | Baja | Muy Alta |
| **Skoll** | ¿Quién hace qué y cómo? | Media | Media |
| **Tyr** | ¿Es seguro lo que hacemos? | Alta | Alta |

---

## 2. Flujo de Integración

### 2.1 Flujo Completo de un Prompt

```mermaid
sequenceDiagram
    participant U as 👤 Usuario
    participant A as 🤖 Agente
    participant F as 🧠 Fenrir
    participant S as ⚙️ Skoll
    participant H as 📋 Hati
    participant T as 🛡️ Tyr

    U->>A: "Implementa OAuth2 con Google"
    A->>F: prompt_analyze(prompt)
    F-->>A: is_relevant: true, confidence: 0.8
    A->>S: agent_activate(context_path)
    S-->>A: AGENTS.md + skills + allowed_tools
    A->>F: intent_save(plan_id, prompt, items)
    F-->>A: intent_id guardado
    A->>H: plan_create(title, risk_level)
    H-->>A: plan_id: "plan_abc123"
    A->>H: checkpoint_open(plan_id, type: "pre")
    H-->>A: can_continue: false
    H->>U: ⏳ Solicita Approval
    U->>H: checkpoint_decide(approved)
    H-->>A: can_continue: true
    A->>T: precommit_validate(files)
    T-->>A: findings: []
    A->>F: intent_verify(plan_id, files)
    F-->>A: coverage_score: 0.95, can_continue: true
    A->>H: checkpoint_decide(approved)
    H-->>A: ✓ Completado
    A->>U: ✅ Código implementado
```

### 2.2 Diagrama de Flujo de Datos

```mermaid
flowchart LR
    subgraph Scanner["🔍 PROJECT SCANNER"]
        Scan["fenrir scan"]
        Bootstrap["fenrir bootstrap"]
    end

    subgraph ProjectFiles["📁 .ragnarok/"]
        SkillsJSON["skills.json"]
        RulesJSON["rules.json"]
        StandardsJSON["standards.json"]
    end

    subgraph SK["⚙️ SKOLL"]
        SkillTable["skills DB"]
        RuleTable["rules DB"]
    end

    subgraph TY["🛡️ TYR"]
        StandardTable["standards DB"]
    end

    Scan -->|Detecta stack/arch| Bootstrap
    Bootstrap -->|Genera| ProjectFiles
    SkillsJSON -->|bootstrap_import| SK
    RulesJSON -->|bootstrap_import| SK
    StandardsJSON -->|bootstrap_import| TY
```

### 2.3 Gatekeeping de Prompts

```mermaid
flowchart TD
    Start["📥 Prompt recibido"]
    Analyze{"prompt_analyze"}
    Relevant{"is_relevant?"}
    Action["suggested_action"]
    Reject["❌ REJECT"]
    Proceed["✅ PROCEED"]
    Clarify["🤔 REQUEST CLARIFICATION"]

    Start --> Analyze
    Analyze --> Relevant

    Relevant -->|"true, confidence > 0.7"| Proceed
    Relevant -->|"true, confidence 0.5-0.7"| Action
    Relevant -->|"false"| Reject
    Relevant -->|"confidence < 0.5"| Clarify

    Action --> Proceed
    Action --> Clarify
```

---

## 3. Plugins Detallados

### 3.1 Fenrir (Memory & Knowledge)

```mermaid
flowchart TB
    subgraph Fenrir["🧠 FENRIR - Memory & Knowledge"]
        subgraph Memory["📚 MEMORY"]
            KG["🕸️ Knowledge Graph<br/>nodes, edges, FTS5"]
            Sessions["📝 Sessions<br/>context, checkpoint_id"]
            Obs["📊 Observations<br/>decisions, prompts, code"]
        end

        subgraph Specs["📋 SPECS MODULE"]
            SpecSave["spec_save"]
            SpecList["spec_list"]
            SpecCheck["spec_check"]
            SpecDelta["spec_delta"]
        end

        subgraph Intent["🎯 INTENT TRACKING"]
            IntentSave["intent_save"]
            IntentVerify["intent_verify"]
            IntentGet["intent_get"]
        end

        subgraph Analysis["🔍 ANALYSIS"]
            PromptAnalyze["prompt_analyze"]
            BiasReport["bias_report"]
            Authority["authority_get/set"]
        end

        subgraph Scanner["🔍 PROJECT SCANNER"]
            ProjectScan["project_scan"]
            ProjectBootstrap["project_bootstrap"]
            AgentsMdGet["agents_md_get"]
        end

        subgraph Incidents["🚨 INCIDENTS"]
            IncidentSave["incident_save"]
            IncidentResolve["incident_resolve"]
            ConflictDetect["conflict_detect"]
        end
    end

    Memory --> Specs
    Specs --> Intent
    Intent --> Analysis
    Scanner --> Analysis
```

#### Fenrir MCP Tools (25)

| Category | Tools |
|----------|-------|
| **Memory** | `mem_session_start`, `mem_session_end`, `mem_save`, `mem_find`, `mem_get_observation` |
| **Specs** | `spec_save`, `spec_list`, `spec_check`, `spec_delta`, `spec_related` |
| **Intent** | `intent_save`, `intent_verify`, `intent_get` |
| **Bias** | `bias_report` |
| **Authority** | `authority_get`, `authority_set` |
| **Incidents** | `incident_save`, `incident_list`, `incident_resolve`, `conflict_detect` |
| **Scanner** | `prompt_analyze`, `project_scan`, `project_bootstrap`, `agents_md_get` |
| **Knowledge** | `knowledge_query`, `knowledge_add` |

### 3.2 Hati (Planning & Approvals)

```mermaid
flowchart TB
    subgraph Hati["📋 HATI - Planning & Approvals"]
        subgraph Plans["📝 PLANS"]
            PlanCreate["plan_create"]
            PlanGet["plan_get"]
            PlanUpdate["plan_update"]
            PlanList["plan_list"]
        end

        subgraph Phases["🔄 PHASES"]
            PhaseAdd["phase_add"]
            PhaseList["phase_list"]
            PhaseComplete["phase_complete"]
        end

        subgraph Checkpoints["⏳ CHECKPOINTS"]
            CpOpen["checkpoint_open"]
            CpDecide["checkpoint_decide"]
            CpStatus["checkpoint_status"]
        end

        subgraph Decisions["✅ DECISIONS"]
            DecisionRecord["decision_record"]
            DecisionGet["decision_get"]
            DecisionList["decision_list"]
            DecisionCancel["decision_cancel"]
        end

        subgraph Quality["📊 QUALITY"]
            QualitySnapshot["quality_snapshot"]
            SpecImpact["spec_impact"]
            SpecStatus["spec_status"]
        end

        subgraph Feedback["💬 FEEDBACK"]
            FeedbackSave["feedback_save"]
            FeedbackList["feedback_list"]
        end
    end

    Plans --> Phases
    Phases --> Checkpoints
    Checkpoints --> Decisions
    Decisions --> Quality
    Quality --> Feedback
```

#### Hati MCP Tools (22)

| Category | Tools |
|----------|-------|
| **Plans** | `plan_create`, `plan_get`, `plan_update`, `plan_list` |
| **Phases** | `phase_add`, `phase_list`, `phase_complete` |
| **Checkpoints** | `checkpoint_open`, `checkpoint_decide`, `checkpoint_status` |
| **Decisions** | `decision_record`, `decision_get`, `decision_list`, `decision_cancel` |
| **Quality** | `quality_snapshot`, `spec_impact`, `spec_status` |
| **Feedback** | `feedback_save`, `feedback_list` |
| **Status** | `hati_status`, `hati_stats` |

### 3.3 Skoll (Skills & Rules)

```mermaid
flowchart TB
    subgraph Skoll["⚙️ SKOLL - Skills & Rules"]
        subgraph Skills["🎯 SKILLS"]
            SkillList["skill_list"]
            SkillLoad["skill_load"]
            SkillSearch["skill_search"]
            SkillVerify["skill_verify"]
            SkillRead["skill_read_file"]
            BootstrapImport["bootstrap_import"]
        end

        subgraph Rules["📏 RULES"]
            RuleList["rule_list"]
            RuleCheck["rule_check"]
            RuleGet["rule_get"]
        end

        subgraph Agents["🤖 AGENTS"]
            AgentList["agent_list"]
            AgentActivate["agent_activate"]
            AgentContext["agent_context"]
            AgentHandoff["agent_handoff"]
        end

        subgraph Workflows["🔀 WORKFLOWS"]
            WorkflowStart["workflow_start"]
            WorkflowProgress["workflow_progress"]
            WorkflowComplete["workflow_complete"]
        end

        subgraph RSAW["⚡ RSAW"]
            DodCheck["dod_check"]
            TeamRegister["team_register"]
            ApiDocsCheck["api_docs_check"]
        end
    end

    Skills --> Rules
    Agents --> Skills
    Workflows --> RSAW
    Skills --> BootstrapImport
```

#### Skoll MCP Tools (28)

| Category | Tools |
|----------|-------|
| **Skills** | `skill_list`, `skill_load`, `skill_search`, `skill_verify`, `skill_read_file`, `bootstrap_import` |
| **Rules** | `rule_list`, `rule_check`, `rule_get` |
| **Agents** | `agent_list`, `agent_activate`, `agent_context`, `agent_handoff` |
| **Workflows** | `workflow_start`, `workflow_progress`, `workflow_complete` |
| **RSAW** | `dod_check`, `team_register`, `api_docs_check`, `api_usage_verify` |

### 3.4 Tyr (Security & Validation)

```mermaid
flowchart TB
    subgraph Tyr["🛡️ TYR - Security & Validation"]
        subgraph Precommit["⚡ PRE-COMMIT"]
            Validate["precommit_validate"]
            Autofix["precommit_autofix"]
        end

        subgraph Packages["📦 PACKAGES"]
            PkgCheck["pkg_check"]
            PkgAudit["pkg_audit"]
            CveAlerts["cve_alerts"]
        end

        subgraph SAST["🔍 SAST"]
            SastRun["sast_run"]
            SastResolve["sast_resolve"]
            SastList["sast_list"]
        end

        subgraph Standards["📐 STANDARDS"]
            StdRun["standard_run"]
            StdRunAll["standard_run_all"]
            StdList["standard_list"]
        end

        subgraph Scope["🎯 SCOPE"]
            ScopeCheck["scope_check"]
            ScopeViolations["scope_violations"]
        end

        subgraph Secrets["🔐 SECRETS"]
            SecretsScan["secrets_scan"]
            SecretsResolve["secrets_resolve"]
        end

        subgraph Security["🛡️ SECURITY"]
            InjectGuard["inject_guard"]
            Sanitize["sanitize"]
            ProactiveScan["proactive_scan"]
            BootstrapImport["bootstrap_import"]
        end
    end

    Precommit --> Packages
    Packages --> SAST
    SAST --> Scope
    Scope --> Secrets
    Standards --> Precommit
    Security --> Precommit
```

#### Tyr MCP Tools (21)

| Category | Tools |
|----------|-------|
| **Precommit** | `precommit_validate`, `precommit_autofix` |
| **Packages** | `pkg_check`, `pkg_audit`, `cve_alerts` |
| **SAST** | `sast_run`, `sast_resolve`, `sast_list` |
| **Standards** | `standard_run`, `standard_run_all`, `standard_list` |
| **Scope** | `scope_check`, `scope_violations` |
| **Secrets** | `secrets_scan`, `secrets_resolve` |
| **Security** | `inject_guard`, `sanitize`, `proactive_scan`, `bootstrap_import` |

---

## 4. Project Scanner & Bootstrap

### 4.1 Flujo del Scanner

```mermaid
flowchart LR
    subgraph Input["📥 INPUT"]
        Cmd["fenrir scan --path ./proyecto"]
    end

    subgraph Layers["🔍 CAPAS DE ANALISIS"]
        Stack["--layer stack<br/>Language, Framework, Package Mgr"]
        Arch["--layer arch<br/>Type, Modules, API, Frontend"]
        Config["--layer config<br/>Config Files"]
        Modules["--layer modules<br/>Project Modules"]
        Patterns["--layer patterns<br/>Testing, CI/CD, Docker"]
    end

    subgraph Output["📤 OUTPUT"]
        ScanResult["Detected Stack/Architecture"]
    end

    Cmd --> Layers
    Stack --> Output
    Arch --> Output
    Config --> Output
    Modules --> Output
    Patterns --> Output
```

### 4.2 Tecnologías Detectadas

#### Lenguajes y Frameworks

| Lenguaje | Frameworks Detectados |
|----------|----------------------|
| **JavaScript/TypeScript** | React, Vue, Svelte, SvelteKit, Next.js, Nuxt, Astro, Gatsby, Remix, Express, Fastify, Vite |
| **Go** | Gin, Gorilla-Mux, Chi, Fiber, Echo |
| **Python** | Django, Flask, FastAPI |
| **Rust** | Actix-Web, Axum, Warp, Rocket |
| **Java** | Spring Boot, Quarkus, Micronaut |

#### Package Managers

| Manager | Ecosistema |
|---------|------------|
| **npm** | Node.js/JavaScript |
| **pip** | Python |
| **cargo** | Rust |
| **maven/gradle** | Java |

#### Test Frameworks

| Framework | Lenguaje |
|-----------|----------|
| Jest, Vitest | JavaScript/TypeScript |
| pytest | Python |
| go test | Go |
| rust test | Rust |

#### CI/CD

| Tool | Detección |
|------|-----------|
| GitHub Actions | `.github/workflows/*.yml` |
| Azure Pipelines | `azure-pipelines.yml` |
| GitLab CI | `.gitlab-ci.yml` |
| Jenkins | `Jenkinsfile` |

#### Databases

| DB | Detección |
|----|-----------|
| SQLite | `db.json`, `migrations/`, `seeds/` |
| PostgreSQL | `postgres.env`, archivos de config |
| MySQL | `mysql.env` |
| MongoDB | `mongodb.env` |

### 4.3 Bootstrap Output

```mermaid
flowchart TB
    subgraph Bootstrap["📦 BOOTSTRAP"]
        Cmd["fenrir bootstrap --path ./proyecto"]
    end

    subgraph RagnarokDir[".ragnarok/"]
        subgraph Files["📄 FILES"]
            SkillsJSON["skills.json"]
            RulesJSON["rules.json"]
            StandardsJSON["standards.json"]
        end
    end

    subgraph AGENTS["📋 AGENTS.MD"]
        Stack["## Project Stack"]
        Rules["## Project Rules"]
        Standards["## Quality Standards"]
        Skills["## Suggested Skills"]
        Cmds["## Commands"]
    end

    subgraph Integration["🔗 INTEGRATION"]
        SK["skoll.bootstrap_import"]
        TY["tyr.bootstrap_import"]
    end

    Cmd --> Bootstrap
    Bootstrap --> RagnarokDir
    RagnarokDir --> AGENTS
    RagnarokDir --> Integration

    SkillsJSON --> SK
    RulesJSON --> SK
    StandardsJSON --> TY
```

---

## 5. Guía de Usuario

### 5.1 🚀 Primeros Pasos - Flujo Completo

```mermaid
flowchart LR
    A["📁 cd tu-proyecto"] --> B["🚀 rag init"]
    B --> C["🔍 rag scan"]
    C --> D["📋 rag integrate"]
    D --> E["🖥️ rag serve"]
    E --> F["✅ Listo!"]
```

#### Paso 1: Inicializar el Ecosistema

```bash
# Navega a tu proyecto
cd mi-proyecto

# Inicializa TODOS los plugins Ragnarok de una vez
rag init --project "mi-proyecto"
```

**¿Qué hace?**
- Crea `.fenrir/`, `.hati/`, `.skoll/`, `.tyr/` en tu `$HOME`
- Genera `config.json` en cada directorio con tu proyecto configurado
- Muestra confirmación con los directorios creados

**Salida esperada:**
```
Ragnarok Init - Initializing all plugins
Project: mi-proyecto
──────────────────────────────────────────────────

📦 Initializing FENRIR...
   Directory: C:\Users\tu\.fenrir
   ✓ Config: C:\Users\tu\.fenrir\config.json

📦 Initializing HATI...
   Directory: C:\Users\tu\.hati
   ✓ Config: C:\Users\tu\.hati\config.json

📦 Initializing SKOLL...
   Directory: C:\Users\tu\.skoll
   ✓ Config: C:\Users\tu\.skoll\config.json

📦 Initializing TYR...
   Directory: C:\Users\tu\.tyr
   ✓ Config: C:\Users\tu\.tyr\config.json

✓ All plugins initialized for project: mi-proyecto
```

---

#### Paso 2: Escanear tu Proyecto

```bash
# Escanear el proyecto actual
rag scan --path .

# Escanar otro directorio
rag scan --path ./frontend

# Escanar sin hacer bootstrap (solo análisis)
rag scan --path . --bootstrap=false
```

**¿Qué hace?**
- Analiza la estructura de tu proyecto
- Detecta: lenguaje, framework, package manager, arquitectura
- Genera `.ragnarok/skills.json`, `.ragnarok/rules.json`, `.ragnarok/standards.json`
- Genera `AGENTS.md` con guidelines para el agente AI

**Salida esperada:**
```
Ragnarok Scan - Analyzing project
Project: C:\path\to\mi-proyecto
──────────────────────────────────────────────────

🔍 Running project analysis...

📦 STACK
   Language: typescript
   Framework: next.js
   Package Manager: npm
   Docker: false | CI/CD: true | Tests: true

🏗️  ARCHITECTURE
   Type: monolith
   Modules: app, components, lib, hooks

📁 MODULES (4)
   • app (Next.js)
   • components (React)
   • lib (Utils)
   • hooks (Custom Hooks)

🔍 PATTERNS DETECTED
   ✓ Unit Testing (jest)
   ✓ CI/CD (GitHub Actions)
   ✓ TypeScript

📦 Generating bootstrap files...
✓ Created: .ragnarok\skills.json
✓ Created: .ragnarok\rules.json
✓ Created: .ragnarok\standards.json

📝 Generating AGENTS.md...
✓ Generated: AGENTS.md

✓ Scan complete!
```

---

#### Paso 3: Ver la Integración

```bash
# Ver qué datos se generaron
rag integrate --path .
```

**¿Qué hace?**
- Lee los archivos `.ragnarok/*.json` generados
- Muestra un resumen de skills, rules y standards detectados
- Te dice cómo importarlos a cada plugin

**Salida esperada:**
```
Ragnarok Integration
Project: C:\path\to\mi-proyecto
──────────────────────────────────────────────────

Bootstrap data loaded:
  Skills: 3
  Rules: 2
  Standards: 2

📦 SKILLS:
  • nextjs (framework): next.js
  • typescript (language): typescript
  • jest (testing): jest

📋 RULES:
  • no-commit-without-tests [high]: Commits must include tests
  • strict-typescript [medium]: Avoid 'any' types

✅ STANDARDS:
  • test-pass (blocks merge): All tests must pass
  • lint (non-blocking): Run linter to check code style

--------------------------------------------------
Next: Import these to plugins via MCP commands
```

---

#### Paso 4: Iniciar los Servidores

```bash
# Iniciar TODOS los servidores MCP
rag serve

# Iniciar en un directorio específico de datos
rag serve --dir ~/mis-datos-ragnarok
```

**¿Qué hace?**
- Inicia fenrir, hati, skoll, tyr en puertos 7437-7440
- Cada servidor queda corriendo en background
- Presiona Ctrl+C para detenerlos

**Salida esperada:**
```
Ragnarok Serve - Starting all MCP servers
Base directory: C:\Users\tu
──────────────────────────────────────────────────

⚠️  Press Ctrl+C to stop all servers

Starting fenrir on port 7437...
✓ fenrir started (PID: 12345)
Starting hati on port 7439...
✓ hati started (PID: 12346)
Starting skoll on port 7438...
✓ skoll started (PID: 12347)
Starting tyr on port 7440...
✓ tyr started (PID: 12348)

✓ All 4 servers running
Press Ctrl+C to stop...
```

---

#### Paso 5: Verificar Salud del Sistema

```bash
# Ver estado de todos los plugins
rag stats --ecosystem

# Ver stats de un plugin específico
rag stats --plugin fenrir
rag stats --plugin hati
rag stats --plugin skoll
rag stats --plugin tyr
```

**¿Qué hace?**
- Consulta el estado de cada plugin via HTTP
- Muestra latencia y datos relevantes

**Salida esperada:**
```
RAGNAROK Ecosystem Health
──────────────────────────────────────────
✓ Fenrir: online (latency: 12ms) [nodes: 47]
✓ Hati: online (latency: 8ms) [plans: 5]
✓ Skoll: online (latency: 15ms) [skills: 23]
✓ Tyr: online (latency: 10ms) [findings: 2]
──────────────────────────────────────────
Overall: ✓ Healthy
```

---

### 5.2 📋 Comandos Ragnarok (rag)

El comando `rag` es el orquestador central. Gestiona todos los plugins desde un solo punto.

```bash
# AYUDA - Ver todos los comandos disponibles
rag

# INICIALIZAR
rag init --project "nombre-proyecto"           # Todos los plugins
rag init --project "nombre" --dir ~/custom     # Directorio custom

# ESCANEAR
rag scan --path ./mi-proyecto                 # Scan + bootstrap
rag scan --path .                             # Scan proyecto actual
rag scan --path ./frontend --bootstrap=false  # Solo scan, sin bootstrap

# SERVIDORES
rag serve                                     # Iniciar todos
rag serve --dir ~/mis-datos                   # Con dir custom
rag stop                                     # Detener todos

# ESTADISTICAS
rag stats --ecosystem                         # Ver todos los plugins
rag stats --plugin fenrir                     # Ver uno específico

# BACKUP Y RESTORE
rag backup --plugin all                       # Backup todos
rag backup --plugin fenrir                    # Backup uno
rag backup --plugin all --dir ~/backups       # Dir custom
rag restore --plugin fenrir --file ~/backups/fenrir_2026-03-25.zip

# INSTALACION
rag install --project "mi-proyecto" --mcp opencode    # Config OpenCode
rag install --project "mi-proyecto" --mcp cursor       # Config Cursor
rag install --project "mi-proyecto" --mcp windsurf    # Config Windsurf

# VERSION
rag version
```

---

### 5.3 🔧 Comandos por Plugin

Cada plugin también tiene sus propios comandos independientes.

#### Fenrir (Memory & Knowledge)

```bash
# Iniciar servidor MCP
fenrir serve --port 7437

# Inicializar con directorio custom
fenrir init --project "mi-proyecto" --dir ~/.fenrir

# Escanear proyecto (capas)
fenrir scan --path ./mi-proyecto                     # Todo
fenrir scan --path ./mi-proyecto --layer stack        # Solo stack
fenrir scan --path ./mi-proyecto --layer arch         # Solo arquitectura
fenrir scan --path ./mi-proyecto --layer patterns     # Solo patrones
fenrir scan --path ./mi-proyecto --layer modules      # Solo módulos
fenrir scan --path ./mi-proyecto --layer config      # Solo config

# Generar estructura .ragnarok/
fenrir bootstrap --path ./mi-proyecto

# Generar AGENTS.md
fenrir init --project "Mi Proyecto"

# Ver stats
fenrir stats

# Version
fenrir version
```

#### Hati (Planning & Approvals)

```bash
# Iniciar servidor MCP
hati serve --port 7439

# Inicializar
hati init --project "mi-proyecto" --dir ~/.hati

# Version
hati version
```

#### Skoll (Skills & Rules)

```bash
# Iniciar servidor MCP
skoll serve --port 7438

# Inicializar
skoll init --project "mi-proyecto" --dir ~/.skoll

# Version
skoll version
```

#### Tyr (Security & Validation)

```bash
# Iniciar servidor MCP
tyr serve --port 7440

# Inicializar
tyr init --project "mi-proyecto" --dir ~/.tyr

# Version
tyr version
```

---

### 5.4 💡 Ejemplos Prácticos

#### Ejemplo 1: Nuevo Proyecto Node.js

```bash
# 1. Crear proyecto
mkdir mi-app && cd mi-app
npm init -y && npm install express react

# 2. Inicializar Ragnarok
rag init --project "mi-app"

# 3. Escanear y detectar stack
rag scan --path .

# 4. Ver qué se detectó
rag integrate --path .

# 5. Iniciar servidores
rag serve
```

#### Ejemplo 2: Proyecto Go con Múltiples Módulos

```bash
# 1. Ir al proyecto
cd mi-proyecto-go

# 2. Inicializar
rag init --project "mi-proyecto-go"

# 3. Scan específico para arquitectura
fenrir scan --path . --layer arch

# 4. Iniciar
rag serve

# 5. Ver stats
rag stats --ecosystem
```

#### Ejemplo 3: Verificar Salud del Sistema

```bash
# Ver estado general
rag stats --ecosystem

# Si algo falla, ver plugin específico
rag stats --plugin fenrir
rag stats --plugin hati

# Hacer backup si todo está bien
rag backup --plugin all
```

#### Ejemplo 4: Recuperar de un Error

```bash
# Si algo sale mal, hacer restore
rag restore --plugin fenrir --file ~/backups/fenrir_2026-03-20.zip

# Reiniciar servidores
rag stop
rag serve
```

#### Ejemplo 5: Configurar en Cursor

```bash
# Generar config para Cursor
rag install --project "mi-proyecto" --mcp cursor

# Seguir las instrucciones del installer
```

---

### 5.5 🔌 Configuración de Agentes AI

#### OpenCode

```bash
rag install --project "mi-proyecto" --mcp opencode
```

#### Cursor

```bash
rag install --project "mi-proyecto" --mcp cursor
```

#### Windsurf

```bash
rag install --project "mi-proyecto" --mcp windsurf
```

---

### 5.6 📁 Estructura de Archivos Generados

```
mi-proyecto/
├── .ragnarok/                    # Datos de bootstrap
│   ├── skills.json               # Skills sugeridos
│   ├── rules.json                # Reglas del proyecto
│   └── standards.json            # Estándares de calidad
│
├── AGENTS.md                     # Guidelines para agentes AI
│
└── [tus archivos de código]
```

```
%USERPROFILE%/
├── .fenrir/                     # Datos de Fenrir
│   ├── fenrir.db                # Knowledge graph
│   └── config.json              # Configuración
│
├── .hati/                       # Datos de Hati
│   ├── hati.db                  # Plans y checkpoints
│   └── config.json
│
├── .skoll/                     # Datos de Skoll
│   ├── skoll.db                # Skills y rules
│   └── config.json
│
└── .tyr/                       # Datos de Tyr
    ├── tyr.db                  # Security findings
    └── config.json
```

---

### 5.7 ⚠️ Solución de Problemas

| Problema | Solución |
|----------|----------|
| `rag: command not found` | Agregar el directorio de binarios al PATH |
| `port already in use` | Cambiar puerto con `--port` o matar proceso |
| `sqlite3: CGO required` | Los binarios pre-compilados no tienen CGO |
| Servidor no responde | Verificar que el proceso está corriendo |
| `bootstrap: no such file` | Ejecutar `rag scan` primero |

---

### 5.8 📊 Flujo Completo Visual

```mermaid
sequenceDiagram
    participant U as 👤 Usuario
    participant R as 🟠 rag
    participant F as 🟣 fenrir
    participant H as 🔵 hati
    participant S as 🟢 skoll
    participant T as 🟡 tyr

    U->>R: rag init --project "mi-proyecto"
    R->>F: fenrir init
    R->>H: hati init
    R->>S: skoll init
    R->>T: tyr init
    R-->>U: ✓ Inicializado

    U->>R: rag scan --path .
    R->>F: fenrir scan
    F-->>R: Stack: TypeScript, Next.js
    R->>F: fenrir bootstrap
    R->>F: fenrir init (AGENTS.md)
    R-->>U: ✓ Scan completo

    U->>R: rag serve
    R->>F: fenrir serve --port 7437
    R->>H: hati serve --port 7439
    R->>S: skoll serve --port 7438
    R->>T: tyr serve --port 7440
    R-->>U: ✓ 4 servidores corriendo

    U->>R: rag stats --ecosystem
    R->>F: GET /stats
    R->>H: GET /stats
    R->>S: GET /stats
    R->>T: GET /stats
    R-->>U: Estado: ✓✓✓✓ Healthy
```

---

## 6. Referencia de Comandos

### 6.1 Comandos Macro (rag)

| Comando | Descripción |
|---------|-------------|
| `rag init --project NAME` | Inicializa TODOS los plugins de una vez |
| `rag scan --path PATH [--bootstrap]` | Escanea proyecto + bootstrap |
| `rag serve [--dir DIR]` | Inicia TODOS los servidores MCP |
| `rag stop` | Detiene TODOS los servidores |
| `rag stats --ecosystem` | Stats unificados del ecosistema |
| `rag stats --plugin NAME` | Stats de un plugin específico |
| `rag backup --plugin NAME` | Backup de plugin(s) |
| `rag restore --plugin NAME --file FILE` | Restaurar desde backup |
| `rag integrate --path PATH` | Ver datos bootstrap |
| `rag install --project NAME --mcp CLIENT` | Generar config MCP |
| `rag version` | Mostrar versión |

### 6.2 Comandos por Plugin

| Plugin | Comando | Descripción |
|--------|---------|-------------|
| fenrir | `serve [--port PORT]` | Iniciar servidor MCP |
| fenrir | `scan --path PATH [--layer LAYER]` | Escanear proyecto |
| fenrir | `bootstrap --path PATH` | Generar estructura .ragnarok |
| fenrir | `init --project NAME` | Inicializar proyecto |
| fenrir | `version` | Mostrar versión |
| hati | `serve [--port PORT]` | Iniciar servidor MCP |
| hati | `init --project NAME` | Inicializar proyecto |
| hati | `version` | Mostrar versión |
| skoll | `serve [--port PORT]` | Iniciar servidor MCP |
| skoll | `init --project NAME` | Inicializar proyecto |
| skoll | `version` | Mostrar versión |
| tyr | `serve [--port PORT]` | Iniciar servidor MCP |
| tyr | `init --project NAME` | Inicializar proyecto |
| tyr | `version` | Mostrar versión |

### 6.3 Puertos Default

| Plugin | Puerto | Protocolo |
|--------|--------|-----------|
| Fenrir | 7437 | stdio / TCP |
| Hati | 7439 | stdio / TCP |
| Skoll | 7438 | stdio / TCP |
| Tyr | 7440 | stdio / TCP |

---

## 7. MCP Tools Catalog

### 7.1 Fenrir Tools (25)

| Category | Tools |
|----------|-------|
| **Memory** | `mem_session_start`, `mem_session_end`, `mem_save`, `mem_find`, `mem_get_observation` |
| **Specs** | `spec_save`, `spec_list`, `spec_check`, `spec_delta`, `spec_related` |
| **Intent** | `intent_save`, `intent_verify`, `intent_get` |
| **Bias** | `bias_report` |
| **Authority** | `authority_get`, `authority_set` |
| **Incidents** | `incident_save`, `incident_list`, `incident_resolve`, `conflict_detect` |
| **Scanner** | `prompt_analyze`, `project_scan`, `project_bootstrap`, `agents_md_get` |
| **Knowledge** | `knowledge_query`, `knowledge_add` |

### 7.2 Hati Tools (22)

| Category | Tools |
|----------|-------|
| **Plans** | `plan_create`, `plan_get`, `plan_update`, `plan_list` |
| **Phases** | `phase_add`, `phase_list`, `phase_complete` |
| **Checkpoints** | `checkpoint_open`, `checkpoint_decide`, `checkpoint_status` |
| **Decisions** | `decision_record`, `decision_get`, `decision_list`, `decision_cancel` |
| **Quality** | `quality_snapshot`, `spec_impact`, `spec_status` |
| **Feedback** | `feedback_save`, `feedback_list` |
| **Status** | `hati_status`, `hati_stats` |

### 7.3 Skoll Tools (28)

| Category | Tools |
|----------|-------|
| **Skills** | `skill_list`, `skill_load`, `skill_search`, `skill_verify`, `skill_read_file`, `bootstrap_import` |
| **Rules** | `rule_list`, `rule_check`, `rule_get` |
| **Agents** | `agent_list`, `agent_activate`, `agent_context`, `agent_handoff` |
| **Workflows** | `workflow_start`, `workflow_progress`, `workflow_complete` |
| **RSAW** | `dod_check`, `team_register`, `api_docs_check`, `api_usage_verify` |

### 7.4 Tyr Tools (21)

| Category | Tools |
|----------|-------|
| **Precommit** | `precommit_validate`, `precommit_autofix` |
| **Packages** | `pkg_check`, `pkg_audit`, `cve_alerts` |
| **SAST** | `sast_run`, `sast_resolve`, `sast_list` |
| **Standards** | `standard_run`, `standard_run_all`, `standard_list` |
| **Scope** | `scope_check`, `scope_violations` |
| **Secrets** | `secrets_scan`, `secrets_resolve` |
| **Security** | `inject_guard`, `sanitize`, `proactive_scan`, `bootstrap_import` |

---

## 8. Configuración

### 8.1 Variables de Entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `FENRIR_DIR` | `~/.fenrir` | Directorio de datos Fenrir |
| `HATI_DIR` | `~/.hati` | Directorio de datos Hati |
| `SKOLL_DIR` | `~/.skoll` | Directorio de datos Skoll |
| `TYR_DIR` | `~/.tyr` | Directorio de datos Tyr |

### 8.2 Estructura de Directorios

```mermaid
graph TD
    UserHome["%USERPROFILE%/"]

    subgraph Fenrir[".fenrir/"]
        FDB["fenrir.db"]
        FConfig["config.json"]
        FSkills["skills/"]
    end

    subgraph Hati[".hati/"]
        HDB["hati.db"]
        HConfig["config.json"]
    end

    subgraph Skoll[".skoll/"]
        SDB["skoll.db"]
        SConfig["config.json"]
        SSkills["skills/"]
        SRules["rules/"]
    end

    subgraph Tyr[".tyr/"]
        TDB["tyr.db"]
        TConfig["config.json"]
        TCache["cache/"]
        TSemgrep["semgrep-rules/"]
    end

    UserHome --> Fenrir
    UserHome --> Hati
    UserHome --> Skoll
    UserHome --> Tyr
```

### 8.3 Configuración MCP (OpenCode/Cursor)

```json
{
  "mcpServers": {
    "fenrir": {
      "command": "fenrir",
      "args": ["serve", "--port", "7437"],
      "env": {
        "FENRIR_DIR": "C:\\Users\\you\\.fenrir"
      }
    },
    "hati": {
      "command": "hati",
      "args": ["serve", "--port", "7439"],
      "env": {
        "HATI_DIR": "C:\\Users\\you\\.hati"
      }
    },
    "skoll": {
      "command": "skoll",
      "args": ["serve", "--port", "7438"],
      "env": {
        "SKOLL_DIR": "C:\\Users\\you\\.skoll"
      }
    },
    "tyr": {
      "command": "tyr",
      "args": ["serve", "--port", "7440"],
      "env": {
        "TYR_DIR": "C:\\Users\\you\\.tyr"
      }
    }
  }
}
```

---

## Documentación Adicional

- [Fenrir PRD](PRD_Fenrir_v1.md) - Memory & Knowledge
- [Hati PRD](PRD_Hati_v1.md) - Planning & Approvals
- [Skoll PRD](PRD_Skoll_v1.md) - RSAW & Skills
- [Tyr PRD](PRD_Tyr_v1.md) - Security & Validation
- [Improvements v1.1](PRD_Ragnarok_v1.1_Improvements.md) - Nuevas features
- [Operations Guide](OPS_Windows.md) - Despliegue en Windows

---

**Version:** v1.1.0  
**License:** MIT
