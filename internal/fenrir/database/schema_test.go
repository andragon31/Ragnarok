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

func TestFenrirSchemaValidation(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_fenrir_test_"+fenrirRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "fenrir.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	tables := []string{
		"sessions", "observations", "nodes", "edges", "specs",
		"spec_deltas", "incidents", "conflicts", "velocity_metrics",
		"pending_rules", "commit_registry", "session_dna", "prompts",
		"intents", "intent_items", "bias_reports",
	}

	for _, table := range tables {
		t.Run("table_"+table, func(t *testing.T) {
			if !fenrirTableExists(db, table) {
				t.Errorf("table %s does not exist", table)
			}
		})
	}
}

func TestFenrirSessionsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_fenrir_test_"+fenrirRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "fenrir.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "project", "module", "started_at", "ended_at",
		"agent_id", "plan_id", "checkpoint_id",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !fenrirColumnExists(db, "sessions", col) {
				t.Errorf("column %s does not exist in sessions", col)
			}
		})
	}
}

func TestFenrirObservationsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_fenrir_test_"+fenrirRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "fenrir.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "session_id", "type", "content", "authority",
		"module", "file", "line", "tags", "authority_by", "authority_at",
		"created_at", "updated_at", "token_count", "is_compressed",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !fenrirColumnExists(db, "observations", col) {
				t.Errorf("column %s does not exist in observations", col)
			}
		})
	}
}

func TestFenrirNodesColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_fenrir_test_"+fenrirRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "fenrir.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "label", "type", "content", "metadata", "authority",
		"created_at", "updated_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !fenrirColumnExists(db, "nodes", col) {
				t.Errorf("column %s does not exist in nodes", col)
			}
		})
	}
}

func TestFenrirSpecsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_fenrir_test_"+fenrirRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "fenrir.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "module", "title", "content", "type",
		"given", "when_cond", "then_cond", "status",
		"implemented_at", "created_at", "updated_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !fenrirColumnExists(db, "specs", col) {
				t.Errorf("column %s does not exist in specs", col)
			}
		})
	}
}

func TestFenrirIntentsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_fenrir_test_"+fenrirRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "fenrir.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "plan_id", "prompt", "module", "created_at", "updated_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !fenrirColumnExists(db, "intents", col) {
				t.Errorf("column %s does not exist in intents", col)
			}
		})
	}
}

func TestFenrirIndexes(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_fenrir_test_"+fenrirRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "fenrir.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedIndexes := []string{
		"idx_observations_session",
		"idx_observations_module",
		"idx_observations_type",
		"idx_observations_created",
		"idx_nodes_label",
		"idx_nodes_type",
		"idx_edges_source",
		"idx_edges_target",
		"idx_specs_module",
		"idx_specs_status",
		"idx_incidents_module",
		"idx_incidents_status",
	}

	for _, idx := range expectedIndexes {
		t.Run("index_"+idx, func(t *testing.T) {
			if !fenrirIndexExists(db, idx) {
				t.Errorf("index %s does not exist", idx)
			}
		})
	}
}

func TestFenrirCRUDOperations(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_fenrir_test_"+fenrirRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "fenrir.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	now := time.Now()

	t.Run("session_crud", func(t *testing.T) {
		sessionID := "test_session_1"
		_, err := db.Exec(`INSERT INTO sessions (id, project, started_at) VALUES (?, ?, ?)`,
			sessionID, "TestProject", now)
		if err != nil {
			t.Fatalf("failed to insert session: %v", err)
		}

		var project string
		err = db.QueryRow(`SELECT project FROM sessions WHERE id = ?`, sessionID).Scan(&project)
		if err != nil {
			t.Fatalf("failed to select session: %v", err)
		}
		if project != "TestProject" {
			t.Errorf("expected project 'TestProject', got '%s'", project)
		}
	})

	t.Run("observation_crud", func(t *testing.T) {
		obsID := "test_obs_1"
		sessionID := "test_session_1"
		now := time.Now()

		_, err := db.Exec(`INSERT INTO observations (id, session_id, type, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			obsID, sessionID, "discovery", "Test observation content", now, now)
		if err != nil {
			t.Fatalf("failed to insert observation: %v", err)
		}

		var content string
		err = db.QueryRow(`SELECT content FROM observations WHERE id = ?`, obsID).Scan(&content)
		if err != nil {
			t.Fatalf("failed to select observation: %v", err)
		}
		if content != "Test observation content" {
			t.Errorf("expected content 'Test observation content', got '%s'", content)
		}
	})

	t.Run("spec_crud", func(t *testing.T) {
		specID := "test_spec_1"
		now := time.Now()

		_, err := db.Exec(`INSERT INTO specs (id, module, title, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			specID, "auth", "User Authentication", "Implement user login", "draft", now, now)
		if err != nil {
			t.Fatalf("failed to insert spec: %v", err)
		}

		var title string
		err = db.QueryRow(`SELECT title FROM specs WHERE id = ?`, specID).Scan(&title)
		if err != nil {
			t.Fatalf("failed to select spec: %v", err)
		}
		if title != "User Authentication" {
			t.Errorf("expected title 'User Authentication', got '%s'", title)
		}
	})
}

func fenrirRandomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func fenrirTableExists(db *sql.DB, tableName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`
	var name string
	err := db.QueryRow(query, tableName).Scan(&name)
	return err == nil
}

func fenrirColumnExists(db *sql.DB, table, column string) bool {
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

func fenrirIndexExists(db *sql.DB, indexName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='index' AND name=?`
	var name string
	err := db.QueryRow(query, indexName).Scan(&name)
	return err == nil
}
