package graph

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

type GraphStore struct {
	db *sql.DB
}

func NewGraphStore(db *sql.DB) *GraphStore {
	return &GraphStore{db: db}
}

func (s *GraphStore) AddNode(id, label, type_, content string, metadata map[string]string) error {
	query := `INSERT INTO nodes (id, label, type, content, metadata, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))`
	metadataJSON := mapToJSON(metadata)
	_, err := s.db.Exec(query, id, label, type_, content, metadataJSON)
	return err
}

func (s *GraphStore) AddEdge(sourceID, targetID, edgeType string) error {
	query := `INSERT INTO edges (source_id, target_id, type, created_at)
			  VALUES (?, ?, ?, datetime('now'))`
	_, err := s.db.Exec(query, sourceID, targetID, edgeType)
	return err
}

func (s *GraphStore) GetNode(id string) (*Node, error) {
	query := `SELECT id, label, type, content, metadata, authority, created_at, updated_at
			  FROM nodes WHERE id = ?`
	node := &Node{}
	var metadata string
	err := s.db.QueryRow(query, id).Scan(&node.ID, &node.Label, &node.Type, &node.Content, &metadata, &node.Authority, &node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return nil, err
	}
	node.Metadata = parseJSON(metadata)
	return node, nil
}

func (s *GraphStore) SearchNodes(query string, limit int) ([]*Node, error) {
	sqlQuery := `SELECT id, label, type, content, metadata, authority, created_at, updated_at
				 FROM nodes_fts WHERE nodes_fts MATCH ? LIMIT ?`
	rows, err := s.db.Query(sqlQuery, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		node := &Node{}
		var metadata, content, authority sql.NullString
		err := rows.Scan(&node.ID, &node.Label, &node.Type, &content, &metadata, &authority, &node.CreatedAt, &node.UpdatedAt)
		if err != nil {
			return nil, err
		}
		node.Content = content.String
		node.Metadata = parseJSON(metadata.String)
		node.Authority = authority.String
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *GraphStore) GetNeighbors(nodeID string) ([]*Node, error) {
	query := `SELECT n.id, n.label, n.type, n.content, n.metadata, n.authority, n.created_at, n.updated_at
			  FROM nodes n
			  JOIN edges e ON (e.target_id = n.id OR e.source_id = n.id)
			  WHERE (e.source_id = ? OR e.target_id = ?) AND n.id != ?`
	rows, err := s.db.Query(query, nodeID, nodeID, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		node := &Node{}
		var metadata, content, authority sql.NullString
		err := rows.Scan(&node.ID, &node.Label, &node.Type, &content, &metadata, &authority, &node.CreatedAt, &node.UpdatedAt)
		if err != nil {
			return nil, err
		}
		node.Content = content.String
		node.Metadata = parseJSON(metadata.String)
		node.Authority = authority.String
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *GraphStore) SearchContext(query string, module string, limit int) ([]*ContextResult, error) {
	var results []*ContextResult

	obsQuery := `SELECT id, 'observation' as type, content, module, authority, created_at
				 FROM observations
				 WHERE content LIKE ? AND (? = '' OR module = ?)
				 ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.Query(obsQuery, "%"+query+"%", module, module, limit)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			r := &ContextResult{}
			var rModule, authority sql.NullString
			rows.Scan(&r.ID, &r.Type, &r.Content, &rModule, &authority, &r.CreatedAt)
			r.Module = rModule.String
			r.Authority = authority.String
			results = append(results, r)
		}
	}

	nodeQuery := `SELECT id, 'node' as type, content, label as module, authority, created_at
				  FROM nodes
				  WHERE content LIKE ? AND (? = '' OR label = ?)
				  ORDER BY created_at DESC LIMIT ?`
	rows2, err := s.db.Query(nodeQuery, "%"+query+"%", module, module, limit)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			r := &ContextResult{}
			var rModule, authority sql.NullString
			rows2.Scan(&r.ID, &r.Type, &r.Content, &rModule, &authority, &r.CreatedAt)
			r.Module = rModule.String
			r.Authority = authority.String
			results = append(results, r)
		}
	}

	specQuery := `SELECT id, 'spec' as type, content, title as module, 'confirmed' as authority, created_at
				  FROM specs
				  WHERE (content LIKE ? OR title LIKE ?) AND (? = '' OR module = ?)
				  ORDER BY created_at DESC LIMIT ?`
	rows3, err := s.db.Query(specQuery, "%"+query+"%", "%"+query+"%", module, module, limit)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			r := &ContextResult{}
			rows3.Scan(&r.ID, &r.Type, &r.Content, &r.Module, &r.Authority, &r.CreatedAt)
			results = append(results, r)
		}
	}

	return results, nil
}

type Node struct {
	ID        string            `json:"id"`
	Label     string            `json:"label"`
	Type      string            `json:"type"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	Authority string            `json:"authority"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

type ContextResult struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Module    string `json:"module"`
	Authority string `json:"authority"`
	CreatedAt string `json:"created_at"`
}

func mapToJSON(m map[string]string) string {
	if m == nil {
		return "{}"
	}
	parts := []string{}
	for k, v := range m {
		parts = append(parts, fmt.Sprintf(`"%s":"%s"`, k, v))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func parseJSON(s string) map[string]string {
	m := make(map[string]string)
	if s == "" || s == "{}" {
		return m
	}
	re := regexp.MustCompile(`"([^"]+)":"([^"]+)"`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		m[match[1]] = match[2]
	}
	return m
}
