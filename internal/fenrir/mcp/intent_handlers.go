package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andragon31/Ragnarok/internal/fenrir/intent"
	"github.com/andragon31/Ragnarok/internal/fenrir/memory"
)

func (s *Server) handleIntentSave(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string   `json:"plan_id"`
		Prompt string   `json:"prompt"`
		Module string   `json:"module,omitempty"`
		Items  []string `json:"items,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	intentObj := &intent.Intent{
		ID:        generateID("intent"),
		PlanID:    params.PlanID,
		Prompt:    params.Prompt,
		Module:    params.Module,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for _, itemDesc := range params.Items {
		intentObj.Items = append(intentObj.Items, &intent.IntentItem{
			ID:          generateID("item"),
			IntentID:    intentObj.ID,
			Description: itemDesc,
			Type:        "feature",
			Status:      "pending",
		})
	}

	store := intent.NewIntentStore(s.db)
	if err := store.Save(intentObj); err != nil {
		return nil, fmt.Errorf("failed to save intent: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         intentObj.ID,
			"plan_id":    intentObj.PlanID,
			"items":      len(intentObj.Items),
			"created_at": intentObj.CreatedAt,
		},
	}, nil
}

func (s *Server) handleIntentVerify(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string              `json:"plan_id"`
		Files  []map[string]string `json:"files,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	var fileInfos []*intent.FileInfo
	for _, f := range params.Files {
		fileInfos = append(fileInfos, &intent.FileInfo{
			Path:    f["path"],
			Content: f["content"],
		})
	}

	store := intent.NewIntentStore(s.db)
	verifier := intent.NewVerifier(store)

	result, err := verifier.Verify(params.PlanID, fileInfos)
	if err != nil {
		return nil, fmt.Errorf("failed to verify intent: %w", err)
	}

	canContinue := result.CoverageScore >= 0.8

	return &Response{
		Result: map[string]interface{}{
			"intent_id":       result.IntentID,
			"plan_id":         result.PlanID,
			"coverage_score":  result.CoverageScore,
			"alignment_score": result.AlignmentScore,
			"covered":         result.Covered,
			"missing":         result.Missing,
			"partial":         result.Partial,
			"suggestions":     result.Suggestions,
			"can_continue":    canContinue,
			"verified_at":     time.Now(),
		},
	}, nil
}

func (s *Server) handleIntentGet(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		PlanID string `json:"plan_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	store := intent.NewIntentStore(s.db)
	intentObj, err := store.GetByPlanID(params.PlanID)
	if err != nil {
		return nil, fmt.Errorf("intent not found: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         intentObj.ID,
			"plan_id":    intentObj.PlanID,
			"prompt":     intentObj.Prompt,
			"module":     intentObj.Module,
			"items":      intentObj.Items,
			"created_at": intentObj.CreatedAt,
		},
	}, nil
}

func (s *Server) handleBiasReport(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Module string `json:"module,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	mStore := memory.NewMemoryStore(s.db)
	stats, _ := mStore.GetStats()

	reports := make([]map[string]interface{}, 0)

	var totalDecisions, totalIncidents int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM observations WHERE type = 'decision'`).Scan(&totalDecisions); err != nil {
		return nil, fmt.Errorf("failed to count decisions: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM incidents`).Scan(&totalIncidents); err != nil {
		return nil, fmt.Errorf("failed to count incidents: %w", err)
	}

	if stats.TotalSessions > 0 {
		decisionRatio := float64(totalDecisions) / float64(stats.TotalSessions)
		if decisionRatio > 10 && totalIncidents == 0 {
			reports = append(reports, map[string]interface{}{
				"bias_type":      "survivorship",
				"severity":       "high",
				"description":    "High decision count with no incidents may indicate survivorship bias",
				"recommendation": "Ensure all failures and incidents are being recorded",
			})
		}
	}

	if totalIncidents > 0 {
		incidentsPerDecision := float64(totalIncidents) / float64(totalDecisions)
		if incidentsPerDecision < 0.1 {
			reports = append(reports, map[string]interface{}{
				"bias_type":      "confirmation",
				"severity":       "medium",
				"description":    "Low incident-to-decision ratio may indicate confirmation bias",
				"recommendation": "Review if negative outcomes are being captured",
			})
		}
	}

	var recentCount, totalCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM observations WHERE created_at > datetime('now', '-30 days')`).Scan(&recentCount); err != nil {
		return nil, fmt.Errorf("failed to count recent observations: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count observations: %w", err)
	}

	if totalCount > 0 && recentCount > 0 {
		recentRatio := float64(recentCount) / float64(totalCount)
		if recentRatio > 0.7 {
			reports = append(reports, map[string]interface{}{
				"bias_type":      "recency",
				"severity":       "medium",
				"description":    fmt.Sprintf("%.0f%% of observations are from the last 30 days", recentRatio*100),
				"recommendation": "Consider historical context when making decisions",
			})
		}
	}

	var exploratoryCount, authoritativeCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM observations WHERE authority = 'exploratory'`).Scan(&exploratoryCount); err != nil {
		return nil, fmt.Errorf("failed to count exploratory observations: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM observations WHERE authority = 'authoritative'`).Scan(&authoritativeCount); err != nil {
		return nil, fmt.Errorf("failed to count authoritative observations: %w", err)
	}

	if exploratoryCount > authoritativeCount*5 {
		reports = append(reports, map[string]interface{}{
			"bias_type":      "authority",
			"severity":       "low",
			"description":    "Most observations are marked as exploratory rather than authoritative",
			"recommendation": "Consider upgrading key observations to authoritative status",
		})
	}

	for _, report := range reports {
		reportID := generateID("bias")
		query := `INSERT INTO bias_reports (id, module, bias_type, severity, description, recommendation, created_at)
				  VALUES (?, ?, ?, ?, ?, ?, ?)`
		s.db.Exec(query, reportID, params.Module, report["bias_type"], report["severity"],
			report["description"], report["recommendation"], time.Now())
	}

	return &Response{
		Result: map[string]interface{}{
			"module":  params.Module,
			"reports": reports,
			"count":   len(reports),
		},
	}, nil
}

func (s *Server) handleMemSessionEnd(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SessionID string `json:"session_id"`
		Summary   string `json:"summary,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	store := memory.NewMemoryStore(s.db)
	if err := store.EndSession(params.SessionID, time.Now()); err != nil {
		return nil, fmt.Errorf("failed to end session: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"session_id": params.SessionID,
			"status":     "ended",
			"ended_at":   time.Now(),
		},
	}, nil
}

func (s *Server) handleMemSavePrompt(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
		Module    string `json:"module,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	obs := &memory.Observation{
		ID:        generateID("prompt"),
		SessionID: params.SessionID,
		Type:      "prompt",
		Content:   params.Content,
		Module:    params.Module,
		Authority: "exploratory",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	store := memory.NewMemoryStore(s.db)
	if err := store.SaveObservation(obs); err != nil {
		return nil, fmt.Errorf("failed to save prompt: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         obs.ID,
			"type":       obs.Type,
			"created_at": obs.CreatedAt,
		},
	}, nil
}

func (s *Server) handleMemSessionCheckpoint(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SessionID string `json:"session_id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	checkpointID := generateID("checkpoint")

	return &Response{
		Result: map[string]interface{}{
			"session_id":    params.SessionID,
			"checkpoint_id": checkpointID,
			"status":        "created",
			"created_at":    time.Now(),
		},
	}, nil
}

func (s *Server) handleMemGetObservation(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ObservationID string `json:"id"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	query := `SELECT id, session_id, type, content, authority, module, file, line, tags, created_at, updated_at
			  FROM observations WHERE id = ?`
	obs := &memory.Observation{}
	var tags string
	var sessionID, authority, module, file sql.NullString
	var line sql.NullInt64
	err := s.db.QueryRow(query, params.ObservationID).Scan(
		&obs.ID, &sessionID, &obs.Type, &obs.Content, &authority,
		&module, &file, &line, &tags, &obs.CreatedAt, &obs.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("observation not found: %w", err)
	}
	obs.SessionID = sessionID.String
	obs.Authority = authority.String
	obs.Module = module.String
	obs.File = file.String
	if line.Valid {
		obs.Line = int(line.Int64)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         obs.ID,
			"session_id": obs.SessionID,
			"type":       obs.Type,
			"content":    obs.Content,
			"authority":  obs.Authority,
			"module":     obs.Module,
			"file":       obs.File,
			"line":       obs.Line,
			"tags":       tags,
			"created_at": obs.CreatedAt,
			"updated_at": obs.UpdatedAt,
		},
	}, nil
}
