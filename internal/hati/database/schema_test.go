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

func TestHatiSchemaValidation(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_hati_test_"+randomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "hati.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	tables := []string{
		"plans", "phases", "tasks", "task_agents", "checkpoints",
		"human_reviews", "notifications", "execution_blockers",
		"plan_recovery", "prds", "prd_requirements", "agent_locks",
		"feedback", "approval_record", "checkpoint_sla", "plan_quality_scores",
		"plan_dependencies", "agent_work",
	}

	for _, table := range tables {
		t.Run("table_"+table, func(t *testing.T) {
			if !tableExists(db, table) {
				t.Errorf("table %s does not exist", table)
			}
		})
	}
}

func tableExists(db *sql.DB, tableName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`
	var name string
	err := db.QueryRow(query, tableName).Scan(&name)
	return err == nil
}

func TestHatiAgentLocksColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_hati_test_"+randomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "hati.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "plan_id", "phase_id", "agent_id", "locked_at", "expires_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !columnExists(db, "agent_locks", col) {
				t.Errorf("column %s does not exist in agent_locks", col)
			}
		})
	}
}

func TestHatiCheckpointColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_hati_test_"+randomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "hati.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "plan_id", "phase_id", "type", "status", "can_continue",
		"risk_level", "spec_delta", "quality_snapshot", "created_at",
		"decided_at", "decided_by", "feedback",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !columnExists(db, "checkpoints", col) {
				t.Errorf("column %s does not exist in checkpoints", col)
			}
		})
	}
}

func TestHatiHumanReviewsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_hati_test_"+randomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "hati.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "review_type", "entity_type", "entity_id", "question",
		"status", "decision", "approver", "notes", "created_at", "decided_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !columnExists(db, "human_reviews", col) {
				t.Errorf("column %s does not exist in human_reviews", col)
			}
		})
	}
}

func TestHatiNotificationsColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_hati_test_"+randomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "hati.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "recipient", "type", "priority", "title", "message",
		"plan_id", "checkpoint_id", "webhook_url", "status", "sent_at", "created_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !columnExists(db, "notifications", col) {
				t.Errorf("column %s does not exist in notifications", col)
			}
		})
	}
}

func TestHatiPlanRecoveryColumns(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_hati_test_"+randomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "hati.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedColumns := []string{
		"id", "plan_id", "phase_id", "agent_id", "detected_state",
		"expected_state", "modified_files", "recovery_needed", "created_at",
	}

	for _, col := range expectedColumns {
		t.Run("column_"+col, func(t *testing.T) {
			if !columnExists(db, "plan_recovery", col) {
				t.Errorf("column %s does not exist in plan_recovery", col)
			}
		})
	}
}

func TestHatiCRUDOperations(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_hati_test_"+randomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "hati.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	t.Run("plan_crud", func(t *testing.T) {
		planID := "test_plan_1"
		now := "2024-01-15T10:00:00Z"

		_, err := db.Exec(`INSERT INTO plans (id, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			planID, "Test Plan", "active", now, now)
		if err != nil {
			t.Fatalf("failed to insert plan: %v", err)
		}

		var title string
		err = db.QueryRow(`SELECT title FROM plans WHERE id = ?`, planID).Scan(&title)
		if err != nil {
			t.Fatalf("failed to select plan: %v", err)
		}
		if title != "Test Plan" {
			t.Errorf("expected title 'Test Plan', got '%s'", title)
		}
	})

	t.Run("phase_crud", func(t *testing.T) {
		planID := "test_plan_1"
		phaseID := "test_phase_1"
		now := "2024-01-15T10:00:00Z"

		_, err := db.Exec(`INSERT INTO phases (id, plan_id, name, status, order_num, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			phaseID, planID, "Test Phase", "pending", 1, now, now)
		if err != nil {
			t.Fatalf("failed to insert phase: %v", err)
		}

		var name string
		err = db.QueryRow(`SELECT name FROM phases WHERE id = ?`, phaseID).Scan(&name)
		if err != nil {
			t.Fatalf("failed to select phase: %v", err)
		}
		if name != "Test Phase" {
			t.Errorf("expected name 'Test Phase', got '%s'", name)
		}
	})

	t.Run("task_crud", func(t *testing.T) {
		phaseID := "test_phase_1"
		taskID := "test_task_1"
		now := "2024-01-15T10:00:00Z"

		_, err := db.Exec(`INSERT INTO tasks (id, phase_id, title, status, priority, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			taskID, phaseID, "Test Task", "pending", "high", now, now)
		if err != nil {
			t.Fatalf("failed to insert task: %v", err)
		}

		var title string
		err = db.QueryRow(`SELECT title FROM tasks WHERE id = ?`, taskID).Scan(&title)
		if err != nil {
			t.Fatalf("failed to select task: %v", err)
		}
		if title != "Test Task" {
			t.Errorf("expected title 'Test Task', got '%s'", title)
		}
	})

	t.Run("task_agents_crud", func(t *testing.T) {
		taskID := "test_task_1"
		agentID := "test_agent_1"
		taskAgentID := "test_ta_1"
		now := "2024-01-15T10:00:00Z"

		_, err := db.Exec(`INSERT INTO task_agents (id, task_id, agent_id, role, status, started_at) VALUES (?, ?, ?, ?, ?, ?)`,
			taskAgentID, taskID, agentID, "executor", "pending", now)
		if err != nil {
			t.Fatalf("failed to insert task_agent: %v", err)
		}

		var role string
		err = db.QueryRow(`SELECT role FROM task_agents WHERE id = ?`, taskAgentID).Scan(&role)
		if err != nil {
			t.Fatalf("failed to select task_agent: %v", err)
		}
		if role != "executor" {
			t.Errorf("expected role 'executor', got '%s'", role)
		}
	})

	t.Run("checkpoint_crud", func(t *testing.T) {
		planID := "test_plan_1"
		checkpointID := "test_checkpoint_1"
		now := "2024-01-15T10:00:00Z"

		_, err := db.Exec(`INSERT INTO checkpoints (id, plan_id, type, status, created_at) VALUES (?, ?, ?, ?, ?)`,
			checkpointID, planID, "milestone", "pending", now)
		if err != nil {
			t.Fatalf("failed to insert checkpoint: %v", err)
		}

		var ctype string
		err = db.QueryRow(`SELECT type FROM checkpoints WHERE id = ?`, checkpointID).Scan(&ctype)
		if err != nil {
			t.Fatalf("failed to select checkpoint: %v", err)
		}
		if ctype != "milestone" {
			t.Errorf("expected type 'milestone', got '%s'", ctype)
		}
	})

	t.Run("human_review_crud", func(t *testing.T) {
		checkpointID := "test_checkpoint_1"
		reviewID := "test_review_1"
		now := "2024-01-15T10:00:00Z"

		_, err := db.Exec(`INSERT INTO human_reviews (id, review_type, entity_type, entity_id, question, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			reviewID, "checkpoint_approval", "checkpoint", checkpointID, "Approve?", "pending", now)
		if err != nil {
			t.Fatalf("failed to insert human_review: %v", err)
		}

		var question string
		err = db.QueryRow(`SELECT question FROM human_reviews WHERE id = ?`, reviewID).Scan(&question)
		if err != nil {
			t.Fatalf("failed to select human_review: %v", err)
		}
		if question != "Approve?" {
			t.Errorf("expected question 'Approve?', got '%s'", question)
		}
	})
}

func TestHatiIndexes(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "ragnarok_hati_test_"+randomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "hati.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	expectedIndexes := []string{
		"idx_plans_session",
		"idx_plans_status",
		"idx_phases_plan",
		"idx_checkpoints_plan",
		"idx_tasks_phase",
		"idx_tasks_status",
		"idx_task_agents_task",
		"idx_task_agents_agent",
		"idx_human_reviews_status",
		"idx_notifications_status",
	}

	for _, idx := range expectedIndexes {
		t.Run("index_"+idx, func(t *testing.T) {
			if !indexExists(db, idx) {
				t.Errorf("index %s does not exist", idx)
			}
		})
	}
}

func randomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
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

func indexExists(db *sql.DB, indexName string) bool {
	query := `SELECT name FROM sqlite_master WHERE type='index' AND name=?`
	var name string
	err := db.QueryRow(query, indexName).Scan(&name)
	return err == nil
}
