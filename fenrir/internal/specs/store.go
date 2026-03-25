package specs

import (
	"database/sql"
	"regexp"
	"strings"
	"time"
)

type SpecStore struct {
	db *sql.DB
}

func NewSpecStore(db *sql.DB) *SpecStore {
	return &SpecStore{db: db}
}

func (s *SpecStore) Save(spec *Spec) error {
	query := `INSERT OR REPLACE INTO specs (id, module, title, content, type, given, when_cond, then_cond, status, implemented_at, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	implementedAt := sql.NullTime{Time: spec.ImplementedAt, Valid: !spec.ImplementedAt.IsZero()}
	_, err := s.db.Exec(query, spec.ID, spec.Module, spec.Title, spec.Content, spec.Type,
		spec.Given, spec.When, spec.Then, spec.Status, implementedAt, spec.CreatedAt, spec.UpdatedAt)
	return err
}

func (s *SpecStore) Get(id string) (*Spec, error) {
	query := `SELECT id, module, title, content, type, given, when_cond, then_cond, status, implemented_at, created_at, updated_at
			  FROM specs WHERE id = ?`
	spec := &Spec{}
	var given, when, then sql.NullString
	var implementedAt sql.NullTime
	err := s.db.QueryRow(query, id).Scan(&spec.ID, &spec.Module, &spec.Title, &spec.Content, &spec.Type,
		&given, &when, &then, &spec.Status, &implementedAt, &spec.CreatedAt, &spec.UpdatedAt)
	if err != nil {
		return nil, err
	}
	spec.Given = given.String
	spec.When = when.String
	spec.Then = then.String
	if implementedAt.Valid {
		spec.ImplementedAt = implementedAt.Time
	}
	return spec, nil
}

func (s *SpecStore) List(module string, status string, limit int) ([]*Spec, error) {
	query := `SELECT id, module, title, content, type, given, when_cond, then_cond, status, implemented_at, created_at, updated_at
			  FROM specs WHERE (? = '' OR module = ?) AND (? = '' OR status = ?)
			  ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.Query(query, module, module, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var specs []*Spec
	for rows.Next() {
		spec := &Spec{}
		var given, when, then sql.NullString
		var implementedAt sql.NullTime
		err := rows.Scan(&spec.ID, &spec.Module, &spec.Title, &spec.Content, &spec.Type,
			&given, &when, &then, &spec.Status, &implementedAt, &spec.CreatedAt, &spec.UpdatedAt)
		if err != nil {
			return nil, err
		}
		spec.Given = given.String
		spec.When = when.String
		spec.Then = then.String
		if implementedAt.Valid {
			spec.ImplementedAt = implementedAt.Time
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func (s *SpecStore) Check(module, changeDescription string) (*SpecImpact, error) {
	impact := &SpecImpact{
		SpecsAffected: []*SpecAffected{},
	}

	allSpecs, err := s.List(module, "active", 100)
	if err != nil {
		return nil, err
	}

	changeLower := strings.ToLower(changeDescription)
	changeWords := extractWords(changeLower)

	for _, spec := range allSpecs {
		contentLower := strings.ToLower(spec.Content)
		contentWords := extractWords(contentLower)

		overlap := countOverlap(changeWords, contentWords)
		if overlap > 0 {
			impactType := "related"
			if strings.Contains(contentLower, changeLower) || strings.Contains(changeLower, contentLower) {
				impactType = "implements"
			}

			impact.SpecsAffected = append(impact.SpecsAffected, &SpecAffected{
				ID:         spec.ID,
				Title:      spec.Title,
				ImpactType: impactType,
				MatchScore: float64(overlap) / float64(len(contentWords)),
			})
		}
	}

	return impact, nil
}

func (s *SpecStore) SaveDelta(delta *SpecDelta) error {
	query := `INSERT INTO spec_deltas (id, spec_id, plan_id, type, description, created_at)
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, delta.ID, delta.SpecID, delta.PlanID, delta.Type, delta.Description, delta.CreatedAt)
	return err
}

type Spec struct {
	ID            string    `json:"id"`
	Module        string    `json:"module"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	Type          string    `json:"type"`
	Given         string    `json:"given,omitempty"`
	When          string    `json:"when,omitempty"`
	Then          string    `json:"then,omitempty"`
	Status        string    `json:"status"`
	ImplementedAt time.Time `json:"implemented_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SpecDelta struct {
	ID          string    `json:"id"`
	SpecID      string    `json:"spec_id"`
	PlanID      string    `json:"plan_id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type SpecImpact struct {
	SpecsAffected []*SpecAffected `json:"specs_affected"`
}

type SpecAffected struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	ImpactType string  `json:"impact_type"`
	MatchScore float64 `json:"match_score"`
}

func extractWords(s string) []string {
	re := regexp.MustCompile(`\w+`)
	return re.FindAllString(s, -1)
}

func countOverlap(a, b []string) int {
	bSet := make(map[string]bool)
	for _, w := range b {
		bSet[w] = true
	}
	count := 0
	for _, w := range a {
		if bSet[w] {
			count++
		}
	}
	return count
}
