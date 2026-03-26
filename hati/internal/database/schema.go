package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func NewDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func InitSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS plans (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'draft',
		risk_level TEXT DEFAULT 'medium',
		spec_impact TEXT,
		module_hints_used TEXT,
		quality_source TEXT DEFAULT 'tyr',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		completed_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS phases (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		risk_level TEXT DEFAULT 'medium',
		status TEXT DEFAULT 'pending',
		order_num INTEGER NOT NULL,
		agents_md_hints TEXT,
		spec_ids_affected TEXT,
		module TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (plan_id) REFERENCES plans(id)
	);

	CREATE TABLE IF NOT EXISTS checkpoints (
		id TEXT PRIMARY KEY,
		plan_id TEXT,
		phase_id TEXT,
		type TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		can_continue INTEGER DEFAULT 0,
		risk_level TEXT,
		spec_delta TEXT,
		quality_snapshot TEXT,
		created_at DATETIME NOT NULL,
		decided_at DATETIME,
		decided_by TEXT,
		feedback TEXT,
		FOREIGN KEY (plan_id) REFERENCES plans(id),
		FOREIGN KEY (phase_id) REFERENCES phases(id)
	);

	CREATE TABLE IF NOT EXISTS feedback (
		id TEXT PRIMARY KEY,
		checkpoint_id TEXT NOT NULL,
		type TEXT NOT NULL,
		content TEXT NOT NULL,
		author TEXT,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (checkpoint_id) REFERENCES checkpoints(id)
	);

	CREATE TABLE IF NOT EXISTS approval_record (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		decision TEXT NOT NULL,
		approver TEXT,
		notes TEXT,
		spec_deltas TEXT,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (plan_id) REFERENCES plans(id)
	);

	CREATE TABLE IF NOT EXISTS plan_quality_scores (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		plan_completeness REAL,
		execution_quality REAL,
		overall_score REAL,
		calculated_at DATETIME NOT NULL,
		FOREIGN KEY (plan_id) REFERENCES plans(id)
	);

	CREATE TABLE IF NOT EXISTS commit_registry (
		id TEXT PRIMARY KEY,
		plan_id TEXT,
		commit_hash TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS plan_revisions (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		feedback_id TEXT,
		previous_state TEXT,
		new_state TEXT,
		changes_summary TEXT,
		status TEXT DEFAULT 'pending',
		created_at DATETIME NOT NULL,
		applied_at DATETIME,
		FOREIGN KEY (plan_id) REFERENCES plans(id),
		FOREIGN KEY (feedback_id) REFERENCES feedback(id)
	);

	CREATE TABLE IF NOT EXISTS execution_blockers (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		checkpoint_id TEXT,
		reason TEXT NOT NULL,
		type TEXT NOT NULL,
		blocked_at DATETIME NOT NULL,
		resolved_at DATETIME,
		FOREIGN KEY (plan_id) REFERENCES plans(id),
		FOREIGN KEY (checkpoint_id) REFERENCES checkpoints(id)
	);

	CREATE INDEX IF NOT EXISTS idx_plans_session ON plans(session_id);
	CREATE INDEX IF NOT EXISTS idx_plans_status ON plans(status);
	CREATE INDEX IF NOT EXISTS idx_phases_plan ON phases(plan_id);
	CREATE INDEX IF NOT EXISTS idx_checkpoints_plan ON checkpoints(plan_id);
	CREATE INDEX IF NOT EXISTS idx_checkpoints_type ON checkpoints(type);
	CREATE INDEX IF NOT EXISTS idx_plan_revisions_plan ON plan_revisions(plan_id);
	CREATE INDEX IF NOT EXISTS idx_execution_blockers_plan ON execution_blockers(plan_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}
