package intent

import (
	"database/sql"
	"time"
)

type IntentStore struct {
	db *sql.DB
}

type Intent struct {
	ID        string        `json:"id"`
	PlanID    string        `json:"plan_id"`
	Prompt    string        `json:"prompt"`
	Module    string        `json:"module,omitempty"`
	Items     []*IntentItem `json:"items,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type IntentItem struct {
	ID          string  `json:"id"`
	IntentID    string  `json:"intent_id"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	MatchScore  float64 `json:"match_score,omitempty"`
}

func NewIntentStore(db *sql.DB) *IntentStore {
	return &IntentStore{db: db}
}

func (s *IntentStore) Save(intent *Intent) error {
	query := `INSERT INTO intents (id, plan_id, prompt, module, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, intent.ID, intent.PlanID, intent.Prompt, intent.Module, intent.CreatedAt, intent.UpdatedAt)
	if err != nil {
		return err
	}

	for _, item := range intent.Items {
		itemQuery := `INSERT INTO intent_items (id, intent_id, description, type, status)
					  VALUES (?, ?, ?, ?, ?)`
		_, err := s.db.Exec(itemQuery, item.ID, intent.ID, item.Description, item.Type, item.Status)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *IntentStore) GetByPlanID(planID string) (*Intent, error) {
	query := `SELECT id, plan_id, prompt, module, created_at, updated_at FROM intents WHERE plan_id = ?`
	intent := &Intent{}
	err := s.db.QueryRow(query, planID).Scan(&intent.ID, &intent.PlanID, &intent.Prompt, &intent.Module, &intent.CreatedAt, &intent.UpdatedAt)
	if err != nil {
		return nil, err
	}

	itemsQuery := `SELECT id, intent_id, description, type, status FROM intent_items WHERE intent_id = ?`
	rows, err := s.db.Query(itemsQuery, intent.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item := &IntentItem{}
		err := rows.Scan(&item.ID, &item.IntentID, &item.Description, &item.Type, &item.Status)
		if err != nil {
			return nil, err
		}
		intent.Items = append(intent.Items, item)
	}

	return intent, nil
}

func (s *IntentStore) Get(id string) (*Intent, error) {
	return s.GetByPlanID(id)
}

func (s *IntentStore) UpdateItemStatus(itemID string, status string, matchScore float64) error {
	query := `UPDATE intent_items SET status = ?, match_score = ? WHERE id = ?`
	_, err := s.db.Exec(query, status, matchScore, itemID)
	return err
}

func (s *IntentStore) List(limit int) ([]*Intent, error) {
	if limit == 0 {
		limit = 50
	}

	query := `SELECT id, plan_id, prompt, module, created_at, updated_at FROM intents ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var intents []*Intent
	for rows.Next() {
		intent := &Intent{}
		err := rows.Scan(&intent.ID, &intent.PlanID, &intent.Prompt, &intent.Module, &intent.CreatedAt, &intent.UpdatedAt)
		if err != nil {
			return nil, err
		}
		intents = append(intents, intent)
	}

	return intents, nil
}

func (s *IntentStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM intent_items WHERE intent_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM intents WHERE id = ?`, id)
	return err
}
