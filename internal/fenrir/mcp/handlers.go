package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/andragon31/Ragnarok/internal/fenrir/graph"
	"github.com/andragon31/Ragnarok/internal/fenrir/memory"
	"github.com/andragon31/Ragnarok/internal/fenrir/specs"
)

func (s *Server) handleSessionStart(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Project string `json:"project"`
		Module  string `json:"module,omitempty"`
		AgentID string `json:"agent_id,omitempty"`
		PlanID  string `json:"plan_id,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	session := &memory.Session{
		ID:        generateID("session"),
		Project:   params.Project,
		Module:    params.Module,
		AgentID:   params.AgentID,
		PlanID:    params.PlanID,
		StartedAt: time.Now(),
	}

	store := memory.NewMemoryStore(s.db)
	if err := store.CreateSession(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	autoInject := s.config.AutoInject
	limit := s.config.AutoInjectLimit

	var recentObs []*memory.Observation
	if autoInject {
		var err error
		recentObs, err = store.GetRecentObservations(limit)
		if err != nil {
			recentObs = []*memory.Observation{}
		}
	}

	return &Response{
		Result: map[string]interface{}{
			"session_id":          session.ID,
			"project":             session.Project,
			"module":              session.Module,
			"auto_injected":       autoInject,
			"recent_observations": recentObs,
		},
	}, nil
}

func (s *Server) handleMemSave(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SessionID string   `json:"session_id"`
		Type      string   `json:"type"`
		Content   string   `json:"content"`
		Module    string   `json:"module,omitempty"`
		File      string   `json:"file,omitempty"`
		Line      int      `json:"line,omitempty"`
		Tags      []string `json:"tags,omitempty"`
		Authority string   `json:"authority,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	obs := &memory.Observation{
		ID:        generateID("obs"),
		SessionID: params.SessionID,
		Type:      params.Type,
		Content:   params.Content,
		Module:    params.Module,
		File:      params.File,
		Line:      params.Line,
		Tags:      params.Tags,
		Authority: params.Authority,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if obs.Authority == "" {
		obs.Authority = "exploratory"
	}

	store := memory.NewMemoryStore(s.db)
	if err := store.SaveObservation(obs); err != nil {
		return nil, fmt.Errorf("failed to save observation: %w", err)
	}

	if s.config.SemanticSearch {
		gStore := graph.NewGraphStore(s.db)
		gStore.AddNode(obs.ID, obs.Type, obs.Type, obs.Content, map[string]string{
			"module": obs.Module,
		})
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         obs.ID,
			"type":       obs.Type,
			"authority":  obs.Authority,
			"created_at": obs.CreatedAt,
		},
	}, nil
}

func (s *Server) handleMemFind(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Query          string `json:"query"`
		Module         string `json:"module,omitempty"`
		Type           string `json:"type,omitempty"`
		Limit          int    `json:"limit,omitempty"`
		IncludeContent bool   `json:"include_content,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 5
	}

	store := memory.NewMemoryStore(s.db)
	graphStore := graph.NewGraphStore(s.db)

	var results []*graph.ContextResult
	var err error

	if s.config.SemanticSearch && params.Module != "" {
		results, err = graphStore.SearchContext(params.Query, params.Module, params.Limit)
	} else {
		results, err = graphStore.SearchContext(params.Query, "", params.Limit)
	}

	if err != nil {
		graphErr := err
		observations, err := store.Search(params.Query, params.Limit)
		if err != nil {
			return nil, fmt.Errorf("graph search failed: %v, fallback search failed: %w", graphErr, err)
		}

		resultItems := make([]map[string]interface{}, 0, len(observations))
		for _, obs := range observations {
			item := map[string]interface{}{
				"id":         obs.ID,
				"type":       obs.Type,
				"module":     obs.Module,
				"authority":  obs.Authority,
				"created_at": obs.CreatedAt,
			}
			if params.IncludeContent {
				item["content"] = obs.Content
			}
			resultItems = append(resultItems, item)
		}

		return &Response{
			Result: map[string]interface{}{
				"query":   params.Query,
				"results": resultItems,
				"count":   len(resultItems),
			},
		}, nil
	}

	resultItems := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		item := map[string]interface{}{
			"id":         r.ID,
			"type":       r.Type,
			"module":     r.Module,
			"authority":  r.Authority,
			"created_at": r.CreatedAt,
		}
		if params.IncludeContent {
			item["content"] = r.Content
		}
		resultItems = append(resultItems, item)
	}

	return &Response{
		Result: map[string]interface{}{
			"query":   params.Query,
			"results": resultItems,
			"count":   len(resultItems),
		},
	}, nil
}

func (s *Server) handleMemContext(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Module string `json:"module"`
		Limit  int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = s.config.AutoInjectLimit
	}

	store := memory.NewMemoryStore(s.db)
	observations, err := store.GetObservationsByModule(params.Module, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"module":       params.Module,
			"observations": observations,
			"count":        len(observations),
		},
	}, nil
}

func (s *Server) handleMemTimeline(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Module string `json:"module,omitempty"`
		Limit  int    `json:"limit,omitempty"`
		Full   bool   `json:"full,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 20
	}

	store := memory.NewMemoryStore(s.db)
	observations, err := store.GetRecentObservations(params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline: %w", err)
	}

	type TimelineEntry struct {
		ID        string    `json:"id"`
		Type      string    `json:"type"`
		Content   string    `json:"content"`
		Module    string    `json:"module"`
		CreatedAt time.Time `json:"created_at"`
	}

	entries := make([]*TimelineEntry, 0, len(observations))
	for _, obs := range observations {
		content := obs.Content
		if !params.Full && len(content) > 200 {
			content = content[:200] + "..."
		}
		entries = append(entries, &TimelineEntry{
			ID:        obs.ID,
			Type:      obs.Type,
			Content:   content,
			Module:    obs.Module,
			CreatedAt: obs.CreatedAt,
		})
	}

	return &Response{
		Result: map[string]interface{}{
			"entries": entries,
			"count":   len(entries),
			"compact": !params.Full,
		},
	}, nil
}

func (s *Server) handleSpecSave(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Module  string `json:"module"`
		Title   string `json:"title"`
		Content string `json:"content"`
		Type    string `json:"type,omitempty"`
		Given   string `json:"given,omitempty"`
		When    string `json:"when,omitempty"`
		Then    string `json:"then,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	spec := &specs.Spec{
		ID:        generateID("spec"),
		Module:    params.Module,
		Title:     params.Title,
		Content:   params.Content,
		Type:      params.Type,
		Given:     params.Given,
		When:      params.When,
		Then:      params.Then,
		Status:    "draft",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if spec.Type == "" {
		spec.Type = "feature"
	}

	store := specs.NewSpecStore(s.db)
	if err := store.Save(spec); err != nil {
		return nil, fmt.Errorf("failed to save spec: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         spec.ID,
			"module":     spec.Module,
			"title":      spec.Title,
			"status":     spec.Status,
			"created_at": spec.CreatedAt,
		},
	}, nil
}

func (s *Server) handleSpecList(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Module string `json:"module,omitempty"`
		Status string `json:"status,omitempty"`
		Limit  int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 50
	}

	store := specs.NewSpecStore(s.db)
	specList, err := store.List(params.Module, params.Status, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list specs: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"specs": specList,
			"count": len(specList),
		},
	}, nil
}

func (s *Server) handleSpecCheck(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Module            string `json:"module"`
		ChangeDescription string `json:"change_description"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	store := specs.NewSpecStore(s.db)
	impact, err := store.Check(params.Module, params.ChangeDescription)
	if err != nil {
		return nil, fmt.Errorf("failed to check specs: %w", err)
	}

	return &Response{
		Result: impact,
	}, nil
}

func (s *Server) handleStats(ctx context.Context, req *Request) (*Response, error) {
	store := memory.NewMemoryStore(s.db)
	stats, err := store.GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"total_observations": stats.TotalObservations,
			"total_sessions":     stats.TotalSessions,
			"total_edges":        stats.TotalEdges,
			"total_specs":        stats.TotalSpecs,
			"open_incidents":     stats.OpenIncidents,
		},
	}, nil
}

func (s *Server) handleIncidentLog(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Module      string `json:"module"`
		Summary     string `json:"summary"`
		Severity    string `json:"severity,omitempty"`
		RelatedSpec string `json:"related_spec,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Severity == "" {
		params.Severity = "medium"
	}

	incident := &memory.Incident{
		ID:          generateID("incident"),
		Module:      params.Module,
		Summary:     params.Summary,
		Severity:    params.Severity,
		Status:      "open",
		RelatedSpec: params.RelatedSpec,
		CreatedAt:   time.Now(),
	}

	store := memory.NewMemoryStore(s.db)
	if err := store.LogIncident(incident); err != nil {
		return nil, fmt.Errorf("failed to log incident: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":         incident.ID,
			"module":     incident.Module,
			"summary":    incident.Summary,
			"severity":   incident.Severity,
			"status":     incident.Status,
			"created_at": incident.CreatedAt,
		},
	}, nil
}

func (s *Server) handleIncidentList(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		Module string `json:"module,omitempty"`
		Status string `json:"status,omitempty"`
		Limit  int    `json:"limit,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 50
	}

	store := memory.NewMemoryStore(s.db)
	incidents, err := store.ListIncidents(params.Module, params.Status, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list incidents: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"incidents": incidents,
			"count":     len(incidents),
		},
	}, nil
}

func (s *Server) handleIncidentResolve(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ID       string `json:"id"`
		Solution string `json:"solution"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	store := memory.NewMemoryStore(s.db)
	if err := store.ResolveIncident(params.ID, params.Solution); err != nil {
		return nil, fmt.Errorf("failed to resolve incident: %w", err)
	}

	incident, _ := store.GetIncident(params.ID)

	return &Response{
		Result: map[string]interface{}{
			"id":       params.ID,
			"status":   "resolved",
			"solution": params.Solution,
			"resolved": incident != nil,
		},
	}, nil
}

func (s *Server) handleConflictList(ctx context.Context, req *Request) (*Response, error) {
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

	store := memory.NewMemoryStore(s.db)
	conflicts, err := store.ListConflicts(params.Status, params.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list conflicts: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"conflicts": conflicts,
			"count":     len(conflicts),
		},
	}, nil
}

func (s *Server) handleConflictResolve(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		ID         string `json:"id"`
		Resolution string `json:"resolution"`
		ResolvedBy string `json:"resolved_by,omitempty"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	store := memory.NewMemoryStore(s.db)
	if err := store.ResolveConflict(params.ID, params.Resolution, params.ResolvedBy); err != nil {
		return nil, fmt.Errorf("failed to resolve conflict: %w", err)
	}

	conflict, _ := store.GetConflict(params.ID)

	return &Response{
		Result: map[string]interface{}{
			"id":         params.ID,
			"status":     "resolved",
			"resolution": params.Resolution,
			"resolved":   conflict != nil,
		},
	}, nil
}

func (s *Server) handleSpecDelta(ctx context.Context, req *Request) (*Response, error) {
	var params struct {
		SpecID      string `json:"spec_id"`
		PlanID      string `json:"plan_id"`
		Type        string `json:"type"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	delta := &specs.SpecDelta{
		ID:          generateID("delta"),
		SpecID:      params.SpecID,
		PlanID:      params.PlanID,
		Type:        params.Type,
		Description: params.Description,
		CreatedAt:   time.Now(),
	}

	store := specs.NewSpecStore(s.db)
	if err := store.SaveDelta(delta); err != nil {
		return nil, fmt.Errorf("failed to save spec delta: %w", err)
	}

	return &Response{
		Result: map[string]interface{}{
			"id":          delta.ID,
			"spec_id":     delta.SpecID,
			"plan_id":     delta.PlanID,
			"type":        delta.Type,
			"description": delta.Description,
			"created_at":  delta.CreatedAt,
		},
	}, nil
}

var idCounter = 0
var idMutex sync.Mutex

func generateID(prefix string) string {
	idMutex.Lock()
	defer idMutex.Unlock()
	idCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), idCounter)
}
