package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

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

	agentID := generateID("agent")
	now := time.Now()

	role := agentTemplate["role"].(string)
	scope := agentTemplate["scope"].(string)
	skills := params.Skills
	if len(skills) == 0 {
		skills = agentTemplate["skills"].([]string)
	}
	capabilities := agentTemplate["capabilities"].([]string)

	skillsJSON, _ := json.Marshal(skills)
	capabilitiesJSON, _ := json.Marshal(capabilities)

	query := `INSERT INTO agents (id, name, role, scope, skills, allowed_tools, capabilities, status, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, '', ?, 'idle', ?, ?)`
	_, err := s.db.Exec(query, agentID, params.Name, role, scope, string(skillsJSON), string(capabilitiesJSON), now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"id":           agentID,
		"name":         params.Name,
		"agent_type":   params.Type,
		"role":         role,
		"scope":        scope,
		"skills":       skills,
		"capabilities": capabilities,
		"status":       "idle",
		"created_at":   now,
	}}, nil
}

func (s *Server) handleAgentGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, name, role, scope, skills, allowed_tools, capabilities, status, current_task, last_heartbeat, created_at
			  FROM agents WHERE id = ?`
	agent := &SpecializedAgent{}
	var skillsJSON, capabilitiesJSON sql.NullString
	var currentTask sql.NullString
	var lastHeartbeat sql.NullTime

	err := s.db.QueryRow(query, params.AgentID).Scan(
		&agent.ID, &agent.Name, &agent.Role, &agent.Scope,
		&skillsJSON, &agent.AllowedTools, &capabilitiesJSON,
		&agent.Status, &currentTask, &lastHeartbeat, &agent.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	if skillsJSON.Valid {
		json.Unmarshal([]byte(skillsJSON.String), &agent.Skills)
	}
	if capabilitiesJSON.Valid {
		json.Unmarshal([]byte(capabilitiesJSON.String), &agent.Capabilities)
	}
	if currentTask.Valid {
		agent.CurrentTask = currentTask.String
	}
	if lastHeartbeat.Valid {
		agent.LastHeartbeat = lastHeartbeat.Time
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
		var createdAt time.Time

		rows.Scan(&id, &name, &agentType, &role, &scope, &status, &lastHeartbeat, &createdAt)

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
	s.db.Exec(taskQuery, taskExecID, params.AgentID, now)

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

	now := time.Now()
	query := `UPDATE agents SET current_task = '', status = 'idle', last_heartbeat = ?, updated_at = ? WHERE id = ?`
	s.db.Exec(query, now, now, params.AgentID)

	taskQuery := `UPDATE agent_tasks SET status = ?, result = ?, error = ?, completed_at = ? WHERE id = ?`
	status := "completed"
	if params.Error != "" {
		status = "failed"
	}
	s.db.Exec(taskQuery, status, params.Result, params.Error, now, params.ExecutionID)

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

	var skillsJSON string
	s.db.QueryRow(`SELECT skills FROM agents WHERE id = ?`, params.AgentID).Scan(&skillsJSON)

	var skills []string
	json.Unmarshal([]byte(skillsJSON), &skills)

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
		s.db.Exec(memberQuery, teamID, agentID, now)
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
	s.db.QueryRow(`SELECT name, project_path, status, created_at FROM teams WHERE id = ?`, params.TeamID).Scan(&teamName, &projectPath, &status, &createdAt)

	rows, _ := s.db.Query(`SELECT agent_id, role FROM team_members WHERE team_id = ?`, params.TeamID)
	members := []map[string]interface{}{}
	for rows.Next() {
		var agentID, role sql.NullString
		rows.Scan(&agentID, &role)
		members = append(members, map[string]interface{}{
			"agent_id": agentID.String,
			"role":     role.String,
		})
	}
	rows.Close()

	return &Response{Result: map[string]interface{}{
		"team_id":      params.TeamID,
		"name":         teamName.String,
		"project_path": projectPath.String,
		"status":       status.String,
		"members":      members,
		"created_at":   createdAt,
	}}, nil
}
