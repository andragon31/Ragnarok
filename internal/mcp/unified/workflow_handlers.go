package unified

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	rootmcp "github.com/andragon31/Ragnarok/internal/mcp"
)

type Request = rootmcp.Request
type Response = rootmcp.Response

type WorkflowResult struct {
	Workflow string                 `json:"workflow"`
	Status   string                 `json:"status"`
	Steps    []WorkflowStep         `json:"steps,omitempty"`
	Results  map[string]interface{} `json:"results,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

type WorkflowStep struct {
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Output interface{} `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func (s *Server) handleWorkflowProjectBootstrap(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
		ProjectName string `json:"project_name,omitempty"`
		PRDDFile    string `json:"prd_file,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := "success"
		if err != nil {
			status = "error"
			steps = append(steps, WorkflowStep{Name: name, Status: status, Error: err.Error()})
		} else {
			steps = append(steps, WorkflowStep{Name: name, Status: status, Output: out})
		}
		results[name] = out
	}

	step("project_scan", func() (interface{}, error) {
		return s.callTool(ctx, "project_scan", map[string]interface{}{"path": params.ProjectPath})
	})

	step("project_bootstrap", func() (interface{}, error) {
		return s.callTool(ctx, "project_bootstrap", map[string]interface{}{
			"path": params.ProjectPath,
			"name": params.ProjectName,
		})
	})

	step("skill_generate", func() (interface{}, error) {
		return s.callTool(ctx, "skill_generate", map[string]interface{}{"project_path": params.ProjectPath})
	})

	step("rules_generate", func() (interface{}, error) {
		return s.callTool(ctx, "rules_generate", map[string]interface{}{"project_path": params.ProjectPath})
	})

	step("standards_generate", func() (interface{}, error) {
		return s.callTool(ctx, "standards_generate", map[string]interface{}{"project_path": params.ProjectPath})
	})

	step("agents_md_get", func() (interface{}, error) {
		return s.callTool(ctx, "agents_md_get", map[string]interface{}{"path": params.ProjectPath})
	})

	return &Response{Result: map[string]interface{}{
		"workflow": "project_bootstrap",
		"status":   "completed",
		"steps":    steps,
		"results":  results,
	}}, nil
}

func (s *Server) handleWorkflowPRDAnalyze(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PRDFile     string `json:"prd_file"`
		ProjectPath string `json:"project_path"`
		PlanTitle   string `json:"plan_title,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := "success"
		if err != nil {
			status = "error"
			steps = append(steps, WorkflowStep{Name: name, Status: status, Error: err.Error()})
		} else {
			steps = append(steps, WorkflowStep{Name: name, Status: status, Output: out})
		}
		results[name] = out
	}

	step("prd_parse", func() (interface{}, error) {
		return s.callTool(ctx, "prd_parse", map[string]interface{}{"file_path": params.PRDFile})
	})

	prdResult := results["prd_parse"]
	prdID := ""
	if prdMap, ok := prdResult.(map[string]interface{}); ok {
		if id, ok := prdMap["prd_id"].(string); ok {
			prdID = id
		}
	}

	step("plan_create_from_prd", func() (interface{}, error) {
		return s.callTool(ctx, "plan_create_from_prd", map[string]interface{}{
			"prd_id": prdID,
			"title":  params.PlanTitle,
		})
	})

	planResult := results["plan_create_from_prd"]
	planID := ""
	if planMap, ok := planResult.(map[string]interface{}); ok {
		if id, ok := planMap["plan_id"].(string); ok {
			planID = id
		}
	}

	step("human_review_create", func() (interface{}, error) {
		return s.callTool(ctx, "human_review_create", map[string]interface{}{
			"review_type": "prd_approval",
			"entity_type": "plan",
			"entity_id":   planID,
			"question":    "¿Apruebas este plan de desarrollo?",
		})
	})

	return &Response{Result: map[string]interface{}{
		"workflow": "prd_analyze",
		"status":   "completed",
		"prd_id":   prdID,
		"plan_id":  planID,
		"steps":    steps,
		"results":  results,
		"message":  "Plan created. Human review required before activation.",
	}}, nil
}

func (s *Server) handleWorkflowAgenticInit(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Title       string   `json:"title"`
		Description string   `json:"description,omitempty"`
		Phases      []string `json:"phases"`
		AgentName   string   `json:"agent_name,omitempty"`
		ProjectPath string   `json:"project_path,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := "success"
		if err != nil {
			status = "error"
			steps = append(steps, WorkflowStep{Name: name, Status: status, Error: err.Error()})
		} else {
			steps = append(steps, WorkflowStep{Name: name, Status: status, Output: out})
		}
		results[name] = out
	}

	step("plan_create", func() (interface{}, error) {
		return s.callTool(ctx, "plan_create", map[string]interface{}{
			"title":       params.Title,
			"description": params.Description,
		})
	})

	planResult := results["plan_create"]
	planID := ""
	if planMap, ok := planResult.(map[string]interface{}); ok {
		if id, ok := planMap["id"].(string); ok {
			planID = id
		}
	}

	phaseIDs := []string{}
	for i, phaseName := range params.Phases {
		phaseResult, err := s.callTool(ctx, "phase_create", map[string]interface{}{
			"plan_id":   planID,
			"title":     phaseName,
			"order_num": i,
		})
		if err == nil {
			if phaseMap, ok := phaseResult.(map[string]interface{}); ok {
				if id, ok := phaseMap["id"].(string); ok {
					phaseIDs = append(phaseIDs, id)
				}
			}
		}
		steps = append(steps, WorkflowStep{Name: "phase_create:" + phaseName, Status: "success", Output: phaseResult})
	}

	step("team_create", func() (interface{}, error) {
		return s.callTool(ctx, "team_create", map[string]interface{}{
			"name":         params.AgentName,
			"project_path": params.ProjectPath,
		})
	})

	teamResult := results["team_create"]
	teamID := ""
	if teamMap, ok := teamResult.(map[string]interface{}); ok {
		if id, ok := teamMap["team_id"].(string); ok {
			teamID = id
		}
	}

	step("human_review_create", func() (interface{}, error) {
		return s.callTool(ctx, "human_review_create", map[string]interface{}{
			"review_type": "team_approval",
			"entity_type": "plan",
			"entity_id":   planID,
			"question":    "¿Asignas los agentes a las fases del plan?",
		})
	})

	return &Response{Result: map[string]interface{}{
		"workflow":  "agentic_init",
		"status":    "completed",
		"plan_id":   planID,
		"phase_ids": phaseIDs,
		"team_id":   teamID,
		"steps":     steps,
		"results":   results,
		"message":   "Plan and phases created. Human review required to assign agents.",
	}}, nil
}

func (s *Server) handleWorkflowPlanDevelop(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID       string `json:"plan_id"`
		AgentID      string `json:"agent_id,omitempty"`
		AutoContinue bool   `json:"auto_continue,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	steps := []WorkflowStep{}

	completedTasks := 0
	blockedTasks := 0

	for {
		taskResult, err := s.callTool(ctx, "task_get_next", map[string]interface{}{
			"plan_id": params.PlanID,
		})
		if err != nil {
			steps = append(steps, WorkflowStep{Name: "task_get_next", Status: "error", Error: err.Error()})
			break
		}

		taskMap, ok := taskResult.(map[string]interface{})
		if !ok || taskMap == nil {
			steps = append(steps, WorkflowStep{Name: "task_get_next", Status: "success", Output: "no more tasks"})
			break
		}

		if allComplete, ok := taskMap["all_complete"].(bool); ok && allComplete {
			steps = append(steps, WorkflowStep{Name: "task_get_next", Status: "success", Output: "all tasks complete"})
			break
		}

		task, ok := taskMap["task"].(map[string]interface{})
		if !ok {
			break
		}

		taskID := task["id"].(string)
		taskTitle := task["title"].(string)

		steps = append(steps, WorkflowStep{Name: "task_start:" + taskTitle, Status: "in_progress"})

		_, err = s.callTool(ctx, "task_update", map[string]interface{}{
			"task_id": taskID,
			"status":  "in_progress",
		})
		if err == nil {
			completedTasks++
			steps = append(steps, WorkflowStep{Name: "task_complete:" + taskTitle, Status: "success"})
		} else {
			blockedTasks++
			steps = append(steps, WorkflowStep{Name: "task_blocked:" + taskTitle, Status: "error", Error: err.Error()})
		}

		if task["milestone"] == true {
			steps = append(steps, WorkflowStep{Name: "checkpoint_create", Status: "in_progress"})
			s.callTool(ctx, "checkpoint_open", map[string]interface{}{
				"plan_id":     params.PlanID,
				"description": "Milestone: " + taskTitle,
			})
			s.callTool(ctx, "human_review_create", map[string]interface{}{
				"review_type": "checkpoint_approval",
				"entity_type": "checkpoint",
				"entity_id":   params.PlanID,
				"question":    "¿Aprobar este checkpoint de milestone?",
			})
		}

		if !params.AutoContinue {
			break
		}
	}

	planProgress, _ := s.callTool(ctx, "plan_progress", map[string]interface{}{"plan_id": params.PlanID})

	return &Response{Result: map[string]interface{}{
		"workflow":        "plan_develop",
		"status":          "completed",
		"completed_tasks": completedTasks,
		"blocked_tasks":   blockedTasks,
		"progress":        planProgress,
		"steps":           steps,
	}}, nil
}

func (s *Server) handleWorkflowSessionStart(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Goal        string `json:"goal"`
		Module      string `json:"module,omitempty"`
		ProjectPath string `json:"project_path,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := "success"
		if err != nil {
			status = "error"
		}
		steps = append(steps, WorkflowStep{Name: name, Status: status, Output: out, Error: err.Error()})
		results[name] = out
	}

	step("mem_session_start", func() (interface{}, error) {
		return s.callTool(ctx, "mem_session_start", map[string]interface{}{
			"goal":   params.Goal,
			"module": params.Module,
		})
	})

	step("mem_context", func() (interface{}, error) {
		return s.callTool(ctx, "mem_context", map[string]interface{}{
			"module": params.Module,
		})
	})

	step("plan_list", func() (interface{}, error) {
		return s.callTool(ctx, "plan_list", map[string]interface{}{
			"status": "active",
		})
	})

	return &Response{Result: map[string]interface{}{
		"workflow": "session_start",
		"status":   "completed",
		"steps":    steps,
		"results":  results,
	}}, nil
}

func (s *Server) handleWorkflowCheckpointCreate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID      string `json:"plan_id"`
		PhaseID     string `json:"phase_id,omitempty"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := "success"
		if err != nil {
			status = "error"
			steps = append(steps, WorkflowStep{Name: name, Status: status, Error: err.Error()})
		} else {
			steps = append(steps, WorkflowStep{Name: name, Status: status, Output: out})
		}
		results[name] = out
	}

	step("checkpoint_open", func() (interface{}, error) {
		return s.callTool(ctx, "checkpoint_open", map[string]interface{}{
			"plan_id":     params.PlanID,
			"phase_id":    params.PhaseID,
			"description": params.Description,
		})
	})

	step("standard_run_all", func() (interface{}, error) {
		return s.callTool(ctx, "standard_run_all", map[string]interface{}{})
	})

	step("sast_run", func() (interface{}, error) {
		return s.callTool(ctx, "sast_run", map[string]interface{}{
			"path": ".",
		})
	})

	step("precommit_validate", func() (interface{}, error) {
		return s.callTool(ctx, "precommit_validate", map[string]interface{}{
			"path": ".",
		})
	})

	step("human_review_create", func() (interface{}, error) {
		return s.callTool(ctx, "human_review_create", map[string]interface{}{
			"review_type": "checkpoint_approval",
			"entity_type": "checkpoint",
			"entity_id":   params.PlanID,
			"question":    "¿Aprobar este checkpoint? Se han ejecutado: standards, SAST, precommit_validate",
		})
	})

	return &Response{Result: map[string]interface{}{
		"workflow": "checkpoint_create",
		"status":   "pending_review",
		"steps":    steps,
		"results":  results,
		"message":  "Checkpoint created. Human approval required.",
	}}, nil
}

func (s *Server) callTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	handler, ok := s.handlers[toolName]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	mcpReq := &Request{
		Method: toolName,
		Params: paramsJSON,
	}

	result, err := handler(ctx, mcpReq)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return result, nil
}

func (s *Server) handleEcosystemDiagnose(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Verbose bool `json:"verbose,omitempty"`
	}
	json.Unmarshal(req.Params, &params)

	diagnostics := map[string]interface{}{
		"version": s.serverVersion,
		"status":  "healthy",
		"issues":  []string{},
	}

	issues := []string{}

	if s.dbPaths == nil {
		s.dbPaths = make(map[string]string)
	}

	for name, path := range s.dbPaths {
		issue := s.checkDatabase(path, name)
		if issue != "" {
			issues = append(issues, fmt.Sprintf("%s: %s", name, issue))
		}
	}

	stats, _ := s.getDatabaseStats()
	diagnostics["database_stats"] = stats

	if len(issues) > 0 {
		diagnostics["status"] = "degraded"
		diagnostics["issues"] = issues
	}

	return &Response{
		Result: diagnostics,
	}, nil
}

func (s *Server) checkDatabase(path, label string) string {
	if path == "" {
		return "database path not configured"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Sprintf("failed to open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Sprintf("ping failed: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&count); err != nil {
		return fmt.Sprintf("query failed: %v", err)
	}

	return ""
}

func (s *Server) getDatabaseStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	fenrirPath := s.dbPaths["fenrir"]
	if fenrirPath != "" {
		if db, err := sql.Open("sqlite", fenrirPath); err == nil {
			defer db.Close()
			var obs, sessions, specs int
			db.QueryRow("SELECT COUNT(*) FROM observations").Scan(&obs)
			db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessions)
			db.QueryRow("SELECT COUNT(*) FROM specs").Scan(&specs)
			stats["fenrir"] = map[string]int{
				"observations": obs,
				"sessions":     sessions,
				"specs":        specs,
			}
		}
	}

	hatiPath := s.dbPaths["hati"]
	if hatiPath != "" {
		if db, err := sql.Open("sqlite", hatiPath); err == nil {
			defer db.Close()
			var plans, phases, tasks int
			db.QueryRow("SELECT COUNT(*) FROM plans").Scan(&plans)
			db.QueryRow("SELECT COUNT(*) FROM phases").Scan(&phases)
			db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&tasks)
			stats["hati"] = map[string]int{
				"plans":  plans,
				"phases": phases,
				"tasks":  tasks,
			}
		}
	}

	skollPath := s.dbPaths["skoll"]
	if skollPath != "" {
		if db, err := sql.Open("sqlite", skollPath); err == nil {
			defer db.Close()
			var skills, rules, agents int
			db.QueryRow("SELECT COUNT(*) FROM skills").Scan(&skills)
			db.QueryRow("SELECT COUNT(*) FROM rules").Scan(&rules)
			db.QueryRow("SELECT COUNT(*) FROM agents").Scan(&agents)
			stats["skoll"] = map[string]int{
				"skills": skills,
				"rules":  rules,
				"agents": agents,
			}
		}
	}

	tyrPath := s.dbPaths["tyr"]
	if tyrPath != "" {
		if db, err := sql.Open("sqlite", tyrPath); err == nil {
			defer db.Close()
			var findings, standards int
			db.QueryRow("SELECT COUNT(*) FROM sast_findings").Scan(&findings)
			db.QueryRow("SELECT COUNT(*) FROM standards").Scan(&standards)
			stats["tyr"] = map[string]int{
				"findings":  findings,
				"standards": standards,
			}
		}
	}

	return stats, nil
}
