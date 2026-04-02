package unified

import (
	"context"
	"encoding/json"
)

// registerHelpHandlers registers all new Fase 3 meta-tools.
func (s *Server) registerHelpHandlers() {
	newTools := map[string]struct {
		fn   func(context.Context, *Request) (*Response, error)
	}{
		"ragnarok_help":        {fn: s.handleRagnarokHelp},
		"ragnarok_status":      {fn: s.handleRagnarokStatus},
		"session_context_full": {fn: s.handleSessionContextFull},
	}
	for name, t := range newTools {
		s.handlers[name] = t.fn
		s.tools = append(s.tools, Tool{
			Name:        name,
			Description: getToolDescription(name),
			InputSchema: json.RawMessage(getToolInputSchema(name)),
		})
	}
}

// handleRagnarokHelp returns usage instructions for Ragnarok.
func (s *Server) handleRagnarokHelp(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Topic string `json:"topic,omitempty"`
	}
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}

	switch params.Topic {
	case "getting_started":
		return &Response{Result: map[string]interface{}{
			"title": "Getting Started with Ragnarok Ecosystem",
			"description": "Follow these steps to initialize a project and start development with AI agents.",
			"steps": []string{
				"1. ragnarok_help(topic='project_init') — learn how to start from a PRD",
				"2. ragnarok_status — verify all modules (Fenrir, Hati, Skoll, Tyr) are healthy",
				"3. workflow_project_lifecycle(project_path, prd_file) — automated full initialization",
				"4. plan_get_active — retrieve the plan created by the lifecycle workflow",
				"5. human_review_pending — check if you need to approve the plan or agents",
				"6. task_get_next(plan_id) — get the highest priority unblocked task",
				"7. [Implement the feature/fix using your coding tools]",
				"8. mem_save(title, type, what, why, learned) — persist key decisions to long-term memory",
				"9. task_update(task_id, status='completed') — mark progress in Hati",
				"10. workflow_checkpoint_create(plan_id) — trigger quality gate and milestone approval",
			},
			"pro_tip": "Run ragnarok_help(topic='workflows') to see advanced orchestration patterns.",
		}}, nil
	case "planning":
		return &Response{Result: map[string]interface{}{
			"title": "Planning & Project Management (Hati Module)",
			"description": "Hati manages the development lifecycle through plans, phases, and tasks.",
			"core_tools": map[string]string{
				"plan_create":          "Create a new plan root. Returns plan_id.",
				"plan_get":             "Retrieve full plan structure, phases, and aggregated progress.",
				"plan_list":            "List plans by status (active, completed, abandoned).",
				"plan_get_active":      "Convenience: get the currently active plan for the workspace.",
				"plan_dashboard":       "High-level overview of plan health, blockers, and velocity.",
				"task_create":          "Add a task to a phase. Required: phase_id, title.",
				"task_get_next":        "Retrieve the next unblocked task according to priority.",
				"task_update":          "Update status (in_progress, completed, blocked).",
				"human_review_pending": "Check for tasks/plans awaiting human decision.",
				"human_review_decide":  "Submit approved/rejected decision for a review request.",
			},
		}}, nil
	case "memory":
		return &Response{Result: map[string]interface{}{
			"title": "Memory & Context Layer (Fenrir Module)",
			"description": "Fenrir provides long-term persistence for observations, decisions, and specs.",
			"core_tools": map[string]string{
				"mem_save":              "Save a development observation. Required: title, type, what, why.",
				"mem_find":              "Search memory store using FTS5 full-text search.",
				"mem_context":           "Get recent history for a specific module or file path.",
				"mem_session_start":     "Initiate a work session with a goal for context grouping.",
				"mem_session_end":       "Finalize session and distill learned context.",
				"session_context_full":  "Retrieve plan + tasks + memory + agents in one call.",
				"spec_save":             "Persist an architectural specification or coding constraint.",
				"spec_check":            "Validate if code complies with registered specifications.",
				"project_scan":          "Deep scan of project to detect stack, architecture, and modules.",
			},
		}}, nil
	case "quality":
		return &Response{Result: map[string]interface{}{
			"title": "Quality Assurance & Standards (Tyr Module)",
			"description": "Tyr enforces coding standards, security, and dependency health.",
			"core_tools": map[string]string{
				"tyr_snapshot":       "Get a full quality snapshot: standards + SAST + metrics.",
				"tyr_bootstrap":      "Import default quality rules and skills for a new project.",
				"sast_run":           "Execute Static Analysis Security Testing scan.",
				"sast_findings":      "Get vulnerability findings filtered by severity.",
				"standard_run_all":   "Run all registered quality standards (lint, tests, etc.).",
				"precommit_validate": "Check code against pre-commit hooks quality baseline.",
				"pkg_check":          "Evaluate a package for CVEs and trust score.",
			},
		}}, nil
	case "orchestration":
		return &Response{Result: map[string]interface{}{
			"title": "Agent Orchestration (Skoll Module)",
			"description": "Skoll manages agent roles, teams, skills, and parallel execution.",
			"core_tools": map[string]string{
				"agent_list":     "List all registered agents and their current availability.",
				"agent_create":   "Register a new agent role (e.g., backend-agent, qa-agent).",
				"agent_activate": "Make an agent available for task assignment in a team.",
				"task_execute":   "Start task execution by a specific agent role.",
				"task_delegate":  "Distribute a task to multiple agents for parallel work.",
				"task_status":    "Track execution progress and agent heartbeats.",
				"team_create":    "Form a coordination team for a specific project path.",
				"skill_list":     "List available automated skills (e.g., 'go-test', 'js-lint').",
			},
		}}, nil
	case "workflows":
		return &Response{Result: map[string]interface{}{
			"title": "Automated Workflows & Lifecycles",
			"description": "High-level orchestrations that combine multiple tools for complex goals.",
			"recommended": []map[string]string{
				{"name": "workflow_project_lifecycle", "desc": "PRD Analysis -> Team Formation -> Planning -> Quality Baseline (Best for new projects)"},
				{"name": "workflow_prd_analyze", "desc": "Scan stack and parse PRD requirements into a Hati plan"},
				{"name": "workflow_plan_develop_v2", "desc": "Multi-agent loop: Get task -> Delegate -> Execute -> Verify"},
				{"name": "workflow_checkpoint_create", "desc": "Quality gate: Run all tests -> SAST -> Human Review -> Plan Update"},
				{"name": "workflow_session_start", "desc": "Load project context and memories for a new development session"},
			},
		}}, nil
	case "project_init":
		return &Response{Result: map[string]interface{}{
			"title": "Project Initialization Guide",
			"description": "Use the integrated lifecycle to set up a project from a PRD file.",
			"command": "workflow_project_lifecycle(project_path='./', prd_file='./PRD.md', title='My App')",
			"actions": []string{
				"1. Fenrir scans your tech stack (Go, React, Python, etc.)",
				"2. Hati extracts functional/non-functional requirements from the PRD",
				"3. Skoll creates a balanced team of specialized AI agents",
				"4. Hati generates an execution plan with phases and estimated tasks",
				"5. Tyr runs a security baseline scan on existing code",
			},
			"benefit": "This gives you full context, a team, and a roadmap in one single call.",
		}}, nil
	default:
		return &Response{Result: map[string]interface{}{
			"id":          "ragnarok-v3",
			"title":       "Ragnarok MCP Ecosystem v2.4.11",
			"description": "The ultimate orchestration layer for autonomous AI software development.",
			"modules":     "Memory (Fenrir), Planning (Hati), Orchestration (Skoll), and Quality (Tyr).",
			"quick_start": "Call ragnarok_help(topic='getting_started') to begin.",
		}}, nil
	}
}

// handleRagnarokStatus returns full ecosystem health status.
func (s *Server) handleRagnarokStatus(ctx context.Context, req *Request) (*Response, error) {
	modules := map[string]interface{}{}
	for name, path := range s.dbPaths {
		modules[name] = map[string]interface{}{"status": "healthy", "db_path": path}
	}

	for toolName, modName := range map[string]string{
		"mem_stats":    "fenrir",
		"hati_stats":   "hati",
		"skoll_status": "skoll",
		"tyr_status":    "tyr",
	} {
		if stats, err := s.callTool(ctx, toolName, map[string]interface{}{}); err == nil {
			if m, ok := modules[modName].(map[string]interface{}); ok {
				m["stats"] = stats
			}
		}
	}

	return &Response{Result: map[string]interface{}{
		"version": "2.4.11",
		"status":  "healthy",
		"modules": modules,
		"tools":   len(s.tools),
	}}, nil
}

// handleSessionContextFull returns comprehensive session context in one call.
func (s *Server) handleSessionContextFull(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id,omitempty"`
	}
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}

	result := map[string]interface{}{}

	if params.PlanID != "" {
		if plan, err := s.callTool(ctx, "plan_get", map[string]interface{}{"id": params.PlanID}); err == nil {
			result["plan"] = plan
		}
		if tasks, err := s.callTool(ctx, "task_list", map[string]interface{}{"plan_id": params.PlanID, "status": "pending"}); err == nil {
			result["pending_tasks"] = tasks
		}
		if nextTask, err := s.callTool(ctx, "task_get_next", map[string]interface{}{"plan_id": params.PlanID}); err == nil {
			result["next_task"] = nextTask
		}
	} else {
		if plans, err := s.callTool(ctx, "plan_list", map[string]interface{}{"status": "active"}); err == nil {
			result["active_plans"] = plans
		}
	}

	if mem, err := s.callTool(ctx, "mem_timeline", map[string]interface{}{"limit": 5}); err == nil {
		result["recent_memory"] = mem
	}
	if agents, err := s.callTool(ctx, "agent_list", map[string]interface{}{}); err == nil {
		result["agents"] = agents
	}
	if reviews, err := s.callTool(ctx, "human_review_pending", map[string]interface{}{}); err == nil {
		result["pending_reviews"] = reviews
	}

	return &Response{Result: result}, nil
}
