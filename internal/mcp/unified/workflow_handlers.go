package unified

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/andragon31/Ragnarok/internal/fenrir/scanner"
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

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	step("project_scan", func() (interface{}, error) {
		return s.callTool(ctx, "project_scan", map[string]interface{}{"path": params.ProjectPath})
	})

	scanResult := results["project_scan"]
	analysis, _ := parseProjectAnalysis(scanResult)

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

	if analysis != nil {
		phaseTemplates := scanner.GeneratePhasesAndTasks(analysis)
		for i, template := range phaseTemplates {
			s.callTool(ctx, "phase_create", map[string]interface{}{
				"plan_id":   planID,
				"title":     template.Name,
				"order_num": i,
			})
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
		"workflow":       "prd_analyze",
		"status":         "completed",
		"prd_id":         prdID,
		"plan_id":        planID,
		"stack_detected": analysis != nil,
		"stack":          getStackFromAnalysis(analysis),
		"steps":          steps,
		"results":        results,
		"message":        "Plan created with stack-based phases. Human review required before activation.",
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
			steps = append(steps, WorkflowStep{Name: name, Status: status, Error: err.Error()})
		} else {
			steps = append(steps, WorkflowStep{Name: name, Status: status, Output: out})
		}
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

	return result.Result, nil
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

func (s *Server) handleWorkflowStackBasedInit(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string   `json:"project_path"`
		Title       string   `json:"title"`
		Phases      []string `json:"phases,omitempty"`
		AgentIDs    []string `json:"agent_ids,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
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

	scanResult := results["project_scan"]
	analysis, err := parseProjectAnalysis(scanResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project scan: %w", err)
	}

	step("plan_create", func() (interface{}, error) {
		stack, arch := getStackInfoSafe(analysis)
		desc := fmt.Sprintf("Plan for %s - %s architecture with %s",
			analysis.Name, arch, stack)
		return s.callTool(ctx, "plan_create", map[string]interface{}{
			"title":       params.Title,
			"description": desc,
		})
	})

	planResult := results["plan_create"]
	planID := ""
	if planMap, ok := planResult.(map[string]interface{}); ok {
		if id, ok := planMap["id"].(string); ok {
			planID = id
		}
	}

	phaseTemplates := scanner.GeneratePhasesAndTasks(analysis)
	recommendedAgents := scanner.GetRecommendedAgents(analysis)

	phaseIDs := []string{}
	taskIDs := []string{}

	for i, template := range phaseTemplates {
		phaseResult, err := s.callTool(ctx, "phase_create", map[string]interface{}{
			"plan_id":   planID,
			"title":     template.Name,
			"order_num": i,
		})
		if err != nil {
			steps = append(steps, WorkflowStep{Name: "phase_create:" + template.Name, Status: "error", Error: err.Error()})
			continue
		}

		phaseID := ""
		if phaseMap, ok := phaseResult.(map[string]interface{}); ok {
			if id, ok := phaseMap["id"].(string); ok {
				phaseID = id
				phaseIDs = append(phaseIDs, id)
			}
		}

		steps = append(steps, WorkflowStep{Name: "phase_create:" + template.Name, Status: "success", Output: phaseResult})

		for _, taskTemplate := range template.Tasks {
			agentIDsForTask := params.AgentIDs
			if len(agentIDsForTask) == 0 {
				for _, at := range taskTemplate.AgentTypes {
					agentID := findAgentByType(at, recommendedAgents)
					if agentID != "" {
						agentIDsForTask = append(agentIDsForTask, agentID)
					}
				}
			}

			taskResult, err := s.callTool(ctx, "task_create", map[string]interface{}{
				"phase_id":    phaseID,
				"title":       taskTemplate.Title,
				"description": taskTemplate.Description,
				"priority":    taskTemplate.Priority,
				"milestone":   taskTemplate.Milestone,
			})
			if err != nil {
				steps = append(steps, WorkflowStep{Name: "task_create:" + taskTemplate.Title, Status: "error", Error: err.Error()})
				continue
			}

			taskID := ""
			if taskMap, ok := taskResult.(map[string]interface{}); ok {
				if id, ok := taskMap["id"].(string); ok {
					taskID = id
					taskIDs = append(taskIDs, id)
				}
			}

			if len(agentIDsForTask) > 0 {
				s.callTool(ctx, "task_assign_agents", map[string]interface{}{
					"task_id":   taskID,
					"agent_ids": agentIDsForTask,
					"role":      "worker",
				})
			}

			steps = append(steps, WorkflowStep{Name: "task_create:" + taskTemplate.Title, Status: "success", Output: taskResult})
		}
	}

	step("human_review_create", func() (interface{}, error) {
		stackLang := getStackLanguage(analysis)
		return s.callTool(ctx, "human_review_create", map[string]interface{}{
			"review_type": "prd_approval",
			"entity_type": "plan",
			"entity_id":   planID,
			"question":    fmt.Sprintf("¿Apruebas este plan de %d fases con %d tareas basado en tu stack de %s?", len(phaseIDs), len(taskIDs), stackLang),
		})
	})

	return &Response{Result: map[string]interface{}{
		"workflow":     "stack_based_init",
		"status":       "completed",
		"plan_id":      planID,
		"phase_ids":    phaseIDs,
		"task_ids":     taskIDs,
		"stack":        getStackLanguage(analysis),
		"architecture": analysis != nil && analysis.Architecture != nil,
		"agents":       recommendedAgents,
		"steps":        steps,
		"results":      results,
		"message":      fmt.Sprintf("Plan created with %d phases and %d tasks based on %s stack", len(phaseIDs), len(taskIDs), getStackLanguage(analysis)),
	}}, nil
}

func (s *Server) handleWorkflowPlanDevelopV2(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID       string `json:"plan_id"`
		AgentID      string `json:"agent_id,omitempty"`
		AutoContinue bool   `json:"auto_continue,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	steps := []WorkflowStep{}

	for {
		taskResult, err := s.callTool(ctx, "task_get_next", map[string]interface{}{
			"plan_id":  params.PlanID,
			"agent_id": params.AgentID,
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
		taskAgents, _ := task["task_agents"].([]interface{})

		steps = append(steps, WorkflowStep{Name: "task_start:" + taskTitle, Status: "in_progress"})

		if len(taskAgents) > 0 {
			for _, ta := range taskAgents {
				taMap, ok := ta.(map[string]interface{})
				if !ok {
					continue
				}
				execResult, err := s.callTool(ctx, "task_execute", map[string]interface{}{
					"task_id":  taskID,
					"agent_id": taMap["agent_id"],
				})
				if err != nil {
					steps = append(steps, WorkflowStep{Name: "task_delegate:" + taskTitle, Status: "error", Error: err.Error()})
				} else {
					steps = append(steps, WorkflowStep{Name: "task_delegate:" + taskTitle, Status: "success", Output: execResult})
				}
			}
		} else {
			s.callTool(ctx, "task_update", map[string]interface{}{
				"task_id": taskID,
				"status":  "in_progress",
			})
		}

		if task["milestone"] == true {
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

		time.Sleep(100 * time.Millisecond)
	}

	planProgress, _ := s.callTool(ctx, "plan_progress", map[string]interface{}{"plan_id": params.PlanID})

	return &Response{Result: map[string]interface{}{
		"workflow": "plan_develop_v2",
		"status":   "completed",
		"progress": planProgress,
		"steps":    steps,
	}}, nil
}

func getStackFromAnalysis(analysis *scanner.ProjectAnalysis) string {
	if analysis == nil || analysis.Stack == nil {
		return ""
	}
	if analysis.Stack.Language != "" {
		return analysis.Stack.Language
	}
	return ""
}

func getStackInfoSafe(analysis *scanner.ProjectAnalysis) (stack, arch string) {
	if analysis == nil {
		return "unknown", "unknown"
	}
	stack = "unknown"
	arch = "unknown"
	if analysis.Stack != nil {
		if analysis.Stack.Framework != "" {
			stack = analysis.Stack.Language + " with " + analysis.Stack.Framework
		} else {
			stack = analysis.Stack.Language
		}
	}
	if analysis.Architecture != nil {
		arch = analysis.Architecture.Type
	}
	return stack, arch
}

func hasFrontend(analysis *scanner.ProjectAnalysis) bool {
	if analysis == nil || analysis.Architecture == nil {
		return false
	}
	return analysis.Architecture.HasFrontend
}

func getStackLanguage(analysis *scanner.ProjectAnalysis) string {
	if analysis == nil || analysis.Stack == nil {
		return ""
	}
	return analysis.Stack.Language
}

func parseProjectAnalysis(result interface{}) (*scanner.ProjectAnalysis, error) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid result type")
	}

	analysis := &scanner.ProjectAnalysis{}

	if name, ok := resultMap["name"].(string); ok {
		analysis.Name = name
	}
	if path, ok := resultMap["path"].(string); ok {
		analysis.Path = path
	}

	if stackMap, ok := resultMap["stack"].(map[string]interface{}); ok {
		analysis.Stack = &scanner.StackInfo{}
		if lang, ok := stackMap["language"].(string); ok {
			analysis.Stack.Language = lang
		}
		if fw, ok := stackMap["framework"].(string); ok {
			analysis.Stack.Framework = fw
		}
		if pkg, ok := stackMap["package_manager"].(string); ok {
			analysis.Stack.PackageMgr = pkg
		}
		if ci, ok := stackMap["ci_tool"].(string); ok {
			analysis.Stack.CITool = ci
		}
		if db, ok := stackMap["db_engine"].(string); ok {
			analysis.Stack.DBEngine = db
		}
		if hasDocker, ok := stackMap["has_docker"].(bool); ok {
			analysis.Stack.HasDocker = hasDocker
		}
		if hasCI, ok := stackMap["has_ci"].(bool); ok {
			analysis.Stack.HasCI = hasCI
		}
		if hasTests, ok := stackMap["has_tests"].(bool); ok {
			analysis.Stack.HasTests = hasTests
		}
	}

	if archMap, ok := resultMap["architecture"].(map[string]interface{}); ok {
		analysis.Architecture = &scanner.ArchitectureInfo{}
		if archType, ok := archMap["type"].(string); ok {
			analysis.Architecture.Type = archType
		}
		if hasAPI, ok := archMap["has_api"].(bool); ok {
			analysis.Architecture.HasAPI = hasAPI
		}
		if hasFE, ok := archMap["has_frontend"].(bool); ok {
			analysis.Architecture.HasFrontend = hasFE
		}
		if isMono, ok := archMap["is_monorepo"].(bool); ok {
			analysis.Architecture.IsMonorepo = isMono
		}
		if feLib, ok := archMap["frontend_lib"].(string); ok {
			analysis.Architecture.FrontendLib = feLib
		}
		if apiFW, ok := archMap["api_framework"].(string); ok {
			analysis.Architecture.APIFramework = apiFW
		}
	}

	return analysis, nil
}

func findAgentByType(agentType string, agents []map[string]string) string {
	for _, agent := range agents {
		if agent["type"] == agentType {
			return agent["name"]
		}
	}
	return ""
}

func (s *Server) handleWorkflowProjectLifecycle(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath  string   `json:"project_path"`
		PRDFile      string   `json:"prd_file,omitempty"`
		Title        string   `json:"title,omitempty"`
		Requirements []string `json:"requirements,omitempty"`
		AutoStart    bool     `json:"auto_start"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ProjectPath == "" {
		return nil, fmt.Errorf("project_path is required")
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

	fmt.Printf("🔄 Project Lifecycle: Starting full cycle\n")
	fmt.Printf("   Project: %s\n", params.ProjectPath)
	if params.PRDFile != "" {
		fmt.Printf("   PRD: %s\n", params.PRDFile)
	}
	if len(params.Requirements) > 0 {
		fmt.Printf("   Requirements: %d\n", len(params.Requirements))
	}
	fmt.Println()

	step("Fenrir: Analyze Project", func() (interface{}, error) {
		return s.callTool(ctx, "project_scan", map[string]interface{}{"path": params.ProjectPath})
	})

	scanResult := results["Fenrir: Analyze Project"]
	analysis, _ := parseProjectAnalysis(scanResult)

	var projectName string
	if params.Title != "" {
		projectName = params.Title
	} else if analysis != nil {
		projectName = analysis.Name
	} else {
		projectName = "Unnamed Project"
	}

	step("Fenrir: Store Project Context", func() (interface{}, error) {
		stackInfo, archInfo := getStackInfoSafe(analysis)
		return s.callTool(ctx, "mem_save", map[string]interface{}{
			"title": "Project: " + projectName,
			"type":  "decision",
			"what":  fmt.Sprintf("Project lifecycle started - Stack: %s, Architecture: %s", stackInfo, archInfo),
			"where": params.ProjectPath,
			"why":   "Initial project analysis and context setup",
		})
	})

	var prdID string
	if params.PRDFile != "" {
		step("Fenrir: Parse PRD", func() (interface{}, error) {
			return s.callTool(ctx, "prd_parse", map[string]interface{}{"file_path": params.PRDFile})
		})

		prdResult := results["Fenrir: Parse PRD"]
		if prdMap, ok := prdResult.(map[string]interface{}); ok {
			if id, ok := prdMap["prd_id"].(string); ok {
				prdID = id
			}
		}

		step("Fenrir: Store Requirements", func() (interface{}, error) {
			var requirements []interface{}
			if prdMap, ok := results["Fenrir: Parse PRD"].(map[string]interface{}); ok {
				if reqs, ok := prdMap["requirements"].([]interface{}); ok {
					requirements = reqs
				}
			}
			for _, req := range requirements {
				if reqText, ok := req.(string); ok {
					s.callTool(ctx, "mem_save", map[string]interface{}{
						"title": "PRD Requirement: " + reqText[:min(50, len(reqText))],
						"type":  "requirement",
						"what":  reqText,
						"where": params.ProjectPath,
					})
				}
			}
			return map[string]interface{}{"requirements_stored": len(requirements)}, nil
		})
	}

	for _, req := range params.Requirements {
		step("Fenrir: Store Requirement: "+req[:min(30, len(req))], func() (interface{}, error) {
			return s.callTool(ctx, "mem_save", map[string]interface{}{
				"title": "Requirement: " + req[:min(50, len(req))],
				"type":  "requirement",
				"what":  req,
				"where": params.ProjectPath,
			})
		})
	}

	planTitle := projectName + " Development Plan"
	if params.Title != "" {
		planTitle = params.Title
	}

	stackInfo, archInfo := getStackInfoSafe(analysis)
	if stackInfo == "unknown" {
		stackInfo = "auto"
	}
	if archInfo == "unknown" {
		archInfo = "auto"
	}
	planDesc := fmt.Sprintf("Stack: %s, Architecture: %s", stackInfo, archInfo)

	var planID string
	step("Hati: Create Development Plan", func() (interface{}, error) {
		return s.callTool(ctx, "plan_create", map[string]interface{}{
			"title":       planTitle,
			"description": planDesc,
			"risk_level":  "medium",
		})
	})

	planResult := results["Hati: Create Development Plan"]
	if planMap, ok := planResult.(map[string]interface{}); ok {
		if id, ok := planMap["id"].(string); ok {
			planID = id
		}
	}

	phaseTemplates := []scanner.PhaseTemplate{}
	if analysis != nil {
		phaseTemplates = scanner.GeneratePhasesAndTasks(analysis)
	} else {
		phaseTemplates = []scanner.PhaseTemplate{
			{Name: "Setup", Tasks: []scanner.TaskTemplate{
				{Title: "Initialize project", Description: "Initialize project", Priority: 5},
				{Title: "Setup dependencies", Description: "Setup dependencies", Priority: 5},
			}},
			{Name: "Backend", Tasks: []scanner.TaskTemplate{
				{Title: "Implement API endpoints", Description: "Implement API endpoints", Priority: 5},
				{Title: "Setup database", Description: "Setup database", Priority: 5},
			}},
			{Name: "Features", Tasks: []scanner.TaskTemplate{
				{Title: "Implement core features", Description: "Implement core features", Priority: 5},
				{Title: "Add business logic", Description: "Add business logic", Priority: 5},
			}},
			{Name: "Testing", Tasks: []scanner.TaskTemplate{
				{Title: "Write unit tests", Description: "Write unit tests", Priority: 5},
				{Title: "Integration testing", Description: "Integration testing", Priority: 5},
			}},
			{Name: "Deployment", Tasks: []scanner.TaskTemplate{
				{Title: "Setup CI/CD", Description: "Setup CI/CD", Priority: 5},
				{Title: "Deploy to production", Description: "Deploy to production", Priority: 5},
			}},
		}
	}

	phaseIDs := []string{}
	taskCount := 0
	for i, template := range phaseTemplates {
		phaseResult, err := s.callTool(ctx, "phase_create", map[string]interface{}{
			"plan_id":   planID,
			"title":     template.Name,
			"order_num": i,
		})
		if err == nil {
			if phaseMap, ok := phaseResult.(map[string]interface{}); ok {
				if id, ok := phaseMap["id"].(string); ok {
					phaseIDs = append(phaseIDs, id)
				}
			}
		}

		for _, task := range template.Tasks {
			taskCount++
			s.callTool(ctx, "task_create", map[string]interface{}{
				"phase_id":    phaseIDs[len(phaseIDs)-1],
				"title":       task.Title,
				"description": task.Description,
				"priority":    task.Priority,
			})
		}
	}

	step("Hati: Store Plan Context", func() (interface{}, error) {
		return s.callTool(ctx, "mem_save", map[string]interface{}{
			"title": "Plan Created: " + planTitle,
			"type":  "decision",
			"what":  fmt.Sprintf("Development plan created with %d phases and tasks", len(phaseTemplates)),
			"where": planID,
			"why":   "Initial planning complete, ready for development",
		})
	})

	step("Skoll: List Available Agents", func() (interface{}, error) {
		return s.callTool(ctx, "agent_list", map[string]interface{}{})
	})

	var availableAgents []map[string]string
	if agentResult, ok := results["Skoll: List Available Agents"].(map[string]interface{}); ok {
		if agents, ok := agentResult["agents"].([]interface{}); ok {
			for _, a := range agents {
				if agentMap, ok := a.(map[string]interface{}); ok {
					availableAgents = append(availableAgents, map[string]string{
						"name": agentMap["name"].(string),
						"type": agentMap["type"].(string),
					})
				}
			}
		}
	}

	recommendedTypes := []string{"backend", "frontend", "qa", "devops"}
	if hasFrontend(analysis) {
		recommendedTypes = []string{"backend", "frontend", "qa", "devops"}
	} else {
		recommendedTypes = []string{"backend", "qa", "devops"}
	}

	for _, agentType := range recommendedTypes {
		agentName := findAgentByType(agentType, availableAgents)
		if agentName != "" {
			step(fmt.Sprintf("Skoll: Register Agent %s", agentType), func() (interface{}, error) {
				return s.callTool(ctx, "agent_activate", map[string]interface{}{
					"agent_id": agentName,
				})
			})
		}
	}

	step("Tyr: Validate Project Dependencies", func() (interface{}, error) {
		stackLang := getStackLanguage(analysis)
		if stackLang != "" {
			return s.callTool(ctx, "pkg_check", map[string]interface{}{
				"name":      stackLang,
				"ecosystem": getEcosystem(stackLang),
			})
		}
		return map[string]interface{}{"status": "skipped", "reason": "no stack detected"}, nil
	})

	step("Tyr: Run SAST Scan", func() (interface{}, error) {
		return s.callTool(ctx, "sast_run", map[string]interface{}{
			"path": params.ProjectPath,
		})
	})

	step("Tyr: Validate Standards", func() (interface{}, error) {
		return s.callTool(ctx, "standard_run_all", map[string]interface{}{})
	})

	reviewQuestion := fmt.Sprintf("¿Apruebas este plan de desarrollo? (Stack: %s, %d fases)", stackInfo, len(phaseTemplates))

	step("Hati: Create Human Review", func() (interface{}, error) {
		return s.callTool(ctx, "human_review_create", map[string]interface{}{
			"review_type": "prd_approval",
			"entity_type": "plan",
			"entity_id":   planID,
			"question":    reviewQuestion,
		})
	})

	response := map[string]interface{}{
		"workflow":       "project_lifecycle",
		"status":         "completed",
		"project_name":   projectName,
		"project_path":   params.ProjectPath,
		"plan_id":        planID,
		"prd_id":         prdID,
		"phases":         len(phaseTemplates),
		"agents":         len(availableAgents),
		"stack_detected": analysis != nil,
		"steps":          steps,
		"auto_start":     params.AutoStart,
		"message":        "Project lifecycle complete. Plan created and validated. Human review required.",
	}

	if params.AutoStart {
		response["next_step"] = "rag continue --plan " + planID
	} else {
		response["next_step"] = "rag continue --plan " + planID + "  # After human approval"
	}

	return &Response{Result: response}, nil
}

func getEcosystem(language string) string {
	switch strings.ToLower(language) {
	case "go":
		return "go"
	case "javascript", "typescript", "node":
		return "npm"
	case "python":
		return "pypi"
	case "rust":
		return "cargo"
	case "java":
		return "maven"
	case "dotnet", "csharp":
		return "nuget"
	default:
		return "npm"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
