package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

const (
	errFailedParseParams   = "failed to parse params: %w"
	errExecutionIDRequired = "execution_id is required"
	errTaskIDRequired      = "task_id is required"
)

type TaskExecution struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"task_id"`
	HatiTaskID  string     `json:"hati_task_id,omitempty"`
	AgentID     string     `json:"agent_id"`
	PhaseID     string     `json:"phase_id,omitempty"`
	Status      string     `json:"status"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	HeartbeatAt *time.Time `json:"heartbeat_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (s *Server) handleTaskExecute(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		TaskID     string `json:"task_id"`
		HatiTaskID string `json:"hati_task_id,omitempty"`
		AgentID    string `json:"agent_id"`
		PhaseID    string `json:"phase_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.TaskID == "" {
		return nil, fmt.Errorf(errTaskIDRequired)
	}
	if params.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	execID := generateID("texec")
	now := time.Now()

	query := `INSERT INTO task_executions (id, task_id, hati_task_id, agent_id, phase_id, status, started_at, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, 'in_progress', ?, ?, ?)`
	_, err := s.db.Exec(query, execID, params.TaskID, params.HatiTaskID, params.AgentID, params.PhaseID, now, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create task execution: %w", err)
	}

	agentQuery := `UPDATE agents SET current_task = ?, status = 'working', last_heartbeat = ?, updated_at = ? WHERE id = ?`
	s.db.Exec(agentQuery, params.TaskID, now, now, params.AgentID)

	return &Response{Result: map[string]interface{}{
		"execution_id": execID,
		"task_id":      params.TaskID,
		"hati_task_id": params.HatiTaskID,
		"agent_id":     params.AgentID,
		"status":       "in_progress",
		"started_at":   now,
	}}, nil
}

func (s *Server) handleTaskDelegate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		TaskID     string   `json:"task_id"`
		HatiTaskID string   `json:"hati_task_id,omitempty"`
		AgentIDs   []string `json:"agent_ids"`
		PhaseID    string   `json:"phase_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.TaskID == "" {
		return nil, fmt.Errorf(errTaskIDRequired)
	}
	
	// Si no hay agent_ids, intentamos buscar por rol
	if len(params.AgentIDs) == 0 {
		params.AgentIDs = s.findAgentsByRole(params.PhaseID)
	}

	if len(params.AgentIDs) == 0 {
		return nil, fmt.Errorf("no active agents found for delegation")
	}

	now := time.Now()
	var executions []map[string]interface{}

	for _, agentID := range params.AgentIDs {
		execID := generateID("texec")
		query := `INSERT INTO task_executions (id, task_id, hati_task_id, agent_id, phase_id, status, started_at, created_at, updated_at)
				  VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?)`
		if _, err := s.db.Exec(query, execID, params.TaskID, params.HatiTaskID, agentID, params.PhaseID, now, now, now); err != nil {
			return nil, fmt.Errorf("failed to delegate to agent %s: %w", agentID, err)
		}
		executions = append(executions, map[string]interface{}{
			"execution_id": execID,
			"agent_id":     agentID,
			"status":       "pending",
		})
	}

	return &Response{Result: map[string]interface{}{
		"task_id":      params.TaskID,
		"hati_task_id": params.HatiTaskID,
		"delegated_to": executions,
		"total_agents": len(params.AgentIDs),
		"created_at":   now,
	}}, nil
}

func (s *Server) handleTaskStatus(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ExecutionID string `json:"execution_id,omitempty"`
		TaskID      string `json:"task_id,omitempty"`
		AgentID     string `json:"agent_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ExecutionID == "" && params.TaskID == "" && params.AgentID == "" {
		return nil, fmt.Errorf("execution_id, task_id, or agent_id is required")
	}

	query := `SELECT id, task_id, hati_task_id, agent_id, phase_id, status, result, error, started_at, completed_at, heartbeat_at, created_at, updated_at
			  FROM task_executions WHERE 1=1`
	var args []interface{}

	if params.ExecutionID != "" {
		query += " AND id = ?"
		args = append(args, params.ExecutionID)
	}
	if params.TaskID != "" {
		query += " AND task_id = ?"
		args = append(args, params.TaskID)
	}
	if params.AgentID != "" {
		query += " AND agent_id = ?"
		args = append(args, params.AgentID)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query task executions: %w", err)
	}
	defer rows.Close()

	executions := []map[string]interface{}{}
	for rows.Next() {
		var exec TaskExecution
		var hatiTaskID, phaseID, result, errorMsg sql.NullString
		var heartbeatAt sql.NullTime
		var completedAt sql.NullTime

		if err := rows.Scan(&exec.ID, &exec.TaskID, &hatiTaskID, &exec.AgentID, &phaseID,
			&exec.Status, &result, &errorMsg, &exec.StartedAt, &completedAt,
			&heartbeatAt, &exec.CreatedAt, &exec.UpdatedAt); err != nil {
			continue
		}

		exec.HatiTaskID = hatiTaskID.String
		exec.PhaseID = phaseID.String
		exec.Result = result.String
		exec.Error = errorMsg.String
		if completedAt.Valid {
			exec.CompletedAt = &completedAt.Time
		}
		if heartbeatAt.Valid {
			exec.HeartbeatAt = &heartbeatAt.Time
		}

		executions = append(executions, map[string]interface{}{
			"id":           exec.ID,
			"task_id":      exec.TaskID,
			"hati_task_id": exec.HatiTaskID,
			"agent_id":     exec.AgentID,
			"phase_id":     exec.PhaseID,
			"status":       exec.Status,
			"result":       exec.Result,
			"error":        exec.Error,
			"started_at":   exec.StartedAt,
			"completed_at": exec.CompletedAt,
			"heartbeat_at": exec.HeartbeatAt,
		})
	}

	return &Response{Result: map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
	}}, nil
}

func (s *Server) handleTaskHeartbeat(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ExecutionID string `json:"execution_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ExecutionID == "" {
		return nil, fmt.Errorf(errExecutionIDRequired)
	}

	now := time.Now()
	query := `UPDATE task_executions SET heartbeat_at = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, now, now, params.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"execution_id": params.ExecutionID,
		"heartbeat_at": now,
	}}, nil
}

func (s *Server) handleTaskComplete(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ExecutionID string `json:"execution_id"`
		Status      string `json:"status"`
		Result      string `json:"result,omitempty"`
		Error       string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ExecutionID == "" {
		return nil, fmt.Errorf(errExecutionIDRequired)
	}
	if params.Status == "" {
		params.Status = "completed"
	}

	now := time.Now()
	var completedStatus string
	if params.Status == "failed" {
		completedStatus = "failed"
	} else {
		completedStatus = "completed"
	}

	query := `UPDATE task_executions SET status = ?, result = ?, error = ?, completed_at = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, completedStatus, params.Result, params.Error, now, now, params.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to complete task execution: %w", err)
	}

	var exec TaskExecution
	s.db.QueryRow(`SELECT task_id, agent_id FROM task_executions WHERE id = ?`, params.ExecutionID).Scan(&exec.TaskID, &exec.AgentID)

	agentQuery := `UPDATE agents SET current_task = '', status = 'idle', last_heartbeat = ?, updated_at = ? WHERE id = ?`
	s.db.Exec(agentQuery, now, now, exec.AgentID)

	var pending, inProgress int
	s.db.QueryRow(`SELECT COUNT(*) FROM task_executions WHERE task_id = ? AND status = 'pending'`, exec.TaskID).Scan(&pending)
	s.db.QueryRow(`SELECT COUNT(*) FROM task_executions WHERE task_id = ? AND status = 'in_progress'`, exec.TaskID).Scan(&inProgress)

	return &Response{Result: map[string]interface{}{
		"execution_id":     params.ExecutionID,
		"task_id":          exec.TaskID,
		"status":           completedStatus,
		"result":           params.Result,
		"error":            params.Error,
		"completed_at":     now,
		"task_pending":     pending,
		"task_in_progress": inProgress,
	}}, nil
}

func (s *Server) handleTaskCancel(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ExecutionID string `json:"execution_id"`
		Reason      string `json:"reason,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf(errFailedParseParams, err)
	}

	if params.ExecutionID == "" {
		return nil, fmt.Errorf("execution_id is required")
	}

	now := time.Now()
	query := `UPDATE task_executions SET status = 'cancelled', error = ?, completed_at = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, params.Reason, now, now, params.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel task execution: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"execution_id": params.ExecutionID,
		"status":       "cancelled",
		"reason":       params.Reason,
		"cancelled_at": now,
	}}, nil
}

func (s *Server) findAgentsByRole(role string) []string {
	var agents []string
	query := `SELECT id FROM agents WHERE is_active = 1`
	var args []interface{}
	if role != "" {
		query += " AND (role = ? OR agent_type = ? OR role LIKE ?)"
		args = append(args, role, role, "%"+role+"%")
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return agents
	}
	defer rows.Close()
	for rows.Next() {
		var aid string
		if err := rows.Scan(&aid); err == nil {
			agents = append(agents, aid)
		}
	}
	return agents
}

func (s *Server) handleWorkflowDeprecate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		WorkflowID string `json:"workflow_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.WorkflowID == "" {
		return nil, fmt.Errorf("workflow_id is required")
	}

	now := time.Now()
	query := `UPDATE workflows SET deprecated = 1, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, now, params.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to deprecate workflow: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"workflow_id": params.WorkflowID,
		"deprecated":  true,
		"note":        "Use task_execute/task_delegate instead of workflows",
		"updated_at":  now,
	}}, nil
}
