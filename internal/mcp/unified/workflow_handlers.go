package unified

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/andragon31/Ragnarok/internal/fenrir/scanner"
	rootmcp "github.com/andragon31/Ragnarok/internal/mcp"
)

// - [x] **Fase 3: Unified - Workflow Colaborativo**
// - [x] Tarea 3.1: Actualizar `workflow_prd_analyze` con el paso de validación.
// - [x] Tarea 3.2: Implementar la generación automática de tareas de revisión humana ante ambigüedades.

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

func getAgentIDFromResult(agentMap map[string]interface{}) (string, bool) {
	if id, ok := agentMap["agent_id"].(string); ok && id != "" {
		return id, true
	}
	if id, ok := agentMap["id"].(string); ok && id != "" {
		return id, true
	}
	return "", false
}

func (s *Server) handleAnalyzeStackWithLLM(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath  string              `json:"project_path"`
		PRDFile      string              `json:"prd_file,omitempty"`
		Requirements []map[string]string `json:"requirements,omitempty"`
		LLMResponse  string              `json:"llm_response,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	if params.LLMResponse != "" {
		llmResult, parseErr := scanner.ParseLLMAnalysisResponse(params.LLMResponse)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse LLM response: %w", parseErr)
		}

		agents := []map[string]string{}
		for _, agent := range llmResult.RecommendedAgents {
			agents = append(agents, map[string]string{
				"name":  agent["name"],
				"type":  agent["type"],
				"role":  agent["role"],
				"scope": agent["scope"],
			})
		}

		return &Response{Result: map[string]interface{}{
			"agents":                agents,
			"reasoning":             llmResult.Reasoning,
			"complexity":            llmResult.Complexity,
			"has_frontend":          llmResult.HasFrontend,
			"has_backend":           llmResult.HasBackend,
			"has_security":          llmResult.HasSecurity,
			"has_devops":            llmResult.HasDevops,
			"has_qa":                llmResult.HasQA,
			"has_docs":              llmResult.HasDocs,
			"recommendation_source": "llm",
		}}, nil
	}

	analysis, err := s.parseProjectAnalysisFromPath(params.ProjectPath)
	if err != nil {
		analysis = &scanner.ProjectAnalysis{
			Architecture: &scanner.ArchitectureInfo{},
			Stack:        &scanner.StackInfo{},
		}
	}

	if params.PRDFile != "" {
		prdResult, err := s.callTool(ctx, "prd_parse", map[string]interface{}{"file_path": params.PRDFile})
		if err == nil {
			if prdMap, ok := prdResult.(map[string]interface{}); ok {
				if reqs, ok := prdMap["requirements"].([]map[string]string); ok {
					params.Requirements = reqs
				} else if reqsRaw, ok := prdMap["requirements"].([]interface{}); ok {
					for _, r := range reqsRaw {
						if reqMap, ok := r.(map[string]interface{}); ok {
							req := map[string]string{}
							if t, ok := reqMap["type"].(string); ok {
								req["type"] = t
							}
							if title, ok := reqMap["title"].(string); ok {
								req["title"] = title
							}
							if id, ok := reqMap["id"].(string); ok {
								req["id"] = id
							}
							params.Requirements = append(params.Requirements, req)
						}
					}
				}
			}
		}
	}

	prompt := scanner.GenerateLLMAnalysisPrompt(analysis, params.Requirements)

	return &Response{Result: map[string]interface{}{
		"llm_prompt":         prompt,
		"instructions":       "Execute this prompt with your LLM and pass the response to this tool again with the llm_response parameter set to the LLM's JSON response.",
		"project_path":       params.ProjectPath,
		"has_analysis":       analysis != nil && analysis.Stack != nil && analysis.Stack.Language != "",
		"requirements_count": len(params.Requirements),
	}}, nil
}

func (s *Server) parseProjectAnalysisFromPath(projectPath string) (*scanner.ProjectAnalysis, error) {
	analyzer := scanner.NewProjectAnalyzer(projectPath)
	return analyzer.Analyze()
}

func (s *Server) handleWorkflowProjectBootstrap(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
		ProjectName string `json:"project_name,omitempty"`
		PRDDFile    string `json:"prd_file,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := statusSuccess
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
		"status":   statusCompleted,
		"steps":    steps,
		"results":  results,
	}}, nil
}

func (s *Server) handleWorkflowPRDAnalyze(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PRDFile     string `json:"prd_file"`
		ProjectPath string `json:"project_path"`
		PlanTitle   string `json:"plan_title,omitempty"`
		LLMResponse string `json:"llm_response,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := statusSuccess
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

	// 1. Scan and parse
	step("project_scan", func() (interface{}, error) {
		return s.callTool(ctx, "project_scan", map[string]interface{}{"path": params.ProjectPath})
	})

	scanResult := results["project_scan"]
	analysis, _ := parseProjectAnalysis(scanResult)

	step("prd_parse", func() (interface{}, error) {
		return s.callTool(ctx, "prd_parse", map[string]interface{}{"file_path": params.PRDFile})
	})

	prdResult := results["prd_parse"]
	prdID, contextualSummary := extractPRDMetadata(prdResult)

	step("prd_validate", func() (interface{}, error) {
		return s.callTool(ctx, "prd_validate", map[string]interface{}{"prd_id": prdID})
	})

	// 2. Plan generation
	projectName := params.PlanTitle
	if projectName == "" {
		projectName = "Project"
	}

	step(logGeneratePlan, func() (interface{}, error) {
		return s.callTool(ctx, "plan_create_from_prd", map[string]interface{}{
			"prd_id":       prdID,
			"project_path": params.ProjectPath,
			"title":        projectName + logExecutionPlan,
		})
	})

	planResult := results[logGeneratePlan]
	planID, _ := extractPlanID(planResult)

	// 3. Agent and team setup
	if analysis == nil {
		analysis = &scanner.ProjectAnalysis{
			Architecture: &scanner.ArchitectureInfo{},
			Stack:        &scanner.StackInfo{},
		}
	}

	requirements := extractRequirements(prdResult)
	requirementsCount := len(requirements)
	var createdAgentIDs []string
	var teamID string
	var pendingLLM bool
	var llmPrompt, llmInstructions string

	step("Skoll: Create Agent Team", func() (interface{}, error) {
		var tMap map[string]string
		var err error

		if params.LLMResponse != "" {
			ids, tid, tMapRet, err := s.setupTeam(ctx, analysis, requirements, params.ProjectPath, projectName, params.LLMResponse)
			createdAgentIDs = ids
			teamID = tid
			tMap = tMapRet
			if err == nil && len(createdAgentIDs) > 0 {
				if teamID == "" && len(createdAgentIDs) > 0 {
					teamResult, _ := s.callTool(ctx, "team_create", map[string]interface{}{
						"name":         projectName + " Team",
						"project_path": params.ProjectPath,
						"agent_ids":    createdAgentIDs,
					})
					if teamMap, ok := teamResult.(map[string]interface{}); ok {
						teamID, _ = teamMap["team_id"].(string)
					}
				}
				return map[string]interface{}{"team_id": teamID, "agents_created": len(createdAgentIDs), "recommendation_source": "llm"}, nil
			}
			if err != nil {
				return nil, err
			}
		}

		ids, tid, tMapRet, err := s.setupTeam(ctx, analysis, requirements, params.ProjectPath, projectName)
		createdAgentIDs = ids
		teamID = tid
		tMap = tMapRet
		result := map[string]interface{}{"team_id": teamID, "agents_created": len(createdAgentIDs)}
		if source, hasSource := tMap["recommendation_source"]; hasSource && source == "pending_llm" {
			pendingLLM = true
			llmPrompt = fmt.Sprintf("%v", tMap["llm_prompt"])
			llmInstructions = fmt.Sprintf("%v", tMap["llm_instructions"])
			result["recommendation_source"] = "pending_llm"
			result["llm_prompt"] = llmPrompt
			result["llm_instructions"] = llmInstructions
			result["has_analysis"] = tMap["has_analysis"]
			result["requirements_count"] = requirementsCount
			result["project_name"] = tMap["project_name"]
			result["project_path"] = tMap["project_path"]
		} else if source, hasSource := tMap["recommendation_source"]; hasSource {
			result["recommendation_source"] = source
		}
		return result, err
	})

	if pendingLLM {
		pendingDisplayMessage := fmt.Sprintf(`📋 **Análisis LLM Requerido:**

El sistema necesita que ejecutes el análisis LLM para determinar los agentes especializados requeridos.

**Proyecto**: %s
**Path**: %s
**Requisitos detectados**: %d

**Próximo paso**: Ejecuta el prompt LLM y pasa la respuesta a analyze_stack_with_llm para crear los agentes.`,
			projectName, params.ProjectPath, requirementsCount)

		return &Response{Result: map[string]interface{}{
			"workflow":              "prd_analyze",
			"status":                "pending_llm",
			"prd_id":                prdID,
			"plan_id":               planID,
			"recommendation_source": "pending_llm",
			"llm_prompt":            llmPrompt,
			"llm_instructions":      llmInstructions,
			"has_analysis":          fmt.Sprintf("%v", requirementsCount),
			"project_name":          projectName,
			"project_path":          params.ProjectPath,
			"stack_detected":        analysis != nil,
			"stack":                 getStackFromAnalysis(analysis),
			"steps":                 steps,
			"results":               results,
			"message":               pendingDisplayMessage,
			"display_to_user":       pendingDisplayMessage,
		}}, nil
	}

	step("human_review_create", func() (interface{}, error) {
		question := "¿Apruebas este plan de desarrollo?"
		if contextualSummary != "" {
			question = fmt.Sprintf("Análisis de Visión:\n%s\n\n%s", contextualSummary, question)
		}
		return s.callTool(ctx, "human_review_create", map[string]interface{}{
			"review_type": "prd_approval",
			"entity_type": "plan",
			"entity_id":   planID,
			"question":    question,
		})
	})

	if projectName == "" {
		projectName = "Project"
	}

	agentNames := "backend-agent, docs-agent"
	if len(createdAgentIDs) > 2 {
		agentNames = "backend-agent, qa-agent, devops-agent, docs-agent"
	}

	displayMessage := fmt.Sprintf(`📋 **Resumen del Análisis PRD:**
- **Plan ID**: %s
- **PRD ID**: %s
- **Equipo Creado**: %s (ID: %s)
- **Agentes Creados**: %d (%s)
- **Stack Detectado**: %s
- **Fases y Tareas**: Generadas desde el PRD

**Próximo paso**: Revisar el plan y aprobar para iniciar el desarrollo.`, planID, prdID, projectName+" Team", teamID, len(createdAgentIDs), agentNames, getStackFromAnalysis(analysis))

	return &Response{Result: map[string]interface{}{
		"workflow":        "prd_analyze",
		"status":          statusCompleted,
		"prd_id":          prdID,
		"plan_id":         planID,
		"team_id":         teamID,
		"agents_created":  len(createdAgentIDs),
		"agent_ids":       createdAgentIDs,
		"stack_detected":  analysis != nil,
		"stack":           getStackFromAnalysis(analysis),
		"steps":           steps,
		"results":         results,
		"message":         displayMessage,
		"display_to_user": displayMessage,
	}}, nil
}

func (s *Server) handleWorkflowTeamSetupFromPRD(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PRDFile     string `json:"prd_file"`
		ProjectPath string `json:"project_path,omitempty"`
		TeamName    string `json:"team_name,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := statusSuccess
		if err != nil {
			status = "error"
			steps = append(steps, WorkflowStep{Name: name, Status: status, Error: err.Error()})
		} else {
			steps = append(steps, WorkflowStep{Name: name, Status: status, Output: out})
		}
		results[name] = out
	}

	if params.TeamName == "" {
		params.TeamName = "ProjectTeam"
	}

	// 1. Scan project if path provided
	var analysis *scanner.ProjectAnalysis
	if params.ProjectPath != "" {
		step("project_scan", func() (interface{}, error) {
			return s.callTool(ctx, "project_scan", map[string]interface{}{"path": params.ProjectPath})
		})
		scanResult := results["project_scan"]
		analysis, _ = parseProjectAnalysis(scanResult)
	}

	// 2. Parse PRD
	step("prd_parse", func() (interface{}, error) {
		return s.callTool(ctx, "prd_parse", map[string]interface{}{"file_path": params.PRDFile})
	})

	prdResult := results["prd_parse"]

	// 3. Get recommendations (default to backend/docs if no analysis)
	var requirements []map[string]string
	if prdResult != nil {
		if prdMap, ok := prdResult.(map[string]interface{}); ok {
			if reqs, ok := prdMap["requirements"].([]map[string]string); ok {
				requirements = reqs
			} else if reqsRaw, ok := prdMap["requirements"].([]interface{}); ok {
				for _, r := range reqsRaw {
					if reqMap, ok := r.(map[string]interface{}); ok {
						req := map[string]string{}
						if t, ok := reqMap["type"].(string); ok {
							req["type"] = t
						}
						if title, ok := reqMap["title"].(string); ok {
							req["title"] = title
						}
						if id, ok := reqMap["id"].(string); ok {
							req["id"] = id
						}
						requirements = append(requirements, req)
					}
				}
			}
		}
	}

	if analysis == nil {
		analysis = &scanner.ProjectAnalysis{
			Architecture: &scanner.ArchitectureInfo{},
			Stack:        &scanner.StackInfo{},
		}
	}
	recommendations := scanner.GetRecommendedAgents(analysis, requirements)

	// 4. Create Agents
	agentIDs := []string{}
	for _, rec := range recommendations {
		recName := rec["name"]
		recType := rec["type"]

		agentResult, err := s.callTool(ctx, "agent_create", map[string]interface{}{
			"name":       recName,
			"agent_type": recType,
			"skills":     []string{rec["role"]},
		})

		if err != nil {
			steps = append(steps, WorkflowStep{Name: "agent_create:" + recName, Status: "error", Error: err.Error()})
			continue
		}

		if agentMap, ok := agentResult.(map[string]interface{}); ok {
			if id, ok := getAgentIDFromResult(agentMap); ok {
				agentIDs = append(agentIDs, id)
				steps = append(steps, WorkflowStep{Name: "agent_create:" + recName, Status: "success", Output: agentMap})
			}
		}
	}

	// 5. Create Team
	step("team_create", func() (interface{}, error) {
		return s.callTool(ctx, "team_create", map[string]interface{}{
			"name":         params.TeamName,
			"project_path": params.ProjectPath,
			"agent_ids":    agentIDs,
		})
	})

	teamResult := results["team_create"]
	teamID := ""
	if teamMap, ok := teamResult.(map[string]interface{}); ok {
		if id, ok := teamMap["team_id"].(string); ok {
			teamID = id
		}
	}

	if params.ProjectPath != "" {
		step("Fenrir: Bootstrap Project Structure", func() (interface{}, error) {
			return s.callTool(ctx, "project_bootstrap", map[string]interface{}{
				"project_path": params.ProjectPath,
			})
		})
	}

	return &Response{Result: map[string]interface{}{
		"workflow": "team_setup_from_prd",
		"status":   statusCompleted,
		"team_id":  teamID,
		"agents":   agentIDs,
		"steps":    steps,
		"results":  results,
		"message":  fmt.Sprintf("Team '%s' created with %d agents based on PRD analysis.", params.TeamName, len(agentIDs)),
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
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := statusSuccess
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
		return nil, fmt.Errorf(errFailedParseParams, err)
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
		"status":          statusCompleted,
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
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := statusSuccess
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
		"status":   statusCompleted,
		"steps":    steps,
		"results":  results,
		"message":  "Session initialized successfully with memory and context.",
	}}, nil
}

func (s *Server) handleWorkflowCheckpointCreate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID      string `json:"plan_id"`
		PhaseID     string `json:"phase_id,omitempty"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := statusSuccess
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

	step("tyr_snapshot", func() (interface{}, error) {
		return s.callTool(ctx, "tyr_snapshot", map[string]interface{}{
			"checkpoint_type": "manual_checkpoint",
		})
	})

	snapshot := results["tyr_snapshot"]
	qualityMsg := "No quality data available."
	if sm, ok := snapshot.(map[string]interface{}); ok {
		score := sm["quality_score"].(float64)
		findings := 0
		if f, ok := sm["findings"].(map[string]interface{}); ok {
			findings = int(f["total_active"].(float64))
		}
		qualityMsg = fmt.Sprintf("Calidad: %.1f/100 | Hallazgos activos: %d", qualityScore(score), findings)
	}

	step("human_review_create", func() (interface{}, error) {
		return s.callTool(ctx, "human_review_create", map[string]interface{}{
			"review_type": "checkpoint_approval",
			"entity_type": "checkpoint",
			"entity_id":   params.PlanID,
			"question":    fmt.Sprintf("¿Aprobar checkpoint? (%s)\nDescripción: %s", qualityMsg, params.Description),
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

	// Tool diagnostics
	toolCount := len(s.tools)
	schemaCount := 0
	descCount := 0
	ghostHandlers := 0

	for _, tool := range s.tools {
		// Check schema
		if tool.InputSchema != nil && len(tool.InputSchema) > 2 {
			schemaCount++
		}

		// Check description length
		if len(tool.Description) >= 80 {
			descCount++
		}

		// Check handler
		if _, ok := s.handlers[tool.Name]; !ok {
			ghostHandlers++
			issues = append(issues, fmt.Sprintf("ghost_handler: %s (no handler registered)", tool.Name))
		}
	}

	diagnostics := map[string]interface{}{
		"version":        s.serverVersion,
		"status":         "healthy",
		"database_stats": stats,
		"tools": map[string]interface{}{
			"total":           toolCount,
			"with_schema":     schemaCount,
			"with_desc_gt_80": descCount,
			"ghost_handlers":  ghostHandlers,
		},
	}

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
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}

	step := func(name string, fn func() (interface{}, error)) {
		out, err := fn()
		status := statusSuccess
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
	recommendedAgents := scanner.GetRecommendedAgents(analysis, nil)

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

			taskParams := map[string]interface{}{
				"phase_id":    phaseID,
				"title":       taskTemplate.Title,
				"description": taskTemplate.Description,
				"priority":    taskTemplate.Priority,
				"milestone":   taskTemplate.Milestone,
			}
			if len(agentIDsForTask) > 0 {
				taskParams["agent_ids"] = agentIDsForTask
			}
			taskResult, err := s.callTool(ctx, "task_create", taskParams)
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
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	steps := []WorkflowStep{}
	taskExecutions := []map[string]interface{}{}
	completedCount := 0
	qualityChecks := []map[string]interface{}{}

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
		taskPhase := task["phase_title"]

		assignedAgentIDsRaw, hasAgents := task["assigned_agent_ids"].([]interface{})
		taskExecution := map[string]interface{}{
			"task_id": taskID,
			"title":   taskTitle,
			"phase":   taskPhase,
			"status":  "in_progress",
			"agents":  []string{},
		}

		steps = append(steps, WorkflowStep{Name: "task_start:" + taskTitle, Status: "in_progress", Output: map[string]interface{}{"task_id": taskID, "title": taskTitle, "phase": taskPhase}})

		var executedBy []string
		var executionID string

		if hasAgents && len(assignedAgentIDsRaw) > 0 {
			for _, agentIDRaw := range assignedAgentIDsRaw {
				agentID, ok := agentIDRaw.(string)
				if !ok || agentID == "" {
					continue
				}
				execResult, err := s.callTool(ctx, "task_execute", map[string]interface{}{
					"task_id":  taskID,
					"agent_id": agentID,
				})
				if err != nil {
					steps = append(steps, WorkflowStep{Name: "task_delegate:" + taskTitle, Status: "error", Error: err.Error()})
					taskExecution["status"] = "error"
				} else {
					executedBy = append(executedBy, agentID)
					if execResultMap, ok := execResult.(map[string]interface{}); ok {
						if eid, ok := execResultMap["execution_id"].(string); ok {
							executionID = eid
						}
					}
					s.callTool(ctx, "task_update", map[string]interface{}{
						"task_id": taskID,
						"status":  "in_progress",
						"notes":   "Delegated to " + agentID + " - awaiting completion",
					})
					steps = append(steps, WorkflowStep{Name: "task_delegate:" + taskTitle, Status: "in_progress", Output: execResult})
				}
			}
		} else {
			assignedType, _ := task["assigned_agent_type"].(string)
			agentIDToUse := params.AgentID
			if agentIDToUse == "" {
				if assignedType != "" {
					availableAgents, err := s.callTool(ctx, "agent_specialized_list", map[string]interface{}{
						"agent_type": assignedType,
					})
					log.Printf("   [DEBUG] Looking for type=%s, result=%+v, err=%v", assignedType, availableAgents, err)
					if err == nil {
						if agentsMap, ok := availableAgents.(map[string]interface{}); ok {
							if agents, ok := agentsMap["agents"].([]map[string]interface{}); ok && len(agents) > 0 {
								agentIDToUse, _ = agents[0]["id"].(string)
								log.Printf("   [DEBUG] Found agent of type %s: %s", assignedType, agentIDToUse)
							}
						}
					}
				}
				if agentIDToUse == "" {
					log.Printf("   [DEBUG] No agent found, trying backend fallback")
					availableAgents, err := s.callTool(ctx, "agent_specialized_list", map[string]interface{}{
						"agent_type": "backend",
					})
					log.Printf("   [DEBUG] Backend fallback result=%+v, err=%v", availableAgents, err)
					if err == nil {
						if agentsMap, ok := availableAgents.(map[string]interface{}); ok {
							if agents, ok := agentsMap["agents"].([]map[string]interface{}); ok && len(agents) > 0 {
								agentIDToUse, _ = agents[0]["id"].(string)
								log.Printf("   [DEBUG] Using backend fallback agent: %s", agentIDToUse)
							}
						}
					}
				}
			}

			if agentIDToUse != "" {
				execResult, err := s.callTool(ctx, "task_execute", map[string]interface{}{
					"task_id":  taskID,
					"agent_id": agentIDToUse,
				})
				if err != nil {
					steps = append(steps, WorkflowStep{Name: "task_execute:" + taskTitle, Status: "error", Error: err.Error()})
					taskExecution["status"] = "error"
				} else {
					executedBy = append(executedBy, agentIDToUse)
					taskExecution["status"] = "in_progress"
					if execResultMap, ok := execResult.(map[string]interface{}); ok {
						if eid, ok := execResultMap["execution_id"].(string); ok {
							executionID = eid
						}
					}
					s.callTool(ctx, "task_update", map[string]interface{}{
						"task_id": taskID,
						"status":  "in_progress",
						"notes":   "Assigned to " + agentIDToUse + " - execution_id: " + executionID + " - awaiting completion",
					})
					steps = append(steps, WorkflowStep{Name: "task_execute:" + taskTitle, Status: "in_progress", Output: map[string]interface{}{
						"execution_id": executionID,
						"task_id":      taskID,
						"agent_id":     agentIDToUse,
						"message":      "Tarea asignada. OpenCode debe ejecutar y llamar task_complete.",
					}})
				}
			} else {
				s.callTool(ctx, "task_update", map[string]interface{}{
					"task_id": taskID,
					"status":  "in_progress",
					"notes":   "No agent assigned - requires manual assignment",
				})
				steps = append(steps, WorkflowStep{Name: "task_start:" + taskTitle, Status: "pending_agent", Output: map[string]interface{}{"task_id": taskID, "message": "No agent assigned"}})
				taskExecution["status"] = "pending_agent"
			}
		}

		taskExecution["agents"] = executedBy
		if executionID != "" {
			taskExecution["execution_id"] = executionID
		}

		taskExecution["agents"] = executedBy

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

		completedCount++
		taskExecutions = append(taskExecutions, taskExecution)

		// FENRIR: Save task assignment to memory (OpenCode must complete the actual work)
		agentList := "agente(s)"
		if len(executedBy) > 0 {
			agentList = fmt.Sprintf("agente(s) %v", executedBy)
		}
		memSaveResult, _ := s.callTool(ctx, "mem_save", map[string]interface{}{
			"title": "Tarea iniciada: " + taskTitle,
			"type":  "task_started",
			"what":  fmt.Sprintf("Tarea '%s' asignada a %s - ejecución en curso. OpenCode debe completar esta tarea.", taskTitle, agentList),
			"where": taskPhase,
			"why":   "Asignación de tarea en plan de desarrollo",
		})
		if memSaveResult != nil {
			steps = append(steps, WorkflowStep{Name: "Fenrir: MemSave", Status: "success"})
		}

		// TYR: Run quality check after task
		qualityResult, _ := s.callTool(ctx, "tyr_snapshot", map[string]interface{}{})
		if qualityMap, ok := qualityResult.(map[string]interface{}); ok {
			qualityChecks = append(qualityChecks, qualityMap)
		}

		if !params.AutoContinue {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	planProgress, _ := s.callTool(ctx, "plan_progress", map[string]interface{}{"plan_id": params.PlanID})

	// FENRIR: Save overall session summary
	s.callTool(ctx, "mem_save", map[string]interface{}{
		"title": fmt.Sprintf("Sesión de desarrollo completada: %d tareas", completedCount),
		"type":  "session_summary",
		"what":  fmt.Sprintf("Se ejecutaron %d tareas en el plan %s", completedCount, params.PlanID),
		"why":   "Ciclo de desarrollo",
	})

	displayMessage := fmt.Sprintf(`📋 **Resumen de Ejecución - Plan %s:**

**Tareas procesadas:** %d

**Detalle de ejecuciones:**
%s

**Checks de calidad:** %d realizados

**Progreso del plan:**
%s

**Nota para OpenCode:** Cada tarea tiene un execution_id. Usa task_complete(execution_id, result="descripción del trabajo realizado") para marcar como completada.

**Próximo paso:** Ejecutar las tareas con OpenCode y llamar task_complete para cada una.`,
		params.PlanID, completedCount,
		func() string {
			out := ""
			for _, t := range taskExecutions {
				status := t["status"]
				title := t["title"]
				agents := t["agents"]
				execID, _ := t["execution_id"].(string)
				if execID != "" {
					out += fmt.Sprintf("  • %s → %v (%s) [exec_id: %s]\n", title, agents, status, execID)
				} else {
					out += fmt.Sprintf("  • %s → %v (%s)\n", title, agents, status)
				}
			}
			return out
		}(),
		len(qualityChecks),
		planProgress)

	return &Response{Result: map[string]interface{}{
		"workflow":        "plan_develop_v2",
		"status":          statusCompleted,
		"tasks_executed":  completedCount,
		"task_executions": taskExecutions,
		"quality_checks":  qualityChecks,
		"progress":        planProgress,
		"steps":           steps,
		"display_to_user": displayMessage,
		"message":         displayMessage,
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

	analysis := &scanner.ProjectAnalysis{
		Architecture: &scanner.ArchitectureInfo{},
		Stack:        &scanner.StackInfo{},
	}

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
		PRDFile     string `json:"prd_file,omitempty"`
		ProjectPath string `json:"project_path"`
		Title       string `json:"title,omitempty"`
		AutoStart   bool   `json:"auto_start"`
		LLMResponse string `json:"llm_response,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ProjectPath == "" {
		params.ProjectPath = "."
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}
	startTime := time.Now()

	shouldBreak := func() bool {
		return time.Since(startTime) > 40*time.Second
	}

	step := func(name string, fn func() (interface{}, error)) {
		if shouldBreak() {
			return
		}
		out, err := fn()
		status := statusSuccess
		if err != nil {
			status = "error"
			steps = append(steps, WorkflowStep{Name: name, Status: status, Error: err.Error()})
		} else {
			steps = append(steps, WorkflowStep{Name: name, Status: statusSuccess, Output: out})
		}
		results[name] = out
	}

	// 1. Fenrir: Analysis
	step("Fenrir: Analyze Project Context", func() (interface{}, error) {
		return s.callTool(ctx, "project_scan", map[string]interface{}{"path": params.ProjectPath})
	})

	scanResult := results["Fenrir: Analyze Project Context"]
	analysis, _ := parseProjectAnalysis(scanResult)

	projectName := params.Title
	if projectName == "" && analysis != nil {
		projectName = analysis.Name
	}
	if projectName == "" {
		projectName = "Unnamed Project"
	}

	// 2. Fenrir: PRD Analysis
	var prdID string
	if params.PRDFile != "" && !shouldBreak() {
		if id, ok := s.getExistingPRD(ctx, params.PRDFile); ok {
			prdID = id
			steps = append(steps, WorkflowStep{Name: logParsePRD, Status: statusSuccess, Output: "Resumed from memory"})
		} else {
			step(logParsePRD, func() (interface{}, error) {
				return s.callTool(ctx, "prd_parse", map[string]interface{}{"file_path": params.PRDFile})
			})

			prdResult := results[logParsePRD]
			if prdMap, ok := prdResult.(map[string]interface{}); ok {
				if id, ok := prdMap["prd_id"].(string); ok {
					prdID = id
					if reqs, ok := prdMap["requirements"].([]interface{}); ok {
						s.savePRDRequirements(ctx, prdID, params.PRDFile, projectName, reqs)
					}
				}
			}
		}
	}

	// 3. Skoll: Create Specialized Agent Team
	var agentIDs []string
	var teamID string
	typeToAgentID := make(map[string]string)
	var pendingLLM bool
	var llmPrompt, llmInstructions string
	if !shouldBreak() {
		if ids, ok := s.getExistingTeam(ctx, projectName+" Team"); ok {
			agentIDs = ids
			steps = append(steps, WorkflowStep{Name: "Skoll: Create Specialized Agent Team", Status: statusSuccess, Output: "Resumed from existing team"})
		} else {
			step("Skoll: Create Specialized Agent Team", func() (interface{}, error) {
				prdResult := results[logParsePRD]
				requirements := extractRequirements(prdResult)

				var ids []string
				var tid string
				var tMap map[string]string
				var err error

				if params.LLMResponse != "" {
					ids, tid, tMap, err = s.setupTeam(ctx, analysis, requirements, params.ProjectPath, projectName, params.LLMResponse)
					if err == nil && len(ids) > 0 {
						agentIDs = ids
						teamID = tid
						typeToAgentID = tMap
						if teamID == "" && len(agentIDs) > 0 {
							teamResult, _ := s.callTool(ctx, "team_create", map[string]interface{}{
								"name":         projectName + " Team",
								"project_path": params.ProjectPath,
								"agent_ids":    agentIDs,
							})
							if teamMap, ok := teamResult.(map[string]interface{}); ok {
								teamID, _ = teamMap["team_id"].(string)
							}
						}
						return map[string]interface{}{"team_id": teamID, "agent_ids": agentIDs, "recommendation_source": "llm"}, nil
					}
					if err != nil {
						return nil, err
					}
				}

				ids, tid, tMap, err = s.setupTeam(ctx, analysis, requirements, params.ProjectPath, projectName)
				agentIDs = ids
				teamID = tid
				typeToAgentID = tMap
				if source, hasSource := tMap["recommendation_source"]; hasSource && source == "pending_llm" {
					pendingLLM = true
					llmPrompt = fmt.Sprintf("%v", tMap["llm_prompt"])
					llmInstructions = fmt.Sprintf("%v", tMap["llm_instructions"])
				}
				return map[string]interface{}{"team_id": teamID, "agent_ids": agentIDs}, err
			})
		}
	}

	if pendingLLM {
		pendingDisplayMessage := fmt.Sprintf(`📋 **Análisis LLM Requerido:**

El sistema necesita que ejecutes el análisis LLM para determinar los agentes especializados requeridos.

**Proyecto**: %s
**Path**: %s

**Próximo paso**: Ejecuta el prompt LLM y pasa la respuesta a analyze_stack_with_llm para crear los agentes.`,
			projectName, params.ProjectPath)

		return &Response{Result: map[string]interface{}{
			"workflow":              "project_lifecycle",
			"status":                "pending_llm",
			"prd_id":                prdID,
			"recommendation_source": "pending_llm",
			"llm_prompt":            llmPrompt,
			"llm_instructions":      llmInstructions,
			"project_name":          projectName,
			"project_path":          params.ProjectPath,
			"stack_detected":        analysis != nil,
			"steps":                 steps,
			"results":               results,
			"message":               pendingDisplayMessage,
			"display_to_user":       pendingDisplayMessage,
		}}, nil
	}

	// 4. Tyr: Quality Scan
	if !shouldBreak() {
		step("Tyr: Run Security and Quality Baseline", func() (interface{}, error) {
			return s.callTool(ctx, "sast_run", map[string]interface{}{"path": params.ProjectPath})
		})
	}

	// 5. Hati: Planning
	var planID string
	planTitle := projectName + logExecutionPlan
	if !shouldBreak() {
		if id, ok := s.getExistingPlan(ctx, planTitle); ok {
			planID = id
			steps = append(steps, WorkflowStep{Name: logGeneratePlan, Status: statusSuccess, Output: "Resumed from existing plan"})
		} else if prdID != "" {
			step(logGeneratePlan, func() (interface{}, error) {
				return s.callTool(ctx, "plan_create_from_prd", map[string]interface{}{
					"prd_id":       prdID,
					"project_path": params.ProjectPath,
					"title":        planTitle,
				})
			})
			planResult := results[logGeneratePlan]
			planID, _ = extractPlanID(planResult)
		}
	}

	// 5b. Fenrir: Bootstrap
	if !shouldBreak() && params.ProjectPath != "" {
		step("Fenrir: Bootstrap Project Structure", func() (interface{}, error) {
			return s.callTool(ctx, "project_bootstrap", map[string]interface{}{
				"project_path": params.ProjectPath,
			})
		})
	}

	// 6. Phases and Tasks creation
	taskCount := 0
	assignmentCount := 0
	if planID != "" && !shouldBreak() {
		existingPhases, _ := s.callTool(ctx, "plan_progress", map[string]interface{}{"plan_id": planID})
		hasData := false
		if epMap, ok := existingPhases.(map[string]interface{}); ok {
			if count, ok := epMap["total_tasks"].(int); ok && count > 0 {
				hasData = true
				taskCount = count
			}
		}

		if !hasData {
			phaseTemplates := scanner.GeneratePhasesAndTasks(analysis)
			for i, phaseTpl := range phaseTemplates {
				if shouldBreak() {
					break
				}
				phaseResult, pErr := s.callTool(ctx, "phase_create", map[string]interface{}{
					"plan_id":   planID,
					"title":     phaseTpl.Name,
					"order_num": i,
				})

				if pErr == nil {
					if phaseMap, ok := phaseResult.(map[string]interface{}); ok {
						if phaseID, ok := phaseMap["id"].(string); ok {
							for _, taskTpl := range phaseTpl.Tasks {
								taskAgentIDs := []string{}
								for _, tType := range taskTpl.AgentTypes {
									if aID, ok := typeToAgentID[tType]; ok {
										taskAgentIDs = append(taskAgentIDs, aID)
									}
								}
								s.callTool(ctx, "task_create", map[string]interface{}{
									"phase_id":    phaseID,
									"title":       taskTpl.Title,
									"description": taskTpl.Description,
									"agent_ids":   taskAgentIDs,
								})
								taskCount++
								if len(taskAgentIDs) > 0 {
									assignmentCount++
								}
							}
						}
					}
				}
			}
		}
	}

	status := statusCompleted
	if shouldBreak() {
		status = "timeout_partial"
	}

	agentInfo := []string{}
	for t, id := range typeToAgentID {
		agentInfo = append(agentInfo, fmt.Sprintf("%s-agent (ID: %s)", t, id))
	}

	displayMessage := fmt.Sprintf("✅ Proyecto orquestado: %s. %d tareas creadas/encontradas.", projectName, taskCount)
	response := map[string]interface{}{
		"workflow":        "project_lifecycle",
		"status":          status,
		"project_name":    projectName,
		"plan_id":         planID,
		"task_count":      taskCount,
		"agents":          agentInfo,
		"display_to_user": displayMessage,
	}

	if params.AutoStart && planID != "" && status == statusCompleted {
		response["next_step"] = "rag continue --plan " + planID
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
func qualityScore(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0.0
	}
}

func (s *Server) handleWorkflowOnboardExisting(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ProjectPath string `json:"project_path"`
		Goal        string `json:"goal,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ProjectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}

	steps := []WorkflowStep{}
	results := map[string]interface{}{}
	startTime := time.Now()

	shouldBreak := func() bool {
		return time.Since(startTime) > 50*time.Second
	}

	step := func(name string, fn func() (interface{}, error)) {
		if shouldBreak() {
			return
		}
		out, err := fn()
		status := "success"
		if err != nil {
			status = "error"
			steps = append(steps, WorkflowStep{Name: name, Status: status, Error: err.Error()})
		} else {
			steps = append(steps, WorkflowStep{Name: name, Status: statusSuccess, Output: out})
		}
		results[name] = out
	}

	log.Printf("🔄 Onboarding Existing Project: %s", params.ProjectPath)

	// 1. Scan project
	step("project_scan", func() (interface{}, error) {
		return s.callTool(ctx, "project_scan", map[string]interface{}{"path": params.ProjectPath})
	})

	// 2. Memory Summary
	step("mem_project_summary", func() (interface{}, error) {
		return s.callTool(ctx, "mem_project_summary", map[string]interface{}{"project_path": params.ProjectPath})
	})

	// 3. Spec Save (Architecture findings)
	scanResult := results["project_scan"]
	if scanResult != nil && !shouldBreak() {
		step("spec_save", func() (interface{}, error) {
			contentJSON, _ := json.MarshalIndent(scanResult, "", "  ")
			return s.callTool(ctx, "spec_save", map[string]interface{}{
				"name":        "Initial Architecture Scan",
				"description": "Auto-generated architecture documentation from project onboarding scan",
				"content":     fmt.Sprintf("## Architecture Findings\n\n```json\n%s\n```", string(contentJSON)),
			})
		})
	}

	// 4. Skill Generate
	step("skill_generate", func() (interface{}, error) {
		return s.callTool(ctx, "skill_generate", map[string]interface{}{"project_path": params.ProjectPath})
	})

	// 5. Rules Generate
	step("rules_generate", func() (interface{}, error) {
		return s.callTool(ctx, "rules_generate", map[string]interface{}{"project_path": params.ProjectPath})
	})

	// 6. Quality Bootstrap
	step("tyr_bootstrap", func() (interface{}, error) {
		return s.callTool(ctx, "tyr_bootstrap", map[string]interface{}{"project_path": params.ProjectPath})
	})

	// 7. Initial SAST
	step("sast_run", func() (interface{}, error) {
		return s.callTool(ctx, "sast_run", map[string]interface{}{"path": params.ProjectPath})
	})

	status := "completed"
	if shouldBreak() {
		status = "partial_timeout"
	}

	return &Response{Result: WorkflowResult{
		Workflow: "workflow_onboard_existing",
		Status:   status,
		Steps:    steps,
		Results:  results,
	}}, nil
}

// Helpers for handleWorkflowPRDAnalyze to reduce complexity

func extractPRDMetadata(prdResult interface{}) (string, string) {
	if prdMap, ok := prdResult.(map[string]interface{}); ok {
		id, _ := prdMap["prd_id"].(string)
		summary, _ := prdMap["contextual_summary"].(string)
		return id, summary
	}
	return "", ""
}

func extractPlanID(planResult interface{}) (string, bool) {
	if planMap, ok := planResult.(map[string]interface{}); ok {
		id, ok := planMap["plan_id"].(string)
		return id, ok
	}
	return "", false
}

func extractRequirements(prdResult interface{}) []map[string]string {
	var requirements []map[string]string
	if prdResult == nil {
		return requirements
	}

	prdMap, ok := prdResult.(map[string]interface{})
	if !ok {
		return requirements
	}

	if reqs, ok := prdMap["requirements"].([]map[string]string); ok {
		return reqs
	}

	if reqsRaw, ok := prdMap["requirements"].([]interface{}); ok {
		for _, r := range reqsRaw {
			if reqMap, ok := r.(map[string]interface{}); ok {
				req := map[string]string{}
				if t, ok := reqMap["type"].(string); ok {
					req["type"] = t
				}
				if title, ok := reqMap["title"].(string); ok {
					req["title"] = title
				}
				if id, ok := reqMap["id"].(string); ok {
					req["id"] = id
				}
				requirements = append(requirements, req)
			}
		}
	}
	return requirements
}

func (s *Server) setupTeam(ctx context.Context, analysis *scanner.ProjectAnalysis, requirements []map[string]string, projectPath, projectName string, llmResponse ...string) ([]string, string, map[string]string, error) {
	var createdAgentIDs []string
	var teamID string
	typeToAgentID := make(map[string]string)
	recommendations := scanner.GetRecommendedAgents(analysis, requirements)

	if len(llmResponse) > 0 && llmResponse[0] != "" {
		llmResult, err := scanner.ParseLLMAnalysisResponse(llmResponse[0])
		if err == nil && len(llmResult.RecommendedAgents) > 0 {
			recommendations = []map[string]string{}
			for _, agent := range llmResult.RecommendedAgents {
				recommendations = append(recommendations, map[string]string{
					"name":  agent["name"],
					"type":  agent["type"],
					"role":  agent["role"],
					"scope": agent["scope"],
				})
			}
			typeToAgentID["recommendation_source"] = "llm"
		}
	} else {
		analyzeResult, err := s.callTool(ctx, "analyze_stack_with_llm", map[string]interface{}{
			"project_path": projectPath,
			"requirements": requirements,
		})
		if err == nil {
			if resultMap, ok := analyzeResult.(map[string]interface{}); ok {
				if agents, ok := resultMap["agents"].([]map[string]string); ok && len(agents) > 0 {
					recommendations = agents
					typeToAgentID["recommendation_source"] = "llm"
				} else if prompt, hasPrompt := resultMap["llm_prompt"].(string); hasPrompt && prompt != "" {
					typeToAgentID["llm_prompt"] = prompt
					typeToAgentID["recommendation_source"] = "pending_llm"
					typeToAgentID["has_analysis"] = fmt.Sprintf("%v", resultMap["has_analysis"])
					typeToAgentID["requirements_count"] = fmt.Sprintf("%v", resultMap["requirements_count"])
					if instructions, hasInstr := resultMap["instructions"].(string); hasInstr {
						typeToAgentID["llm_instructions"] = instructions
					}
					if projectName == "" {
						projectName = "Project"
					}
					typeToAgentID["project_name"] = projectName
					typeToAgentID["project_path"] = projectPath
					return []string{}, "", typeToAgentID, nil
				}
			}
		}
	}

	for _, rec := range recommendations {
		recName := fmt.Sprintf("%v", rec["name"])
		recType := fmt.Sprintf("%v", rec["type"])
		recRole := fmt.Sprintf("%v", rec["role"])

		agentResult, err := s.callTool(ctx, "agent_create", map[string]interface{}{
			"name":       recName,
			"agent_type": recType,
			"skills":     []string{recRole},
		})

		if err == nil {
			if agentMap, ok := agentResult.(map[string]interface{}); ok {
				if id, ok := getAgentIDFromResult(agentMap); ok {
					createdAgentIDs = append(createdAgentIDs, id)
					typeToAgentID[recType] = id
				}
			}
		}
	}

	if len(createdAgentIDs) > 0 {
		teamResult, _ := s.callTool(ctx, "team_create", map[string]interface{}{
			"name":         projectName + " Team",
			"project_path": projectPath,
			"agent_ids":    createdAgentIDs,
		})
		if teamMap, ok := teamResult.(map[string]interface{}); ok {
			teamID, _ = teamMap["team_id"].(string)
		}
	}

	return createdAgentIDs, teamID, typeToAgentID, nil
}

func (s *Server) getExistingPRD(ctx context.Context, prdFile string) (string, bool) {
	existingPRDs, _ := s.callTool(ctx, "mem_search", map[string]interface{}{
		"query": prdFile,
		"type":  "prd_id",
	})

	if prdMap, ok := existingPRDs.(map[string]interface{}); ok {
		if matches, ok := prdMap["matches"].([]interface{}); ok && len(matches) > 0 {
			if first, ok := matches[0].(map[string]interface{}); ok {
				prdID, ok := first["content"].(string)
				return prdID, ok
			}
		}
	}
	return "", false
}

func (s *Server) savePRDRequirements(ctx context.Context, prdID, prdFile, projectName string, requirements []interface{}) {
	if len(requirements) == 0 {
		return
	}
	reqsJSON, _ := json.Marshal(requirements)
	s.callTool(ctx, "mem_save", map[string]interface{}{
		"content":  string(reqsJSON),
		"type":     "requirement_batch",
		"metadata": map[string]string{"project": projectName, "source": "prd_parse", "prd_file": prdFile},
	})
	s.callTool(ctx, "mem_save", map[string]interface{}{
		"content":  prdID,
		"type":     "prd_id",
		"metadata": map[string]string{"file": prdFile},
	})
}

func (s *Server) getExistingTeam(ctx context.Context, teamName string) ([]string, bool) {
	existingTeams, _ := s.callTool(ctx, "team_list", map[string]interface{}{})
	if teamMap, ok := existingTeams.(map[string]interface{}); ok {
		if teams, ok := teamMap["teams"].([]interface{}); ok {
			for _, t := range teams {
				if tm, ok := t.(map[string]interface{}); ok {
					if tm["name"] == teamName {
						var ids []string
						if agentIDs, ok := tm["agent_ids"].([]interface{}); ok {
							for _, id := range agentIDs {
								if sid, ok := id.(string); ok {
									ids = append(ids, sid)
								}
							}
						}
						return ids, true
					}
				}
			}
		}
	}
	return nil, false
}

func (s *Server) getExistingPlan(ctx context.Context, planTitle string) (string, bool) {
	existingPlans, _ := s.callTool(ctx, "plan_list", map[string]interface{}{})
	if planMap, ok := existingPlans.(map[string]interface{}); ok {
		if plans, ok := planMap["plans"].([]interface{}); ok {
			for _, p := range plans {
				if pm, ok := p.(map[string]interface{}); ok {
					if pm["title"] == planTitle {
						id, ok := pm["plan_id"].(string)
						return id, ok
					}
				}
			}
		}
	}
	return "", false
}
