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
	fmt.Println("Path:", dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("TABLES FOUND:")
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Println("-", name)
	}
}
