package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	schema := `
	-- Enable foreign keys
	PRAGMA foreign_keys = ON;

	-- Providers table
	CREATE TABLE IF NOT EXISTS providers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		endpoint TEXT NOT NULL,
		api_key_encrypted TEXT,
		description TEXT,
		website TEXT,
		support_email TEXT,
		documentation_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_checked TIMESTAMP,
		is_active BOOLEAN DEFAULT 1,
		reliability_score REAL DEFAULT 0.0,
		average_response_time_ms INTEGER DEFAULT 0
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		fmt.Printf("SQL Error: %v\n", err)
		fmt.Printf("Schema: %s\n", schema)
	} else {
		fmt.Println("Schema executed successfully")
	}
}