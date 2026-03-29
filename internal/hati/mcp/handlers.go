package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

	var currentPlan Plan
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
	if params.Title != "" && params.Title != currentPlan.Title {
		changesSummary += fmt.Sprintf("title: '%s' -> '%s'; ", currentPlan.Title, params.Title)
	}
	if params.Description != "" && params.Description != currentPlan.Description {
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

func (s *Server) handlePlanCompleteness(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT COUNT(*), SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) FROM phases WHERE plan_id = ?`
	var total, completed int
	if err := s.db.QueryRow(query, params.PlanID).Scan(&total, &completed); err != nil {
		return nil, fmt.Errorf("failed to get phase progress: %w", err)
	}

	score := 0.0
	if total > 0 {
		score = float64(completed) / float64(total)
	}

	return &Response{
		Result: map[string]interface{}{
			"plan_id":      params.PlanID,
			"total_phases": total,
			"completed":    completed,
			"score":        score,
		},
	}, nil
}

func (s *Server) handlePlanQuality(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT plan_completeness, execution_quality, overall_score FROM plan_quality_scores WHERE plan_id = ?`
	var pqs struct {
		PlanCompleteness float64 `json:"plan_completeness"`
		ExecutionQuality float64 `json:"execution_quality"`
		OverallScore     float64 `json:"overall_score"`
	}

	err := s.db.QueryRow(query, params.PlanID).Scan(&pqs.PlanCompleteness, &pqs.ExecutionQuality, &pqs.OverallScore)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"plan_id": params.PlanID,
				"score":   0.0,
				"note":    "no quality scores calculated yet",
			},
		}, nil
	}

	return &Response{
		Result: map[string]interface{}{
			"plan_id":           params.PlanID,
			"plan_completeness": pqs.PlanCompleteness,
			"execution_quality": pqs.ExecutionQuality,
			"overall_score":     pqs.OverallScore,
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

func (s *Server) handleCheckpointDecide(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointID string `json:"checkpoint_id"`
		Decision     string `json:"decision"`
		Approver     string `json:"approver,omitempty"`
		Notes        string `json:"notes,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	canContinue := params.Decision == "approved"

	query := `UPDATE checkpoints SET status = ?, decided_at = ?, decided_by = ?, feedback = ?, can_continue = ? WHERE id = ?`
	_, err := s.db.Exec(query, params.Decision, time.Now(), params.Approver, params.Notes, boolToInt(canContinue), params.CheckpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to decide checkpoint: %w", err)
	}

	record := &ApprovalRecord{
		ID:        generateID("record"),
		PlanID:    "",
		Decision:  params.Decision,
		Approver:  params.Approver,
		Notes:     params.Notes,
		CreatedAt: time.Now(),
	}

	recQuery := `INSERT INTO approval_record (id, plan_id, decision, approver, notes, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := s.db.Exec(recQuery, record.ID, record.PlanID, record.Decision, record.Approver, record.Notes, record.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to create approval record: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":           params.CheckpointID,
			"decision":     params.Decision,
			"can_continue": canContinue,
			"decided_at":   time.Now(),
		},
	}, nil
}

func (s *Server) handleCheckpointStatus(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointID string `json:"checkpoint_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, plan_id, phase_id, type, status, can_continue, risk_level, created_at, decided_at, decided_by, feedback
			  FROM checkpoints WHERE id = ?`
	cp := &Checkpoint{}
	var decidedAt, decidedBy, feedback sql.NullString
	var canContinueInt int
	err := s.db.QueryRow(query, params.CheckpointID).Scan(
		&cp.ID, &cp.PlanID, &cp.PhaseID, &cp.Type, &cp.Status, &canContinueInt,
		&cp.RiskLevel, &cp.CreatedAt, &decidedAt, &decidedBy, &feedback,
	)
	if err != nil {
		return nil, fmt.Errorf("checkpoint not found: %w", err)
	}
	cp.CanContinue = canContinueInt == 1
	cp.DecidedAt, _ = time.Parse("2006-01-02 15:04:05", decidedAt.String)
	cp.DecidedBy = decidedBy.String
	cp.Feedback = feedback.String

	return &Response{
		Result: cp,
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
	_, err := s.db.Exec(query, phase.ID, phase.PlanID, phase.Name, phase.RiskLevel, phase.Status, phase.OrderNum, phase.Module, phase.CreatedAt, phase.UpdatedAt)
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

func (s *Server) handleFeedbackRequest(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointID string `json:"checkpoint_id"`
		Type         string `json:"type"`
		Content      string `json:"content"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	fb := &Feedback{
		ID:           generateID("fb"),
		CheckpointID: params.CheckpointID,
		Type:         params.Type,
		Content:      params.Content,
		CreatedAt:    time.Now(),
	}

	query := `INSERT INTO feedback (id, checkpoint_id, type, content, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, fb.ID, fb.CheckpointID, fb.Type, fb.Content, fb.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create feedback request: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":            fb.ID,
			"checkpoint_id": fb.CheckpointID,
			"type":          fb.Type,
			"status":        "pending",
			"created_at":    fb.CreatedAt,
		},
	}, nil
}

func (s *Server) handleFeedbackReceive(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		FeedbackID string `json:"feedback_id"`
		Content    string `json:"content"`
		Author     string `json:"author,omitempty"`
		Type       string `json:"type,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	feedbackType := params.Type
	if feedbackType == "" {
		feedbackType = detectFeedbackType(params.Content)
	}

	isRejection := isRejectionFeedback(params.Content, feedbackType)

	result := map[string]interface{}{
		"feedback_id":  params.FeedbackID,
		"received":     true,
		"content":      params.Content,
		"author":       params.Author,
		"type":         feedbackType,
		"is_rejection": isRejection,
		"received_at":  time.Now(),
	}

	if isRejection {
		var cp Checkpoint
		cpQuery := `SELECT id, plan_id, phase_id FROM checkpoints WHERE id = ?`
		err := s.db.QueryRow(cpQuery, params.FeedbackID).Scan(&cp.ID, &cp.PlanID, &cp.PhaseID)
		if err == nil && cp.PlanID != "" {
			updateCpQuery := `UPDATE checkpoints SET status = 'rejected', can_continue = 0, feedback = ? WHERE id = ?`
			if _, err := s.db.Exec(updateCpQuery, params.Content, cp.ID); err != nil {
				return nil, fmt.Errorf("failed to update checkpoint: %w", err)
			}

			blockerID := generateID("block")
			blockReason := fmt.Sprintf("User rejection: %s", params.Content)
			blockerQuery := `INSERT INTO execution_blockers (id, plan_id, checkpoint_id, reason, type, blocked_at) VALUES (?, ?, ?, ?, 'user_rejection', ?)`
			if _, err := s.db.Exec(blockerQuery, blockerID, cp.PlanID, cp.ID, blockReason, time.Now()); err != nil {
				return nil, fmt.Errorf("failed to insert blocker: %w", err)
			}

			abandonQuery := `UPDATE plans SET status = 'needs_revision', updated_at = ? WHERE id = ?`
			if _, err := s.db.Exec(abandonQuery, time.Now(), cp.PlanID); err != nil {
				return nil, fmt.Errorf("failed to update plan status: %w", err)
			}

			result["action_triggered"] = "plan_revise"
			result["plan_id"] = cp.PlanID
			result["checkpoint_id"] = cp.ID
			result["message"] = "Rejection detected. Plan marked for revision. Use plan_revise to update and plan_restart to continue."
		}
	}

	return &Response{Result: result}, nil
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

func (s *Server) handlePlanRestart(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID     string `json:"plan_id"`
		FromPhase  int    `json:"from_phase,omitempty"`
		ClearState bool   `json:"clear_state,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	var currentStatus string
	statusQuery := `SELECT status FROM plans WHERE id = ?`
	if err := s.db.QueryRow(statusQuery, params.PlanID).Scan(&currentStatus); err != nil {
		return nil, fmt.Errorf("failed to get plan status: %w", err)
	}

	if currentStatus == "needs_revision" {
		return nil, fmt.Errorf("plan has pending revisions, use plan_revise first")
	}

	if currentStatus == "in_progress" {
		return nil, fmt.Errorf("plan is already in progress")
	}

	startPhase := 1
	if params.FromPhase > 0 {
		startPhase = params.FromPhase
	}

	resetPhasesQuery := `UPDATE phases SET status = 'pending' WHERE plan_id = ? AND order_num >= ?`
	if _, err := s.db.Exec(resetPhasesQuery, params.PlanID, startPhase); err != nil {
		return nil, fmt.Errorf("failed to reset phases: %w", err)
	}

	clearBlockersQuery := `UPDATE execution_blockers SET resolved_at = ? WHERE plan_id = ? AND resolved_at IS NULL`
	if _, err := s.db.Exec(clearBlockersQuery, time.Now(), params.PlanID); err != nil {
		return nil, fmt.Errorf("failed to clear blockers: %w", err)
	}

	updatePlanQuery := `UPDATE plans SET status = 'in_progress', updated_at = ? WHERE id = ?`
	if _, err := s.db.Exec(updatePlanQuery, time.Now(), params.PlanID); err != nil {
		return nil, fmt.Errorf("failed to update plan status: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":           params.PlanID,
			"status":       "in_progress",
			"restarted_at": time.Now(),
			"from_phase":   startPhase,
			"message":      fmt.Sprintf("Plan restarted from phase %d", startPhase),
		},
	}, nil
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

func (s *Server) handleFeedbackEscalate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		FeedbackID string `json:"feedback_id"`
		Reason     string `json:"reason"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"feedback_id":  params.FeedbackID,
			"status":       "escalated",
			"reason":       params.Reason,
			"escalated_at": time.Now(),
		},
	}, nil
}

func (s *Server) handleRecordList(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id,omitempty"`
		Limit  int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 50
	}

	query := `SELECT id, plan_id, decision, approver, notes, created_at FROM approval_record WHERE (? = '' OR plan_id = ?) LIMIT ?`
	rows, err := s.db.Query(query, params.PlanID, params.PlanID, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	var records []*ApprovalRecord
	for rows.Next() {
		rec := &ApprovalRecord{}
		var approver, notes sql.NullString
		err := rows.Scan(&rec.ID, &rec.PlanID, &rec.Decision, &approver, &notes, &rec.CreatedAt)
		if err != nil {
			return nil, err
		}
		rec.Approver = approver.String
		rec.Notes = notes.String
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating records: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"records": records,
			"count":   len(records),
		},
	}, nil
}

func (s *Server) handleRecordGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		RecordID string `json:"record_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, plan_id, decision, approver, notes, spec_deltas, created_at FROM approval_record WHERE id = ?`
	rec := &ApprovalRecord{}
	var specDeltas sql.NullString
	err := s.db.QueryRow(query, params.RecordID).Scan(&rec.ID, &rec.PlanID, &rec.Decision, &rec.Approver, &rec.Notes, &specDeltas, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("record not found: %w", err)
	}
	rec.SpecDeltas = specDeltas.String

	return &Response{
		Result: rec,
	}, nil
}

func (s *Server) handleRecordExport(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
		Format string `json:"format,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Format == "" {
		params.Format = "markdown"
	}

	return &Response{
		Result: map[string]interface{}{
			"plan_id":  params.PlanID,
			"format":   params.Format,
			"exported": true,
			"note":     "export functionality pending implementation",
		},
	}, nil
}

func (s *Server) handleModuleHints(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Modules []string `json:"modules"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	hints := make([]map[string]interface{}, 0)
	for _, mod := range params.Modules {
		hints = append(hints, map[string]interface{}{
			"module":            mod,
			"source":            "none",
			"action":            "none",
			"message":           "",
			"applied_to_phases": []string{},
		})
	}

	return &Response{
		Result: map[string]interface{}{
			"hints": hints,
			"count": len(hints),
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

func (s *Server) handleLearningAnswer(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointID string `json:"checkpoint_id"`
		Answer       string `json:"answer"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"checkpoint_id": params.CheckpointID,
			"answer":        params.Answer,
			"recorded":      true,
		},
	}, nil
}

func (s *Server) handleHatiStatus(ctx context.Context, req *Request) (*Response, error) {
	var totalPlans, activePlans, completedPlans int

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM plans`).Scan(&totalPlans); err != nil {
		totalPlans = -1
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM plans WHERE status = 'draft' OR status = 'in_progress'`).Scan(&activePlans); err != nil {
		activePlans = -1
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM plans WHERE status = 'completed'`).Scan(&completedPlans); err != nil {
		completedPlans = -1
	}

	return &Response{
		Result: map[string]interface{}{
			"status":          "operational",
			"total_plans":     totalPlans,
			"active_plans":    activePlans,
			"completed_plans": completedPlans,
			"version":         "1.4.0",
		},
	}, nil
}

func (s *Server) handleHatiStats(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Metric string `json:"metric,omitempty"`
	}

	json.Unmarshal(req.Params, &params)

	return &Response{
		Result: map[string]interface{}{
			"metric":         params.Metric,
			"fast_approvals": 0,
			"quality":        0.0,
			"rejections":     0,
			"learning":       0,
			"reliability":    0.0,
			"specs":          0,
			"note":           "stats require more implementation",
		},
	}, nil
}

func (s *Server) handleHatiCommitInfo(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CommitHash string `json:"commit_hash"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, plan_id, commit_hash, created_at FROM commit_registry WHERE commit_hash = ?`
	var cr struct {
		ID         string    `json:"id"`
		PlanID     string    `json:"plan_id"`
		CommitHash string    `json:"commit_hash"`
		CreatedAt  time.Time `json:"created_at"`
	}

	err := s.db.QueryRow(query, params.CommitHash).Scan(&cr.ID, &cr.PlanID, &cr.CommitHash, &cr.CreatedAt)
	if err != nil {
		return &Response{
			Result: map[string]interface{}{
				"commit_hash": params.CommitHash,
				"plan_id":     "",
				"found":       false,
			},
		}, nil
	}

	return &Response{
		Result: map[string]interface{}{
			"commit_hash": cr.CommitHash,
			"plan_id":     cr.PlanID,
			"found":       true,
		},
	}, nil
}

func (s *Server) handleHatiRegisterCommit(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID     string `json:"plan_id"`
		CommitHash string `json:"commit_hash"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	cr := &struct {
		ID         string
		PlanID     string
		CommitHash string
		CreatedAt  time.Time
	}{
		ID:         generateID("commit"),
		PlanID:     params.PlanID,
		CommitHash: params.CommitHash,
		CreatedAt:  time.Now(),
	}

	query := `INSERT INTO commit_registry (id, plan_id, commit_hash, created_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.Exec(query, cr.ID, cr.PlanID, cr.CommitHash, cr.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to register commit: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":          cr.ID,
			"plan_id":     cr.PlanID,
			"commit_hash": cr.CommitHash,
			"registered":  true,
			"created_at":  cr.CreatedAt,
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

func (s *Server) handleNotificationAck(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		NotificationID string `json:"notification_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	now := time.Now()
	query := `UPDATE notifications SET status = 'acknowledged', sent_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, now, params.NotificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to acknowledge notification: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":              params.NotificationID,
			"status":          "acknowledged",
			"acknowledged_at": now,
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

func (s *Server) handlePlanRecover(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID        string   `json:"plan_id"`
		AgentID       string   `json:"agent_id,omitempty"`
		ModifiedFiles []string `json:"modified_files,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.PlanID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	var currentPhase struct {
		ID       string
		Name     string
		Status   string
		OrderNum int
	}
	phaseQuery := `SELECT id, name, status, order_num FROM phases WHERE plan_id = ? AND status = 'in_progress' ORDER BY order_num LIMIT 1`
	err := s.db.QueryRow(phaseQuery, params.PlanID).Scan(&currentPhase.ID, &currentPhase.Name, &currentPhase.Status, &currentPhase.OrderNum)

	var recoveryState string
	var recoveryNeeded bool

	if err == sql.ErrNoRows {
		var planStatus string
		if err := s.db.QueryRow(`SELECT status FROM plans WHERE id = ?`, params.PlanID).Scan(&planStatus); err != nil {
			recoveryState = "unknown"
			recoveryNeeded = false
			return &Response{Result: map[string]interface{}{
				"recovery_needed": recoveryNeeded,
				"recovery_state":  recoveryState,
			}}, nil
		}

		if planStatus == "in_progress" {
			recoveryState = "agent_disconnected"
			recoveryNeeded = true
		} else {
			recoveryState = "normal"
			recoveryNeeded = false
		}
	} else if err != nil {
		recoveryState = "error_querying_phase"
		recoveryNeeded = true
	} else {
		if params.AgentID != "" {
			recoveryState = "agent_active"
			recoveryNeeded = false
		} else {
			recoveryState = "phase_in_progress_no_agent"
			recoveryNeeded = true
		}
	}

	var filesJSON string
	if len(params.ModifiedFiles) > 0 {
		filesData, _ := json.Marshal(params.ModifiedFiles)
		filesJSON = string(filesData)
	}

	recoveryID := generateID("recovery")
	createdAt := time.Now()

	insertQuery := `INSERT INTO plan_recovery (id, plan_id, phase_id, agent_id, detected_state, expected_state, modified_files, recovery_needed, created_at) 
	                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := s.db.Exec(insertQuery, recoveryID, params.PlanID, currentPhase.ID, params.AgentID, recoveryState, "completed", filesJSON, boolToInt(recoveryNeeded), createdAt); err != nil {
		return nil, fmt.Errorf("failed to insert recovery record: %w", err)
	}

	result := map[string]interface{}{
		"plan_id":         params.PlanID,
		"current_phase":   currentPhase.Name,
		"detected_state":  recoveryState,
		"recovery_needed": recoveryNeeded,
		"recovery_id":     recoveryID,
	}

	if recoveryNeeded {
		result["suggested_actions"] = []string{
			"plan_restart --from-phase " + strconv.Itoa(currentPhase.OrderNum),
			"plan_abandon --plan_id " + params.PlanID + " --reason agent_disconnected",
		}
		result["message"] = "Recovery needed. Agent appears disconnected or phase is inconsistent."
	} else {
		result["message"] = "Plan appears healthy, no recovery needed."
	}

	return &Response{Result: result}, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Server) handlePlanLock(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID    string `json:"plan_id"`
		PhaseID   string `json:"phase_id,omitempty"`
		AgentID   string `json:"agent_id"`
		ExpiresIn int    `json:"expires_in_minutes,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.PlanID == "" || params.AgentID == "" {
		return nil, fmt.Errorf("plan_id and agent_id are required")
	}

	var existingLock struct {
		AgentID string
	}
	lockQuery := `SELECT agent_id FROM agent_locks WHERE plan_id = ? AND (phase_id = ? OR (? = '' AND phase_id IS NULL)) AND expires_at > ?`
	now := time.Now()
	err := s.db.QueryRow(lockQuery, params.PlanID, params.PhaseID, params.PhaseID, now).Scan(&existingLock.AgentID)

	if err == nil {
		if existingLock.AgentID != params.AgentID {
			return nil, fmt.Errorf("plan is locked by agent %s", existingLock.AgentID)
		}
		return &Response{
			Result: map[string]interface{}{
				"plan_id":  params.PlanID,
				"phase_id": params.PhaseID,
				"locked":   true,
				"message":  "Lock already held by this agent",
			},
		}, nil
	}

	lockID := generateID("lock")
	expiresAt := now.Add(time.Duration(params.ExpiresIn) * time.Minute)
	if params.ExpiresIn <= 0 {
		expiresAt = now.Add(30 * time.Minute)
	}

	insertQuery := `INSERT INTO agent_locks (id, plan_id, phase_id, agent_id, locked_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(insertQuery, lockID, params.PlanID, params.PhaseID, params.AgentID, now, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"lock_id":    lockID,
			"plan_id":    params.PlanID,
			"phase_id":   params.PhaseID,
			"agent_id":   params.AgentID,
			"locked":     true,
			"expires_at": expiresAt,
		},
	}, nil
}

func (s *Server) handlePlanUnlock(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID  string `json:"plan_id"`
		PhaseID string `json:"phase_id,omitempty"`
		AgentID string `json:"agent_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.PlanID == "" || params.AgentID == "" {
		return nil, fmt.Errorf("plan_id and agent_id are required")
	}

	query := `DELETE FROM agent_locks WHERE plan_id = ? AND agent_id = ? AND (phase_id = ? OR (? = '' AND phase_id IS NULL))`
	result, err := s.db.Exec(query, params.PlanID, params.AgentID, params.PhaseID, params.PhaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to release lock: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	return &Response{
		Result: map[string]interface{}{
			"plan_id":  params.PlanID,
			"phase_id": params.PhaseID,
			"agent_id": params.AgentID,
			"released": rowsAffected > 0,
		},
	}, nil
}

func (s *Server) handleAgentRegisterWork(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		AgentID   string `json:"agent_id"`
		AgentName string `json:"agent_name,omitempty"`
		PlanID    string `json:"plan_id"`
		PhaseID   string `json:"phase_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.AgentID == "" || params.PlanID == "" {
		return nil, fmt.Errorf("agent_id and plan_id are required")
	}

	unregQuery := `UPDATE agent_work SET status = 'inactive', heartbeat_at = ? WHERE agent_id = ? AND status = 'active'`
	s.db.Exec(unregQuery, time.Now(), params.AgentID)

	workID := generateID("work")
	now := time.Now()
	insertQuery := `INSERT INTO agent_work (id, agent_id, agent_name, plan_id, phase_id, status, started_at, heartbeat_at) VALUES (?, ?, ?, ?, ?, 'active', ?, ?)`
	_, err := s.db.Exec(insertQuery, workID, params.AgentID, params.AgentName, params.PlanID, params.PhaseID, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to register work: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"work_id":    workID,
			"agent_id":   params.AgentID,
			"plan_id":    params.PlanID,
			"phase_id":   params.PhaseID,
			"status":     "active",
			"started_at": now,
		},
	}, nil
}

func (s *Server) handleAgentUnregisterWork(ctx context.Context, req *Request) (*Response, error) {
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
	query := `UPDATE agent_work SET status = 'completed', heartbeat_at = ? WHERE agent_id = ? AND status = 'active'`
	result, err := s.db.Exec(query, now, params.AgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to unregister: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	return &Response{
		Result: map[string]interface{}{
			"agent_id":    params.AgentID,
			"completed":   rowsAffected > 0,
			"finished_at": now,
		},
	}, nil
}

func (s *Server) handleAgentListWork(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id,omitempty"`
		Status string `json:"status,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, agent_id, agent_name, plan_id, phase_id, status, started_at, heartbeat_at FROM agent_work WHERE 1=1`
	args := []interface{}{}

	if params.PlanID != "" {
		query += " AND plan_id = ?"
		args = append(args, params.PlanID)
	}
	if params.Status != "" {
		query += " AND status = ?"
		args = append(args, params.Status)
	}

	query += " ORDER BY started_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list work: %w", err)
	}
	defer rows.Close()

	var works []map[string]interface{}
	for rows.Next() {
		var id, agentID, agentName, planID, phaseID, status string
		var startedAt, heartbeatAt time.Time
		if err := rows.Scan(&id, &agentID, &agentName, &planID, &phaseID, &status, &startedAt, &heartbeatAt); err != nil {
			continue
		}

		work := map[string]interface{}{
			"id":           id,
			"agent_id":     agentID,
			"agent_name":   agentName,
			"plan_id":      planID,
			"phase_id":     phaseID,
			"status":       status,
			"started_at":   startedAt,
			"heartbeat_at": heartbeatAt,
		}
		works = append(works, work)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating works: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"works": works,
			"count": len(works),
		},
	}, nil
}

func (s *Server) handleCheckpointSetSLA(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointID         string `json:"checkpoint_id"`
		SLAHours             int    `json:"sla_hours"`
		EscalationRecipients string `json:"escalation_recipients,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.CheckpointID == "" || params.SLAHours <= 0 {
		return nil, fmt.Errorf("checkpoint_id and sla_hours (>0) are required")
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(params.SLAHours) * time.Hour)

	slaID := generateID("sla")
	insertQuery := `INSERT INTO checkpoint_sla (id, checkpoint_id, sla_hours, escalation_recipients, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(insertQuery, slaID, params.CheckpointID, params.SLAHours, params.EscalationRecipients, now, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to set SLA: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"sla_id":        slaID,
			"checkpoint_id": params.CheckpointID,
			"sla_hours":     params.SLAHours,
			"expires_at":    expiresAt,
			"created_at":    now,
		},
	}, nil
}

func (s *Server) handleCheckpointEscalate(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		CheckpointID         string `json:"checkpoint_id"`
		EscalationRecipients string `json:"escalation_recipients"`
		Reason               string `json:"reason,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.CheckpointID == "" {
		return nil, fmt.Errorf("checkpoint_id is required")
	}

	var slaID string
	var slaHours int
	var recipients string
	slaQuery := `SELECT id, sla_hours, escalation_recipients FROM checkpoint_sla WHERE checkpoint_id = ?`
	err := s.db.QueryRow(slaQuery, params.CheckpointID).Scan(&slaID, &slaHours, &recipients)
	if err != nil {
		recipients = params.EscalationRecipients
	}

	now := time.Now()
	if slaID != "" {
		updateQuery := `UPDATE checkpoint_sla SET escalated_at = ? WHERE id = ?`
		if _, err := s.db.Exec(updateQuery, now, slaID); err != nil {
			return nil, fmt.Errorf("failed to update SLA escalated_at: %w", err)
		}
	}

	var cpTitle string
	var planID string
	cpQuery := `SELECT type, plan_id FROM checkpoints WHERE id = ?`
	if err := s.db.QueryRow(cpQuery, params.CheckpointID).Scan(&cpTitle, &planID); err != nil {
		cpTitle = "Unknown"
		planID = ""
	}

	if recipients != "" {
		go s.sendWebhook(recipients, map[string]interface{}{
			"type":         "sla_escalation",
			"checkpoint":   params.CheckpointID,
			"plan_id":      planID,
			"reason":       params.Reason,
			"message":      fmt.Sprintf("Checkpoint %s has exceeded SLA and requires immediate attention", cpTitle),
			"escalated_at": now,
		})
	}

	return &Response{
		Result: map[string]interface{}{
			"checkpoint_id": params.CheckpointID,
			"escalated":     true,
			"escalated_at":  now,
			"recipients":    recipients,
		},
	}, nil
}

func (s *Server) handleCheckpointCheckSLA(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	now := time.Now()
	query := `SELECT cs.id, cs.checkpoint_id, cs.sla_hours, cs.escalation_recipients, cs.created_at, cs.expires_at, cs.escalated_at, c.type, c.plan_id
	          FROM checkpoint_sla cs
	          JOIN checkpoints c ON cs.checkpoint_id = c.id
	          WHERE cs.expires_at < ? AND cs.escalated_at IS NULL AND c.status = 'pending'`

	args := []interface{}{now}
	if params.PlanID != "" {
		query += " AND c.plan_id = ?"
		args = append(args, params.PlanID)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to check SLA: %w", err)
	}
	defer rows.Close()

	var expired []map[string]interface{}
	for rows.Next() {
		var id, cpID, recipients, cpType, pid string
		var slaHours int
		var createdAt, expiresAt time.Time
		var escalatedAt *time.Time
		if err := rows.Scan(&id, &cpID, &slaHours, &recipients, &createdAt, &expiresAt, &escalatedAt, &cpType, &pid); err != nil {
			continue
		}

		item := map[string]interface{}{
			"sla_id":        id,
			"checkpoint_id": cpID,
			"sla_hours":     slaHours,
			"expires_at":    expiresAt,
			"overdue_hours": int(now.Sub(expiresAt).Hours()),
			"type":          cpType,
			"plan_id":       pid,
		}

		if escalatedAt != nil {
			item["escalated_at"] = escalatedAt
		}

		expired = append(expired, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expired checkpoints: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"expired_checkpoints": expired,
			"count":               len(expired),
			"checked_at":          now,
		},
	}, nil
}
