package unified

import (
	"context"
	"encoding/json"
	"fmt"
)

// registerHelpHandlers registers all new Fase 3 meta-tools.
func (s *Server) registerHelpHandlers() {
	newTools := map[string]struct {
		fn   func(context.Context, *Request) (*Response, error)
	}{
		"ragnarok_help":        {fn: s.handleRagnarokHelp},
		"ragnarok_status":      {fn: s.handleRagnarokStatus},
		"plan_get_active":      {fn: s.handlePlanGetActive},
		"session_context_full": {fn: s.handleSessionContextFull},
		"quality_gate":         {fn: s.handleQualityGate},
		"plan_dashboard":       {fn: s.handlePlanDashboard},
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
			"title": "Getting Started with Ragnarok",
			"steps": []string{
				"1. ragnarok_status — verify all modules are healthy",
				"2. project_scan(path) — detect stack and architecture",
				"3. workflow_project_lifecycle(project_path) — auto-create full plan",
				"4. human_review_pending() — check for approvals needed",
				"5. task_get_next(plan_id) — get next task to work on",
				"6. [do the work]",
				"7. task_update(task_id, status=completed)",
				"8. mem_save(title, type, what, why) — persist what you learned",
				"9. Repeat from step 5 until {all_complete: true}",
				"10. workflow_checkpoint_create(plan_id) — quality milestone",
			},
		}}, nil
	case "planning":
		return &Response{Result: map[string]interface{}{
			"title": "Planning Tools (Hati module)",
			"tools": map[string]string{
				"plan_create":          "Create a plan — returns plan_id. Required: title",
				"plan_get":             "Get plan details — required: id",
				"plan_list":            "List plans by status",
				"plan_get_active":      "Get active plan (no id needed)",
				"plan_dashboard":       "Full plan overview: phases + tasks + progress",
				"phase_create":         "Add phase — required: plan_id, title",
				"task_create":          "Add task — required: phase_id, title",
				"task_get_next":        "Next pending task — required: plan_id",
				"task_update":          "Update task status — required: task_id",
				"task_assign_agents":   "Assign agents to task — required: task_id, agent_ids",
				"human_review_pending": "List pending human approvals",
				"human_review_decide":  "Approve/reject — required: review_id, decision",
			},
		}}, nil
	case "memory":
		return &Response{Result: map[string]interface{}{
			"title": "Memory Tools (Fenrir module)",
			"tools": map[string]string{
				"mem_save":              "Save observation — required: title, type",
				"mem_find":              "Search memories — required: query",
				"mem_context":           "Context for a module path",
				"mem_session_start":     "Start session — required: goal",
				"mem_session_end":       "End session with summary",
				"session_context_full":  "Full context in one call",
				"spec_save":             "Save a spec/constraint",
				"spec_check":            "Verify spec compliance",
			},
		}}, nil
	case "quality":
		return &Response{Result: map[string]interface{}{
			"title": "Quality Tools (Tyr module)",
			"tools": map[string]string{
				"quality_gate":       "Full quality check in one call — required: path",
				"sast_run":           "SAST security scan",
				"sast_findings":      "Get security findings",
				"standard_run_all":   "Run all quality standards",
				"precommit_validate": "Validate pre-commit hooks — required: path",
				"pkg_check":          "Check package for CVEs — required: name",
			},
		}}, nil
	case "orchestration":
		return &Response{Result: map[string]interface{}{
			"title": "Orchestration Tools (Skoll module)",
			"tools": map[string]string{
				"agent_list":     "List all agents with status",
				"agent_create":   "Create agent — required: name, role",
				"task_execute":   "Execute task — required: task_id, agent_id",
				"task_delegate":  "Delegate to agents — required: task_id, agent_ids",
				"task_complete":  "Complete execution — required: execution_id",
				"task_heartbeat": "Keep-alive — required: execution_id",
				"skill_list":     "List available skills",
				"team_create":    "Create team — required: name",
			},
		}}, nil
	case "workflows":
		return &Response{Result: map[string]interface{}{
			"title": "Recommended Workflows",
			"recommended": []map[string]string{
				{"name": "workflow_project_lifecycle", "desc": "Recommended full start: PRD Analysis -> Skoll Agents & Team -> Hati Planning -> Task Assignment -> Tyr Quality Baseline"},
				{"name": "workflow_team_setup_from_prd", "desc": "PRD Analysis -> Skoll Agents -> Team Creation"},
				{"name": "workflow_stack_based_init", "desc": "Stack-based plan initialization with Hati"},
				{"name": "workflow_plan_develop_v2", "desc": "Multi-agent development loop with task delegation"},
				{"name": "workflow_checkpoint_create", "desc": "Quality milestone with human review and validation"},
			},
		}}, nil
	case "project_init":
		return &Response{Result: map[string]interface{}{
			"title": "Project Initialization from PRD",
			"description": "The best way to start a project is using the integrated lifecycle workflow.",
			"command_example": "workflow_project_lifecycle(project_path='./path', prd_file='./PRD.md', title='Project Name')",
			"what_it_does": []string{
				"1. Analyzes tech stack and architecture (Fenrir)",
				"2. Parses the PRD to extract requirements (Hati)",
				"3. Creates specialized agents in Skoll (Backend, Frontend, etc.)",
				"4. Forms a project team and assigns them (Skoll)",
				"5. Generates a multi-phase development plan (Hati)",
				"6. Auto-assigns agents to matching tasks",
				"7. Runs a security baseline scan (Tyr)",
				"8. Stores all context in persistent memory (Fenrir)",
			},
			"next_steps": "After initialization, wait for 'human_review_pending' to approve the plan, then start 'task_get_next'.",
		}}, nil
	default:
		return &Response{Result: map[string]interface{}{
			"title":       "Ragnarok MCP Ecosystem v2.2.4",
			"description": "Orchestrates AI agents for software development. Modules: Fenrir (memory), Hati (planning), Skoll (orchestration), Tyr (quality).",
			"quick_start": "Call ragnarok_help(topic='getting_started') or topic='project_init' for PRD-driven setup.",
			"topics":      []string{"getting_started", "project_init", "planning", "memory", "quality", "orchestration", "workflows"},
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
		"tyr_stats":    "tyr",
	} {
		if stats, err := s.callTool(ctx, toolName, map[string]interface{}{}); err == nil {
			if m, ok := modules[modName].(map[string]interface{}); ok {
				m["stats"] = stats
			}
		}
	}

	return &Response{Result: map[string]interface{}{
		"version": "2.2.4",
		"modules": modules,
		"tools":   len(s.tools),
	}}, nil
}

// handlePlanGetActive returns the active plan without requiring a plan_id.
func (s *Server) handlePlanGetActive(ctx context.Context, req *Request) (*Response, error) {
	result, err := s.callTool(ctx, "plan_list", map[string]interface{}{"status": "active"})
	if err != nil {
		return nil, fmt.Errorf("failed to get active plans: %w", err)
	}
	return &Response{Result: result}, nil
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

// handleQualityGate runs a full quality check in one call.
func (s *Server) handleQualityGate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Path   string `json:"path"`
		PlanID string `json:"plan_id,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}
	if params.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	results := map[string]interface{}{"path": params.Path, "passed": true}
	var failures []string

	if sast, err := s.callTool(ctx, "sast_run", map[string]interface{}{"path": params.Path}); err == nil {
		results["sast"] = sast
	}
	if standards, err := s.callTool(ctx, "standard_run_all", map[string]interface{}{}); err == nil {
		results["standards"] = standards
	} else {
		failures = append(failures, "standards: "+err.Error())
	}
	if precommit, err := s.callTool(ctx, "precommit_validate", map[string]interface{}{"path": params.Path}); err == nil {
		results["precommit"] = precommit
	} else {
		failures = append(failures, "precommit: "+err.Error())
	}
	if findings, err := s.callTool(ctx, "sast_findings", map[string]interface{}{"severity": "critical"}); err == nil {
		results["critical_findings"] = findings
	}

	if len(failures) > 0 {
		results["passed"] = false
		results["failures"] = failures
	}
	return &Response{Result: results}, nil
}

// handlePlanDashboard returns a full plan dashboard.
func (s *Server) handlePlanDashboard(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}
	if params.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	dashboard := map[string]interface{}{"plan_id": params.PlanID}
	if plan, err := s.callTool(ctx, "plan_get", map[string]interface{}{"id": params.PlanID}); err == nil {
		dashboard["plan"] = plan
	}
	if progress, err := s.callTool(ctx, "plan_progress", map[string]interface{}{"plan_id": params.PlanID}); err == nil {
		dashboard["progress"] = progress
	}
	if tasks, err := s.callTool(ctx, "task_list", map[string]interface{}{"plan_id": params.PlanID}); err == nil {
		dashboard["tasks"] = tasks
	}
	if blockers, err := s.callTool(ctx, "plan_blockers", map[string]interface{}{"id": params.PlanID}); err == nil {
		dashboard["blockers"] = blockers
	}
	return &Response{Result: dashboard}, nil
}
