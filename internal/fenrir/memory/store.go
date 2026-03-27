package memory

import (
	"database/sql"
	"fmt"
	"time"
)

type Session struct {
	ID           string      `json:"id"`
	Project      string      `json:"project"`
	Module       string      `json:"module,omitempty"`
	StartedAt    time.Time   `json:"started_at"`
	EndedAt      time.Time   `json:"ended_at,omitempty"`
	AgentID      string      `json:"agent_id,omitempty"`
	PlanID       string      `json:"plan_id,omitempty"`
	CheckpointID string      `json:"checkpoint_id,omitempty"`
	DNA          *SessionDNA `json:"dna,omitempty"`
}

type SessionDNA struct {
	SessionID       string   `json:"session_id"`
	FilesRead       int      `json:"files_read"`
	FilesWritten    int      `json:"files_written"`
	CommandsRun     int      `json:"commands_run"`
	DecisionsMade   int      `json:"decisions_made"`
	QualityScore    float64  `json:"quality_score"`
	ToolsUsed       []string `json:"tools_used"`
	DurationSeconds int      `json:"duration_seconds"`
}

type Observation struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	Type         string    `json:"type"` // decision, incident, discovery, prompt, spec
	Content      string    `json:"content"`
	Authority    string    `json:"authority"` // exploratory, confirmed, authoritative
	Module       string    `json:"module,omitempty"`
	File         string    `json:"file,omitempty"`
	Line         int       `json:"line,omitempty"`
	Tags         []string  `json:"tags"`
	AuthorityBy  string    `json:"authority_by,omitempty"`
	AuthorityAt  time.Time `json:"authority_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ReadCount    int       `json:"read_count"`
	IsCompressed bool      `json:"is_compressed"`
	TokenCount   int       `json:"token_count"`
}

type Edge struct {
	ID        string    `json:"id"`
	SourceID  string    `json:"source_id"`
	TargetID  string    `json:"target_id"`
	Type      string    `json:"type"` // caused_by, related_to, implements, extends
	SessionID string    `json:"session_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Spec struct {
	ID            string    `json:"id"`
	Module        string    `json:"module"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	Type          string    `json:"type"` // feature, bugfix, refactor, docs
	Given         string    `json:"given,omitempty"`
	When          string    `json:"when,omitempty"`
	Then          string    `json:"then,omitempty"`
	Status        string    `json:"status"` // draft, active, implemented, violated
	ImplementedAt time.Time `json:"implemented_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SpecDelta struct {
	ID          string    `json:"id"`
	SpecID      string    `json:"spec_id"`
	PlanID      string    `json:"plan_id"`
	Type        string    `json:"type"` // implemented, extended, modified, violated
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Incident struct {
	ID          string    `json:"id"`
	Module      string    `json:"module"`
	Summary     string    `json:"summary"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	Status      string    `json:"status"`   // open, acknowledged, resolved
	RelatedSpec string    `json:"related_spec,omitempty"`
	Solution    string    `json:"solution,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	ResolvedAt  time.Time `json:"resolved_at,omitempty"`
}

type MemoryStore struct {
	db *sql.DB
}

func NewMemoryStore(db *sql.DB) *MemoryStore {
	return &MemoryStore{db: db}
}

func (s *MemoryStore) CreateSession(session *Session) error {
	query := `INSERT INTO sessions (id, project, module, started_at, agent_id, plan_id, checkpoint_id)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, session.ID, session.Project, session.Module, session.StartedAt, session.AgentID, session.PlanID, session.CheckpointID)
	return err
}

func (s *MemoryStore) EndSession(sessionID string, endedAt time.Time) error {
	query := `UPDATE sessions SET ended_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, endedAt, sessionID)
	return err
}

func (s *MemoryStore) SaveObservation(obs *Observation) error {
	query := `INSERT INTO observations (id, session_id, type, content, authority, module, file, line, tags, created_at, updated_at, token_count)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	tagsJSON := fmt.Sprintf("[%s]", joinStrings(obs.Tags, ","))
	_, err := s.db.Exec(query, obs.ID, obs.SessionID, obs.Type, obs.Content, obs.Authority, obs.Module, obs.File, obs.Line, tagsJSON, obs.CreatedAt, obs.UpdatedAt, obs.TokenCount)
	return err
}

func (s *MemoryStore) Search(query string, limit int) ([]*Observation, error) {
	sqlQuery := `SELECT id, session_id, type, content, authority, module, file, line, tags, created_at, updated_at, token_count
				 FROM observations_fts WHERE observations_fts MATCH ? LIMIT ?`
	rows, err := s.db.Query(sqlQuery, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []*Observation
	for rows.Next() {
		obs := &Observation{}
		var tags string
		var sessionID, module, file, authority sql.NullString
		var line sql.NullInt64
		var tokenCount sql.NullInt64
		err := rows.Scan(&obs.ID, &sessionID, &obs.Type, &obs.Content, &authority, &module, &file, &line, &tags, &obs.CreatedAt, &obs.UpdatedAt, &tokenCount)
		if err != nil {
			return nil, err
		}
		obs.SessionID = sessionID.String
		obs.Module = module.String
		obs.File = file.String
		obs.Authority = authority.String
		if line.Valid {
			obs.Line = int(line.Int64)
		}
		if tokenCount.Valid {
			obs.TokenCount = int(tokenCount.Int64)
		}
		observations = append(observations, obs)
	}
	return observations, nil
}

func (s *MemoryStore) GetObservationsByModule(module string, limit int) ([]*Observation, error) {
	query := `SELECT id, session_id, type, content, authority, module, file, line, tags, created_at, updated_at, token_count
			  FROM observations WHERE module = ? ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.Query(query, module, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []*Observation
	for rows.Next() {
		obs := &Observation{}
		var tags string
		var sessionID, module, file, authority sql.NullString
		var line sql.NullInt64
		var tokenCount sql.NullInt64
		err := rows.Scan(&obs.ID, &sessionID, &obs.Type, &obs.Content, &authority, &module, &file, &line, &tags, &obs.CreatedAt, &obs.UpdatedAt, &tokenCount)
		if err != nil {
			return nil, err
		}
		obs.SessionID = sessionID.String
		obs.Module = module.String
		obs.File = file.String
		obs.Authority = authority.String
		if line.Valid {
			obs.Line = int(line.Int64)
		}
		if tokenCount.Valid {
			obs.TokenCount = int(tokenCount.Int64)
		}
		observations = append(observations, obs)
	}
	return observations, nil
}

func (s *MemoryStore) GetRecentObservations(limit int) ([]*Observation, error) {
	query := `SELECT id, session_id, type, content, authority, module, file, line, tags, created_at, updated_at, token_count
			  FROM observations ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []*Observation
	for rows.Next() {
		obs := &Observation{}
		var tags string
		var sessionID, module, file, authority sql.NullString
		var line sql.NullInt64
		var tokenCount sql.NullInt64
		err := rows.Scan(&obs.ID, &sessionID, &obs.Type, &obs.Content, &authority, &module, &file, &line, &tags, &obs.CreatedAt, &obs.UpdatedAt, &tokenCount)
		if err != nil {
			return nil, err
		}
		obs.SessionID = sessionID.String
		obs.Module = module.String
		obs.File = file.String
		obs.Authority = authority.String
		if line.Valid {
			obs.Line = int(line.Int64)
		}
		if tokenCount.Valid {
			obs.TokenCount = int(tokenCount.Int64)
		}
		observations = append(observations, obs)
	}
	return observations, nil
}

func (s *MemoryStore) GetStats() (*Stats, error) {
	stats := &Stats{}

	s.db.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&stats.TotalObservations)
	s.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&stats.TotalSessions)
	s.db.QueryRow(`SELECT COUNT(*) FROM edges`).Scan(&stats.TotalEdges)
	s.db.QueryRow(`SELECT COUNT(*) FROM specs`).Scan(&stats.TotalSpecs)
	s.db.QueryRow(`SELECT COUNT(*) FROM incidents WHERE status = 'open'`).Scan(&stats.OpenIncidents)

	return stats, nil
}

type Stats struct {
	TotalObservations int `json:"total_observations"`
	TotalSessions     int `json:"total_sessions"`
	TotalEdges        int `json:"total_edges"`
	TotalSpecs        int `json:"total_specs"`
	OpenIncidents     int `json:"open_incidents"`
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
