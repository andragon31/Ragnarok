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
	rows, _ := db.Query("SELECT id, name, plan_id, status FROM phases")
	for rows.Next() {
		var id, name, planID, status string
		rows.Scan(&id, &name, &planID, &status)
		fmt.Printf("  ID: %s, Name: %s, Plan: %s, Status: %s\n", id, name, planID, status)
	}
	rows.Close()

	fmt.Println("\nTABLE: tasks")
	rows, _ = db.Query("SELECT id, phase_id, title, status FROM tasks")
	for rows.Next() {
		var id, phaseID, title, status string
		rows.Scan(&id, &phaseID, &title, &status)
		fmt.Printf("  ID: %s, Phase: %s, Title: %s, Status: %s\n", id, phaseID, title, status)
	}
	rows.Close()
}
