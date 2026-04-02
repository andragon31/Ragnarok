package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handlePlanCreate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SessionID   string   `json:"session_id,omitempty"`
		Title       string   `json:"title"`
		Description string   `json:"description,omitempty"`
		RiskLevel   string   `json:"risk_level,omitempty"`
		Phases      []string `json:"phases,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.RiskLevel == "" {
		params.RiskLevel = "medium"
	}

	plan := &Plan{
		ID:            generateID("plan"),
		SessionID:     params.SessionID,
		Title:         params.Title,
		Description:   params.Description,
		Status:        "draft",
		RiskLevel:     params.RiskLevel,
		QualitySource: "tyr",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	query := `INSERT INTO plans (id, session_id, title, description, status, risk_level, quality_source, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, plan.ID, plan.SessionID, plan.Title, plan.Description, plan.Status, plan.RiskLevel, plan.QualitySource, plan.CreatedAt, plan.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         plan.ID,
			"title":      plan.Title,
			"status":     plan.Status,
			"risk_level": plan.RiskLevel,
			"created_at": plan.CreatedAt,
		},
	}, nil
}

func (s *Server) handlePlanGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, session_id, title, description, status, risk_level, spec_impact, module_hints_used, quality_source, created_at, updated_at, completed_at
			  FROM plans WHERE id = ?`
	plan := &Plan{}
	var sessionID, description, specImpact, moduleHints sql.NullString
	var completedAt sql.NullTime
	err := s.db.QueryRow(query, params.PlanID).Scan(
		&plan.ID, &sessionID, &plan.Title, &description, &plan.Status,
		&plan.RiskLevel, &specImpact, &moduleHints, &plan.QualitySource,
		&plan.CreatedAt, &plan.UpdatedAt, &completedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}
	plan.SessionID = sessionID.String
	plan.Description = description.String
	plan.SpecImpact = specImpact.String
	plan.ModuleHintsUsed = moduleHints.String
	if completedAt.Valid {
		plan.CompletedAt = completedAt.Time
	}

	return &Response{
		Result: plan,
	}, nil
}

func (s *Server) handlePlanList(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Status string `json:"status,omitempty"`
		Limit  int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 50
	}

	var query string
	var args []interface{}

	if params.Status == "" || params.Status == "all" {
		query = `SELECT id, session_id, title, description, status, risk_level, created_at, updated_at
				FROM plans ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{params.Limit}
	} else {
		query = `SELECT id, session_id, title, description, status, risk_level, created_at, updated_at
				FROM plans WHERE status = ? ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{params.Status, params.Limit}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		plan := &Plan{}
		var sessionID, description sql.NullString
		err := rows.Scan(&plan.ID, &sessionID, &plan.Title, &description, &plan.Status, &plan.RiskLevel, &plan.CreatedAt, &plan.UpdatedAt)
		if err != nil {
			return nil, err
		}
		plan.SessionID = sessionID.String
		plan.Description = description.String
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating plans: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"plans": plans,
			"count": len(plans),
		},
	}, nil
}

func (s *Server) handlePlanRevise(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID      string   `json:"plan_id"`
		Title       string   `json:"title,omitempty"`
		Description string   `json:"description,omitempty"`
		NewPhases   []string `json:"new_phases,omitempty"`
		Notes       string   `json:"notes,omitempty"`
		RevisionID  string   `json:"revision_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	var currentPlan struct {
		ID          string
		Title       sql.NullString
		Description sql.NullString
		Status      string
	}
	query := `SELECT id, title, description, status FROM plans WHERE id = ?`
	err := s.db.QueryRow(query, params.PlanID).Scan(&currentPlan.ID, &currentPlan.Title, &currentPlan.Description, &currentPlan.Status)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plan not found: %s", params.PlanID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to query plan: %w", err)
	}

	revisionID := params.RevisionID
	if revisionID == "" {
		revisionID = generateID("rev")
	}

	var changesSummary string
	if params.Title != "" && params.Title != currentPlan.Title.String {
		changesSummary += fmt.Sprintf("title: '%s' -> '%s'; ", currentPlan.Title.String, params.Title)
	}
	if params.Description != "" && params.Description != currentPlan.Description.String {
		changesSummary += fmt.Sprintf("description updated; ")
	}
	if len(params.NewPhases) > 0 {
		changesSummary += fmt.Sprintf("phases updated (%d new phases); ", len(params.NewPhases))
	}

	if changesSummary == "" {
		changesSummary = "no changes detected"
	}

	prevState := fmt.Sprintf("status=%s", currentPlan.Status)
	newState := "needs_revision"

	insertRev := `INSERT INTO plan_revisions (id, plan_id, previous_state, new_state, changes_summary, status, created_at) VALUES (?, ?, ?, ?, ?, 'pending', ?)`
	if _, err := s.db.Exec(insertRev, revisionID, params.PlanID, prevState, newState, changesSummary, time.Now()); err != nil {
		return nil, fmt.Errorf("failed to create plan revision: %w", err)
	}

	if params.RevisionID != "" {
		updateRevQuery := `UPDATE plan_revisions SET status = 'applied', applied_at = ? WHERE id = ?`
		if _, err := s.db.Exec(updateRevQuery, time.Now(), params.RevisionID); err != nil {
			return nil, fmt.Errorf("failed to update revision status: %w", err)
		}
	}

	if params.Title != "" {
		updatePlanQuery := `UPDATE plans SET title = ?, status = 'needs_revision', updated_at = ? WHERE id = ?`
		if _, err := s.db.Exec(updatePlanQuery, params.Title, time.Now(), params.PlanID); err != nil {
			return nil, fmt.Errorf("failed to update plan title: %w", err)
		}
	}

	if params.Description != "" {
		updatePlanQuery := `UPDATE plans SET description = ?, status = 'needs_revision', updated_at = ? WHERE id = ?`
		if _, err := s.db.Exec(updatePlanQuery, params.Description, time.Now(), params.PlanID); err != nil {
			return nil, fmt.Errorf("failed to update plan description: %w", err)
		}
	}

	if len(params.NewPhases) > 0 {
		for i, phaseName := range params.NewPhases {
			phaseID := generateID("phase")
			insertPhase := `INSERT INTO phases (id, plan_id, name, status, order_num, created_at, updated_at) VALUES (?, ?, ?, 'pending', ?, ?, ?)`
			if _, err := s.db.Exec(insertPhase, phaseID, params.PlanID, phaseName, i+1, time.Now(), time.Now()); err != nil {
				return nil, fmt.Errorf("failed to insert phase: %w", err)
			}
		}
		updatePlanQuery := `UPDATE plans SET status = 'needs_revision', updated_at = ? WHERE id = ?`
		if _, err := s.db.Exec(updatePlanQuery, time.Now(), params.PlanID); err != nil {
			return nil, fmt.Errorf("failed to update plan status: %w", err)
		}
	}

	blockerQuery := `INSERT INTO execution_blockers (id, plan_id, reason, type, blocked_at) VALUES (?, ?, ?, 'revision_required', ?)`
	if _, err := s.db.Exec(blockerQuery, generateID("block"), params.PlanID, changesSummary, time.Now()); err != nil {
		return nil, fmt.Errorf("failed to insert blocker: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":              params.PlanID,
			"revision_id":     revisionID,
			"status":          "needs_revision",
			"changes_summary": changesSummary,
			"revised_at":      time.Now(),
			"notes":           params.Notes,
		},
	}, nil
}

func (s *Server) handlePlanAbandon(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
		Reason string `json:"reason,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `UPDATE plans SET status = 'abandoned', updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), params.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to abandon plan: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         params.PlanID,
			"status":     "abandoned",
			"reason":     params.Reason,
			"updated_at": time.Now(),
		},
	}, nil
}

func (s *Server) handlePlanComplete(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `UPDATE plans SET status = 'completed', completed_at = ?, updated_at = ? WHERE id = ?`
	now := time.Now()
	_, err := s.db.Exec(query, now, now, params.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to complete plan: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":           params.PlanID,
			"status":       "completed",
			"completed_at": now,
		},
	}, nil
}

func (s *Server) handleCheckpointOpen(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID    string `json:"plan_id"`
		PhaseID   string `json:"phase_id,omitempty"`
		Type      string `json:"type"`
		RiskLevel string `json:"risk_level,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	cp := &Checkpoint{
		ID:          generateID("cp"),
		PlanID:      params.PlanID,
		PhaseID:     params.PhaseID,
		Type:        params.Type,
		Status:      "open",
		CanContinue: false,
		RiskLevel:   params.RiskLevel,
		CreatedAt:   time.Now(),
	}

	query := `INSERT INTO checkpoints (id, plan_id, phase_id, type, status, can_continue, risk_level, created_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, cp.ID, cp.PlanID, cp.PhaseID, cp.Type, cp.Status, 0, cp.RiskLevel, cp.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to open checkpoint: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":           cp.ID,
			"plan_id":      cp.PlanID,
			"phase_id":     cp.PhaseID,
			"type":         cp.Type,
			"status":       cp.Status,
			"can_continue": false,
			"created_at":   cp.CreatedAt,
		},
	}, nil
}

func (s *Server) handlePhaseStart(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID    string `json:"plan_id"`
		Name      string `json:"name"`
		RiskLevel string `json:"risk_level,omitempty"`
		Module    string `json:"module,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	existingQuery := `SELECT id, status FROM phases WHERE plan_id = ? AND name = ?`
	var existingID string
	var existingStatus string
	err := s.db.QueryRow(existingQuery, params.PlanID, params.Name).Scan(&existingID, &existingStatus)
	if err == nil {
		updateQuery := `UPDATE phases SET status = 'in_progress', updated_at = ? WHERE id = ?`
		_, err := s.db.Exec(updateQuery, time.Now(), existingID)
		if err != nil {
			return nil, fmt.Errorf("failed to update phase: %w", err)
		}
		return &Response{
			Result: map[string]interface{}{
				"id":         existingID,
				"plan_id":    params.PlanID,
				"name":       params.Name,
				"status":     "in_progress",
				"order_num":  0,
				"started_at": time.Now(),
			},
		}, nil
	}

	orderQuery := `SELECT COALESCE(MAX(order_num), 0) + 1 FROM phases WHERE plan_id = ?`
	var orderNum int
	if err := s.db.QueryRow(orderQuery, params.PlanID).Scan(&orderNum); err != nil {
		return nil, fmt.Errorf("failed to get order number: %w", err)
	}

	phase := &Phase{
		ID:        generateID("phase"),
		PlanID:    params.PlanID,
		Name:      params.Name,
		RiskLevel: params.RiskLevel,
		Status:    "in_progress",
		OrderNum:  orderNum,
		Module:    params.Module,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `INSERT INTO phases (id, plan_id, name, risk_level, status, order_num, module, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(query, phase.ID, phase.PlanID, phase.Name, phase.RiskLevel, phase.Status, phase.OrderNum, phase.Module, phase.CreatedAt, phase.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to start phase: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         phase.ID,
			"plan_id":    phase.PlanID,
			"name":       phase.Name,
			"status":     phase.Status,
			"order_num":  phase.OrderNum,
			"started_at": phase.CreatedAt,
		},
	}, nil
}

func (s *Server) handlePhaseReport(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PhaseID     string `json:"phase_id"`
		Content     string `json:"content"`
		WhyApproach string `json:"why_approach,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `UPDATE phases SET status = 'completed', updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), params.PhaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to update phase: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"phase_id":     params.PhaseID,
			"status":       "completed",
			"report":       params.Content,
			"why_approach": params.WhyApproach,
			"completed_at": time.Now(),
		},
	}, nil
}

func detectFeedbackType(content string) string {
	contentLower := strings.ToLower(content)
	if strings.Contains(contentLower, "reject") || strings.Contains(contentLower, "not correct") {
		return "rejection"
	}
	if strings.Contains(contentLower, "approve") || strings.Contains(contentLower, "lgtm") {
		return "approval"
	}
	if strings.Contains(contentLower, "escalate") || strings.Contains(contentLower, "urgent") {
		return "escalation"
	}
	return "general"
}

func isRejectionFeedback(content string, feedbackType string) bool {
	if feedbackType == "rejection" {
		return true
	}
	contentLower := strings.ToLower(content)
	rejectionPhrases := []string{
		"no es correcto", "not correct", "wrong", "incorrect",
		"esto no", "that's wrong", "not right", "doesn't look right",
		"no estoy de acuerdo", "i disagree", "reject", "rechazo",
		"no debería", "should not", "please stop", "hold on",
		"espera", "wait", "hold", "revisa esto", "review this",
	}
	for _, phrase := range rejectionPhrases {
		if strings.Contains(contentLower, phrase) {
			return true
		}
	}
	return false
}

func (s *Server) handlePlanResume(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID     string `json:"plan_id"`
		RevisionID string `json:"revision_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	var blockerCount int
	blockerQuery := `SELECT COUNT(*) FROM execution_blockers WHERE plan_id = ? AND resolved_at IS NULL`
	if err := s.db.QueryRow(blockerQuery, params.PlanID).Scan(&blockerCount); err != nil {
		return nil, fmt.Errorf("failed to check blockers: %w", err)
	}

	if blockerCount > 0 {
		return nil, fmt.Errorf("plan has %d unresolved blockers, resolve them first", blockerCount)
	}

	if params.RevisionID != "" {
		applyRevQuery := `UPDATE plan_revisions SET status = 'applied', applied_at = ? WHERE id = ?`
		if _, err := s.db.Exec(applyRevQuery, time.Now(), params.RevisionID); err != nil {
			return nil, fmt.Errorf("failed to apply revision: %w", err)
		}
	}

	resumeQuery := `UPDATE plans SET status = 'in_progress', updated_at = ? WHERE id = ?`
	if _, err := s.db.Exec(resumeQuery, time.Now(), params.PlanID); err != nil {
		return nil, fmt.Errorf("failed to resume plan: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         params.PlanID,
			"status":     "in_progress",
			"resumed_at": time.Now(),
			"message":    "Plan resumed successfully",
		},
	}, nil
}

func (s *Server) handlePlanBlockers(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	query := `SELECT id, checkpoint_id, reason, type, blocked_at, resolved_at 
	          FROM execution_blockers WHERE plan_id = ? ORDER BY blocked_at DESC`

	rows, err := s.db.Query(query, params.PlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to query blockers: %w", err)
	}
	defer rows.Close()

	var blockers []map[string]interface{}
	for rows.Next() {
		var id, checkpointID, reason, blkType string
		var blockedAt time.Time
		var resolvedAt *time.Time
		if err := rows.Scan(&id, &checkpointID, &reason, &blkType, &blockedAt, &resolvedAt); err != nil {
			continue
		}

		blocker := map[string]interface{}{
			"id":            id,
			"checkpoint_id": checkpointID,
			"reason":        reason,
			"type":          blkType,
			"blocked_at":    blockedAt,
			"resolved":      resolvedAt != nil,
		}
		if resolvedAt != nil {
			blocker["resolved_at"] = resolvedAt
		}
		blockers = append(blockers, blocker)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating blockers: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"plan_id":  params.PlanID,
			"blockers": blockers,
			"count":    len(blockers),
		},
	}, nil
}

func (s *Server) handleCheckpointApprove(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointID string `json:"checkpoint_id"`
		Approver     string `json:"approver,omitempty"`
		Notes        string `json:"notes,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.CheckpointID == "" {
		return nil, fmt.Errorf("checkpoint_id is required")
	}

	var planID string
	cpQuery := `SELECT plan_id FROM checkpoints WHERE id = ?`
	if err := s.db.QueryRow(cpQuery, params.CheckpointID).Scan(&planID); err != nil {
		return nil, fmt.Errorf("failed to get checkpoint plan: %w", err)
	}

	updateCp := `UPDATE checkpoints SET status = 'approved', can_continue = 1, decided_at = ?, decided_by = ?, feedback = ? WHERE id = ?`
	if _, err := s.db.Exec(updateCp, time.Now(), params.Approver, params.Notes, params.CheckpointID); err != nil {
		return nil, fmt.Errorf("failed to update checkpoint: %w", err)
	}

	resolveBlockers := `UPDATE execution_blockers SET resolved_at = ? WHERE plan_id = ? AND checkpoint_id = ? AND type = 'user_rejection'`
	if _, err := s.db.Exec(resolveBlockers, time.Now(), planID, params.CheckpointID); err != nil {
		return nil, fmt.Errorf("failed to resolve blockers: %w", err)
	}

	recordQuery := `INSERT INTO approval_record (id, plan_id, decision, approver, notes, created_at) VALUES (?, ?, 'approved', ?, ?, ?)`
	if _, err := s.db.Exec(recordQuery, generateID("record"), planID, params.Approver, params.Notes, time.Now()); err != nil {
		return nil, fmt.Errorf("failed to create approval record: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":           params.CheckpointID,
			"decision":     "approved",
			"can_continue": true,
			"approved_at":  time.Now(),
			"approved_by":  params.Approver,
		},
	}, nil
}

func (s *Server) handleSpecImpact(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"plan_id": params.PlanID,
			"specs":   []interface{}{},
			"count":   0,
			"note":    "spec impact requires Fenrir integration",
		},
	}, nil
}

func (s *Server) handleQualitySnapshot(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointType string `json:"checkpoint_type"`
		RiskLevel      string `json:"risk_level,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"snapshot":        map[string]interface{}{},
			"source":          "tyr",
			"checkpoint_type": params.CheckpointType,
			"note":            "quality snapshot requires Tyr integration",
		},
	}, nil
}

func (s *Server) handleNotificationSend(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Recipient    string `json:"recipient"`
		Type         string `json:"type"`
		Priority     string `json:"priority,omitempty"`
		Title        string `json:"title"`
		Message      string `json:"message"`
		PlanID       string `json:"plan_id,omitempty"`
		CheckpointID string `json:"checkpoint_id,omitempty"`
		WebhookURL   string `json:"webhook_url,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Recipient == "" || params.Title == "" || params.Message == "" {
		return nil, fmt.Errorf("recipient, title, and message are required")
	}

	if params.Type == "" {
		params.Type = "checkpoint_pending"
	}
	if params.Priority == "" {
		params.Priority = "normal"
	}

	notifID := generateID("notif")
	createdAt := time.Now()

	query := `INSERT INTO notifications (id, recipient, type, priority, title, message, plan_id, checkpoint_id, webhook_url, status, created_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)`
	_, err := s.db.Exec(query, notifID, params.Recipient, params.Type, params.Priority, params.Title, params.Message, params.PlanID, params.CheckpointID, params.WebhookURL, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	if params.WebhookURL != "" {
		go s.sendWebhook(params.WebhookURL, map[string]interface{}{
			"id":         notifID,
			"type":       params.Type,
			"title":      params.Title,
			"message":    params.Message,
			"plan_id":    params.PlanID,
			"priority":   params.Priority,
			"created_at": createdAt,
		})
	}

	return &Response{
		Result: map[string]interface{}{
			"id":          notifID,
			"status":      "sent",
			"sent_at":     createdAt,
			"has_webhook": params.WebhookURL != "",
		},
	}, nil
}

func (s *Server) sendWebhook(url string, payload map[string]interface{}) {
	data, _ := json.Marshal(payload)
	http.Post(url, "application/json", strings.NewReader(string(data)))
}

func (s *Server) handleNotificationList(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Status    string `json:"status,omitempty"`
		Recipient string `json:"recipient,omitempty"`
		PlanID    string `json:"plan_id,omitempty"`
		Limit     int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 50
	}

	query := `SELECT id, recipient, type, priority, title, message, plan_id, checkpoint_id, status, sent_at, created_at 
	          FROM notifications WHERE 1=1`
	args := []interface{}{}

	if params.Status != "" {
		query += " AND status = ?"
		args = append(args, params.Status)
	}
	if params.Recipient != "" {
		query += " AND recipient = ?"
		args = append(args, params.Recipient)
	}
	if params.PlanID != "" {
		query += " AND plan_id = ?"
		args = append(args, params.PlanID)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, params.Limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var id, recipient, notifType, priority, title, message, planID, cpID string
		var status string
		var sentAt, createdAt *time.Time
		if err := rows.Scan(&id, &recipient, &notifType, &priority, &title, &message, &planID, &cpID, &status, &sentAt, &createdAt); err != nil {
			continue
		}

		n := map[string]interface{}{
			"id":         id,
			"recipient":  recipient,
			"type":       notifType,
			"priority":   priority,
			"title":      title,
			"message":    message,
			"plan_id":    planID,
			"status":     status,
			"created_at": createdAt,
		}
		if sentAt != nil {
			n["sent_at"] = sentAt
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notifications: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"notifications": notifications,
			"count":         len(notifications),
		},
	}, nil
}

func (s *Server) handlePlanDependencies(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID          string `json:"plan_id"`
		DependsOnPlanID string `json:"depends_on_plan_id,omitempty"`
		DependencyType  string `json:"dependency_type,omitempty"`
		Action          string `json:"action,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	if params.Action == "list" {
		query := `SELECT pd.id, pd.plan_id, pd.depends_on_plan_id, pd.dependency_type, pd.status,
		                 p.title as dependent_title
		          FROM plan_dependencies pd
		          JOIN plans p ON pd.depends_on_plan_id = p.id
		          WHERE pd.plan_id = ?`
		rows, err := s.db.Query(query, params.PlanID)
		if err != nil {
			return nil, fmt.Errorf("failed to list dependencies: %w", err)
		}
		defer rows.Close()

		var deps []map[string]interface{}
		for rows.Next() {
			var id, planID, dependsOn, depType, status, depTitle string
			if err := rows.Scan(&id, &planID, &dependsOn, &depType, &status, &depTitle); err != nil {
				continue
			}
			deps = append(deps, map[string]interface{}{
				"id":               id,
				"plan_id":          planID,
				"depends_on":       dependsOn,
				"dependency_type":  depType,
				"status":           status,
				"depends_on_title": depTitle,
			})
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating dependencies: %w", err)
		}

		return &Response{
			Result: map[string]interface{}{
				"plan_id":      params.PlanID,
				"dependencies": deps,
				"count":        len(deps),
			},
		}, nil
	}

	if params.DependsOnPlanID != "" {
		if params.DependencyType == "" {
			params.DependencyType = "blocking"
		}

		depID := generateID("dep")
		query := `INSERT INTO plan_dependencies (id, plan_id, depends_on_plan_id, dependency_type, status) VALUES (?, ?, ?, ?, 'pending')`
		_, err := s.db.Exec(query, depID, params.PlanID, params.DependsOnPlanID, params.DependencyType)
		if err != nil {
			return nil, fmt.Errorf("failed to add dependency: %w", err)
		}

		return &Response{
			Result: map[string]interface{}{
				"id":                 depID,
				"plan_id":            params.PlanID,
				"depends_on_plan_id": params.DependsOnPlanID,
				"dependency_type":    params.DependencyType,
				"status":             "pending",
			},
		}, nil
	}

	return &Response{
		Result: map[string]interface{}{
			"plan_id": params.PlanID,
			"message": "Use action=list to view dependencies or provide depends_on_plan_id to add",
		},
	}, nil
}
