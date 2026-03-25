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
	CREATE TABLE IF NOT EXISTS pkg_cache (
		id TEXT PRIMARY KEY,
		ecosystem TEXT NOT NULL,
		name TEXT NOT NULL,
		version TEXT,
		exists_pkg INTEGER NOT NULL,
		trusted INTEGER DEFAULT 1,
		cve_count INTEGER DEFAULT 0,
		license TEXT,
		transitive_license_risk TEXT DEFAULT 'none',
		downloads INTEGER DEFAULT 0,
		age_days INTEGER DEFAULT 0,
		response TEXT,
		cached_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sast_findings (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		rule_id TEXT NOT NULL,
		file TEXT NOT NULL,
		line INTEGER,
		message TEXT NOT NULL,
		severity TEXT NOT NULL,
		owasp TEXT,
		cwe TEXT,
		status TEXT DEFAULT 'open',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		resolved_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS audit_log (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		tool TEXT NOT NULL,
		action_type TEXT NOT NULL,
		target TEXT,
		risk_level TEXT DEFAULT 'low',
		result TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS standards (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		category TEXT,
		last_result TEXT,
		pass_rate REAL DEFAULT 0.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS standards_results (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		standard_id TEXT NOT NULL,
		checkpoint TEXT,
		passed INTEGER NOT NULL,
		metric_value REAL,
		output TEXT,
		duration_ms INTEGER,
		ran_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS scope_violations (
		id TEXT PRIMARY KEY,
		session_id TEXT,
		module TEXT,
		violation_type TEXT,
		target TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS cve_alerts (
		id TEXT PRIMARY KEY,
		package_id TEXT NOT NULL,
		cve_id TEXT NOT NULL,
		severity TEXT NOT NULL,
		summary TEXT,
		detected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		acknowledged INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS secrets_findings (
		id TEXT PRIMARY KEY,
		file TEXT NOT NULL,
		line INTEGER,
		type TEXT NOT NULL,
		secret_preview TEXT,
		status TEXT DEFAULT 'open',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_pkg_cache_name ON pkg_cache(ecosystem, name);
	CREATE INDEX IF NOT EXISTS idx_pkg_cache_expiry ON pkg_cache(expires_at);
	CREATE INDEX IF NOT EXISTS idx_sast_findings_status ON sast_findings(status);
	CREATE INDEX IF NOT EXISTS idx_sast_findings_severity ON sast_findings(severity);
	CREATE INDEX IF NOT EXISTS idx_audit_log_session ON audit_log(session_id);
	CREATE INDEX IF NOT EXISTS idx_standards_results_session ON standards_results(session_id);
	CREATE INDEX IF NOT EXISTS idx_scope_violations_session ON scope_violations(session_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}
