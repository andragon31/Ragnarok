package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	var specImpact, moduleHints sql.NullString
	var completedAt sql.NullTime
	err := s.db.QueryRow(query, params.PlanID).Scan(
		&plan.ID, &plan.SessionID, &plan.Title, &plan.Description, &plan.Status,
		&plan.RiskLevel, &specImpact, &moduleHints, &plan.QualitySource,
		&plan.CreatedAt, &plan.UpdatedAt, &completedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}
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

	query := `SELECT id, session_id, title, description, status, risk_level, created_at, updated_at
			  FROM plans WHERE (? = '' OR status = ?) ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.Query(query, params.Status, params.Status, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	defer rows.Close()

	var plans []*Plan
	for rows.Next() {
		plan := &Plan{}
		err := rows.Scan(&plan.ID, &plan.SessionID, &plan.Title, &plan.Description, &plan.Status, &plan.RiskLevel, &plan.CreatedAt, &plan.UpdatedAt)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
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
	s.db.Exec(insertRev, revisionID, params.PlanID, prevState, newState, changesSummary, time.Now())

	if params.RevisionID != "" {
		updateRevQuery := `UPDATE plan_revisions SET status = 'applied', applied_at = ? WHERE id = ?`
		s.db.Exec(updateRevQuery, time.Now(), params.RevisionID)
	}

	if params.Title != "" {
		updatePlanQuery := `UPDATE plans SET title = ?, status = 'needs_revision', updated_at = ? WHERE id = ?`
		s.db.Exec(updatePlanQuery, params.Title, time.Now(), params.PlanID)
	}

	if params.Description != "" {
		updatePlanQuery := `UPDATE plans SET description = ?, status = 'needs_revision', updated_at = ? WHERE id = ?`
		s.db.Exec(updatePlanQuery, params.Description, time.Now(), params.PlanID)
	}

	if len(params.NewPhases) > 0 {
		for i, phaseName := range params.NewPhases {
			phaseID := generateID("phase")
			insertPhase := `INSERT INTO phases (id, plan_id, name, status, order_num, created_at, updated_at) VALUES (?, ?, ?, 'pending', ?, ?, ?)`
			s.db.Exec(insertPhase, phaseID, params.PlanID, phaseName, i+1, time.Now(), time.Now())
		}
		updatePlanQuery := `UPDATE plans SET status = 'needs_revision', updated_at = ? WHERE id = ?`
		s.db.Exec(updatePlanQuery, time.Now(), params.PlanID)
	}

	blockerQuery := `INSERT INTO execution_blockers (id, plan_id, reason, type, blocked_at) VALUES (?, ?, ?, 'revision_required', ?)`
	s.db.Exec(blockerQuery, generateID("block"), params.PlanID, changesSummary, time.Now())

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
	s.db.QueryRow(query, params.PlanID).Scan(&total, &completed)

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
	s.db.Exec(recQuery, record.ID, record.PlanID, record.Decision, record.Approver, record.Notes, record.CreatedAt)

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
	s.db.QueryRow(orderQuery, params.PlanID).Scan(&orderNum)

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
			s.db.Exec(updateCpQuery, params.Content, cp.ID)

			blockerID := generateID("block")
			blockReason := fmt.Sprintf("User rejection: %s", params.Content)
			blockerQuery := `INSERT INTO execution_blockers (id, plan_id, checkpoint_id, reason, type, blocked_at) VALUES (?, ?, ?, ?, 'user_rejection', ?)`
			s.db.Exec(blockerQuery, blockerID, cp.PlanID, cp.ID, blockReason, time.Now())

			abandonQuery := `UPDATE plans SET status = 'needs_revision', updated_at = ? WHERE id = ?`
			s.db.Exec(abandonQuery, time.Now(), cp.PlanID)

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
	s.db.QueryRow(statusQuery, params.PlanID).Scan(&currentStatus)

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
	s.db.Exec(resetPhasesQuery, params.PlanID, startPhase)

	clearBlockersQuery := `UPDATE execution_blockers SET resolved_at = ? WHERE plan_id = ? AND resolved_at IS NULL`
	s.db.Exec(clearBlockersQuery, time.Now(), params.PlanID)

	updatePlanQuery := `UPDATE plans SET status = 'in_progress', updated_at = ? WHERE id = ?`
	s.db.Exec(updatePlanQuery, time.Now(), params.PlanID)

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
	s.db.QueryRow(blockerQuery, params.PlanID).Scan(&blockerCount)

	if blockerCount > 0 {
		return nil, fmt.Errorf("plan has %d unresolved blockers, resolve them first", blockerCount)
	}

	if params.RevisionID != "" {
		applyRevQuery := `UPDATE plan_revisions SET status = 'applied', applied_at = ? WHERE id = ?`
		s.db.Exec(applyRevQuery, time.Now(), params.RevisionID)
	}

	resumeQuery := `UPDATE plans SET status = 'in_progress', updated_at = ? WHERE id = ?`
	s.db.Exec(resumeQuery, time.Now(), params.PlanID)

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
		rows.Scan(&id, &checkpointID, &reason, &blkType, &blockedAt, &resolvedAt)

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
	s.db.QueryRow(cpQuery, params.CheckpointID).Scan(&planID)

	updateCp := `UPDATE checkpoints SET status = 'approved', can_continue = 1, decided_at = ?, decided_by = ?, feedback = ? WHERE id = ?`
	s.db.Exec(updateCp, time.Now(), params.Approver, params.Notes, params.CheckpointID)

	resolveBlockers := `UPDATE execution_blockers SET resolved_at = ? WHERE plan_id = ? AND checkpoint_id = ? AND type = 'user_rejection'`
	s.db.Exec(resolveBlockers, time.Now(), planID, params.CheckpointID)

	recordQuery := `INSERT INTO approval_record (id, plan_id, decision, approver, notes, created_at) VALUES (?, ?, 'approved', ?, ?, ?)`
	s.db.Exec(recordQuery, generateID("record"), planID, params.Approver, params.Notes, time.Now())

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
		err := rows.Scan(&rec.ID, &rec.PlanID, &rec.Decision, &rec.Approver, &rec.Notes, &rec.CreatedAt)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
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
	s.db.QueryRow(`SELECT COUNT(*) FROM plans`).Scan(&totalPlans)
	s.db.QueryRow(`SELECT COUNT(*) FROM plans WHERE status = 'draft' OR status = 'in_progress'`).Scan(&activePlans)
	s.db.QueryRow(`SELECT COUNT(*) FROM plans WHERE status = 'completed'`).Scan(&completedPlans)

	return &Response{
		Result: map[string]interface{}{
			"status":          "operational",
			"total_plans":     totalPlans,
			"active_plans":    activePlans,
			"completed_plans": completedPlans,
			"version":         "1.0.0",
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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
