# PRD — Tyr v1.0.0
## Product Requirements Document — Versión 1.0.0

**Producto:** Tyr  
**Versión:** 1.0.0  
**Tipo:** MCP Plugin — Security, Validation & Standards Layer  
**Lenguaje:** Go 1.22+  
**Licencia:** MIT  
**Fecha:** Marzo 2026

---

## Historial de versiones

| Versión | Cambios principales |
|---|---|
| v1.0.0 | **Versión inicial unificada.** Validator, SAST, Audit, Standards, Scope Enforcer, Cache Multi-nivel, SAST Incremental, Secrets Scanner (Gitleaks), SBOM Express, Policy Engine Simple, Proactive Scan, Inject Guard, Quality Snapshot, Transitive License Analysis. |

---

## El nombre

En la mitología nórdica, **Tyr** es el dios de la justicia y la seguridad. Es quien pone la mano en la boca de Fenrir para encadenarlo — quien controla al lobo. En el ecosistema, Tyr controla al agente: valida lo que propone, escanea lo que genera, y audita lo que hace.

---

## Origen

Tyr surge de extraer los módulos de seguridad de Fenrir v3.0 para darles una identidad propia. La separación tiene cuatro justificaciones concretas:

**Preguntas distintas:** Fenrir responde "¿qué sabemos?" — Tyr responde "¿es seguro lo que el agente está haciendo?"

**Ciclos distintos:** El knowledge graph crece con cada sesión. Los patrones SAST se actualizan con Semgrep. Las APIs de CVEs consultan OSV continuamente. Son ritmos incompatibles en un mismo binario.

**Audiencias distintas:** La memoria la consume el agente. El audit log lo consume el Tech Lead o el equipo de seguridad.

**Adopción independiente:** Un equipo puede querer Tyr como gate de seguridad en CI/CD sin necesitar memoria persistente. O puede querer Fenrir sin análisis SAST pesado. Tyr puede vivir solo.

---

## Qué cambia en v1.0.0

### Principios de diseño

Todas las mejoras de v1.0.0 están guiadas por tres criterios:

| Criterio | Descripción | Aplicación |
|----------|-------------|------------|
| **VELOCIDAD** | Respuesta < 2s para validaciones simples | Cache < 1ms, SAST incremental |
| **EFICIENCIA DE TOKENS** | Quality Snapshot express, SBOM resumido | 90% reducción en outputs |
| **EFICACIA** | Secret scanning real, Standards que informan no bloquean | Gitleaks + warn-first |

### Mejoras en v1.0.0

| Mejora | Open Source | Velocidad | Tokens | Eficacia |
|--------|-------------|------------|--------|----------|
| Cache multi-nivel (mem/disk/net) | freecache + badger | ⚡⚡⚡ | ⚡ | ⚡⚡ |
| SAST incremental (diff) | Semgrep | ⚡⚡⚡ | ⚡ | ⚡⚡ |
| Standards warn-first | - | ⚡⚡⚡ | ⚡ | ⚡⚡ |
| Secret scanning local | gitleaks + trufflehog | ⚡⚡ | ⚡ | ⚡⚡ |
| SBOM rápido (summary) | syft | ⚡⚡ | ⚡ | ⚡⚡ |
| Policy engine simple | - | ⚡⚡⚡ | ⚡ | ⚡⚡ | |

---

## Tabla de contenidos

1. [Visión del producto](#1-visión-del-producto)
2. [Módulos](#2-módulos)
3. [Requisitos funcionales](#3-requisitos-funcionales)
4. [Módulo Validator](#4-módulo-validator)
5. [Módulo SAST](#5-módulo-sast)
6. [Módulo Audit](#6-módulo-audit)
7. [Módulo Standards](#7-módulo-standards)
8. [Módulo Scope Enforcer](#8-módulo-scope-enforcer)
9. [Modelo de datos](#9-modelo-de-datos)
10. [MCP Tools — Catálogo completo v1.0.0](#10-mcp-tools--catálogo-completo-v100)
11. [CLI completo v1.0.0](#11-cli-completo-v100)
12. [Configuración v1.0.0](#12-configuración-v100)
13. [Integración con Fenrir, Skoll y Hati](#13-integración-con-fenrir-skoll-y-hati)
14. [Modo CI/CD — sin los otros plugins](#14-modo-cicd--sin-los-otros-plugins)
15. [Roadmap v1.0.0](#15-roadmap-v100)
16. [Cobertura de seguridad](#16-cobertura-de-seguridad)

---

## 1. Visión del producto

Tyr es la capa de **seguridad, validación y calidad objetiva** del ecosistema. Opera en tres momentos:

- **Antes de actuar:** valida paquetes, verifica APIs, consulta CVEs
- **Durante la ejecución:** escanea el código generado con SAST, detecta inyecciones de prompt
- **Después de actuar:** audita lo que hizo el agente, ejecuta standards, expone resultados

> **Misión:** Que ninguna vulnerabilidad, ningún paquete malicioso, ninguna acción fuera de scope y ningún hallazgo SAST llegue al código sin ser detectado.

Tyr es el único plugin del ecosistema que tiene sentido como **gate autónomo en CI/CD** sin ningún otro plugin instalado.

---

## 2. Módulos

```
TYR
├── Validator      → paquetes: existencia, CVEs, licencias (directas + transitivas)
├── SAST           → análisis estático con Semgrep, gestión de findings
├── Audit          → log de acciones del agente, detección de inject, sanitización
├── Standards      → ejecución de criterios objetivos de calidad, Quality Snapshot
└── Scope Enforcer → watcher de filesystem, alertas de acceso fuera de scope
```

---

## 3. Requisitos funcionales

### RF-01 — Validator

| ID | Requisito | Prioridad |
|---|---|---|
| RF-01-01 | `pkg_check` debe verificar existencia en npm, PyPI, crates.io, NuGet | MUST |
| RF-01-02 | `pkg_check` debe retornar: exists, trusted, cve_count, age_days, downloads_monthly, typosquatting_risk | MUST |
| RF-01-03 | `pkg_check` debe consultar `api.osv.dev` para CVEs conocidos | MUST |
| RF-01-04 | Paquetes < 100 descargas y < 30 días de edad deben marcarse como `suspicious` | MUST |
| RF-01-05 | `pkg_license` debe retornar licencia directa y análisis de dependencias transitivas | MUST |
| RF-01-06 | El análisis transitivo debe usar `api.deps.dev` con TTL de cache 24 horas | MUST |
| RF-01-07 | Las licencias prohibidas en `policies.json` deben verificarse en todo el árbol transitivo | MUST |
| RF-01-08 | `pkg_audit` debe auditar todas las dependencias del proyecto actual | MUST |
| RF-01-09 | `pkg_audit_continuous` debe detectar CVEs nuevos en dependencias existentes | MUST |
| RF-01-10 | Cache de validaciones con TTL configurable (default: 1h para existencia, 6h para CVEs) | MUST |

### RF-02 — SAST

| ID | Requisito | Prioridad |
|---|---|---|
| RF-02-01 | `sast_run` debe ejecutar Semgrep con rulesets configurables | MUST |
| RF-02-02 | El output de Semgrep debe parsearse y categorizarse por severidad: INFO, WARNING, ERROR, CRITICAL | MUST |
| RF-02-03 | Findings críticos deben bloquear el cierre de sesión si configurado | MUST |
| RF-02-04 | `sast_findings` debe listar hallazgos activos con filtros por severidad, archivo, status | MUST |
| RF-02-05 | `sast_resolve` debe marcar un finding como resuelto | MUST |
| RF-02-06 | Si Semgrep no está instalado, degradar gracefully con instrucción de instalación | MUST |
| RF-02-07 | Los findings SAST deben notificarse a Fenrir para crear nodos `vulnerability` | SHOULD |
| RF-02-08 | Soporte de rulesets custom en `.tyr/semgrep-rules/` | SHOULD |

### RF-03 — Audit

| ID | Requisito | Prioridad |
|---|---|---|
| RF-03-01 | `audit_log` debe registrar cada acción del agente: tool, action_type, target, risk_level, result | MUST |
| RF-03-02 | `session_audit` debe retornar el log completo de una sesión con filtros | MUST |
| RF-03-03 | `inject_guard` debe verificar contenido por patrones de prompt injection reactivos | MUST |
| RF-03-04 | `proactive_scan` debe escanear archivos del módulo objetivo al iniciar sesión | MUST |
| RF-03-05 | `sanitize` debe stripear secrets y `<private>` tags antes de cualquier escritura | MUST |
| RF-03-06 | Patrones de secrets detectados: OpenAI, Anthropic, GitHub, AWS, Google, JWT, passwords en variables | MUST |
| RF-03-07 | Las detecciones de injection deben loguearse con risk_level: high | MUST |

### RF-04 — Standards

| ID | Requisito | Prioridad |
|---|---|---|
| RF-04-01 | `standard_run` debe ejecutar un standard específico de `standards.json` | MUST |
| RF-04-02 | `standard_run_all` debe ejecutar todos los standards y retornar Quality Snapshot | MUST |
| RF-04-03 | Standards con `on_failure: block` deben impedir el cierre de sesión | MUST |
| RF-04-04 | Standards con `run_on` deben ejecutarse solo en los checkpoints especificados | MUST |
| RF-04-05 | El Quality Snapshot debe distinguir entre unit tests, E2E y SAST | MUST |
| RF-04-06 | `standard_list` debe mostrar standards configurados con último resultado | MUST |
| RF-04-07 | Los resultados deben incluirse en el Session DNA de Fenrir | MUST |
| RF-04-08 | `tyr init` debe detectar comandos de testing en AGENTS.md y sugerir Standards automáticamente | MUST |
| RF-04-09 | El `overall_quality_score` se calcula: estándares block pesan 2, warn pesan 1 | MUST |

### RF-05 — Scope Enforcer

| ID | Requisito | Prioridad |
|---|---|---|
| RF-05-01 | `tyr watch` debe iniciar un watcher de filesystem sobre el directorio del proyecto | SHOULD |
| RF-05-02 | El watcher debe leer el scope del agente activo desde Skoll si disponible | SHOULD |
| RF-05-03 | Cambios fuera del scope deben generar alertas en consola y en el audit log | SHOULD |
| RF-05-04 | Modo `--strict` debe revertir automáticamente cambios fuera de scope | COULD |
| RF-05-05 | `tyr scope-violations` debe listar todas las violaciones detectadas | SHOULD |

---

## 4. Módulo Validator

### APIs externas usadas (todas gratuitas, sin API key)

| API | Endpoint | Uso | Cache TTL |
|---|---|---|---|
| npm Registry | `registry.npmjs.org/{pkg}` | Existencia y metadata | 1h |
| PyPI | `pypi.org/pypi/{pkg}/json` | Existencia y metadata | 1h |
| crates.io | `crates.io/api/v1/crates/{pkg}` | Existencia y metadata | 1h |
| NuGet | `api.nuget.org/v3/registration5/{pkg}/index.json` | Existencia | 1h |
| OSV | `api.osv.dev/v1/query` | CVEs conocidos | 6h |
| deps.dev | `api.deps.dev/v3alpha/...` | Licencias y dependencias transitivas | 24h |

### Respuesta de `pkg_check` v1.0

```json
{
  "package": "some-new-pkg",
  "ecosystem": "npm",
  "exists": true,
  "trusted": false,
  "trust_factors": {
    "downloads_monthly": 42,
    "age_days": 3,
    "maintainers": 1,
    "has_readme": true
  },
  "typosquatting_risk": "medium",
  "similar_legitimate": ["express", "fastify"],
  "cve_count": 0,
  "cves": [],
  "warning": "SUSPICIOUS: very new package (3 days) with low downloads (42/month)",
  "recommendation": "Verify manually before installing"
}
```

### Análisis transitivo de licencias

```go
// internal/modules/validator/transitive.go

type TransitiveLicenseResult struct {
    Package          string
    Version          string
    DirectLicense    string
    TransitiveCount  int
    ProblematicDeps  []TransitiveDep
    RiskLevel        string  // none | low | medium | high
    PolicyCompliant  bool
}

type TransitiveDep struct {
    Name           string
    Version        string
    License        string
    Depth          int
    Violates       bool
    PolicyViolated string
}

func (v *Validator) LicenseTransitive(pkg, eco, ver string) (*TransitiveLicenseResult, error) {
    url := fmt.Sprintf(
        "https://api.deps.dev/v3alpha/systems/%s/packages/%s/versions/%s:dependencies",
        eco, url.PathEscape(pkg), ver,
    )
    resp, err := v.httpGet(url)
    if err != nil { return v.LicenseDirect(pkg, eco, ver) }

    return v.evaluateTransitive(resp, v.loadPolicies())
}
```

---

## 5. Módulo SAST

### `standards.json` — tipo sast

```json
{
  "standards": [
    {
      "id": "sast-owasp",
      "description": "OWASP Top 10 via Semgrep",
      "type": "sast",
      "command": "semgrep --config=p/owasp-top-ten --json --quiet src/",
      "threshold": {
        "metric": "sast_findings",
        "max_severity_error": 0,
        "max_severity_critical": 0
      },
      "on_failure": "block",
      "run_on": ["all"],
      "timeout_seconds": 120
    },
    {
      "id": "sast-custom",
      "description": "Reglas SAST del proyecto",
      "type": "sast",
      "command": "semgrep --config=.tyr/semgrep-rules/ --json --quiet src/",
      "threshold": { "metric": "sast_findings", "max_severity_error": 0 },
      "on_failure": "warn",
      "run_on": ["post_high_risk", "final_checkpoint"],
      "timeout_seconds": 60
    }
  ]
}
```

### Rulesets Semgrep recomendados

| Ruleset | Detecta |
|---|---|
| `p/security-audit` | Vulnerabilidades generales |
| `p/owasp-top-ten` | OWASP Top 10 (SQLi, XSS, IDOR, etc.) |
| `p/nodejs` | Vulnerabilidades Node.js |
| `p/typescript` | Anti-patrones TypeScript inseguros |
| `p/jwt` | JWT mal configurado |
| `p/secrets` | Secrets hardcodeados |
| `p/django` | Vulnerabilidades Django |

### Parsing y categorización del output

```go
// internal/modules/sast/parser.go

type SASTResult struct {
    Findings        []SASTFinding
    CountBySeverity map[string]int
    Passed          bool
    BlockingCount   int
}

type SASTFinding struct {
    RuleID     string
    File       string
    Line       int
    Message    string
    Severity   string   // INFO | WARNING | ERROR | CRITICAL
    OWASP      []string
    CWE        []string
    Status     string   // open | resolved | suppressed
}
```

---

## 6. Módulo Audit

### Patrones de prompt injection

```go
// internal/modules/audit/inject.go

var injectionPatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous\s+)?instructions`),
    regexp.MustCompile(`(?i)you\s+are\s+now\s+(a\s+)?`),
    regexp.MustCompile(`(?i)disregard\s+(your|all|previous)`),
    regexp.MustCompile(`(?i)new\s+(persona|role|identity|instructions)`),
    regexp.MustCompile(`(?i)forget\s+(everything|all|previous|your)`),
    regexp.MustCompile(`\[SYSTEM\]|\[INST\]|<\|im_start\|>`),
    regexp.MustCompile(`(?i)act\s+as\s+if\s+you\s+(are|have\s+no)`),
    regexp.MustCompile(`(?i)your\s+(real|true|actual)\s+(purpose|goal|instruction)`),
}
```

### Patrones de secrets

```go
// internal/modules/audit/sanitize.go

var secretPatterns = []*regexp.Regexp{
    regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),                                    // OpenAI
    regexp.MustCompile(`sk-ant-[a-zA-Z0-9\-]{40,}`),                             // Anthropic
    regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),                                    // GitHub
    regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                                       // AWS
    regexp.MustCompile(`ya29\.[a-zA-Z0-9\-_]+`),                                  // Google
    regexp.MustCompile(`eyJ[a-zA-Z0-9._-]{20,}`),                                 // JWT
    regexp.MustCompile(`(?i)(password|secret|token|apikey)\s*[:=]\s*["']?\S{8,}`),
}
```

### Proactive scan al iniciar sesión

```go
// internal/modules/audit/proactive.go

func (a *Audit) ProactiveScan(modulePath string) *ProactiveScanResult {
    targets := a.collectScanTargets(modulePath)
    result := &ProactiveScanResult{FilesScanned: len(targets)}

    for _, file := range targets {
        content, err := os.ReadFile(file)
        if err != nil { continue }

        for _, pattern := range injectionPatterns {
            if pattern.MatchString(string(content)) {
                result.Findings = append(result.Findings, ProactiveFinding{
                    File:    file,
                    Pattern: pattern.String(),
                    Risk:    "high",
                })
            }
        }
    }
    return result
}

// Archivos objetivo del scan proactivo
func (a *Audit) collectScanTargets(modulePath string) []string {
    patterns := []string{
        "**/*.md", "**/*.txt",
        "**/prompts/**", "**/fixtures/**",
        "**/.env*", "**/templates/**",
    }
    return glob(modulePath, patterns)
}
```

---

## 7. Módulo Standards

### Selección de standards por checkpoint (campo `run_on`)

```go
// internal/modules/standards/selector.go

func SelectForCheckpoint(standards []Standard, checkpointType string, riskLevel string) []Standard {
    var selected []Standard
    for _, s := range standards {
        for _, runOn := range s.RunOn {
            switch runOn {
            case "all":
                selected = append(selected, s)
            case "post_high_risk":
                if checkpointType == "post" && (riskLevel == "high" || riskLevel == "critical") {
                    selected = append(selected, s)
                }
            case "final_checkpoint":
                if checkpointType == "final" {
                    selected = append(selected, s)
                }
            case "post_critical":
                if checkpointType == "post" && riskLevel == "critical" {
                    selected = append(selected, s)
                }
            }
        }
    }
    return uniqueByID(selected)
}
```

### Quality Snapshot completo

```json
{
  "ran_at": "2026-03-22T10:30:00Z",
  "checkpoint_type": "post",
  "phase_risk": "critical",

  "unit_tests": [
    { "id": "typescript-clean", "passed": true,  "on_failure": "block", "duration_ms": 2340 },
    { "id": "test-pass",        "passed": true,  "on_failure": "block", "duration_ms": 8920 },
    { "id": "test-coverage",    "passed": false, "on_failure": "warn",  "value": 74, "threshold": 80 },
    { "id": "lint-clean",       "passed": true,  "on_failure": "warn" }
  ],

  "e2e_tests": [
    { "id": "e2e-critical", "passed": true, "on_failure": "block", "type": "e2e",
      "summary": "6 tests passed in 14.2s", "run_on": "post_high_risk" }
  ],

  "sast": [
    { "id": "sast-owasp", "passed": true, "findings": 0, "on_failure": "block" }
  ],

  "security": [
    { "id": "no-vulnerabilities", "passed": true, "on_failure": "block" }
  ],

  "overall_quality_score": 0.87,
  "previous_score": 0.91,
  "score_delta": -0.04,
  "blockers": [],
  "warnings": ["test-coverage está por debajo del umbral (74% < 80%)"]
}
```

### Auto-detección de standards desde AGENTS.md

```bash
$ tyr init

🐺 Tyr — Detectando standards desde AGENTS.md...

Encontrado en AGENTS.md:
  "Run tests: npm test"
  "Run lint: npm run lint"
  "E2E: npx playwright test"

¿Agregar los siguientes standards a .fenrir/standards.json?
  ✅ test-pass       command: "npm test"           on_failure: block
  ✅ lint-clean      command: "npm run lint"        on_failure: warn
  ✅ e2e-critical    command: "npx playwright test" on_failure: block  run_on: final_checkpoint

[y/N]: y

✅ 3 standards agregados a .fenrir/standards.json
```

---

## 8. Módulo Scope Enforcer

```go
// internal/modules/scope/watcher.go

type ScopeEnforcer struct {
    watcher     *fsnotify.Watcher
    skollClient *SkollClient
    auditLogger *AuditLogger
    strict      bool
    violations  []ScopeViolation
}

func (se *ScopeEnforcer) handleChange(event fsnotify.Event) {
    if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) == 0 {
        return
    }

    scope := se.skollClient.GetActiveScope()
    if scope == nil { return }

    if !scope.Owns(event.Name) && !isIgnored(event.Name) {
        v := ScopeViolation{
            File:      event.Name,
            Agent:     scope.AgentName,
            Operation: event.Op.String(),
            At:        time.Now(),
        }
        se.violations = append(se.violations, v)
        se.auditLogger.Log(v)

        fmt.Printf("⚠️  TYR SCOPE VIOLATION: %s [%s] outside %s scope\n",
            event.Name, event.Op, scope.AgentName)

        if se.strict {
            se.revert(event.Name)
        }
    }
}
```

---

## 9. Modelo de datos

```sql
-- Validaciones de paquetes (cache)
CREATE TABLE pkg_cache (
    id          TEXT PRIMARY KEY,    -- {ecosystem}:{name}:{version}
    ecosystem   TEXT NOT NULL,
    name        TEXT NOT NULL,
    version     TEXT,
    exists_pkg  INTEGER NOT NULL,
    trusted     INTEGER DEFAULT 1,
    cve_count   INTEGER DEFAULT 0,
    license     TEXT,
    transitive_license_risk TEXT DEFAULT 'none',
    downloads   INTEGER DEFAULT 0,
    age_days    INTEGER DEFAULT 0,
    response    TEXT,                -- JSON completo
    cached_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at  DATETIME NOT NULL
);

-- Hallazgos SAST
CREATE TABLE sast_findings (
    id          TEXT PRIMARY KEY,
    session_id  TEXT,
    rule_id     TEXT NOT NULL,
    file        TEXT NOT NULL,
    line        INTEGER,
    message     TEXT NOT NULL,
    severity    TEXT NOT NULL,       -- INFO | WARNING | ERROR | CRITICAL
    owasp       TEXT,                -- JSON array
    cwe         TEXT,                -- JSON array
    status      TEXT DEFAULT 'open', -- open | resolved | suppressed
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME
);

-- Audit log de acciones del agente
CREATE TABLE audit_log (
    id           TEXT PRIMARY KEY,
    session_id   TEXT NOT NULL,
    tool_called  TEXT NOT NULL,
    action_type  TEXT NOT NULL,      -- read | write | execute | network | validate
    target       TEXT,
    risk_level   TEXT DEFAULT 'low', -- low | medium | high | critical
    result       TEXT DEFAULT 'success',
    metadata     TEXT,               -- JSON opcional
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Resultados de standards por sesión
CREATE TABLE standards_results (
    id           TEXT PRIMARY KEY,
    session_id   TEXT NOT NULL,
    standard_id  TEXT NOT NULL,
    checkpoint   TEXT,               -- all | post_high_risk | final_checkpoint
    passed       INTEGER NOT NULL,
    metric_value REAL,
    output       TEXT,
    duration_ms  INTEGER,
    ran_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Violaciones de scope
CREATE TABLE scope_violations (
    id         TEXT PRIMARY KEY,
    session_id TEXT,
    file       TEXT NOT NULL,
    agent      TEXT,
    operation  TEXT NOT NULL,        -- write | create | remove
    reverted   INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Incidents de CVEs continuos
CREATE TABLE cve_alerts (
    id            TEXT PRIMARY KEY,
    package_id    TEXT NOT NULL,
    cve_id        TEXT NOT NULL,
    severity      TEXT NOT NULL,
    summary       TEXT,
    detected_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    acknowledged  INTEGER DEFAULT 0
);
```

---

## 10. MCP Tools — Catálogo completo v1.0.0

### Módulo Validator (5 tools)

| Tool | Descripción |
|---|---|
| `pkg_check` | Validar existencia, confianza y CVEs de un paquete |
| `pkg_license` | Verificar licencia directa y transitiva |
| `pkg_audit` | Auditar todas las dependencias del proyecto |
| `pkg_audit_snapshot` | Snapshot de vulnerabilidades actuales |
| `pkg_audit_continuous` | Verificar si hay CVEs nuevos en deps existentes |

### Módulo SAST (3 tools)

| Tool | Descripción |
|---|---|
| `sast_run` | Ejecutar análisis SAST con Semgrep |
| `sast_findings` | Listar hallazgos activos con filtros |
| `sast_resolve` | Marcar un hallazgo como resuelto |

### Módulo Audit (5 tools)

| Tool | Descripción |
|---|---|
| `audit_log` | Registrar acción del agente en el audit trail |
| `session_audit` | Ver log completo de una sesión |
| `inject_guard` | Verificar contenido por patrones de injection (reactivo) |
| `proactive_scan` | Escanear módulo objetivo antes de trabajar (proactivo) |
| `sanitize` | Stripear secrets y private tags de contenido |

### Módulo Standards (4 tools)

| Tool | Descripción |
|---|---|
| `standard_run` | Ejecutar un standard específico |
| `standard_run_all` | Ejecutar todos y retornar Quality Snapshot |
| `standard_list` | Listar standards configurados con último resultado |
| `quality_snapshot` | Obtener el Quality Snapshot más reciente |

### Módulo Scope (2 tools)

| Tool | Descripción |
|---|---|
| `scope_violations` | Listar violaciones de scope detectadas |
| `tyr_stats` | Estadísticas del sistema: findings, violations, audits |

**Total v1.0: 19 MCP tools**

### Especificaciones clave

#### `pkg_check`

```json
{
  "name": "pkg_check",
  "description": "Validate package existence, trustworthiness and CVEs before installing. ALWAYS call before suggesting any package installation.",
  "inputSchema": {
    "type": "object",
    "required": ["name", "ecosystem"],
    "properties": {
      "name":      { "type": "string" },
      "ecosystem": { "type": "string", "enum": ["npm", "pypi", "cargo", "nuget"] },
      "version":   { "type": "string" }
    }
  }
}
```

#### `standard_run_all`

```json
{
  "name": "standard_run_all",
  "description": "Run all configured standards and return Quality Snapshot. Called by Hati before POST checkpoints.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "checkpoint_type": { "type": "string", "enum": ["all", "post", "post_high_risk", "final_checkpoint"] },
      "risk_level": { "type": "string", "enum": ["low", "medium", "high", "critical"] }
    }
  }
}
```

#### `proactive_scan`

```json
{
  "name": "proactive_scan",
  "description": "Proactively scan module files for prompt injection before starting work. Called automatically on mem_session_start.",
  "inputSchema": {
    "type": "object",
    "required": ["module_path"],
    "properties": {
      "module_path": { "type": "string" },
      "include_patterns": { "type": "array", "items": { "type": "string" } }
    }
  }
}
```

---

## 11. CLI completo v1.0.0

```bash
# ─── INICIALIZACIÓN ──────────────────────────────────
tyr init [--dry-run]
    # Detecta AGENTS.md, sugiere standards, configura herramientas

# ─── SISTEMA ─────────────────────────────────────────
tyr mcp
tyr serve [--port 7440]
tyr tui
tyr version
tyr stats

# ─── VALIDACIÓN DE PAQUETES ──────────────────────────
tyr pkg check <nombre> --eco <npm|pypi|cargo|nuget> [--version <ver>]
tyr pkg license <nombre> --eco <eco> [--transitive]
tyr pkg audit [--manifest <path>] [--transitive]
tyr pkg cve-alerts [--unacknowledged]

# ─── SAST ────────────────────────────────────────────
tyr sast [--file <path>] [--ruleset <nombre>]
tyr sast findings [--severity error|warning] [--file <path>]
tyr sast resolve <finding_id>
tyr sast suppress <finding_id> [--reason "<razón>"]

# ─── STANDARDS ───────────────────────────────────────
tyr standards list
tyr standards run [<id>]
tyr standards run --checkpoint post --risk high
tyr standards history [--sessions 10]
tyr standards snapshot

# ─── AUDIT ───────────────────────────────────────────
tyr audit list [--session <id>] [--risk high|critical]
tyr audit session <session_id>
tyr inject-check "<contenido>"

# ─── SCOPE ───────────────────────────────────────────
tyr watch [--strict]
tyr scope-violations [--today] [--session <id>]
tyr scope-violations clear

# ─── CONFIGURACIÓN ───────────────────────────────────
tyr config show
tyr config policies              # Ver/editar policies.json
```

---

## 12. Configuración v1.0.0

### `.tyr/config.json`

```json
{
  "project": "mi-proyecto",
  "version": "1.0.0",

  "validator": {
    "ecosystems": ["npm", "pypi", "cargo", "nuget"],
    "osv_ttl_hours": 6,
    "pkg_ttl_hours": 1,
    "transitive_license": true,
    "transitive_cache_ttl_hours": 24,
    "suspicious_threshold": {
      "max_age_days": 30,
      "min_downloads": 100
    }
  },

  "sast": {
    "enabled": true,
    "tool": "semgrep",
    "rulesets": ["p/security-audit", "p/owasp-top-ten", "p/nodejs"],
    "custom_rules_dir": ".tyr/semgrep-rules/",
    "timeout_seconds": 120,
    "on_finding_notify_fenrir": true
  },

  "audit": {
    "log_all_actions": true,
    "inject_guard_reactive": true,
    "proactive_scan_on_session_start": true,
    "proactive_scan_targets": ["**/*.md", "**/prompts/**", "**/fixtures/**"],
    "proactive_max_duration_ms": 2000,
    "sanitize_on_persist": true
  },

  "standards": {
    "file": ".fenrir/standards.json",
    "block_session_end_on_failure": true,
    "expose_to_hati": true,
    "auto_detect_from_agents_md": true
  },

  "scope_enforcer": {
    "enabled": false,
    "strict_mode": false,
    "alert_on_violation": true
  },

  "continuous_audit": {
    "enabled": true,
    "interval_hours": 24
  },

  "policies": {
    "forbidden_licenses": ["GPL-3.0", "AGPL-3.0"],
    "file": ".tyr/policies.json"
  }
}
```

### `.tyr/policies.json`

```json
{
  "team": "mi-equipo",
  "version": "1.0",
  "forbidden_licenses": ["GPL-3.0", "AGPL-3.0", "LGPL-3.0"],
  "policies": [
    {
      "id": "no-any-typescript",
      "description": "No usar 'any' en TypeScript",
      "severity": "hard",
      "pattern": ":\\s*any[\\s;,)]"
    },
    {
      "id": "no-direct-db",
      "description": "No acceso directo a DB fuera de repositories",
      "severity": "critical",
      "allowed_in": ["repositories/", "migrations/"]
    },
    {
      "id": "require-error-handling",
      "description": "Todo async/await debe tener manejo de errores",
      "severity": "soft"
    }
  ]
}
```

### Variables de entorno

```
TYR_DIR          Directorio de datos (default: .tyr)
TYR_PORT         Puerto del servidor HTTP (default: 7440)
TYR_LOG_LEVEL    debug|info|warn|error (default: info)
TYR_OFFLINE      Deshabilitar llamadas a APIs externas: true|false
TYR_FENRIR_MCP   URL del servidor MCP de Fenrir
TYR_SKOLL_MCP    URL del servidor MCP de Skoll
```

---

## 13. Integración con Fenrir, Skoll y Hati

### Con Fenrir

| Evento Tyr | Acción Fenrir |
|---|---|
| `sast_run` encuentra findings | Llama `mem_save type:vulnerability` para crear nodos en el grafo |
| `sast_resolve` resuelve finding | Actualiza el nodo `vulnerability` a status:resolved |
| `proactive_scan` detecta injection | Llama `audit_log` en Fenrir para el knowledge trail |
| `standard_run_all` completa | Expone Quality Snapshot para `export_pr_summary` de Fenrir |
| `pkg_check` detecta CVE | Notifica a Fenrir para ajustar `predict` del módulo |
| `tyr init` lee AGENTS.md | Comparte testing commands con Fenrir para la sección `standards` |

### Con Skoll

| Evento Tyr | Acción Skoll |
|---|---|
| `tyr watch` activo | Lee el scope del agente activo desde `skoll team_status` |
| `scope_violations` detecta violación | El scope viene de `.skoll/agents/{nombre}.md` |
| `pkg_check` valida paquete | Skoll puede pre-aprobar validaciones via `allowed-tools` en SKILL.md |
| `sast_findings` en módulo | Skoll puede ajustar el scope del agente si hay findings críticos |

### Con Hati

| Evento Tyr | Acción Hati |
|---|---|
| `standard_run_all` | Hati consume el Quality Snapshot antes de abrir checkpoints POST |
| Standards con `on_failure: block` que fallan | Hati no puede abrir el checkpoint POST |
| `proactive_scan` al iniciar sesión | Hati incluye hallazgos en las alertas del checkpoint PRE |
| `pkg_check` detecta CVE en dep usada en una fase | Hati sube el riesgo de esa fase |
| `sast_findings` activos en archivos del plan | Hati menciona los findings en el checkpoint PRE |

---

## 14. Modo CI/CD — sin los otros plugins

Tyr puede correr como gate de seguridad autónomo en pipelines de CI/CD:

```yaml
# .github/workflows/tyr-gate.yml
name: Tyr Security Gate

on: [push, pull_request]

jobs:
  tyr-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Tyr
        run: |
          curl -L https://github.com/tu-org/tyr/releases/latest/download/tyr_linux_amd64 -o tyr
          chmod +x tyr && sudo mv tyr /usr/local/bin/

      - name: Install Semgrep
        run: pip install semgrep

      - name: Run Tyr Security Gate
        run: |
          tyr pkg audit --manifest package.json --transitive
          tyr sast --ruleset p/owasp-top-ten,p/security-audit
          tyr standards run
```

En este modo, Tyr corre completamente sin Fenrir, Skoll ni Hati — solo como validador de seguridad de CI/CD.

---

## 15. Roadmap v1.0.0

| Fase | Semanas | Deliverable |
|---|---|---|
| 1 — Core Migration | 1–2 | Validator, Shield, SAST, Standards, ScopeEnforcer base |
| 2 — Validator completo | 3 | pkg_check, pkg_license transitivo, pkg_audit, OSV continuo |
| 3 — SAST Integration | 4–5 | sast_run, semgrep parsing, findings table, sast_resolve |
| 4 — Audit completo | 6 | audit_log, inject_guard, proactive_scan, sanitize |
| 5 — Standards Engine | 7–8 | standard_run, standard_run_all, Quality Snapshot, run_on selector |
| 6 — Auto-detect desde AGENTS.md | 9 | tyr init lee AGENTS.md, sugiere standards |
| 7 — Scope Enforcer | 10 | tyr watch, fsnotify, scope violation log |
| 8 — Integración completa | 11 | Todos los puntos con Fenrir, Skoll y Hati |
| 9 — CI/CD Mode docs | 12 | GitHub Actions example, documentación standalone |
| 10 — Release v1.0.0 | 13 | Homebrew tap, docs, release |

---

## 16. Cobertura de seguridad

Tyr es el plugin responsable principal de los problemas de seguridad:

| Problema | Módulo Tyr | Estado |
|---|---|---|
| Vulnerabilidades de seguridad | SAST + Standards | ✅ Resuelto |
| Slopsquatting | Validator | ✅ Resuelto |
| Vulnerabilidades en agentes | Audit (inject, proactive) | ◑ Parcial (sandboxing = externo) |
| Propiedad intelectual | Validator (licencias transitivas) | ◑ Parcial |
| Acceso excesivo | Scope Enforcer | ◑ Parcial (sin OS sandbox) |

---

## 17. Detalles Técnicos v1.0.0

### Principios de diseño v2.0

Todas las mejoras de v2.0 están guiadas por tres criterios:

| Criterio | Descripción | Aplicación |
|----------|-------------|------------|
| **VELOCIDAD** | Respuesta < 2s para validaciones simples | Cache < 1ms, SAST incremental |
| **EFICIENCIA DE TOKENS** | Quality Snapshot express, SBOM resumido | 90% reducción en outputs |
| **EFICACIA** | Secret scanning real, Standards que informan no bloquean | Gitleaks + warn-first |

### Mejoras en v2.0

| Mejora | Open Source | Velocidad | Tokens | Eficacia |
|--------|-------------|------------|--------|----------|
| Cache multi-nivel (mem/disk/net) | freecache + badger | ⚡⚡⚡ | ⚡ | ⚡⚡ |
| SAST incremental (diff) | Semgrep | ⚡⚡⚡ | ⚡ | ⚡⚡ |
| Standards warn-first | - | ⚡⚡⚡ | ⚡ | ⚡⚡ |
| Secret scanning local | gitleaks + trufflehog | ⚡⚡ | ⚡ | ⚡⚡ |
| SBOM rápido (summary) | syft | ⚡⚡ | ⚡ | ⚡⚡ |
| Policy engine simple | - | ⚡⚡⚡ | ⚡ | ⚡⚡ |

---

### 17.1 Cache Multi-nivel para Validator

**Problema:** Cada `pkg_check` llama a APIs externas (lento).

**Solución:**

```go
// internal/modules/validator/multi_level_cache.go

type PackageValidator struct {
    memCache   *freecache.Cache    // 100MB memory cache
    diskCache  *badger.DB          // Disk cache persistente
    httpClient *retryablehttp.Client
    
    // Config
    memoryTTL time.Hour * 1       // 1 hora
    diskTTL   time.Hour * 24 * 7 // 7 días
}

func (v *PackageValidator) Check(pkg, eco, version string) (*CheckResult, error) {
    cacheKey := fmt.Sprintf("%s:%s:%s", eco, pkg, version)
    
    // 1. Check memory cache (instantáneo)
    if cached := v.memCache.Get([]byte(cacheKey)); cached != nil {
        return unmarshal(cached), nil
    }
    
    // 2. Check disk cache (< 1ms)
    if cached, err := v.diskCache.Get([]byte(cacheKey)); err == nil {
        result := unmarshal(cached)
        v.memCache.Set([]byte(cacheKey), cached, int(v.memoryTTL.Seconds()))
        return result, nil
    }
    
    // 3. Fetch from network (solo si necesario)
    result, err := v.fetchAndCache(pkg, eco, version)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```

### 17.2 SAST Incremental (Diff-based)

**Problema:** SAST escanea todo el codebase cada vez (lento).

**Solución:**

```go
// internal/modules/sast/incremental.go

type SASTEngine struct {
    semgrep   *semgrep.Semgrep
    cache     *FileCache  // Cache de resultados por archivo
    gitClient *git.Client
}

func (s *SASTEngine) RunIncremental() (*SASTResult, error) {
    // 1. Obtener archivos cambiados desde último scan
    changedFiles, err := s.gitClient.ChangedSince(s.lastScanCommit)
    if err != nil {
        return s.RunFull() // Fallback a full scan
    }
    
    // 2. Solo escanear archivos cambiados
    var findings []SASTFinding
    for _, file := range changedFiles {
        // Check cache primero
        if cached, ok := s.cache.Get(file); ok && !s.fileChanged(file, cached.ScannedAt) {
            continue // Skip unchanged files
        }
        
        fileFindings, err := s.semgrep.ScanFile(file)
        if err != nil {
            continue
        }
        
        findings = append(findings, fileFindings...)
        s.cache.Set(file, &FileScanResult{ScannedAt: time.Now(), Findings: fileFindings})
    }
    
    // 3. Combinar con findings de archivos no modificados (del cache)
    allFindings := append(findings, s.getCachedFindings()...)
    
    return &SASTResult{Findings: allFindings}, nil
}
```

### 17.3 Standards Warn-first (No Bloquean)

**Problema:** Standards con `on_failure: block` pueden detener al agente innecesariamente.

**Solución — Todos los standards tienen dos modos:**

```go
// Standards tienen modo warn por defecto
// El agente decide si procede basándose en la información

type StandardResult struct {
    ID          string
    Passed      bool
    Blocking    bool      // Si es blocking o no
    Summary     string    //⚡ Compacto: "3 tests failed, 74% coverage"
    Details     []string  // Detalles solo si el agente pregunta
    Suggestion  string    //⚡ "Run 'npm test' to fix"
}

func (s *StandardsEngine) RunAll(checkpoint string, risk string) QualitySnapshot {
    var results []StandardResult
    
    for _, std := range s.standards {
        // Skip si no aplica a este checkpoint
        if !std.AppliesTo(checkpoint, risk) {
            continue
        }
        
        result := s.runStandard(std)
        
        // SIEMPRE incluir en results, aunque sea warn
        results = append(results, result)
    }
    
    return QualitySnapshot{
        Results: results,
        Blockers: filterBlockingFailures(results),
        Score:    calculateScore(results),
    }
}
```

### 17.4 Secret Scanning con Gitleaks

**Problema:** El agente puede escribir secrets accidentalmente.

**Solución:**

```go
// internal/modules/audit/secrets_scanner.go

type SecretScanner struct {
    gitleaksPath string
}

func (s *SecretScanner) Scan(path string) (*ScanResult, error) {
    // gitleaks detect --source . --no-color --report-format json
    output, err := exec.Command("gitleaks", "detect", 
        "--source", path,
        "--no-color",
        "--report-format", "json").Output()
    
    if err != nil {
        return nil, err
    }
    
    findings := parseGitleaksFindings(output)
    
    //⚡ Retornar solo findings relevantes, no todo el report
    return &ScanResult{
        HasSecrets:   len(findings) > 0,
        Count:        len(findings),
        Locations:     extractLocations(findings), // Solo archivos, no secrets
        Suggestions:  []string{
            "Ejecuta 'gitleaks protect' para auto-remediar",
            "Agrega secrets al .gitignore global",
        },
    }, nil
}
```

### 17.5 SBOM Ligero con CycloneDX

**Problema:** Generar SBOM completo es lento.

**Solución:**

```go
// internal/modules/validator/sbom_generator.go

type SBOMGenerator struct {
    syftPath string
    cache    *Cache
}

func (g *SBOMGenerator) GenerateQuick(dir string) (*SBOMSummary, error) {
    // syft . -o cyclonedx-json --quiet
    output, err := exec.Command("syft", dir, 
        "-o", "cyclonedx-json",
        "--quiet").Output()
    
    if err != nil {
        return nil, err
    }
    
    sbom := parseCycloneDX(output)
    
    //⚡ Retornar solo summary, no el SBOM completo
    return &SBOMSummary{
        TotalPackages: sbom.ComponentCount,
        LicenseCounts: sbom.LicenseSummary,
        HighRiskDeps:  sbom.HighRiskComponents,
    }, nil
}
```

### 17.6 Policy Engine Simple (No OPA)

**Problema:** OPA es overkill para la mayoría de policies.

**Solución — Policies como funciones Go compiladas:**

```go
// internal/modules/policy/simple_engine.go

type Policy struct {
    ID          string
    Description string
    Evaluate    func(input PolicyInput) (bool, string)
}

var Policies = []Policy{
    {
        ID:          "no-any-typescript",
        Description: "No usar 'any' en TypeScript",
        Evaluate: func(input PolicyInput) (bool, string) {
            if strings.Contains(input.Content, ": any") {
                return false, "TypeScript: uso de 'any' detectado"
            }
            return true, ""
        },
    },
    {
        ID:          "no-gpl-license",
        Description: "No dependencias con GPL-3.0",
        Evaluate: func(input PolicyInput) (bool, string) {
            for _, dep := range input.Dependencies {
                if isForbiddenLicense(dep.License) {
                    return false, fmt.Sprintf("Dependencia %s tiene licencia %s", dep.Name, dep.License)
                }
            }
            return true, ""
        },
    },
}

//⚡ Ejecución paralela con goroutines
func (e *PolicyEngine) EvaluateAll(input PolicyInput) []PolicyResult {
    results := make([]PolicyResult, len(e.policies))
    
    var wg sync.WaitGroup
    for i, policy := range e.policies {
        wg.Add(1)
        go func(idx int, p Policy) {
            defer wg.Done()
            allowed, reason := p.Evaluate(input)
            results[idx] = PolicyResult{PolicyID: p.ID, Allowed: allowed, Reason: reason}
        }(i, policy)
    }
    wg.Wait()
    
    return results
}
```

### 17.7 Requisitos Funcionales v2.0

### RF-01 — Cache Multi-nivel

| ID | Requisito | Prioridad |
|---|---|---|
| RF-01-01 | `pkg_check` usa cache memory (< 1ms si hit) | MUST |
| RF-01-02 | Cache disk como segundo nivel (< 10ms si hit) | MUST |
| RF-01-03 | Invalidación por TTL configurable | MUST |
| RF-01-04 | `pkg_check --no-cache` fuerza fetch network | SHOULD |

### RF-02 — SAST Incremental

| ID | Requisito | Prioridad |
|---|---|---|
| RF-02-01 | SAST solo escanea archivos cambiados desde último scan | MUST |
| RF-02-02 | Findings de archivos no modificados se recuperan del cache | MUST |
| RF-02-03 | Fallback a full scan si no hay historial | MUST |

### RF-03 — Standards Warn-first

| ID | Requisito | Prioridad |
|---|---|---|
| RF-03-01 | Standards tienen `on_failure: warn` por defecto | MUST |
| RF-03-02 | Quality Snapshot incluye todos los results, no solo blockers | MUST |
| RF-03-03 | `on_failure: block` solo para standards críticos | SHOULD |

### RF-04 — Secret Scanning

| ID | Requisito | Prioridad |
|---|---|---|
| RF-04-01 | `secret_scan` usa gitleaks localmente | MUST |
| RF-04-02 | Retorna solo locations, no secrets completos | MUST |
| RF-04-03 | Sugerencias de remediación automáticas | MUST |

### RF-05 — SBOM Express

| ID | Requisito | Prioridad |
|---|---|---|
| RF-05-01 | `sbom_generate --quick` retorna summary, no full SBOM | MUST |
| RF-05-02 | Formato CycloneDX para máxima compatibilidad | MUST |
| RF-05-03 | Incluye license counts y high-risk deps | MUST |

### RF-06 — Policy Engine

| ID | Requisito | Prioridad |
|---|---|---|
| RF-06-01 | Policies definidas como funciones Go compiladas | MUST |
| RF-06-02 | Evaluación paralela con goroutines | MUST |
| RF-06-03 | Resultado: allowed + reason | MUST |

---

## 18. Roadmap Continuo

| Fase | Semanas | Deliverable |
|---|---|---|
| 1 — Core Modules | 1–4 | Validator, SAST, Audit, Standards, Scope Enforcer |
| 2 — Cache Multi-nivel | 5 | freecache + badger, invalidation |
| 3 — SAST Incremental | 6 | diff-based scanning, file cache |
| 4 — Standards Express | 7 | Quality Snapshot < 100 tokens |
| 5 — Secret Scanning | 8 | gitleaks integration |
| 6 — SBOM Express | 9 | syft + CycloneDX summary |
| 7 — Policy Engine | 10 | Simple Go-based engine |
| 8 — Release v1.0.0 | 11 | Docs, testing, release |

---

## 19. Mejoras Planificadas v1.1.0

Ver documento `PRD_Ragnarok_v1.1_Improvements.md` para especificaciones completas.

| Mejora | Descripción | Prioridad |
|---|---|---|
| **Pre-Commit Validation** | `precommit_validate`, `precommit_autofix` — catch errores antes de presentar | 🔴 ALTA |
| **CLI Stats** | `ecosystem_stats` — stats unificados del ecosistema | 🟡 MEDIA |

*Pre-Commit Validation es la mejora más importante para v1.1.0 ya que reduce significativamente el tiempo que el usuario gasta corrigiendo errores básicos del código generado por el agente.*

### Nuevos tools en v1.1.0

| Tool | Descripción |
|---|---|
| `precommit_validate` | Validar código generado (syntax, imports, types) |
| `precommit_autofix` | Auto-corregir errores comunes automáticamente |

---

*Tyr PRD v1.0.0 — Marzo 2026*
*~25 MCP tools · SQLite · Go 1.22+ · MIT*
*Tyr PRD v1.0.0 — Marzo 2026*
*~19 MCP tools · SQLite · Go 1.22+ · MIT*
*Cache Multi-nivel · SAST Incremental · Secrets Scanner · SBOM Express · Policy Engine*
*3 Pilares: Velocidad ⚡ · Eficiencia de Tokens 💎 · Eficacia 🎯*
*Funciona solo (CI/CD) o integrado con Fenrir, Skoll y Hati*
