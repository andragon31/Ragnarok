# PRD — Ragnarok Improvements v1.1
## Product Requirements Document — Versión 1.1.0

**Producto:** Ecosistema RAGNAROK  
**Versión:** 1.1.0  
**Tipo:** Mejoras y Nuevos Módulos  
**Lenguaje:** Go 1.22+  
**Licencia:** MIT  
**Fecha:** Marzo 2026

---

## Contexto y Alcance

```
PERFIL DE USUARIO:
├── OS: Windows 10/11 (principal)
├── Agentes: OpenCode, Cursor, Windsurf, Claude Code, Gemini CLI
├── Contexto: Desarrollo individual o equipo pequeño (≤5 devs)
├── Proyecto: 1 proyecto activo a la vez
└── Red: Conexión internet estándar

MEJORAS PRIORIZADAS PARA v1.1.0:
├── 1. Pre-Commit Validation (🔴 ALTA)
├── 2. Intent Verifier (🔴 ALTA)
├── 3. Bias Detector (🟡 MEDIA)
├── 4. CLI Stats (🟡 MEDIA)
└── 5. Backup Automation (🟡 MEDIA)
```

---

## Tabla de contenidos

1. [Resumen y Prioridades](#1-resumen-y-prioridades)
2. [Pre-Commit Validation](#2-pre-commit-validation)
3. [Intent Verifier](#3-intent-verifier)
4. [Bias Detector](#4-bias-detector)
5. [CLI Stats](#5-cli-stats)
6. [Backup Automation](#6-backup-automation)
7. [Roadmap de implementación](#7-roadmap-de-implementación)

---

## 1. Resumen y Prioridades

| # | Mejora | Prioridad | Impacto en usuario |
|---|---|---|---|
| 1 | **Pre-Commit Validation** | 🔴 ALTA | El agente ya no presenta código con errores de sintaxis/imports |
| 2 | **Intent Verifier** | 🔴 ALTA | El agente sabe si está resolviendo el problema correcto |
| 3 | **Bias Detector** | 🟡 MEDIA | Se detectan sesgos en el conocimiento del proyecto |
| 4 | **CLI Stats** | 🟡 MEDIA | El usuario puede ver salud del sistema fácilmente |
| 5 | **Backup Automation** | 🟡 MEDIA | No se pierde el knowledge graph |

### Mejoras NO priorizadas para v1.1.0

Estas mejoras están en el backlog para v1.2+:

| Mejora | Razón de descarte |
|---|---|
| Async Checkpoints | El usuario está disponible para approve/reject en tiempo real |
| Distributed Sync (CRDT) | Single-user, no hay equipos distribuidos |
| Health Dashboard (web) | CLI simple es suficiente para el usuario |
| Multi-tenancy | No aplica en contexto single-user |

---

## 2. Pre-Commit Validation

### 2.1 Visión del producto

El agente (OpenCode, Cursor, Windsurf) genera código con errores básicos que el usuario tiene que corregir manualmente. **Pre-Commit Validation** es un hook que valida el código antes de que el agente lo presente, catch errores de sintaxis, imports faltantes, y types incorrectos.

### 2.2 Flujo

```
┌─────────────────────────────────────────────────────────────┐
│              PRE-COMMIT VALIDATION FLOW                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Agent genera código                                           │
│        │                                                      │
│        ▼                                                      │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │
│  │  Syntax     │───▶│  Import     │───▶│  Type       │   │
│  │  Check     │    │  Resolve    │    │  Check      │   │
│  └─────────────┘    └─────────────┘    └─────────────┘   │
│        │                  │                  │               │
│        ▼                  ▼                  ▼               │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │
│  │  Error!     │    │  Missing!  │    │  Wrong!     │   │
│  │  → Auto-fix │    │  → Suggest │    │  → Warning  │   │
│  └─────────────┘    └─────────────┘    └─────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 2.3 Implementación

```go
// internal/precommit/validator.go

type PreCommitValidator struct {
    syntaxChecker  *SyntaxChecker
    importResolver *ImportResolver
    typeChecker   *TypeChecker
    semgrepRunner *SemgrepRunner
    config        *ValidatorConfig
}

type ValidationResult struct {
    Passed     bool
    DurationMs int64
    Errors     []*ValidationError
    Warnings   []*ValidationWarning
}

type ValidationError struct {
    Type    string  // syntax, import, type
    File    string
    Line    int
    Message string
    Fixed   bool
}

func (v *PreCommitValidator) Validate(files []*FileChange) *ValidationResult {
    result := &ValidationResult{Passed: true}
    
    // 1. Syntax check (más rápido)
    for _, file := range files {
        if errs := v.syntaxChecker.Check(file); len(errs) > 0 {
            result.Passed = false
            result.Errors = append(result.Errors, errs...)
        }
    }
    
    // 2. Import resolution
    for _, file := range files {
        if errs := v.importResolver.Resolve(file); len(errs) > 0 {
            result.Passed = false
            result.Errors = append(result.Errors, errs...)
        }
    }
    
    // 3. Type checking (TypeScript, Go, Rust)
    for _, file := range files {
        if errs := v.typeChecker.Check(file); len(errs) > 0 {
            result.Passed = false
            result.Errors = append(result.Errors, errs...)
        }
    }
    
    return result
}
```

### 2.4 Auto-fix para errores comunes

```go
// Errors que se pueden auto-corregir:
var AutoFixableErrors = map[string]bool{
    "missing-import":         true,  // Agregar import faltante
    "unused-import":          true,  // Eliminar import no usado
    "formatting":             true,  // rustfmt, prettier, gofmt
    "missing-semicolon":      true,  // TypeScript
    "trailing-whitespace":    true,  // Limpiar whitespace
    "duplicate-import":       true,  // Eliminar duplicados
}

func (v *PreCommitValidator) TryAutoFix(err *ValidationError) *ValidationError {
    if !AutoFixableErrors[err.Type] {
        return err
    }
    
    switch err.Type {
    case "missing-import":
        return v.fixMissingImport(err)
    case "formatting":
        return v.fixFormatting(err)
    // ...
    }
}
```

### 2.5 Integración con Tyr

```go
// En tyr como hook:

func (t *Tyr) PreCommitHook(ctx context.Context, req *PreCommitRequest) (*PreCommitResponse, error) {
    files := t.parseFiles(req.CodeChanges)
    
    result := t.validator.Validate(files)
    
    // Intentar auto-fix
    if !result.Passed && req.AllowAutofix {
        fixed := t.validator.TryAutoFix(result.Errors)
        result.Errors = fixed
        result.Passed = len(result.Errors) == 0
    }
    
    return &PreCommitResponse{
        Passed:     result.Passed,
        Errors:     result.Errors,
        Warnings:   result.Warnings,
        CanContinue: result.Passed || req.AllowPartial,
    }, nil
}
```

### 2.6 Requisitos funcionales

| ID | Requisito | Prioridad |
|---|---|---|
| RF-01-01 | Validar syntax de código generado antes de presentar | MUST |
| RF-01-02 | Verificar que imports existen y son resolubles | MUST |
| RF-01-03 | Type checking para TypeScript, Go, Python | MUST |
| RF-01-04 | Auto-fix para errores comunes | SHOULD |
| RF-01-05 | Tiempo máximo de validación: 30 segundos | MUST |
| RF-01-06 | Integración como hook en Tyr (llamado por agente) | MUST |

---

## 3. Intent Verifier

### 3.1 Visión del producto

El agente no sabe si está resolviendo el problema real. Puede implementar algo que "funciona" pero no es lo que el usuario pidió. **Intent Verifier** guarda la intención original y la compara con el código implementado al final.

### 3.2 Flujo

```
┌─────────────────────────────────────────────────────────────┐
│              INTENT VERIFICATION FLOW                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Usuario pide algo                                        │
│        │                                                      │
│        ▼                                                      │
│  2. ┌──────────────┐                                       │
│     │ intent_save   │ ← Guardar intención con embedding      │
│     └──────────────┘                                       │
│        │                                                      │
│        ▼                                                      │
│  3. Agent trabaja                                            │
│        │                                                      │
│        ▼                                                      │
│  4. ┌──────────────┐                                       │
│     │ intent_verify │ ← Comparar vs código implementado      │
│     └──────────────┘                                       │
│        │                                                      │
│        ▼                                                      │
│  5. Coverage: 85% ✓                                          │
│     Missing: "validación de errores"                         │
│     → Alertar al usuario                                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 3.3 Implementación

```go
// internal/intent_verifier/verifier.go

type IntentVerifier struct {
    intentStore   *IntentStore
    codeAnalyzer  *CodeAnalyzer
    embeddingModel embeddings.Model
}

type Intent struct {
    ID        string
    PlanID    string
    Prompt    string
    Embedding []float32
    Items     []*IntentItem
    CreatedAt time.Time
}

type IntentItem struct {
    ID          string
    Description string
    Type        string  // feature, fix, refactor, config
    Status      string  // pending, covered, partial, missing
}

type VerificationResult struct {
    IntentID       string
    CoverageScore  float64  // 0.0 - 1.0
    AlignmentScore float64  // 0.0 - 1.0
    
    Covered []string  // Items cubiertos
    Missing []string  // Items no implementados
    Partial []string  // Items parcialmente
    
    Suggestions []string
}

func (v *IntentVerifier) Verify(planID string, changedFiles []*File) (*VerificationResult, error) {
    // 1. Obtener intención original
    intent, err := v.intentStore.GetByPlanID(planID)
    if err != nil {
        return nil, err
    }
    
    // 2. Extraer features del código
    codeFeatures, err := v.codeAnalyzer.Analyze(changedFiles)
    if err != nil {
        return nil, err
    }
    
    // 3. Matching semántico
    result := &VerificationResult{IntentID: intent.ID}
    
    for _, item := range intent.Items {
        matched := v.findBestMatch(item, codeFeatures)
        
        if matched == nil {
            result.Missing = append(result.Missing, item.Description)
            continue
        }
        
        if matched.Score > 0.8 {
            result.Covered = append(result.Covered, item.Description)
        } else {
            result.Partial = append(result.Partial, item.Description)
        }
    }
    
    result.CoverageScore = float64(len(result.Covered)) / float64(len(intent.Items))
    
    return result, nil
}

func (v *IntentVerifier) findBestMatch(item *IntentItem, features []*CodeFeature) *Match {
    itemEmbedding, _ := v.embeddingModel.Embed(item.Description)
    
    var best *Match
    for _, feature := range features {
        featureEmbedding, _ := v.embeddingModel.Embed(feature.Description())
        score := cosineSimilarity(itemEmbedding, featureEmbedding)
        
        if score > 0.6 && (best == nil || score > best.Score) {
            best = &Match{Item: item, Feature: feature, Score: score}
        }
    }
    return best
}
```

### 3.4 Análisis de código

```go
// internal/intent_verifier/code_analyzer.go

type CodeAnalyzer struct {
    astParsers map[string]ASTParser
}

type CodeFeature struct {
    Type      string  // function, class, endpoint, config
    Name      string
    File      string
    Signature string
    Imports   []string
}

func (a *CodeAnalyzer) Analyze(files []*File) ([]*CodeFeature, error) {
    var features []*CodeFeature
    
    for _, file := range files {
        parser := a.astParsers[file.Language]
        if parser == nil {
            continue
        }
        
        parsed, err := parser.Parse(file.Content)
        if err != nil {
            continue
        }
        
        features = append(features, parsed.ExtractFeatures()...)
    }
    
    return features, nil
}
```

### 3.5 Requisitos funcionales

| ID | Requisito | Prioridad |
|---|---|---|
| RF-02-01 | `intent_save` debe guardar intención con embedding | MUST |
| RF-02-02 | `intent_verify` debe comparar intención vs código al completar plan | MUST |
| RF-02-03 | Coverage score >= 0.8 para aprobación automática | MUST |
| RF-02-04 | Items missing deben mostrarse como alertas | MUST |
| RF-02-05 | Se integra con Hati checkpoint system | MUST |

---

## 4. Bias Detector

### 4.1 Visión del producto

El agente puede desarrollar "puntos ciegos" hacia ciertos patrones si el knowledge graph está sesgado. Bias Detector analiza el grafo y detecta sesgos para alert al usuario.

### 4.2 Tipos de sesgos detectados

| Sesgo | Descripción | Señal |
|---|---|---|
| Recency | Decisiones antiguas inapropiadamente seguidas | >70% de data es del último año |
| Authority | Información de baja autoridad tratada como hecho | Nodos "exploratory" usados como "authoritative" |
| Confirmation | Solo se guardan decisiones exitosas | 10x más decisions que incidents |
| Survivorship | Casos fallidos no se registran | Incidents subreportados |

### 4.3 Implementación

```go
// internal/bias_detector/detector.go

type BiasDetector struct {
    graphStore *GraphStore
}

type BiasReport struct {
    ModuleID       string
    BiasType       string  // recency, authority, confirmation, survivorship
    Severity       string  // high, medium, low
    Description    string
    Recommendation string
}

func (d *BiasDetector) Analyze(moduleID string) ([]*BiasReport, error) {
    var reports []*BiasReport
    
    // Recency bias
    if report := d.detectRecencyBias(moduleID); report != nil {
        reports = append(reports, report)
    }
    
    // Authority bias
    if report := d.detectAuthorityBias(moduleID); report != nil {
        reports = append(reports, report)
    }
    
    // Confirmation/survivorship bias
    if report := d.detectSurvivorshipBias(moduleID); report != nil {
        reports = append(reports, report)
    }
    
    return reports, nil
}

func (d *BiasDetector) detectRecencyBias(moduleID string) *BiasReport {
    decisions := d.graphStore.GetDecisionsByYear(moduleID)
    total := sum(decisions)
    recentRatio := float64(decisions[currentYear]) / float64(total)
    
    if recentRatio > 0.7 {
        return &BiasReport{
            BiasType:       "recency",
            Severity:       "medium",
            Description:    fmt.Sprintf("%.0f%% de decisiones son del último año", recentRatio*100),
            Recommendation: "Revise decisiones de años anteriores que aún pueden aplicar",
        }
    }
    return nil
}

func (d *BiasDetector) detectSurvivorshipBias(moduleID string) *BiasReport {
    incidents := d.graphStore.GetIncidentCount(moduleID)
    decisions := d.graphStore.GetDecisionCount(moduleID)
    
    // Si hay 10x más decisiones que incidents, posible survivorship
    if incidents > 0 && decisions/incidents > 10 {
        return &BiasReport{
            BiasType:       "survivorship",
            Severity:       "high",
            Description:    "Incidencias subreportadas vs decisiones tomadas",
            Recommendation: "Revise si todos los incidentes están siendo registrados",
        }
    }
    return nil
}
```

### 4.4 Requisitos funcionales

| ID | Requisito | Prioridad |
|---|---|---|
| RF-03-01 | Detectar recency bias cuando >70% de data es reciente | MUST |
| RF-03-02 | Detectar survivorship bias cuando ratio incidents/decisions es bajo | MUST |
| RF-03-03 | Generar bias report con recomendaciones | MUST |
| RF-03-04 | Se notifica al usuario cuando bias alto es detectado | SHOULD |

---

## 5. CLI Stats

### 5.1 Visión del producto

El usuario necesita saber si el sistema está funcionando bien sin abrir un dashboard web. CLI Stats proporciona información rápida via comandos.

### 5.2 Comandos

```bash
# Stats de cada plugin
fenrir stats
hati stats
skoll stats
tyr stats

# Ejemplo de salida:
# Fenrir Stats:
# ├── Sessions: 47 active, 234 total
# ├── Knowledge Graph:
# │   ├── Nodes: 1,234 (↑ 12 this week)
# │   ├── Edges: 4,567
# │   └── Authority: 89% confirmed, 11% exploratory
# ├── Performance:
# │   ├── Avg query latency: 23ms (P99: 89ms)
# │   ├── Cache hit rate: 94%
# │   └── FTS queries: 1,234
# └── Health:
#     └── Status: ✓ Healthy

# Ecosystem unificado
rag stats --ecosystem

# Ejemplo:
# RAGNAROK Ecosystem Health:
# ├── Fenrir: ✓ (nodes: 1,234, latency: 23ms)
# ├── Hati: ✓ (active plans: 2, pending: 0)
# ├── Skoll: ✓ (skills: 12, rules: 8)
# ├── Tyr: ✓ (findings: 3, cache: 67%)
# └── Overall: ✓ Healthy
```

### 5.3 Implementación

```go
// cmd/stats.go

type StatsCommand struct{}

func (c *StatsCommand) Run(ctx *Context) error {
    // Get stats from each plugin
    fenrirStats, _ := c.fenrir.GetStats()
    hatiStats, _ := c.hati.GetStats()
    skollStats, _ := c.skoll.GetStats()
    tyrStats, _ := c.tyr.GetStats()
    
    // Print unified output
    return c.printUnifiedStats(fenrirStats, hatiStats, skollStats, tyrStats)
}

func (c *StatsCommand) printUnifiedStats(stats ...interface{}) error {
    w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
    
    fmt.Fprintln(w, "RAGNAROK Ecosystem Health:")
    fmt.Fprintln(w, "├── Fenrir: ✓ (nodes:", stats[0].(*FenrirStats).NodeCount, ")")
    fmt.Fprintln(w, "├── Hati: ✓ (active plans:", stats[1].(*HatiStats).ActivePlans, ")")
    fmt.Fprintln(w, "├── Skoll: ✓ (skills:", stats[2].(*SkollStats).ActiveSkills, ")")
    fmt.Fprintln(w, "├── Tyr: ✓ (open findings:", stats[3].(*TyrStats).OpenFindings, ")")
    fmt.Fprintln(w, "└── Overall:", c.calculateOverallHealth(stats), "\n")
    
    return w.Flush()
}
```

### 5.4 Requisitos funcionales

| ID | Requisito | Prioridad |
|---|---|---|
| RF-04-01 | `fenrir stats` muestra nodes, edges, latency, cache | MUST |
| RF-04-02 | `hati stats` muestra active plans, pending checkpoints | MUST |
| RF-04-03 | `skoll stats` muestra skills, rules, team | MUST |
| RF-04-04 | `tyr stats` muestra findings, cache, security status | MUST |
| RF-04-05 | `rag stats --ecosystem` muestra health unificado | MUST |

---

## 6. Backup Automation

### 6.1 Visión del producto

Perder el knowledge graph es catastrófico. Backup Automation hace backups automáticos del estado de cada plugin para poder recuperar.

### 6.2 Estrategia de backup

```
┌─────────────────────────────────────────────────────────────┐
│                    BACKUP STRATEGY                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  LOCALLY (siempre):                                          │
│  ├── Git commit automático del knowledge graph               │
│  ├── SQLite WAL para recoverability                          │
│  └── Cache en disco                                          │
│                                                                  │
│  PERIODIC (recomendado):                                     │
│  ├── Daily: Export a JSON                                    │
│  ├── Weekly: Backup a OneDrive/Google Drive                  │
│  └── Mensual: Backup a disco externo                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 6.3 Script de backup para Windows

```powershell
# backup_ragnarok.ps1 (ver OPS_Windows.md para detalles completos)

$BackupDir = "$env:USERPROFILE\OneDrive\RagnarokBackups"
$Date = Get-Date -Format "yyyy-MM-dd"

# Backup de cada plugin
$Plugins = @("fenrir", "hati", "skoll", "tyr")
foreach ($plugin in $Plugins) {
    $pluginDir = "$env:USERPROFILE\.$plugin"
    $backupFile = "$BackupDir\$plugin`_$Date.zip"
    
    if (Test-Path $pluginDir) {
        Compress-Archive -Path $pluginDir -DestinationPath $backupFile -Force
    }
}

# Limpiar backups antiguos (> 30 días)
Get-ChildItem $BackupDir -Filter "*.zip" | 
    Where-Object { $_.LastWriteTime -lt (Get-Date).AddDays(-30) } |
    Remove-Item
```

### 6.4 Restauración

```powershell
# restore_ragnarok.ps1

param(
    [Parameter(Mandatory=$true)]
    [string]$BackupFile,
    [Parameter(Mandatory=$true)]
    [ValidateSet("fenrir", "hati", "skoll", "tyr")]
    [string]$Plugin
)

$pluginDir = "$env:USERPROFILE\.$Plugin"
$tempBackup = "$env:TEMP\$Plugin`_backup.zip"

# Backup actual
if (Test-Path $pluginDir) {
    Compress-Archive -Path $pluginDir -DestinationPath $tempBackup -Force
}

# Restaurar
Expand-Archive -Path $BackupFile -DestinationPath $pluginDir -Force

Write-Host "Restored $Plugin. Restart MCP server for changes to take effect."
```

### 6.5 Requisitos funcionales

| ID | Requisito | Prioridad |
|---|---|---|
| RF-05-01 | Export a JSON del state completo | MUST |
| RF-05-02 | Script de backup para Windows (Tarea Programada) | MUST |
| RF-05-03 | Script de restauración | MUST |
| RF-05-04 | Integración con OneDrive/Google Drive | SHOULD |

---

## 7. Roadmap de implementación

| Fase | Semanas | Deliverable | Prioridad |
|---|---|---|---|
| **Fase 1: Pre-Commit Validation** | 1-3 | Syntax/import/type check, auto-fix | 🔴 ALTA |
| **Fase 2: Intent Verifier** | 2-4 | Intent store, code analyzer, semantic matching | 🔴 ALTA |
| **Fase 3: CLI Stats** | 3-4 | Unified stats command | 🟡 MEDIA |
| **Fase 4: Bias Detector** | 4-5 | Sesgo detection, reports | 🟡 MEDIA |
| **Fase 5: Backup Automation** | 4-5 | Scripts de backup/restore | 🟡 MEDIA |
| **Fase 6: Integración** | 5-6 | Todos los módulos integrados | MUST |
| **Fase 7: Testing & Docs** | 6-7 | Unit tests, integration tests, OPS_Windows.md | MUST |
| **Fase 8: Release v1.1.0** | 7-8 | GA release | MUST |

---

## Nuevos MCP tools en v1.1.0

| Plugin | Tool | Descripción |
|---|---|---|
| Tyr | `precommit_validate` | Validar código generado (syntax, imports, types) |
| Tyr | `precommit_autofix` | Auto-corregir errores comunes |
| Fenrir | `intent_save` | Guardar intención original con embedding |
| Fenrir | `intent_verify` | Comparar intención vs código implementado |
| Fenrir | `bias_report` | Generar reporte de sesgos detectados |
| Rag | `ecosystem_stats` | Stats unificados del ecosistema |

---

## Conte de tools actualizado

| Plugin | v1.0.0 | v1.1.0 | Total |
|---|---|---|---|
| Fenrir | 42 | +3 | **45** |
| Hati | 26 | +1 | **27** |
| Skoll | 29 | +1 | **30** |
| Tyr | 19 | +2 | **21** |
| **Total** | **116** | **+7** | **123** |

---

*Ragnarok Improvements PRD v1.1.0 — Marzo 2026*
*Enfoque: Windows single-user con agentes (OpenCode, Cursor, Windsurf)*
*Pre-Commit Validation · Intent Verifier · Bias Detector · CLI Stats · Backup Automation*
*3 Pilares: Velocidad ⚡ · Eficiencia de Tokens 💎 · Eficacia 🎯*
