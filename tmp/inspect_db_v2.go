package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".ragnarok", "hati.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("PLAN PROGRESS CHECK")
	rows, _ := db.Query(`
		SELECT p.id, p.name, 
		(SELECT COUNT(*) FROM tasks t JOIN phases ph ON t.phase_id = ph.id WHERE ph.plan_id = p.id) as total,
		(SELECT COUNT(*) FROM tasks t JOIN phases ph ON t.phase_id = ph.id WHERE ph.plan_id = p.id AND t.status IN ('pending', 'blocked')) as pending
		FROM plans p
		ORDER BY p.created_at DESC LIMIT 3
	`)
	for rows.Next() {
		var id, name string
		var total, pending int
		rows.Scan(&id, &name, &total, &pending)
		fmt.Printf("Plan: %s (%s) | Total: %d | Pending: %d\n", id, name, total, pending)
	}
	rows.Close()

	fmt.Println("\nLAST 5 TASKS")
	rows, _ = db.Query("SELECT t.id, t.phase_id, t.title, t.status FROM tasks t ORDER BY t.created_at DESC LIMIT 5")
	for rows.Next() {
		var id, phaseID, title, status string
		rows.Scan(&id, &phaseID, &title, &status)
		fmt.Printf("Task: %s | Status: %s | Title: %s\n", id, status, title)
	}
	rows.Close()
}
