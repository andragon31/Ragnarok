package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func columnExists(db *sql.DB, table, column string) bool {
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	rows, err := db.Query(query)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt_value interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk); err != nil {
			continue
		}
		if name == column {
			return true
		}
	}
	return false
}

type SpecializedAgent struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"agent_type"`
	Role          string    `json:"role"`
	Scope         string    `json:"scope"`
	Skills        []string  `json:"skills"`
	AllowedTools  []string  `json:"allowed_tools"`
	Capabilities  []string  `json:"capabilities"`
	Status        string    `json:"status"`
	CurrentTask   string    `json:"current_task,omitempty"`
	LastHeartbeat time.Time `json:"last_heartbeat,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type AgentTask struct {
	ID          string     `json:"id"`
	AgentID     string     `json:"agent_id"`
	TaskType    string     `json:"task_type"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

var AgentTypes = map[string]map[string]interface{}{
	"backend": {
		"role":         "Backend Developer",
		"scope":        "api, database, grpc, backend services",
		"skills":       []string{"go", "python", "api", "database", "grpc", "sql", "nosql"},
		"capabilities": []string{"implement_endpoints", "design_database", "write_tests", "optimize_performance"},
	},
	"frontend": {
		"role":         "Frontend Developer",
		"scope":        "ui, ux, components, state management",
		"skills":       []string{"react", "vue", "typescript", "css", "html", "webpack", "vite"},
		"capabilities": []string{"build_ui", "implement_state", "style_components", "integrate_api"},
	},
	"qa": {
		"role":         "QA Engineer",
		"scope":        "testing, quality, automation",
		"skills":       []string{"testing", "jest", "cypress", "pytest", "e2e", "integration"},
		"capabilities": []string{"write_tests", "automate_testing", "quality_audit", "performance_testing"},
	},
	"devops": {
		"role":         "DevOps Engineer",
		"scope":        "deployment, infrastructure, ci/cd",
		"skills":       []string{"docker", "kubernetes", "terraform", "ci_cd", "aws", "gcp", "azure"},
		"capabilities": []string{"deploy", "manage_infra", "setup_ci_cd", "monitor"},
	},
	"security": {
		"role":         "Security Engineer",
		"scope":        "security, audit, compliance",
		"skills":       []string{"security", "audit", "sast", "compliance", "penetration_testing"},
		"capabilities": []string{"security_audit", "vulnerability_scan", "compliance_check", "secure_code_review"},
	},
	"docs": {
		"role":         "Technical Writer",
		"scope":        "documentation, guides, api docs",
		"skills":       []string{"markdown", "api_docs", "guides", "diagrams"},
		"capabilities": []string{"write_docs", "generate_api_docs", "create_guides", "update_changelog"},
	},
}

var DefaultToolsByType = map[string][]string{
	"backend":  {"Bash", "Read", "Write", "Edit", "Glob", "Grep", "TodoWrite", "MemSave", "MemFind", "MemContext", "PlanCreate", "PlanGet", "TaskCreate", "TaskUpdate", "TaskComplete", "PhaseUpdate", "CheckpointOpen", "HumanReviewCreate", "HumanReviewPending", "HumanReviewDecide", "SkillSearch", "SkillLoad", "RuleCheck", "NotificationSend", "ProjectScan", "SpecSave", "SpecCheck", "AgentList", "AgentGet", "AgentHeartbeat", "AgentCompleteTask", "TeamGet", "PlanProgress", "PlanDashboard", "PhaseReport", "PlanBlockers", "TaskSetBlocker"},
	"frontend": {"Bash", "Read", "Write", "Edit", "Glob", "Grep", "TodoWrite", "MemSave", "MemFind", "MemContext", "PlanCreate", "PlanGet", "TaskCreate", "TaskUpdate", "TaskComplete", "PhaseUpdate", "CheckpointOpen", "HumanReviewCreate", "HumanReviewPending", "HumanReviewDecide", "SkillSearch", "SkillLoad", "RuleCheck", "NotificationSend", "ProjectScan", "SpecSave", "SpecCheck", "AgentList", "AgentGet", "AgentHeartbeat", "AgentCompleteTask", "TeamGet", "PlanProgress", "PlanDashboard", "PhaseReport", "PlanBlockers", "TaskSetBlocker"},
	"devops":   {"Bash", "Read", "Write", "Edit", "Glob", "Grep", "TodoWrite", "MemSave", "MemFind", "MemContext", "PlanCreate", "PlanGet", "TaskCreate", "TaskUpdate", "TaskComplete", "PhaseUpdate", "CheckpointOpen", "HumanReviewCreate", "HumanReviewPending", "HumanReviewDecide", "SkillSearch", "SkillLoad", "RuleCheck", "NotificationSend", "ProjectScan", "SpecSave", "SpecCheck", "AgentList", "AgentGet", "AgentHeartbeat", "AgentCompleteTask", "TeamGet", "PlanProgress", "PlanDashboard", "PhaseReport", "PlanBlockers", "TaskSetBlocker", "SastRun", "QualitySnapshot", "PrecommitValidate", "PkgAudit"},
	"qa":       {"Bash", "Read", "Glob", "Grep", "MemSave", "MemFind", "MemContext", "PlanCreate", "PlanGet", "TaskCreate", "TaskUpdate", "TaskComplete", "PhaseUpdate", "CheckpointOpen", "HumanReviewCreate", "HumanReviewPending", "HumanReviewDecide", "SkillSearch", "SkillLoad", "RuleCheck", "NotificationSend", "ProjectScan", "AgentList", "AgentGet", "AgentHeartbeat", "AgentCompleteTask", "TeamGet", "PlanProgress", "PlanDashboard", "PhaseReport", "PlanBlockers", "TaskSetBlocker", "QualitySnapshot", "PrecommitValidate", "PkgAudit", "PkgCheck"},
	"security": {"Read", "Glob", "Grep", "MemSave", "MemFind", "MemContext", "PlanCreate", "PlanGet", "TaskCreate", "TaskUpdate", "TaskComplete", "PhaseUpdate", "CheckpointOpen", "HumanReviewCreate", "HumanReviewPending", "HumanReviewDecide", "SkillSearch", "SkillLoad", "RuleCheck", "NotificationSend", "ProjectScan", "SpecSave", "SpecCheck", "AgentList", "AgentGet", "AgentHeartbeat", "AgentCompleteTask", "TeamGet", "PlanProgress", "PlanDashboard", "PhaseReport", "PlanBlockers", "TaskSetBlocker", "SastRun", "PkgAudit", "PkgCheck"},
	"docs":     {"Read", "Write", "Glob", "MemSave", "MemFind", "MemContext", "HumanReviewCreate", "HumanReviewPending", "SkillSearch", "SkillLoad", "RuleCheck", "NotificationSend", "ProjectScan", "SpecSave", "AgentList", "AgentGet", "TeamGet", "PlanProgress", "PlanDashboard", "PlanBlockers"},
}

func GetToolsForAgentType(agentType string) []string {
	if tools, ok := DefaultToolsByType[agentType]; ok {
		return tools
	}
	return DefaultToolsByType["backend"]
}

func (s *Server) handleAgentCreate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Name   string   `json:"name"`
		Type   string   `json:"agent_type"`
		Skills []string `json:"skills,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Name == "" || params.Type == "" {
		return nil, fmt.Errorf("name and agent_type are required")
	}

	agentTemplate, ok := AgentTypes[params.Type]
	if !ok {
		return nil, fmt.Errorf("unknown agent type: %s", params.Type)
	}

	now := time.Now()
	role := agentTemplate["role"].(string)
	scope := agentTemplate["scope"].(string)
	skills := params.Skills
	if len(skills) == 0 {
		skills = agentTemplate["skills"].([]string)
	}
	capabilities := agentTemplate["capabilities"].([]string)

	allowedTools := GetToolsForAgentType(params.Type)
	_ = allowedTools // suppress unused warning

	skillsJSON, _ := json.Marshal(skills)
	capabilitiesJSON, _ := json.Marshal(capabilities)
	allowedToolsJSON, _ := json.Marshal(allowedTools)

	hasCapabilities := columnExists(s.db, "agents", "capabilities")

	var existingID string
	err := s.db.QueryRow(`SELECT id FROM agents WHERE name = ?`, params.Name).Scan(&existingID)
	if err == nil {
		return &Response{Result: map[string]interface{}{
			"agent_id":       existingID,
			"name":           params.Name,
			"agent_type":     params.Type,
			"role":           role,
			"scope":          scope,
			"skills":         skills,
			"capabilities":   capabilities,
			"status":         "idle",
			"created_at":     now,
			"already_exists": true,
		}}, nil
	}

	agentID := generateID("agent")

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	var query string
	var args []interface{}

	if hasCapabilities {
		query = `INSERT INTO agents (id, name, agent_type, role, scope, skills, allowed_tools, capabilities, status, created_at, updated_at)
				  VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'idle', ?, ?)`
		args = []interface{}{agentID, params.Name, params.Type, role, scope, string(skillsJSON), string(allowedToolsJSON), string(capabilitiesJSON), "idle", now.Format(time.RFC3339), now.Format(time.RFC3339)}
	} else {
		query = `INSERT INTO agents (id, name, agent_type, role, scope, skills, allowed_tools, status, created_at, updated_at)
				  VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'idle', ?, ?)`
		args = []interface{}{agentID, params.Name, params.Type, role, scope, string(skillsJSON), string(allowedToolsJSON), "idle", now.Format(time.RFC3339), now.Format(time.RFC3339)}
	}

	_, err = tx.Exec(query, args...)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"agent_id":      agentID,
		"name":          params.Name,
		"agent_type":    params.Type,
		"role":          role,
		"scope":         scope,
		"skills":        skills,
		"allowed_tools": allowedTools,
		"capabilities":  capabilities,
		"status":        "idle",
		"created_at":    now,
	}}, nil
}

func (s *Server) handleAgentGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	agent := &SpecializedAgent{}
	var skillsJSON, allowedToolsJSON, capabilitiesJSON, agentTypeNull sql.NullString
	var currentTask sql.NullString
	var lastHeartbeatNull sql.NullString
	var createdAtStr sql.NullString

	hasCapabilities := columnExists(s.db, "agents", "capabilities")

	if hasCapabilities {
		query := `SELECT id, name, agent_type, role, scope, skills, allowed_tools, capabilities, status, current_task, last_heartbeat, created_at
				  FROM agents WHERE id = ?`
		err := s.db.QueryRow(query, params.AgentID).Scan(
			&agent.ID, &agent.Name, &agentTypeNull, &agent.Role, &agent.Scope,
			&skillsJSON, &allowedToolsJSON, &capabilitiesJSON,
			&agent.Status, &currentTask, &lastHeartbeatNull, &createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("agent not found: %w", err)
		}
	} else {
		query := `SELECT id, name, agent_type, role, scope, skills, allowed_tools, status, current_task, last_heartbeat, created_at
				  FROM agents WHERE id = ?`
		err := s.db.QueryRow(query, params.AgentID).Scan(
			&agent.ID, &agent.Name, &agentTypeNull, &agent.Role, &agent.Scope,
			&skillsJSON, &allowedToolsJSON,
			&agent.Status, &currentTask, &lastHeartbeatNull, &createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("agent not found: %w", err)
		}
	}

	if agentTypeNull.Valid {
		agent.Type = agentTypeNull.String
	}
	if createdAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, createdAtStr.String); err == nil {
			agent.CreatedAt = t
		}
	}
	if lastHeartbeatNull.Valid {
		if t, err := time.Parse(time.RFC3339, lastHeartbeatNull.String); err == nil {
			agent.LastHeartbeat = t
		}
	}

	if skillsJSON.Valid {
		json.Unmarshal([]byte(skillsJSON.String), &agent.Skills)
	}
	if allowedToolsJSON.Valid {
		json.Unmarshal([]byte(allowedToolsJSON.String), &agent.AllowedTools)
	}
	if capabilitiesJSON.Valid {
		json.Unmarshal([]byte(capabilitiesJSON.String), &agent.Capabilities)
	}
	if currentTask.Valid {
		agent.CurrentTask = currentTask.String
	}

	return &Response{Result: agent}, nil
}

func (s *Server) handleSpecializedAgentList(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Type   string `json:"agent_type,omitempty"`
		Status string `json:"status,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, name, agent_type, role, scope, status, last_heartbeat, created_at FROM agents WHERE 1=1`
	var args []interface{}

	if params.Type != "" {
		query += " AND agent_type = ?"
		args = append(args, params.Type)
	}
	if params.Status != "" {
		query += " AND status = ?"
		args = append(args, params.Status)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	agents := []map[string]interface{}{}
	for rows.Next() {
		var id, name, agentType, role, scope, status sql.NullString
		var lastHeartbeat sql.NullTime
		var createdAtStr sql.NullString

		if err := rows.Scan(&id, &name, &agentType, &role, &scope, &status, &lastHeartbeat, &createdAtStr); err != nil {
			continue
		}

		var createdAt time.Time
		if createdAtStr.Valid {
			createdAt, _ = time.Parse(time.RFC3339, createdAtStr.String)
		}

		agent := map[string]interface{}{
			"id":         id.String,
			"name":       name.String,
			"agent_type": agentType.String,
			"role":       role.String,
			"scope":      scope.String,
			"status":     status.String,
			"created_at": createdAt,
		}
		if lastHeartbeat.Valid {
			agent["last_heartbeat"] = lastHeartbeat.Time
		}
		agents = append(agents, agent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agents: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"agents": agents,
		"count":  len(agents),
	}}, nil
}

func (s *Server) handleAgentAssignTask(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID string `json:"agent_id"`
		TaskID  string `json:"task_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if params.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	now := time.Now()
	query := `UPDATE agents SET current_task = ?, status = 'working', last_heartbeat = ?, updated_at = ? WHERE id = ?`
	result, err := s.db.Exec(query, params.TaskID, now, now, params.AgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to assign task: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("agent not found")
	}

	taskQuery := `INSERT INTO agent_tasks (id, agent_id, task_type, description, status, started_at)
				  VALUES (?, ?, 'development', '', 'in_progress', ?)`
	taskExecID := generateID("agent_task")
	if _, err := s.db.Exec(taskQuery, taskExecID, params.AgentID, now); err != nil {
		return nil, fmt.Errorf("failed to create task record: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"agent_id":     params.AgentID,
		"task_id":      params.TaskID,
		"execution_id": taskExecID,
		"status":       "assigned",
		"assigned_at":  now,
	}}, nil
}

func (s *Server) handleAgentCompleteTask(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID     string `json:"agent_id"`
		ExecutionID string `json:"execution_id"`
		Result      string `json:"result,omitempty"`
		Error       string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if params.ExecutionID == "" {
		return nil, fmt.Errorf("execution_id is required")
	}

	now := time.Now()
	query := `UPDATE agents SET current_task = '', status = 'idle', last_heartbeat = ?, updated_at = ? WHERE id = ?`
	if _, err := s.db.Exec(query, now, now, params.AgentID); err != nil {
		return nil, fmt.Errorf("failed to update agent status: %w", err)
	}

	taskQuery := `UPDATE agent_tasks SET status = ?, result = ?, error = ?, completed_at = ? WHERE id = ?`
	status := "completed"
	if params.Error != "" {
		status = "failed"
	}
	if _, err := s.db.Exec(taskQuery, status, params.Result, params.Error, now, params.ExecutionID); err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"agent_id":     params.AgentID,
		"execution_id": params.ExecutionID,
		"status":       status,
		"completed_at": now,
	}}, nil
}

func (s *Server) handleAgentHeartbeat(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	now := time.Now()
	query := `UPDATE agents SET last_heartbeat = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, now, now, params.AgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"agent_id":       params.AgentID,
		"last_heartbeat": now,
	}}, nil
}

func (s *Server) handleAgentSkillsGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	var skillsJSON string
	if err := s.db.QueryRow(`SELECT skills FROM agents WHERE id = ?`, params.AgentID).Scan(&skillsJSON); err != nil {
		return nil, fmt.Errorf("failed to get agent skills: %w", err)
	}

	var skills []string
	if err := json.Unmarshal([]byte(skillsJSON), &skills); err != nil {
		return nil, fmt.Errorf("failed to parse skills JSON: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"agent_id": params.AgentID,
		"skills":   skills,
	}}, nil
}

func (s *Server) handleTeamCreate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Name        string   `json:"name"`
		ProjectPath string   `json:"project_path,omitempty"`
		AgentIDs    []string `json:"agent_ids"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	teamID := generateID("team")
	now := time.Now()

	query := `INSERT INTO teams (id, name, project_path, status, created_at, updated_at)
			  VALUES (?, ?, ?, 'active', ?, ?)`
	_, err := s.db.Exec(query, teamID, params.Name, params.ProjectPath, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	for _, agentID := range params.AgentIDs {
		memberQuery := `INSERT INTO team_members (team_id, agent_id, role, joined_at) VALUES (?, ?, 'member', ?)`
		if _, err := s.db.Exec(memberQuery, teamID, agentID, now); err != nil {
			return nil, fmt.Errorf("failed to add team member: %w", err)
		}
	}

	return &Response{Result: map[string]interface{}{
		"team_id":    teamID,
		"name":       params.Name,
		"members":    len(params.AgentIDs),
		"status":     "active",
		"created_at": now,
	}}, nil
}

func (s *Server) handleTeamGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		TeamID string `json:"team_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	var teamName, projectPath, status sql.NullString
	var createdAt time.Time
	if err := s.db.QueryRow(`SELECT name, project_path, status, created_at FROM teams WHERE id = ?`, params.TeamID).Scan(&teamName, &projectPath, &status, &createdAt); err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	rows, err := s.db.Query(`SELECT agent_id, role FROM team_members WHERE team_id = ?`, params.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list team members: %w", err)
	}
	defer rows.Close()

	members := []map[string]interface{}{}
	for rows.Next() {
		var agentID, role sql.NullString
		if err := rows.Scan(&agentID, &role); err != nil {
			continue
		}
		members = append(members, map[string]interface{}{
			"agent_id": agentID.String,
			"role":     role.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating team members: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"team_id":      params.TeamID,
		"name":         teamName.String,
		"project_path": projectPath.String,
		"status":       status.String,
		"members":      members,
		"created_at":   createdAt,
	}}, nil
}
