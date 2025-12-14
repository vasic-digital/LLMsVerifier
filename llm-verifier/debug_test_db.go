package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Simulate the test environment
	tempDir := "/tmp/test_debug"
	dbPath := filepath.Join(tempDir, "test.db")
	
	// Create directory
	// os.MkdirAll(tempDir, 0755)
	
	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Try to execute the exact schema from database.go
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

-- Verification results table
CREATE TABLE IF NOT EXISTS verification_results (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	model_id INTEGER NOT NULL,
	verification_type TEXT NOT NULL,
	started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	completed_at TIMESTAMP,
	status TEXT DEFAULT 'running',
	error_message TEXT,
	exists BOOLEAN,
	responsive BOOLEAN,
	overloaded BOOLEAN,
	latency_ms INTEGER,
	supports_tool_use BOOLEAN DEFAULT 0,
	supports_function_calling BOOLEAN DEFAULT 0,
	supports_code_generation BOOLEAN DEFAULT 0,
	supports_code_completion BOOLEAN DEFAULT 0,
	supports_code_review BOOLEAN DEFAULT 0,
	supports_code_explanation BOOLEAN DEFAULT 0,
	supports_embeddings BOOLEAN DEFAULT 0,
	supports_reranking BOOLEAN DEFAULT 0,
	supports_image_generation BOOLEAN DEFAULT 0,
	supports_audio_generation BOOLEAN DEFAULT 0,
	supports_video_generation BOOLEAN DEFAULT 0,
	supports_mcps BOOLEAN DEFAULT 0,
	supports_lsps BOOLEAN DEFAULT 0,
	supports_multimodal BOOLEAN DEFAULT 0,
	supports_streaming BOOLEAN DEFAULT 0,
	supports_json_mode BOOLEAN DEFAULT 0,
	supports_structured_output BOOLEAN DEFAULT 0,
	supports_reasoning BOOLEAN DEFAULT 0,
	supports_parallel_tool_use BOOLEAN DEFAULT 0,
	max_parallel_calls INTEGER DEFAULT 0,
	supports_batch_processing BOOLEAN DEFAULT 0,
	code_language_support TEXT,
	code_debugging BOOLEAN DEFAULT 0,
	code_optimization BOOLEAN DEFAULT 0,
	test_generation BOOLEAN DEFAULT 0,
	documentation_generation BOOLEAN DEFAULT 0,
	refactoring BOOLEAN DEFAULT 0,
	error_resolution BOOLEAN DEFAULT 0,
	architecture_design BOOLEAN DEFAULT 0,
	security_assessment BOOLEAN DEFAULT 0,
	pattern_recognition BOOLEAN DEFAULT 0,
	debugging_accuracy REAL DEFAULT 0.0,
	max_handled_depth INTEGER DEFAULT 0,
	code_quality_score REAL DEFAULT 0.0,
	logic_correctness_score REAL DEFAULT 0.0,
	runtime_efficiency_score REAL DEFAULT 0.0,
	overall_score REAL DEFAULT 0.0,
	code_capability_score REAL DEFAULT 0.0,
	responsiveness_score REAL DEFAULT 0.0,
	reliability_score REAL DEFAULT 0.0,
	feature_richness_score REAL DEFAULT 0.0,
	value_proposition_score REAL DEFAULT 0.0,
	score_details TEXT,
	avg_latency_ms INTEGER DEFAULT 0,
	p95_latency_ms INTEGER DEFAULT 0,
	min_latency_ms INTEGER DEFAULT 0,
	max_latency_ms INTEGER DEFAULT 0,
	throughput_rps REAL DEFAULT 0.0,
	raw_request TEXT,
	raw_response TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
);
`

	_, err = db.Exec(schema)
	if err != nil {
		fmt.Printf("Schema execution failed: %v\n", err)
		
		// Try to find the problematic line
		lines := []string{}
		currentLine := ""
		for _, char := range schema {
			currentLine += string(char)
			if char == '\n' {
				lines = append(lines, currentLine)
				currentLine = ""
			}
		}
		if currentLine != "" {
			lines = append(lines, currentLine)
		}
		
		fmt.Printf("Schema has %d lines\n", len(lines))
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), "exists") {
				fmt.Printf("Line %d contains 'exists': %s", i+1, line)
			}
		}
	} else {
		fmt.Println("Schema executed successfully")
	}
}