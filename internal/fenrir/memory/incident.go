package memory

import (
	"time"
)

func (s *MemoryStore) LogIncident(incident *Incident) error {
	query := `INSERT INTO incidents (id, module, summary, severity, status, related_spec, created_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, incident.ID, incident.Module, incident.Summary, incident.Severity, incident.Status, incident.RelatedSpec, incident.CreatedAt)
	return err
}

func (s *MemoryStore) ListIncidents(module string, status string, limit int) ([]*Incident, error) {
	query := `SELECT id, module, summary, severity, status, related_spec, solution, created_at, resolved_at
			  FROM incidents WHERE (? = '' OR module = ?) AND (? = '' OR status = ?)
			  ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.Query(query, module, module, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []*Incident
	for rows.Next() {
		inc := &Incident{}
		err := rows.Scan(&inc.ID, &inc.Module, &inc.Summary, &inc.Severity, &inc.Status, &inc.RelatedSpec, &inc.Solution, &inc.CreatedAt, &inc.ResolvedAt)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, inc)
	}
	return incidents, nil
}

func (s *MemoryStore) ResolveIncident(id string, solution string) error {
	query := `UPDATE incidents SET status = 'resolved', solution = ?, resolved_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, solution, time.Now(), id)
	return err
}

func (s *MemoryStore) GetIncident(id string) (*Incident, error) {
	query := `SELECT id, module, summary, severity, status, related_spec, solution, created_at, resolved_at
			  FROM incidents WHERE id = ?`
	inc := &Incident{}
	err := s.db.QueryRow(query, id).Scan(&inc.ID, &inc.Module, &inc.Summary, &inc.Severity, &inc.Status, &inc.RelatedSpec, &inc.Solution, &inc.CreatedAt, &inc.ResolvedAt)
	if err != nil {
		return nil, err
	}
	return inc, nil
}
