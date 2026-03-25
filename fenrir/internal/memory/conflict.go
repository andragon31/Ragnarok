package memory

import (
	"fmt"
	"time"
)

var conflictIDCounter = 0

func generateConflictID(prefix string) string {
	conflictIDCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), conflictIDCounter)
}

type Conflict struct {
	ID            string    `json:"id"`
	EntityType    string    `json:"entity_type"`
	EntityID      string    `json:"entity_id"`
	LocalContent  string    `json:"local_content,omitempty"`
	RemoteContent string    `json:"remote_content,omitempty"`
	Resolution    string    `json:"resolution,omitempty"`
	ResolvedBy    string    `json:"resolved_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	ResolvedAt    time.Time `json:"resolved_at,omitempty"`
}

func (s *MemoryStore) ListConflicts(status string, limit int) ([]*Conflict, error) {
	query := `SELECT id, entity_type, entity_id, local_content, remote_content, resolution, resolved_by, created_at, resolved_at
			  FROM conflicts WHERE (? = '' OR resolution = ?) AND (? = '' OR resolution != ?)
			  ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.Query(query, status, status, status, "resolved", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []*Conflict
	for rows.Next() {
		cf := &Conflict{}
		err := rows.Scan(&cf.ID, &cf.EntityType, &cf.EntityID, &cf.LocalContent, &cf.RemoteContent, &cf.Resolution, &cf.ResolvedBy, &cf.CreatedAt, &cf.ResolvedAt)
		if err != nil {
			return nil, err
		}
		conflicts = append(conflicts, cf)
	}
	return conflicts, nil
}

func (s *MemoryStore) ResolveConflict(id string, resolution string, resolvedBy string) error {
	query := `UPDATE conflicts SET resolution = ?, resolved_by = ?, resolved_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, resolution, resolvedBy, time.Now(), id)
	return err
}

func (s *MemoryStore) GetConflict(id string) (*Conflict, error) {
	query := `SELECT id, entity_type, entity_id, local_content, remote_content, resolution, resolved_by, created_at, resolved_at
			  FROM conflicts WHERE id = ?`
	cf := &Conflict{}
	err := s.db.QueryRow(query, id).Scan(&cf.ID, &cf.EntityType, &cf.EntityID, &cf.LocalContent, &cf.RemoteContent, &cf.Resolution, &cf.ResolvedBy, &cf.CreatedAt, &cf.ResolvedAt)
	if err != nil {
		return nil, err
	}
	return cf, nil
}

func (s *MemoryStore) SaveConflict(conflict *Conflict) error {
	query := `INSERT INTO conflicts (id, entity_type, entity_id, local_content, remote_content, created_at)
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, conflict.ID, conflict.EntityType, conflict.EntityID, conflict.LocalContent, conflict.RemoteContent, conflict.CreatedAt)
	return err
}

func (s *MemoryStore) CreateConflict(entityType, entityID, localContent, remoteContent string) (*Conflict, error) {
	conflict := &Conflict{
		ID:            generateConflictID("conflict"),
		EntityType:    entityType,
		EntityID:      entityID,
		LocalContent:  localContent,
		RemoteContent: remoteContent,
		CreatedAt:     time.Now(),
	}

	query := `INSERT INTO conflicts (id, entity_type, entity_id, local_content, remote_content, created_at)
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, conflict.ID, conflict.EntityType, conflict.EntityID, conflict.LocalContent, conflict.RemoteContent, conflict.CreatedAt)
	if err != nil {
		return nil, err
	}
	return conflict, nil
}
