package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Read the actual schema from database.go
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

	-- Models table
	CREATE TABLE IF NOT EXISTS models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id INTEGER NOT NULL,
		model_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		version TEXT,
		architecture TEXT,
		parameter_count INTEGER,
		context_window_tokens INTEGER,
		max_output_tokens INTEGER,
		training_data_cutoff DATE,
		release_date DATE,
		is_multimodal BOOLEAN DEFAULT 0,
		supports_vision BOOLEAN DEFAULT 0,
		supports_audio BOOLEAN DEFAULT 0,
		supports_video BOOLEAN DEFAULT 0,
		supports_reasoning BOOLEAN DEFAULT 0,
		open_source BOOLEAN DEFAULT 0,
		deprecated BOOLEAN DEFAULT 0,
		tags TEXT,
		language_support TEXT,
		use_case TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_verified TIMESTAMP,
		verification_status TEXT DEFAULT 'pending',
		overall_score REAL DEFAULT 0.0,
		code_capability_score REAL DEFAULT 0.0,
		responsiveness_score REAL DEFAULT 0.0,
		reliability_score REAL DEFAULT 0.0,
		feature_richness_score REAL DEFAULT 0.0,
		value_proposition_score REAL DEFAULT 0.0,
		FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
	);
	`

	// Split and execute statements individually
	statements := strings.Split(schema, ";")
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		
		fmt.Printf("Executing statement %d: %.50s...\n", i+1, stmt)
		_, err := db.Exec(stmt)
		if err != nil {
			fmt.Printf("ERROR in statement %d: %v\n", i+1, err)
			fmt.Printf("Full statement: %s\n", stmt)
			return
		}
		fmt.Printf("Statement %d executed successfully\n", i+1)
	}
	
	fmt.Println("All statements executed successfully")
}