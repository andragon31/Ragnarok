package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

type Task struct {
	ID                string     `json:"id"`
	PhaseID           string     `json:"phase_id"`
	PRDRequirementID  string     `json:"prd_requirement_id,omitempty"`
	Title             string     `json:"title"`
	Description       string     `json:"description,omitempty"`
	Status            string     `json:"status"`
	Priority          int        `json:"priority"`
	AssignedAgentIDs  []string   `json:"assigned_agent_ids,omitempty"`
	AssignedAgentType string     `json:"assigned_agent_type,omitempty"`
	EstimatedHours    float64    `json:"estimated_hours,omitempty"`
	ActualHours       float64    `json:"actual_hours,omitempty"`
	Notes             string     `json:"notes,omitempty"`
	Blocker           string     `json:"blocker,omitempty"`
	Milestone         bool       `json:"milestone"`
	Subtasks          []string   `json:"subtasks,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type TaskAgent struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"task_id"`
	AgentID     string     `json:"agent_id"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type PRD struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Version   string    `json:"version"`
	Content   string    `json:"content,omitempty"`
	FilePath  string    `json:"file_path,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PRDRequirement struct {
	ID                 string   `json:"id"`
	PRDID              string   `json:"prd_id"`
	ReqType            string   `json:"req_type"`
	Priority           string   `json:"priority"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Status             string   `json:"status"`
}

type HumanReview struct {
	ID         string     `json:"id"`
	ReviewType string     `json:"review_type"`
	EntityType string     `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
	Question   string     `json:"question,omitempty"`
	Decision   string     `json:"decision,omitempty"`
	Approver   string     `json:"approver,omitempty"`
	Notes      string     `json:"notes,omitempty"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	DecidedAt  *time.Time `json:"decided_at,omitempty"`
}

func (s *Server) handleTaskCreate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PhaseID           string   `json:"phase_id"`
		PRDRequirementID  string   `json:"prd_requirement_id,omitempty"`
		Title             string   `json:"title"`
		Description       string   `json:"description,omitempty"`
		Priority          int      `json:"priority,omitempty"`
		AssignedAgentIDs  []string `json:"assigned_agent_ids,omitempty"`
		AssignedAgentType string   `json:"assigned_agent_type,omitempty"`
		EstimatedHours    float64  `json:"estimated_hours,omitempty"`
		Milestone         bool     `json:"milestone,omitempty"`
		Subtasks          []string `json:"subtasks,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if params.PhaseID == "" {
		return nil, fmt.Errorf("phase_id is required")
	}

	taskID := generateID("task")
	now := time.Now()

	subtasksJSON, _ := json.Marshal(params.Subtasks)
	agentIDsJSON, _ := json.Marshal(params.AssignedAgentIDs)

	assignedAgentType := params.AssignedAgentType
	if assignedAgentType == "" {
		assignedAgentType = "worker"
	}

	query := `INSERT INTO tasks (id, phase_id, prd_requirement_id, title, description, status, priority, assigned_agent_ids, assigned_agent_type, estimated_hours, milestone, subtasks, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?, ?)`
	var prdReqID interface{} = nil
	if params.PRDRequirementID != "" {
		prdReqID = params.PRDRequirementID
	}
	_, err := s.db.Exec(query, taskID, params.PhaseID, prdReqID, params.Title, params.Description,
		params.Priority, string(agentIDsJSON), assignedAgentType, params.EstimatedHours,
		params.Milestone, string(subtasksJSON), now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	if len(params.AssignedAgentIDs) > 0 {
		for _, agentID := range params.AssignedAgentIDs {
			taID := generateID("taskagent")
			insertQuery := `INSERT INTO task_agents (id, task_id, agent_id, role, status, started_at) VALUES (?, ?, ?, 'worker', 'pending', ?)`
			s.db.Exec(insertQuery, taID, taskID, agentID, now)
		}
	}

	return &Response{
		Result: map[string]interface{}{
			"id":                 taskID,
			"phase_id":           params.PhaseID,
			"title":              params.Title,
			"status":             "pending",
			"priority":           params.Priority,
			"assigned_agent_ids": params.AssignedAgentIDs,
			"created_at":         now,
		},
	}, nil
}

func (s *Server) handleTaskGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		TaskID string `json:"task_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, phase_id, prd_requirement_id, title, description, status, priority, assigned_agent_ids, assigned_agent_type, estimated_hours, actual_hours, notes, blocker, milestone, subtasks, completed_at, created_at, updated_at
			  FROM tasks WHERE id = ?`
	task := &Task{}
	var prdReqID, agentIDsJSON, agentType, desc, notes, blocker sql.NullString
	var estimated, actual sql.NullFloat64
	var subtasks sql.NullString
	var completedAt sql.NullTime
	var milestone int

	err := s.db.QueryRow(query, params.TaskID).Scan(
		&task.ID, &task.PhaseID, &prdReqID, &task.Title, &desc,
		&task.Status, &task.Priority, &agentIDsJSON, &agentType,
		&estimated, &actual, &notes, &blocker, &milestone,
		&subtasks, &completedAt, &task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	task.PRDRequirementID = prdReqID.String
	task.Description = desc.String
	task.Notes = notes.String
	task.Blocker = blocker.String
	if agentIDsJSON.Valid && agentIDsJSON.String != "" {
		json.Unmarshal([]byte(agentIDsJSON.String), &task.AssignedAgentIDs)
	}
	task.AssignedAgentType = agentType.String
	task.EstimatedHours = estimated.Float64
	task.ActualHours = actual.Float64
	task.Milestone = milestone == 1
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if subtasks.Valid && subtasks.String != "" {
		json.Unmarshal([]byte(subtasks.String), &task.Subtasks)
	}

	taskAgents, _ := s.getTaskAgents(params.TaskID)

	return &Response{Result: map[string]interface{}{
		"task":        task,
		"task_agents": taskAgents,
	}}, nil
}

func (s *Server) handleTaskGetNext(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID    string `json:"plan_id"`
		AgentType string `json:"agent_type,omitempty"`
		AgentID   string `json:"agent_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT t.id, t.phase_id, t.prd_requirement_id, t.title, t.description, t.status, t.priority, 
			  t.assigned_agent_ids, t.assigned_agent_type, t.estimated_hours, t.actual_hours, t.notes, 
			  t.blocker, t.milestone, t.subtasks, t.completed_at, t.created_at, t.updated_at, p.name as phase_title
			  FROM tasks t
			  JOIN phases p ON t.phase_id = p.id
			  WHERE p.plan_id = ? AND t.status IN ('pending', 'blocked')
			  ORDER BY t.milestone DESC, t.priority DESC, t.created_at ASC
			  LIMIT 1`

	task := &Task{}
	var prdReqID, agentIDsJSON, agentType, desc, notes, blocker, phaseTitle sql.NullString
	var estimated, actual sql.NullFloat64
	var subtasks sql.NullString
	var completedAt sql.NullTime
	var milestone int

	err := s.db.QueryRow(query, params.PlanID).Scan(
		&task.ID, &task.PhaseID, &prdReqID, &task.Title, &desc,
		&task.Status, &task.Priority, &agentIDsJSON, &agentType,
		&estimated, &actual, &notes, &blocker, &milestone,
		&subtasks, &completedAt, &task.CreatedAt, &task.UpdatedAt, &phaseTitle,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &Response{
				Result: map[string]interface{}{
					"message":      "no pending tasks",
					"all_complete": true,
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to get next task: %w", err)
	}

	task.PRDRequirementID = prdReqID.String
	task.Description = desc.String
	task.Notes = notes.String
	task.Blocker = blocker.String
	if agentIDsJSON.Valid && agentIDsJSON.String != "" {
		json.Unmarshal([]byte(agentIDsJSON.String), &task.AssignedAgentIDs)
	}
	task.AssignedAgentType = agentType.String
	task.EstimatedHours = estimated.Float64
	task.ActualHours = actual.Float64
	task.Milestone = milestone == 1
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if subtasks.Valid && subtasks.String != "" {
		json.Unmarshal([]byte(subtasks.String), &task.Subtasks)
	}

	return &Response{Result: map[string]interface{}{
		"task":        task,
		"phase_title": phaseTitle.String,
	}}, nil
}

func (s *Server) handleTaskUpdate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		TaskID      string  `json:"task_id"`
		Status      string  `json:"status,omitempty"`
		Notes       string  `json:"notes,omitempty"`
		ActualHours float64 `json:"actual_hours,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	now := time.Now()
	var completedAt *time.Time
	if params.Status == "completed" {
		completedAt = &now
	}

	query := `UPDATE tasks SET status = COALESCE(NULLIF(?, ''), status), 
			  notes = COALESCE(NULLIF(?, ''), notes),
			  actual_hours = CASE WHEN ? > 0 THEN ? ELSE actual_hours END,
			  completed_at = COALESCE(?, completed_at),
			  updated_at = ?
			  WHERE id = ?`
	_, err := s.db.Exec(query, params.Status, params.Notes, params.ActualHours, params.ActualHours, completedAt, now, params.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"task_id":    params.TaskID,
		"status":     params.Status,
		"updated_at": now,
	}}, nil
}

func (s *Server) handleTaskSetBlocker(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		TaskID  string `json:"task_id"`
		Blocker string `json:"blocker"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	now := time.Now()
	query := `UPDATE tasks SET blocker = ?, status = 'blocked', updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, params.Blocker, now, params.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to set blocker: %w", err)
	}

	query = `INSERT INTO execution_blockers (id, plan_id, reason, type, blocked_at)
			 SELECT ?, p.id, ?, 'task_blocker', ?
			 FROM tasks t JOIN phases ph ON t.phase_id = ph.id JOIN plans p ON ph.plan_id = p.id
			 WHERE t.id = ?`
	blockerID := generateID("blocker")
	if _, err := s.db.Exec(query, blockerID, params.Blocker, now, params.TaskID); err != nil {
		return nil, fmt.Errorf("failed to set task blocker: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"task_id": params.TaskID,
		"blocker": params.Blocker,
		"status":  "blocked",
	}}, nil
}

func (s *Server) handleTaskList(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PhaseID string `json:"phase_id,omitempty"`
		PlanID  string `json:"plan_id,omitempty"`
		Status  string `json:"status,omitempty"`
		Limit   int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 50
	}

	query := `SELECT t.id, t.phase_id, t.prd_requirement_id, t.title, t.description, t.status, t.priority, 
			  t.assigned_agent_ids, t.assigned_agent_type, t.milestone, t.blocker, t.completed_at, t.created_at
			  FROM tasks t`
	var args []interface{}
	var conditions []string

	if params.PhaseID != "" {
		conditions = append(conditions, "t.phase_id = ?")
		args = append(args, params.PhaseID)
	}
	if params.PlanID != "" {
		conditions = append(conditions, "ph.plan_id = ?")
		args = append(args, params.PlanID)
		query += " JOIN phases ph ON t.phase_id = ph.id"
	}
	if params.Status != "" {
		conditions = append(conditions, "t.status = ?")
		args = append(args, params.Status)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY t.milestone DESC, t.priority DESC, t.created_at ASC LIMIT ?"
	args = append(args, params.Limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	tasks := []map[string]interface{}{}
	for rows.Next() {
		var id, phaseID, prdReqID, title, description, status, priority, agentIDsJSON, agentType, blocker sql.NullString
		var milestone int
		var completedAt sql.NullTime
		var createdAtStr sql.NullString

		if err := rows.Scan(&id, &phaseID, &prdReqID, &title, &description, &status, &priority, &agentIDsJSON, &agentType, &milestone, &blocker, &completedAt, &createdAtStr); err != nil {
			log.Printf("Error scanning task in task_list: %v", err)
			continue
		}

		createdAt, _ := time.Parse(time.RFC3339, createdAtStr.String)
		if createdAt.IsZero() {
			createdAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr.String)
		}

		var agentIDs []string
		if agentIDsJSON.Valid && agentIDsJSON.String != "" {
			json.Unmarshal([]byte(agentIDsJSON.String), &agentIDs)
		}

		task := map[string]interface{}{
			"id":          id.String,
			"phase_id":    phaseID.String,
			"prd_req_id":  prdReqID.String,
			"title":       title.String,
			"description": description.String,
			"status":      status.String,
			"priority":    priority.String,
			"agent_ids":   agentIDs,
			"agent_type":  agentType.String,
			"milestone":   milestone == 1,
			"blocker":     blocker.String,
			"created_at":  createdAt.Format(time.RFC3339),
		}
		if completedAt.Valid {
			task["completed_at"] = completedAt.Time
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}}, nil
}

func (s *Server) handlePhaseCreate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID      string `json:"plan_id"`
		Title       string `json:"title"`
		Description string `json:"description,omitempty"`
		Order       int    `json:"order_num,omitempty"`
		AgentHints  string `json:"agent_hints,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.PlanID == "" || params.Title == "" {
		return nil, fmt.Errorf("plan_id and title are required")
	}

	phaseID := generateID("phase")
	now := time.Now()

	query := `INSERT INTO phases (id, plan_id, name, description, order_num, status, agents_md_hints, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?)`
	_, err := s.db.Exec(query, phaseID, params.PlanID, params.Title, params.Description, params.Order, params.AgentHints, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create phase: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"id":         phaseID,
		"phase_id":   phaseID,
		"plan_id":    params.PlanID,
		"title":      params.Title,
		"status":     "pending",
		"created_at": now,
	}}, nil
}

func (s *Server) handlePhaseUpdate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PhaseID string `json:"phase_id"`
		Status  string `json:"status,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	now := time.Now()
	query := `UPDATE phases SET status = COALESCE(NULLIF(?, ''), status), updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, params.Status, now, params.PhaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to update phase: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"id":         params.PhaseID,
		"phase_id":   params.PhaseID,
		"status":     params.Status,
		"updated_at": now,
	}}, nil
}

func (s *Server) handlePlanProgress(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT COUNT(*) FROM tasks t 
			  JOIN phases ph ON t.phase_id = ph.id 
			  WHERE ph.plan_id = ?`
	var total, completed int
	if err := s.db.QueryRow(query, params.PlanID).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	query = `SELECT COUNT(*) FROM tasks t 
			JOIN phases ph ON t.phase_id = ph.id 
			WHERE ph.plan_id = ? AND t.status = 'completed'`
	if err := s.db.QueryRow(query, params.PlanID).Scan(&completed); err != nil {
		return nil, fmt.Errorf("failed to count completed tasks: %w", err)
	}

	var progress float64
	if total > 0 {
		progress = float64(completed) / float64(total)
	}

	query = `SELECT p.id, p.title, p.status, p.created_at,
			 (SELECT COUNT(*) FROM phases WHERE plan_id = p.id) as phase_count,
			 (SELECT COUNT(*) FROM phases WHERE plan_id = p.id AND status = 'completed') as completed_phases
			 FROM plans p WHERE p.id = ?`
	var planID, planTitle, planStatus string
	var createdAt time.Time
	var phaseCount, completedPhases int

	if err := s.db.QueryRow(query, params.PlanID).Scan(&planID, &planTitle, &planStatus, &createdAt, &phaseCount, &completedPhases); err != nil {
		return nil, fmt.Errorf("failed to get plan details: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"id":               params.PlanID,
		"plan_id":          params.PlanID,
		"title":            planTitle,
		"status":           planStatus,
		"total_tasks":      total,
		"completed_tasks":  completed,
		"pending_tasks":    total - completed,
		"progress":         progress,
		"progress_percent": fmt.Sprintf("%.1f%%", progress*100),
		"phase_count":      phaseCount,
		"completed_phases": completedPhases,
	}}, nil
}

func (s *Server) handleHumanReviewCreate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ReviewType string `json:"review_type"`
		EntityType string `json:"entity_type"`
		EntityID   string `json:"entity_id"`
		Question   string `json:"question"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ReviewType == "" || params.EntityID == "" {
		return nil, fmt.Errorf("review_type and entity_id are required")
	}

	reviewID := generateID("review")
	now := time.Now()

	query := `INSERT INTO human_reviews (id, review_type, entity_type, entity_id, question, status, created_at)
			  VALUES (?, ?, ?, ?, ?, 'pending', ?)`
	_, err := s.db.Exec(query, reviewID, params.ReviewType, params.EntityType, params.EntityID, params.Question, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"id":          reviewID,
		"review_id":   reviewID,
		"review_type": params.ReviewType,
		"entity_id":   params.EntityID,
		"status":      "pending",
		"question":    params.Question,
		"created_at":  now,
	}}, nil
}

func (s *Server) handleHumanReviewDecide(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ReviewID string `json:"review_id"`
		Decision string `json:"decision"`
		Approver string `json:"approver,omitempty"`
		Notes    string `json:"notes,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.ReviewID == "" || params.Decision == "" {
		return nil, fmt.Errorf("review_id and decision are required")
	}

	now := time.Now()
	query := `UPDATE human_reviews SET decision = ?, approver = ?, notes = ?, status = ?, decided_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, params.Decision, params.Approver, params.Notes, "decided", now, params.ReviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to decide review: %w", err)
	}

	var reviewType, entityID string
	if err := s.db.QueryRow(`SELECT review_type, entity_id FROM human_reviews WHERE id = ?`, params.ReviewID).Scan(&reviewType, &entityID); err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	if params.Decision == "approved" && reviewType == "prd_approval" {
		if _, err := s.db.Exec(`UPDATE plans SET status = 'active' WHERE id = ?`, entityID); err != nil {
			return nil, fmt.Errorf("failed to activate plan: %w", err)
		}
	}

	return &Response{Result: map[string]interface{}{
		"id":         params.ReviewID,
		"review_id":  params.ReviewID,
		"decision":   params.Decision,
		"status":     "decided",
		"decided_at": now,
	}}, nil
}

func (s *Server) handleHumanReviewPending(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ReviewType string `json:"review_type,omitempty"`
		EntityType string `json:"entity_type,omitempty"`
	}

	query := `SELECT id, review_type, entity_type, entity_id, question, status, created_at FROM human_reviews WHERE status = 'pending'`
	var args []interface{}

	if params.ReviewType != "" {
		query += " AND review_type = ?"
		args = append(args, params.ReviewType)
	}
	if params.EntityType != "" {
		query += " AND entity_type = ?"
		args = append(args, params.EntityType)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending reviews: %w", err)
	}
	defer rows.Close()

	reviews := []map[string]interface{}{}
	for rows.Next() {
		var id, reviewType, entityType, entityID, question sql.NullString
		var status string
		var createdAt time.Time

		if err := rows.Scan(&id, &reviewType, &entityType, &entityID, &question, &status, &createdAt); err != nil {
			continue
		}

		reviews = append(reviews, map[string]interface{}{
			"id":          id.String,
			"review_id":   id.String,
			"review_type": reviewType.String,
			"entity_type": entityType.String,
			"entity_id":   entityID.String,
			"question":    question.String,
			"status":      status,
			"created_at":  createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reviews: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"pending_reviews": reviews,
		"count":           len(reviews),
	}}, nil
}

func (s *Server) handlePRDParse(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	content := params.Content
	if content == "" && params.FilePath != "" {
		var err error
		content, err = osReadFile(params.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
	}

	prdID := generateID("prd")
	now := time.Now()

	title := extractPRDTitle(content)
	version := extractPRDVersion(content)

	query := `INSERT INTO prds (id, title, version, content, file_path, status, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, 'draft', ?, ?)`
	_, err := s.db.Exec(query, prdID, title, version, content, params.FilePath, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create PRD: %w", err)
	}

	requirements, err := s.extractRequirements(content, prdID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract requirements: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"id":                 prdID,
		"prd_id":             prdID,
		"title":              title,
		"version":            version,
		"requirements":       requirements,
		"requirements_count": len(requirements),
	}}, nil
}

func (s *Server) handlePRDRequirementsExtract(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PRDID string `json:"prd_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, req_type, priority, title, description, acceptance_criteria, status FROM prd_requirements WHERE prd_id = ?`
	rows, err := s.db.Query(query, params.PRDID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requirements: %w", err)
	}
	defer rows.Close()

	requirements := []map[string]interface{}{}
	for rows.Next() {
		var id, reqType, priority, title, desc, ac sql.NullString
		var status string
		if err := rows.Scan(&id, &reqType, &priority, &title, &desc, &ac, &status); err != nil {
			continue
		}

		var acList []string
		if ac.Valid {
			json.Unmarshal([]byte(ac.String), &acList)
		}

		requirements = append(requirements, map[string]interface{}{
			"id":                  id.String,
			"req_id":              id.String,
			"type":                reqType.String,
			"priority":            priority.String,
			"title":               title.String,
			"description":         desc.String,
			"acceptance_criteria": acList,
			"status":              status,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating requirements: %w", err)
	}

	return &Response{Result: map[string]interface{}{
		"id":           params.PRDID,
		"prd_id":       params.PRDID,
		"requirements": requirements,
		"count":        len(requirements),
	}}, nil
}

func (s *Server) handlePlanCreateFromPRD(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PRDID       string   `json:"prd_id"`
		Title       string   `json:"title,omitempty"`
		Description string   `json:"description,omitempty"`
		Phases      []string `json:"phases,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	requirementsQuery := `SELECT id, req_type, priority, title, description FROM prd_requirements WHERE prd_id = ?`
	rows, err := s.db.Query(requirementsQuery, params.PRDID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requirements: %w", err)
	}
	defer rows.Close()

	type reqInfo struct {
		ID          string
		Type        string
		Priority    string
		Title       string
		Description string
	}
	requirements := []reqInfo{}
	for rows.Next() {
		var r reqInfo
		var desc sql.NullString
		if err := rows.Scan(&r.ID, &r.Type, &r.Priority, &r.Title, &desc); err != nil {
			continue
		}
		r.Description = desc.String
		requirements = append(requirements, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating requirements: %w", err)
	}

	type taskInfo struct {
		PhaseIndex int
		Title      string
		Type       string
	}

	phaseNames := params.Phases
	if len(phaseNames) == 0 {
		phaseNames = s.extractPhasesFromPRD(params.PRDID)
		if len(phaseNames) == 0 {
			phaseNames = []string{"Setup", "Backend", "Frontend", "Testing", "Deploy"}
		}
	}

	planID := generateID("plan")
	now := time.Now()

	planTitle := params.Title
	if planTitle == "" {
		planTitle = "Development Plan from PRD"
	}

	query := `INSERT INTO plans (id, title, description, status, created_at, updated_at)
			  VALUES (?, ?, ?, 'draft', ?, ?)`
	_, err = s.db.Exec(query, planID, planTitle, params.Description, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	tasksToCreate := []taskInfo{}

	phaseMap := map[string]int{}
	for i, phase := range phaseNames {
		phaseID := generateID("phase")
		phaseQuery := `INSERT INTO phases (id, plan_id, name, order_num, status, created_at, updated_at)
					   VALUES (?, ?, ?, ?, 'pending', ?, ?)`
		if _, err := s.db.Exec(phaseQuery, phaseID, planID, phase, i, now, now); err != nil {
			return nil, fmt.Errorf("failed to create phase: %w", err)
		}
		phaseMap[phase] = i
	}

	phaseIdx := 0
	for _, req := range requirements {
		phaseIdx = (phaseIdx + 1) % len(phaseNames)
		taskTitle := fmt.Sprintf("[%s] %s", strings.ToUpper(req.Type), req.Title)
		tasksToCreate = append(tasksToCreate, taskInfo{
			PhaseIndex: phaseIdx,
			Title:      taskTitle,
			Type:       req.Type,
		})
	}

	if len(tasksToCreate) == 0 && len(requirements) > 0 {
		for i, req := range requirements {
			phaseIdx := i % len(phaseNames)
			taskTitle := fmt.Sprintf("[%s] %s", strings.ToUpper(req.Type), req.Title)
			tasksToCreate = append(tasksToCreate, taskInfo{
				PhaseIndex: phaseIdx,
				Title:      taskTitle,
				Type:       req.Type,
			})
		}
	}

	taskIDs := []string{}
	for _, t := range tasksToCreate {
		taskID := generateID("task")
		taskQuery := `INSERT INTO tasks (id, phase_id, title, status, priority, created_at, updated_at)
					  VALUES (?, (SELECT id FROM phases WHERE plan_id = ? AND order_num = ?), ?, 'pending', ?, ?, ?)`
		priority := 1
		if t.Type == "high" {
			priority = 3
		} else if t.Type == "medium" {
			priority = 2
		}
		if _, err := s.db.Exec(taskQuery, taskID, planID, t.PhaseIndex, t.Title, priority, now, now); err != nil {
			return nil, fmt.Errorf("failed to create task: %w", err)
		}
		taskIDs = append(taskIDs, taskID)
	}

	return &Response{Result: map[string]interface{}{
		"id":                  planID,
		"plan_id":             planID,
		"title":               planTitle,
		"prd_id":              params.PRDID,
		"phases_created":      len(phaseNames),
		"tasks_created":       len(taskIDs),
		"requirements_linked": len(requirements),
		"status":              "draft",
		"human_review_needed": true,
	}}, nil
}

func (s *Server) handlePlanActivate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	now := time.Now()
	query := `UPDATE plans SET status = 'active', updated_at = ? WHERE id = ?`
	result, err := s.db.Exec(query, now, params.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to activate plan: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("plan not found")
	}

	return &Response{Result: map[string]interface{}{
		"plan_id":    params.PlanID,
		"status":     "active",
		"updated_at": now,
	}}, nil
}

func extractPRDTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || strings.HasPrefix(strings.ToLower(line), "title:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
	}
	return "Untitled PRD"
}

func extractPRDVersion(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "version:"))
		}
	}
	return "1.0"
}

func (s *Server) extractPhasesFromPRD(prdID string) []string {
	var content string
	contentQuery := `SELECT content FROM prds WHERE id = ?`
	row := s.db.QueryRow(contentQuery, prdID)
	if err := row.Scan(&content); err != nil {
		return nil
	}

	skipSections := map[string]bool{
		"tabla de contenidos":  true,
		"control de versiones": true,
		"glosario":             true,
		"resumen":              true,
		"anexos":               true,
		"appendix":             true,
	}

	re := regexp.MustCompile(`(?m)^##?\s*\d+\.?\s*([A-ZÁÉÍÓÚÑ][A-Za-zÁÉÍÓÚÑáéíóúñ\s–-]+?)(?:\n|—|-|\s*$)`)
	matches := re.FindAllStringSubmatch(content, -1)

	var phases []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		title := strings.TrimSpace(match[1])
		title = strings.ReplaceAll(title, "–", "-")
		title = strings.ReplaceAll(title, "  ", " ")
		title = strings.Trim(title, " ")

		lower := strings.ToLower(title)
		if skipSections[lower] || title == "" || len(title) < 4 {
			continue
		}
		if seen[title] {
			continue
		}
		seen[title] = true
		phases = append(phases, title)
		if len(phases) >= 12 {
			break
		}
	}

	if len(phases) < 3 {
		return nil
	}
	return phases
}

func (s *Server) extractRequirements(content string, prdID string) ([]map[string]string, error) {
	requirements := []map[string]string{}

	reWithBracket := regexp.MustCompile(`(?m)^[-*]\s+\[([^\]]+)\]\s*[:.-]?\s*(.+)`)
	reWithoutBracket := regexp.MustCompile(`(?m)^[-*]\s+(.+)`)

	matchesWithBracket := reWithBracket.FindAllStringSubmatch(content, -1)
	for _, match := range matchesWithBracket {
		reqID := match[1]
		title := strings.TrimSpace(match[2])
		if title == "" {
			continue
		}
		reqType := "functional"
		lowerTitle := strings.ToLower(title)
		if strings.Contains(lowerTitle, "performance") ||
			strings.Contains(lowerTitle, "security") ||
			strings.Contains(lowerTitle, "scalability") ||
			strings.Contains(lowerTitle, "compliance") ||
			strings.Contains(lowerTitle, "rate limit") ||
			strings.Contains(lowerTitle, "owasp") {
			reqType = "non-functional"
		}
		requirements = append(requirements, map[string]string{
			"id":    reqID,
			"type":  reqType,
			"title": title,
		})
		reqQuery := `INSERT INTO prd_requirements (id, prd_id, req_type, priority, title, status) VALUES (?, ?, ?, 'medium', ?, 'pending')`
		_, err := s.db.Exec(reqQuery, generateID("req"), prdID, reqType, title)
		if err != nil {
			return nil, fmt.Errorf("failed to insert requirement: %w", err)
		}
	}

	for i, match := range reWithoutBracket.FindAllStringSubmatch(content, -1) {
		title := strings.TrimSpace(match[1])
		if title == "" || strings.HasPrefix(title, "[") {
			continue
		}
		reqID := fmt.Sprintf("REQ-%03d", i+1)
		reqType := "functional"
		lowerTitle := strings.ToLower(title)
		if strings.Contains(lowerTitle, "performance") ||
			strings.Contains(lowerTitle, "security") ||
			strings.Contains(lowerTitle, "scalability") ||
			strings.Contains(lowerTitle, "compliance") ||
			strings.Contains(lowerTitle, "rate limit") ||
			strings.Contains(lowerTitle, "owasp") {
			reqType = "non-functional"
		}
		requirements = append(requirements, map[string]string{
			"id":    reqID,
			"type":  reqType,
			"title": title,
		})
		reqQuery := `INSERT INTO prd_requirements (id, prd_id, req_type, priority, title, status) VALUES (?, ?, ?, 'medium', ?, 'pending')`
		_, err := s.db.Exec(reqQuery, generateID("req"), prdID, reqType, title)
		if err != nil {
			return nil, fmt.Errorf("failed to insert requirement: %w", err)
		}
	}

	if len(requirements) == 0 {
		reSection := regexp.MustCompile(`(?m)^##?\s*\d+\.?\s*([A-ZÁÉÍÓÚÑ][A-Za-zÁÉÍÓÚÑáéíóúñ\s]+?)(?:\n|—|-|$)`)
		sectionMatches := reSection.FindAllStringSubmatch(content, -1)
		seenTitles := make(map[string]bool)

		for i, match := range sectionMatches {
			if len(match) < 2 {
				continue
			}
			title := strings.TrimSpace(match[1])
			title = strings.ReplaceAll(title, "–", "-")
			title = strings.Trim(title, " ")

			skipWords := []string{"Tabla de Contenidos", "Control de Versiones", "Glosario", "Resumen", "Anexos", "Appendix"}
			skip := false
			for _, w := range skipWords {
				if strings.Contains(strings.ToLower(title), strings.ToLower(w)) {
					skip = true
					break
				}
			}
			if skip || title == "" || len(title) < 3 {
				continue
			}
			if seenTitles[title] {
				continue
			}
			seenTitles[title] = true

			reqType := "functional"
			lowerTitle := strings.ToLower(title)
			if strings.Contains(lowerTitle, "seguridad") ||
				strings.Contains(lowerTitle, "arquitectura") ||
				strings.Contains(lowerTitle, "stack") ||
				strings.Contains(lowerTitle, "infraestructura") ||
				strings.Contains(lowerTitle, "operaciones") {
				reqType = "non-functional"
			}

			reqID := fmt.Sprintf("SEC-%03d", i+1)
			requirements = append(requirements, map[string]string{
				"id":    reqID,
				"type":  reqType,
				"title": title,
			})

			reqQuery := `INSERT INTO prd_requirements (id, prd_id, req_type, priority, title, status) VALUES (?, ?, ?, 'medium', ?, 'pending')`
			_, err := s.db.Exec(reqQuery, generateID("req"), prdID, reqType, title)
			if err != nil {
				return nil, fmt.Errorf("failed to insert requirement: %w", err)
			}
		}
	}

	return requirements, nil
}

func osReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Server) getTaskAgents(taskID string) ([]TaskAgent, error) {
	query := `SELECT id, task_id, agent_id, role, status, started_at, completed_at, result, error FROM task_agents WHERE task_id = ?`
	rows, err := s.db.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []TaskAgent
	for rows.Next() {
		var ta TaskAgent
		var role, result, errorMsg sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(&ta.ID, &ta.TaskID, &ta.AgentID, &role, &ta.Status, &startedAt, &completedAt, &result, &errorMsg); err != nil {
			continue
		}
		ta.Role = role.String
		ta.Result = result.String
		ta.Error = errorMsg.String
		if startedAt.Valid {
			ta.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			ta.CompletedAt = &completedAt.Time
		}
		agents = append(agents, ta)
	}
	return agents, rows.Err()
}

func (s *Server) handleTaskAssignAgents(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		TaskID   string   `json:"task_id"`
		AgentIDs []string `json:"agent_ids"`
		Role     string   `json:"role,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if len(params.AgentIDs) == 0 {
		return nil, fmt.Errorf("agent_ids are required")
	}

	now := time.Now()
	role := params.Role
	if role == "" {
		role = "worker"
	}

	agentIDsJSON, _ := json.Marshal(params.AgentIDs)
	updateQuery := `UPDATE tasks SET assigned_agent_ids = ?, updated_at = ? WHERE id = ?`
	if _, err := s.db.Exec(updateQuery, string(agentIDsJSON), now, params.TaskID); err != nil {
		return nil, fmt.Errorf("failed to update task agents: %w", err)
	}

	var assigned []map[string]interface{}
	for _, agentID := range params.AgentIDs {
		taID := generateID("taskagent")
		insertQuery := `INSERT INTO task_agents (id, task_id, agent_id, role, status, started_at) VALUES (?, ?, ?, ?, 'assigned', ?)`
		if _, err := s.db.Exec(insertQuery, taID, params.TaskID, agentID, role, now); err != nil {
			return nil, fmt.Errorf("failed to assign agent %s: %w", agentID, err)
		}
		assigned = append(assigned, map[string]interface{}{
			"task_agent_id": taID,
			"agent_id":      agentID,
			"role":          role,
			"status":        "assigned",
		})
	}

	return &Response{Result: map[string]interface{}{
		"task_id":         params.TaskID,
		"agents_assigned": assigned,
		"updated_at":      now,
	}}, nil
}
