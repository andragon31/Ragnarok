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

func TestSkollSchemaValidation(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_skoll_test_"+skollRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "skoll.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	tables := []string{
		"agents", "skills", "rules", "task_executions",
		"workflows", "teams", "team_members", "agent_tasks",
		"pending_rules", "team_context",
	}

	for _, table := range tables {
		t.Run("table_"+table, func(t *testing.T) {
			if !skollTableExists(db, table) {
				t.Errorf("table %s does not exist", table)
			}
		})
	}
}

func TestSkollAgentsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_skoll_test_"+skollRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "skoll.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "name", "agent_type", "role", "scope",
		"skills", "allowed_tools", "capabilities",
		"status", "current_task", "is_active",
		"last_active", "last_heartbeat", "created_at", "updated_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !skollColumnExists(db, "agents", col) {
				t.Errorf("column %s does not exist in agents", col)
			}
		})
	}
}

func TestSkollSkillsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_skoll_test_"+skollRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "skoll.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "name", "description", "framework",
		"min_version", "max_version", "source", "tags",
		"allowed_tools", "path", "created_at", "updated_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !skollColumnExists(db, "skills", col) {
				t.Errorf("column %s does not exist in skills", col)
			}
		})
	}
}

func TestSkollTaskExecutionsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_skoll_test_"+skollRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "skoll.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "task_id", "hati_task_id", "agent_id", "phase_id",
		"status", "result", "error",
		"started_at", "completed_at", "heartbeat_at",
		"created_at", "updated_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !skollColumnExists(db, "task_executions", col) {
				t.Errorf("column %s does not exist in task_executions", col)
			}
		})
	}
}

func TestSkollWorkflowsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_skoll_test_"+skollRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "skoll.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "name", "type", "status", "description",
		"phases", "standards", "is_active", "deprecated",
		"created_at", "updated_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !skollColumnExists(db, "workflows", col) {
				t.Errorf("column %s does not exist in workflows", col)
			}
		})
	}
}

func TestSkollIndexes(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_skoll_test_"+skollRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "skoll.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedIndexes := []string{
		"idx_skills_name",
		"idx_skills_framework",
		"idx_agents_active",
		"idx_agents_name",
		"idx_task_executions_task",
		"idx_task_executions_agent",
		"idx_task_executions_status",
		"idx_workflows_deprecated",
	}

	for _, idx := range expectedIndexes {
		t.Run("index_"+idx, func(t *testing.T) {
			if !skollIndexExists(db, idx) {
				t.Errorf("index %s does not exist", idx)
			}
		})
	}
}

func TestSkollCRUDOperations(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_skoll_test_"+skollRandomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "skoll.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	now := time.Now()

	t.Run("agent_crud", func(t *testing.T) {
		agentID := "test_agent_1"
		_, err := db.Exec(`INSERT INTO agents (id, name, agent_type, role, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			agentID, "TestAgent", "backend", "worker", "idle", now, now)
		if err != nil {
			t.Fatalf("failed to insert agent: %v", err)
		}

		var name string
		err = db.QueryRow(`SELECT name FROM agents WHERE id = ?`, agentID).Scan(&name)
		if err != nil {
			t.Fatalf("failed to select agent: %v", err)
		}
		if name != "TestAgent" {
			t.Errorf("expected name 'TestAgent', got '%s'", name)
		}
	})

	t.Run("skill_crud", func(t *testing.T) {
		skillID := "test_skill_1"
		_, err := db.Exec(`INSERT INTO skills (id, name, description, framework, min_version, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			skillID, "TestSkill", "Test skill", "go", "1.0.0", "local", now, now)
		if err != nil {
			t.Fatalf("failed to insert skill: %v", err)
		}

		var name string
		err = db.QueryRow(`SELECT name FROM skills WHERE id = ?`, skillID).Scan(&name)
		if err != nil {
			t.Fatalf("failed to select skill: %v", err)
		}
		if name != "TestSkill" {
			t.Errorf("expected name 'TestSkill', got '%s'", name)
		}
	})

	t.Run("task_execution_crud", func(t *testing.T) {
		execID := "test_exec_1"
		agentID := "test_agent_1"
		now := time.Now()

		_, err := db.Exec(`INSERT INTO task_executions (id, task_id, agent_id, status, started_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			execID, "task_1", agentID, "in_progress", now, now, now)
		if err != nil {
			t.Fatalf("failed to insert task_execution: %v", err)
		}

		var status string
		err = db.QueryRow(`SELECT status FROM task_executions WHERE id = ?`, execID).Scan(&status)
		if err != nil {
			t.Fatalf("failed to select task_execution: %v", err)
		}
		if status != "in_progress" {
			t.Errorf("expected status 'in_progress', got '%s'", status)
		}
	})

	t.Run("team_crud", func(t *testing.T) {
		teamID := "test_team_1"
		_, err := db.Exec(`INSERT INTO teams (id, name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			teamID, "TestTeam", "active", now, now)
		if err != nil {
			t.Fatalf("failed to insert team: %v", err)
		}

		var name string
		err = db.QueryRow(`SELECT name FROM teams WHERE id = ?`, teamID).Scan(&name)
		if err != nil {
			t.Fatalf("failed to select team: %v", err)
		}
		if name != "TestTeam" {
			t.Errorf("expected name 'TestTeam', got '%s'", name)
		}
	})
}

func skollRandomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func skollTableExists(db *sql.DB, tableName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`
	var name string
	err := db.QueryRow(query, tableName).Scan(&name)
	return err == nil
}

func skollColumnExists(db *sql.DB, table, column string) bool {
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

func skollIndexExists(db *sql.DB, indexName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='index' AND name=?`
	var name string
	err := db.QueryRow(query, indexName).Scan(&name)
	return err == nil
}
