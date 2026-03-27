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

	CREATE TABLE IF NOT EXISTS notifications (
		id TEXT PRIMARY KEY,
		recipient TEXT NOT NULL,
		type TEXT NOT NULL,
		priority TEXT DEFAULT 'normal',
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		plan_id TEXT,
		checkpoint_id TEXT,
		webhook_url TEXT,
		status TEXT DEFAULT 'pending',
		sent_at DATETIME,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (plan_id) REFERENCES plans(id),
		FOREIGN KEY (checkpoint_id) REFERENCES checkpoints(id)
	);

	CREATE TABLE IF NOT EXISTS plan_dependencies (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		depends_on_plan_id TEXT NOT NULL,
		dependency_type TEXT DEFAULT 'blocking',
		status TEXT DEFAULT 'pending',
		FOREIGN KEY (plan_id) REFERENCES plans(id),
		FOREIGN KEY (depends_on_plan_id) REFERENCES plans(id)
	);

	CREATE TABLE IF NOT EXISTS plan_recovery (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		phase_id TEXT,
		agent_id TEXT,
		detected_state TEXT NOT NULL,
		expected_state TEXT NOT NULL,
		modified_files TEXT,
		recovery_needed INTEGER DEFAULT 0,
		resolved_at DATETIME,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (plan_id) REFERENCES plans(id),
		FOREIGN KEY (phase_id) REFERENCES phases(id)
	);

	CREATE TABLE IF NOT EXISTS agent_locks (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		phase_id TEXT,
		agent_id TEXT NOT NULL,
		locked_at DATETIME NOT NULL,
		expires_at DATETIME,
		FOREIGN KEY (plan_id) REFERENCES plans(id),
		FOREIGN KEY (phase_id) REFERENCES phases(id)
	);

	CREATE TABLE IF NOT EXISTS agent_work (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		agent_name TEXT,
		plan_id TEXT NOT NULL,
		phase_id TEXT,
		status TEXT DEFAULT 'active',
		started_at DATETIME NOT NULL,
		heartbeat_at DATETIME,
		FOREIGN KEY (plan_id) REFERENCES plans(id),
		FOREIGN KEY (phase_id) REFERENCES phases(id)
	);

	CREATE TABLE IF NOT EXISTS prds (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		version TEXT DEFAULT '1.0',
		content TEXT,
		file_path TEXT,
		status TEXT DEFAULT 'draft',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS prd_requirements (
		id TEXT PRIMARY KEY,
		prd_id TEXT NOT NULL,
		req_type TEXT NOT NULL,
		priority TEXT DEFAULT 'medium',
		title TEXT NOT NULL,
		description TEXT,
		acceptance_criteria TEXT,
		status TEXT DEFAULT 'pending',
		FOREIGN KEY (prd_id) REFERENCES prds(id)
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		phase_id TEXT NOT NULL,
		prd_requirement_id TEXT,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'pending',
		priority INTEGER DEFAULT 0,
		assigned_agent_id TEXT,
		assigned_agent_type TEXT,
		estimated_hours REAL,
		actual_hours REAL,
		notes TEXT,
		blocker TEXT,
		milestone INTEGER DEFAULT 0,
		subtasks TEXT,
		completed_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (phase_id) REFERENCES phases(id),
		FOREIGN KEY (prd_requirement_id) REFERENCES prd_requirements(id)
	);

	CREATE TABLE IF NOT EXISTS human_reviews (
		id TEXT PRIMARY KEY,
		review_type TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		question TEXT,
		decision TEXT,
		approver TEXT,
		notes TEXT,
		status TEXT DEFAULT 'pending',
		created_at DATETIME NOT NULL,
		decided_at DATETIME,
		FOREIGN KEY (entity_id) REFERENCES plans(id)
	);

	CREATE TABLE IF NOT EXISTS checkpoint_sla (
		id TEXT PRIMARY KEY,
		checkpoint_id TEXT NOT NULL,
		sla_hours INTEGER NOT NULL,
		escalation_level INTEGER DEFAULT 1,
		escalation_recipients TEXT,
		created_at DATETIME NOT NULL,
		expires_at DATETIME,
		escalated_at DATETIME,
		FOREIGN KEY (checkpoint_id) REFERENCES checkpoints(id)
	);

	CREATE INDEX IF NOT EXISTS idx_plans_session ON plans(session_id);
	CREATE INDEX IF NOT EXISTS idx_plans_status ON plans(status);
	CREATE INDEX IF NOT EXISTS idx_phases_plan ON phases(plan_id);
	CREATE INDEX IF NOT EXISTS idx_checkpoints_plan ON checkpoints(plan_id);
	CREATE INDEX IF NOT EXISTS idx_checkpoints_type ON checkpoints(type);
	CREATE INDEX IF NOT EXISTS idx_plan_revisions_plan ON plan_revisions(plan_id);
	CREATE INDEX IF NOT EXISTS idx_execution_blockers_plan ON execution_blockers(plan_id);
	CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
	CREATE INDEX IF NOT EXISTS idx_notifications_plan ON notifications(plan_id);
	CREATE INDEX IF NOT EXISTS idx_plan_dependencies_plan ON plan_dependencies(plan_id);
	CREATE INDEX IF NOT EXISTS idx_plan_recovery_plan ON plan_recovery(plan_id);
	CREATE INDEX IF NOT EXISTS idx_agent_locks_plan ON agent_locks(plan_id);
	CREATE INDEX IF NOT EXISTS idx_agent_work_plan ON agent_work(plan_id);
	CREATE INDEX IF NOT EXISTS idx_checkpoint_sla_expires ON checkpoint_sla(expires_at);
	CREATE INDEX IF NOT EXISTS idx_prds_status ON prds(status);
	CREATE INDEX IF NOT EXISTS idx_prd_requirements_prd ON prd_requirements(prd_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_phase ON tasks(phase_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_assigned ON tasks(assigned_agent_id);
	CREATE INDEX IF NOT EXISTS idx_human_reviews_entity ON human_reviews(entity_type, entity_id);
	CREATE INDEX IF NOT EXISTS idx_human_reviews_status ON human_reviews(status);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}
