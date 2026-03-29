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
	dbPath := filepath.Join(home, ".ragnarok", ".hati", "hati.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("--- PLANS ---")
	rows, _ := db.Query("SELECT id, title FROM plans")
	for rows != nil && rows.Next() {
		var id, title string
		rows.Scan(&id, &title)
		fmt.Printf("[%s] %s\n", id, title)
	}
	if rows != nil { rows.Close() }

	fmt.Println("\n--- PHASES ---")
	rows, _ = db.Query("SELECT id, plan_id, name FROM phases")
	for rows != nil && rows.Next() {
		var id, pID, name string
		rows.Scan(&id, &pID, &name)
		fmt.Printf("[%s] Plan: %s | %s\n", id, pID, name)
	}
	if rows != nil { rows.Close() }

	fmt.Println("\n--- TASKS ---")
	rows, _ = db.Query("SELECT id, phase_id, title FROM tasks")
	for rows != nil && rows.Next() {
		var id, phID, title string
		rows.Scan(&id, &phID, &title)
		fmt.Printf("[%s] Phase: %s | %s\n", id, phID, title)
	}
	if rows != nil { rows.Close() }
}
