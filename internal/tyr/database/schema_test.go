package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestTyrSchemaValidation(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_tyr_test_"+tyrRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tyr.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	tables := []string{
		"pkg_cache", "sast_findings", "audit_log", "standards",
		"standards_results", "scope_violations", "cve_alerts", "secrets_findings",
	}

	for _, table := range tables {
		t.Run("table_"+table, func(t *testing.T) {
			if !tyrTableExists(db, table) {
				t.Errorf("table %s does not exist", table)
			}
		})
	}
}

func TestTyrPkgCacheColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_tyr_test_"+tyrRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tyr.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "ecosystem", "name", "version", "exists_pkg",
		"trusted", "cve_count", "license", "transitive_license_risk",
		"downloads", "age_days", "response", "cached_at", "expires_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !tyrColumnExists(db, "pkg_cache", col) {
				t.Errorf("column %s does not exist in pkg_cache", col)
			}
		})
	}
}

func TestTyrSastFindingsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_tyr_test_"+tyrRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tyr.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "session_id", "rule_id", "file", "line",
		"message", "severity", "owasp", "cwe",
		"status", "created_at", "resolved_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !tyrColumnExists(db, "sast_findings", col) {
				t.Errorf("column %s does not exist in sast_findings", col)
			}
		})
	}
}

func TestTyrStandardsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_tyr_test_"+tyrRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tyr.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "name", "description", "category", "last_result", "pass_rate", "created_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !tyrColumnExists(db, "standards", col) {
				t.Errorf("column %s does not exist in standards", col)
			}
		})
	}
}

func TestTyrStandardsResultsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_tyr_test_"+tyrRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tyr.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "session_id", "standard_id", "checkpoint",
		"passed", "metric_value", "output", "duration_ms", "ran_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !tyrColumnExists(db, "standards_results", col) {
				t.Errorf("column %s does not exist in standards_results", col)
			}
		})
	}
}

func TestTyrIndexes(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_tyr_test_"+tyrRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tyr.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedIndexes := []string{
		"idx_pkg_cache_name",
		"idx_pkg_cache_expiry",
		"idx_sast_findings_status",
		"idx_sast_findings_severity",
		"idx_audit_log_session",
		"idx_standards_results_session",
		"idx_scope_violations_session",
	}

	for _, idx := range expectedIndexes {
		t.Run("index_"+idx, func(t *testing.T) {
			if !tyrIndexExists(db, idx) {
				t.Errorf("index %s does not exist", idx)
			}
		})
	}
}

func TestTyrCRUDOperations(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_tyr_test_"+tyrRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tyr.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	t.Run("standard_crud", func(t *testing.T) {
		stdID := "test_std_1"
		_, err := db.Exec(`INSERT INTO standards (id, name, description, category) VALUES (?, ?, ?, ?)`,
			stdID, "Test Standard", "Test description", "security")
		if err != nil {
			t.Fatalf("failed to insert standard: %v", err)
		}

		var name string
		err = db.QueryRow(`SELECT name FROM standards WHERE id = ?`, stdID).Scan(&name)
		if err != nil {
			t.Fatalf("failed to select standard: %v", err)
		}
		if name != "Test Standard" {
			t.Errorf("expected name 'Test Standard', got '%s'", name)
		}
	})

	t.Run("standards_result_crud", func(t *testing.T) {
		resultID := "test_result_1"
		stdID := "test_std_1"
		now := time.Now()

		_, err := db.Exec(`INSERT INTO standards_results (id, session_id, standard_id, passed, ran_at) VALUES (?, ?, ?, ?, ?)`,
			resultID, "session_1", stdID, 1, now)
		if err != nil {
			t.Fatalf("failed to insert standard result: %v", err)
		}

		var passed int
		err = db.QueryRow(`SELECT passed FROM standards_results WHERE id = ?`, resultID).Scan(&passed)
		if err != nil {
			t.Fatalf("failed to select standard result: %v", err)
		}
		if passed != 1 {
			t.Errorf("expected passed 1, got %d", passed)
		}
	})

	t.Run("audit_log_crud", func(t *testing.T) {
		logID := "test_log_1"

		_, err := db.Exec(`INSERT INTO audit_log (id, tool, action_type) VALUES (?, ?, ?)`,
			logID, "test_tool", "create")
		if err != nil {
			t.Fatalf("failed to insert audit log: %v", err)
		}

		var tool string
		err = db.QueryRow(`SELECT tool FROM audit_log WHERE id = ?`, logID).Scan(&tool)
		if err != nil {
			t.Fatalf("failed to select audit log: %v", err)
		}
		if tool != "test_tool" {
			t.Errorf("expected tool 'test_tool', got '%s'", tool)
		}
	})
}

func tyrRandomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func tyrTableExists(db *sql.DB, tableName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`
	var name string
	err := db.QueryRow(query, tableName).Scan(&name)
	return err == nil
}

func tyrColumnExists(db *sql.DB, table, column string) bool {
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

func tyrIndexExists(db *sql.DB, indexName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='index' AND name=?`
	var name string
	err := db.QueryRow(query, indexName).Scan(&name)
	return err == nil
}
