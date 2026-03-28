package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

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
	tables := `
	CREATE TABLE IF NOT EXISTS skills (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		license TEXT,
		compatibility TEXT,
		framework TEXT,
		min_version TEXT,
		max_version TEXT,
		last_verified DATETIME,
		source TEXT DEFAULT 'local',
		tags TEXT,
		has_scripts INTEGER DEFAULT 0,
		has_references INTEGER DEFAULT 0,
		has_assets INTEGER DEFAULT 0,
		allowed_tools TEXT,
		path TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS rules (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		content TEXT NOT NULL,
		severity TEXT DEFAULT 'medium',
		scope TEXT DEFAULT 'global',
		status TEXT DEFAULT 'active',
		is_active INTEGER DEFAULT 1,
		source TEXT DEFAULT 'local',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		agent_type TEXT,
		role TEXT,
		scope TEXT,
		skills TEXT,
		allowed_tools TEXT,
		capabilities TEXT,
		status TEXT DEFAULT 'idle',
		current_task TEXT,
		is_active INTEGER DEFAULT 0,
		last_active DATETIME,
		last_heartbeat DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS teams (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		project_path TEXT,
		status TEXT DEFAULT 'active',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS team_members (
		id TEXT PRIMARY KEY,
		team_id TEXT NOT NULL,
		agent_id TEXT NOT NULL,
		role TEXT DEFAULT 'member',
		joined_at DATETIME NOT NULL,
		FOREIGN KEY (team_id) REFERENCES teams(id),
		FOREIGN KEY (agent_id) REFERENCES agents(id)
	);

	CREATE TABLE IF NOT EXISTS agent_tasks (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		task_type TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'pending',
		result TEXT,
		error TEXT,
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		FOREIGN KEY (agent_id) REFERENCES agents(id)
	);

	CREATE TABLE IF NOT EXISTS workflows (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT DEFAULT 'started',
		description TEXT,
		phases TEXT,
		standards TEXT,
		is_active INTEGER DEFAULT 1,
		deprecated INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS task_executions (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		hati_task_id TEXT,
		agent_id TEXT NOT NULL,
		phase_id TEXT,
		status TEXT DEFAULT 'pending',
		result TEXT,
		error TEXT,
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		heartbeat_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS pending_rules (
		id TEXT PRIMARY KEY,
		rule_id TEXT NOT NULL,
		proposed_by TEXT,
		reason TEXT,
		status TEXT DEFAULT 'pending',
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS team_context (
		id TEXT PRIMARY KEY,
		module TEXT NOT NULL UNIQUE,
		scope TEXT,
		skills TEXT,
		rules TEXT,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name);
	CREATE INDEX IF NOT EXISTS idx_skills_framework ON skills(framework);
	CREATE INDEX IF NOT EXISTS idx_skills_source ON skills(source);
	CREATE INDEX IF NOT EXISTS idx_rules_category ON rules(category);
	CREATE INDEX IF NOT EXISTS idx_rules_active ON rules(is_active);
	CREATE INDEX IF NOT EXISTS idx_agents_active ON agents(is_active);
	CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);
	CREATE INDEX IF NOT EXISTS idx_task_executions_task ON task_executions(task_id);
	CREATE INDEX IF NOT EXISTS idx_task_executions_agent ON task_executions(agent_id);
	CREATE INDEX IF NOT EXISTS idx_task_executions_status ON task_executions(status);
	`

	indexes := `
	CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name);
	CREATE INDEX IF NOT EXISTS idx_skills_framework ON skills(framework);
	CREATE INDEX IF NOT EXISTS idx_skills_source ON skills(source);
	CREATE INDEX IF NOT EXISTS idx_rules_category ON rules(category);
	CREATE INDEX IF NOT EXISTS idx_rules_active ON rules(is_active);
	CREATE INDEX IF NOT EXISTS idx_agents_active ON agents(is_active);
	CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);
	CREATE INDEX IF NOT EXISTS idx_task_executions_task ON task_executions(task_id);
	CREATE INDEX IF NOT EXISTS idx_task_executions_agent ON task_executions(task_id);
	CREATE INDEX IF NOT EXISTS idx_task_executions_status ON task_executions(status);
	`

	_, err := db.Exec(tables)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	db.Exec(indexes)

	if columnExists(db, "workflows", "deprecated") {
		db.Exec(`CREATE INDEX IF NOT EXISTS idx_workflows_deprecated ON workflows(deprecated)`)
	}

	return nil
}

func runMigrations(db *sql.DB) error {
	migrations := []struct {
		table   string
		column  string
		colType string
		addSQL  string
	}{
		{"agents", "capabilities", "TEXT", `ALTER TABLE agents ADD COLUMN capabilities TEXT`},
		{"agents", "agent_type", "TEXT", `ALTER TABLE agents ADD COLUMN agent_type TEXT`},
		{"agents", "allowed_tools", "TEXT", `ALTER TABLE agents ADD COLUMN allowed_tools TEXT`},
		{"workflows", "deprecated", "INTEGER", `ALTER TABLE workflows ADD COLUMN deprecated INTEGER DEFAULT 0`},
	}

	for _, m := range migrations {
		if !columnExists(db, m.table, m.column) {
			_, err := db.Exec(m.addSQL)
			if err != nil && !strings.Contains(err.Error(), "duplicate column") && !strings.Contains(err.Error(), "already exists") {
				log.Printf("Migration warning for %s.%s: %v", m.table, m.column, err)
			}
		}
	}

	return nil
}

func columnExists(db *sql.DB, table, column string) bool {
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	rows, err := db.Query(query)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt_value interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk); err != nil {
			continue
		}
		if name == column {
			return true
		}
	}
	return false
}
