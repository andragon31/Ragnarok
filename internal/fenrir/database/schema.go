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

	db.Exec("PRAGMA foreign_keys = ON")

	return db, nil
}

func InitSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		project TEXT NOT NULL,
		module TEXT,
		started_at DATETIME NOT NULL,
		ended_at DATETIME,
		agent_id TEXT,
		plan_id TEXT,
		checkpoint_id TEXT
	);

	CREATE TABLE IF NOT EXISTS observations (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		type TEXT NOT NULL,
		content TEXT NOT NULL,
		authority TEXT DEFAULT 'exploratory',
		module TEXT,
		file TEXT,
		line INTEGER,
		tags TEXT,
		authority_by TEXT,
		authority_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		token_count INTEGER DEFAULT 0,
		is_compressed INTEGER DEFAULT 0,
		metadata TEXT,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE TABLE IF NOT EXISTS nodes (
		id TEXT PRIMARY KEY,
		label TEXT NOT NULL,
		type TEXT NOT NULL,
		content TEXT,
		metadata TEXT,
		authority TEXT DEFAULT 'exploratory',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS edges (
		id TEXT PRIMARY KEY,
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		type TEXT NOT NULL,
		session_id TEXT,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (source_id) REFERENCES nodes(id),
		FOREIGN KEY (target_id) REFERENCES nodes(id),
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE TABLE IF NOT EXISTS specs (
		id TEXT PRIMARY KEY,
		module TEXT NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		type TEXT DEFAULT 'feature',
		given TEXT,
		when_cond TEXT,
		then_cond TEXT,
		status TEXT DEFAULT 'draft',
		implemented_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS spec_deltas (
		id TEXT PRIMARY KEY,
		spec_id TEXT NOT NULL,
		plan_id TEXT NOT NULL,
		type TEXT NOT NULL,
		description TEXT,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (spec_id) REFERENCES specs(id)
	);

	CREATE TABLE IF NOT EXISTS incidents (
		id TEXT PRIMARY KEY,
		module TEXT NOT NULL,
		summary TEXT NOT NULL,
		severity TEXT DEFAULT 'medium',
		status TEXT DEFAULT 'open',
		related_spec TEXT,
		solution TEXT,
		created_at DATETIME NOT NULL,
		resolved_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS conflicts (
		id TEXT PRIMARY KEY,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		local_content TEXT,
		remote_content TEXT,
		resolution TEXT,
		resolved_by TEXT,
		created_at DATETIME NOT NULL,
		resolved_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS velocity_metrics (
		id TEXT PRIMARY KEY,
		module TEXT NOT NULL,
		date DATE NOT NULL,
		quality_score REAL,
		velocity_score REAL,
		lines_added INTEGER DEFAULT 0,
		lines_removed INTEGER DEFAULT 0,
		test_coverage REAL,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS pending_rules (
		id TEXT PRIMARY KEY,
		rule_content TEXT NOT NULL,
		module TEXT,
		proposed_by TEXT,
		status TEXT DEFAULT 'pending',
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS commit_registry (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		commit_hash TEXT NOT NULL,
		plan_id TEXT,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS session_dna (
		session_id TEXT PRIMARY KEY,
		files_read INTEGER DEFAULT 0,
		files_written INTEGER DEFAULT 0,
		commands_run INTEGER DEFAULT 0,
		decisions_made INTEGER DEFAULT 0,
		quality_score REAL DEFAULT 0.0,
		tools_used TEXT,
		duration_seconds INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE TABLE IF NOT EXISTS prompts (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		content TEXT NOT NULL,
		module TEXT,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_observations_session ON observations(session_id);
	CREATE INDEX IF NOT EXISTS idx_observations_module ON observations(module);
	CREATE INDEX IF NOT EXISTS idx_observations_type ON observations(type);
	CREATE INDEX IF NOT EXISTS idx_observations_created ON observations(created_at);

	CREATE INDEX IF NOT EXISTS idx_nodes_label ON nodes(label);
	CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);

	CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
	CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);

	CREATE INDEX IF NOT EXISTS idx_specs_module ON specs(module);
	CREATE INDEX IF NOT EXISTS idx_specs_status ON specs(status);

	CREATE INDEX IF NOT EXISTS idx_incidents_module ON incidents(module);
	CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);

	CREATE TABLE IF NOT EXISTS intents (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		prompt TEXT NOT NULL,
		module TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS intent_items (
		id TEXT PRIMARY KEY,
		intent_id TEXT NOT NULL,
		description TEXT NOT NULL,
		type TEXT DEFAULT 'feature',
		status TEXT DEFAULT 'pending',
		match_score REAL DEFAULT 0.0,
		FOREIGN KEY (intent_id) REFERENCES intents(id)
	);

	CREATE TABLE IF NOT EXISTS bias_reports (
		id TEXT PRIMARY KEY,
		module TEXT NOT NULL,
		bias_type TEXT NOT NULL,
		severity TEXT NOT NULL,
		description TEXT,
		recommendation TEXT,
		created_at DATETIME NOT NULL
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(content, content=observations, content_rowid=rowid);
	CREATE VIRTUAL TABLE IF NOT EXISTS nodes_fts USING fts5(content, content=nodes, content_rowid=rowid);

	CREATE TABLE IF NOT EXISTS rule_registry (
		id TEXT PRIMARY KEY,
		fingerprint TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		pattern TEXT,
		content TEXT,
		severity TEXT DEFAULT 'medium',
		source TEXT DEFAULT 'generated',
		usage_count INTEGER DEFAULT 0,
		project_count INTEGER DEFAULT 0,
		tags TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS rule_usage_log (
		id TEXT PRIMARY KEY,
		rule_id TEXT NOT NULL,
		project_path TEXT,
		action TEXT,
		was_violation INTEGER DEFAULT 0,
		checked_at DATETIME NOT NULL,
		FOREIGN KEY (rule_id) REFERENCES rule_registry(id)
	);

	CREATE INDEX IF NOT EXISTS idx_rule_registry_fingerprint ON rule_registry(fingerprint);
	CREATE INDEX IF NOT EXISTS idx_rule_registry_category ON rule_registry(category);
	CREATE INDEX IF NOT EXISTS idx_rule_usage_log_rule ON rule_usage_log(rule_id);
	CREATE INDEX IF NOT EXISTS idx_rule_usage_log_project ON rule_usage_log(project_path);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Simple migration for existing DBs
	db.Exec(`ALTER TABLE observations ADD COLUMN metadata TEXT`)

	return nil
}
