# Changelog

All notable changes to Ragnarok are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [2.2.4] - 2026-03-29

### Added
- CI/CD workflow (`.github/workflows/ci.yml`)
- Release automation with GoReleaser (`.github/workflows/release.yml`)
- Cross-platform build configuration (`.goreleaser.yaml`)
- Linux/macOS installer (`install.sh`)
- Dynamic version detection in installers
- `rag setup cursor`, `rag setup windsurf`, `rag setup gemini` commands

### Fixed
- `install_quick.ps1` URL (was pointing to non-existent `ragnarok-ecosystem` org)
- Removed broken `irm|iex` detection block in `install.ps1`
- `opencode.json` now uses `rag` instead of hardcoded absolute path
- Makefile now uses dynamic versioning and proper cross-platform paths

### Changed
- `install.ps1` now downloads precompiled binaries instead of building from source
- `go.work` removed (single module project)

---

## [2.1.0] - Migration Guide

### Overview

v2.1.0 introduced a significant API redesign. 59 MCP functions were consolidated into a streamlined workflow system. This guide helps you migrate from v2.0.x to v2.1.0+.

### Breaking Changes Summary

| Module | Functions Removed | Replacement |
|--------|------------------|-------------|
| HATI | 27 | `plan_create`, `plan_get`, `plan_revise`, `checkpoint_*` |
| Fenrir | 9 | `session_*`, `spec_*`, `graph_*` |
| Skoll | 17 | `workflow_*`, `skill_*`, `rule_*` |
| Tyr | 6 | `audit_*`, `scan_*`, `guard_*` |

### HATI Module Migrations

#### `plan_lock` → `plan_update`
```json
// Before (v2.0.x)
{"method": "plan_lock", "params": {"plan_id": "..."}}

// After (v2.1.0+)
{"method": "plan_update", "params": {"plan_id": "...", "status": "locked"}}
```

#### `checkpoint_decide` → `checkpoint_approve`
```json
// Before (v2.0.x)
{"method": "checkpoint_decide", "params": {"checkpoint_id": "...", "decision": "approve"}}

// After (v2.1.0+)
{"method": "checkpoint_approve", "params": {"checkpoint_id": "...", "approver": "...", "notes": "..."}}
```

#### `notification_ack` → Use `notification_list` and mark as read externally
```json
// Before (v2.0.x)
{"method": "notification_ack", "params": {"notification_id": "..."}}

// After (v2.1.0+)
// Notifications are now retrieved and managed by the client
{"method": "notification_list", "params": {}}
```

#### `feedback_*` handlers consolidated
```json
// Before (v2.0.x) - separate handlers for each feedback type
{"method": "feedback_create", "params": {...}}
{"method": "feedback_update", "params": {...}}

// After (v2.1.0+) - use checkpoint workflow
{"method": "checkpoint_open", "params": {"plan_id": "...", "type": "review"}}
```

### Fenrir Module Migrations

#### `bias_report` → Use `session_analyze`
```json
// Before (v2.0.x)
{"method": "bias_report", "params": {"session_id": "..."}}

// After (v2.1.0+)
{"method": "session_analyze", "params": {"session_id": "..."}}
```

#### `intent_history`, `intent_predict` → `session_start`
```json
// Before (v2.0.x)
{"method": "intent_history", "params": {"session_id": "..."}}

// After (v2.1.0+)
{"method": "session_start", "params": {"goal": "...", "project_path": "..."}}
```

#### `incident_*`, `conflict_*` → Use `graph_query`
```json
// Before (v2.0.x)
{"method": "incident_report", "params": {...}}

// After (v2.1.0+)
{"method": "graph_query", "params": {"type": "incidents", "filters": {...}}}
```

### Skoll Module Migrations

#### `workflow_start`, `workflow_execute`, `workflow_status` → `workflow_*` handlers
```json
// Before (v2.0.x)
{"method": "workflow_start", "params": {"workflow_id": "..."}}

// After (v2.1.0+)
{"method": "workflow_stack_based_init", "params": {"project_path": "..."}}
// or
{"method": "workflow_plan_develop_v2", "params": {"plan_id": "..."}}
```

#### `task_pending`, `task_assign`, `task_complete` → `task_*` handlers
```json
// Before (v2.0.x)
{"method": "task_pending", "params": {"agent_id": "..."}}

// After (v2.1.0+)
{"method": "task_get_next", "params": {"agent_id": "..."}}
// then
{"method": "task_update", "params": {"task_id": "...", "status": "completed"}}
```

#### `rule_pending`, `rule_evaluate` → `rule_*` handlers
```json
// Before (v2.0.x)
{"method": "rule_pending", "params": {}}

// After (v2.1.0+)
{"method": "rule_list", "params": {"status": "pending"}}
```

### Tyr Module Migrations

#### `audit_log`, `audit_query` → `audit_snapshot`
```json
// Before (v2.0.x)
{"method": "audit_log", "params": {"project_path": "..."}}

// After (v2.1.0+)
{"method": "audit_snapshot", "params": {"project_path": "..."}}
```

#### `inject_guard`, `proactive_scan` → `quality_snapshot`
```json
// Before (v2.0.x)
{"method": "inject_guard", "params": {"file": "..."}}

// After (v2.1.0+)
{"method": "quality_snapshot", "params": {"project_path": "..."}}
```

### New Workflow System

v2.1.0 introduces a unified workflow system. Instead of calling individual functions:

#### Old Pattern (v2.0.x)
```
plan_lock → task_pending → task_assign → task_complete → checkpoint_decide → ...
```

#### New Pattern (v2.1.0+)
```
workflow_stack_based_init → workflow_plan_develop_v2 → workflow_checkpoint_create
```

The new workflow system provides:
- Automatic phase and task generation
- Built-in checkpoint and human review gates
- Multi-agent delegation
- Context preservation across sessions

### Recommended Migration Steps

1. **Audit current usage**: List all MCP calls your agent/integration makes
2. **Map to new handlers**: Use the migration tables above
3. **Update parameter structures**: New handlers may have different param schemas
4. **Test thoroughly**: The new workflow system may behave differently
5. **Consider workflow API**: If making many API calls, consider using the new workflow system

### Getting Help

If you encounter issues migrating:
- Check the [AGENTS.md](AGENTS.md) for current API documentation
- Run `rag doctor` to verify your installation
- Review example integrations in `examples/`

---

## [2.0.x] - Previous Versions

v2.0.x releases are no longer supported. Please upgrade to v2.1.0 or later.

---

*This migration guide was created to help users of Ragnarok v2.0.x transition to v2.1.0+. For the most up-to-date API documentation, see [AGENTS.md](AGENTS.md).*
