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

	fmt.Println("TABLE: phases")
	rows, _ := db.Query("SELECT id, name, plan_id FROM phases")
	pMap := make(map[string]bool)
	for rows.Next() {
		var id, name, pID string
		rows.Scan(&id, &name, &pID)
		fmt.Printf("Phase: %s | Plan: %s | Name: %s\n", id, pID, name)
		pMap[id] = true
	}
	rows.Close()

	fmt.Println("\nTABLE: tasks")
	rows, _ = db.Query("SELECT id, phase_id, title FROM tasks")
	for rows.Next() {
		var id, phID, title string
		rows.Scan(&id, &phID, &title)
		exists := pMap[phID]
		fmt.Printf("Task: %s | PhaseID: %s (Exists: %v) | Title: %s\n", id, phID, exists, title)
	}
	rows.Close()
}
