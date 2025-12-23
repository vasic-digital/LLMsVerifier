package database

import (
	"strings"
	"testing"
)

func TestValidateTableName(t *testing.T) {
	tests := []struct {
		name    string
		table   string
		wantErr bool
	}{
		{
			name:    "Valid table",
			table:   "users",
			wantErr: false,
		},
		{
			name:    "Another valid table",
			table:   "models",
			wantErr: false,
		},
		{
			name:    "Invalid table - SQL injection",
			table:   "users; DROP TABLE users; --",
			wantErr: true,
		},
		{
			name:    "Invalid table - not in whitelist",
			table:   "hacker_table",
			wantErr: true,
		},
		{
			name:    "Empty table name",
			table:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTableName(tt.table)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTableName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateColumnName(t *testing.T) {
	tests := []struct {
		name     string
		column   string
		wantErr  bool
	}{
		{
			name:    "Valid column",
			column:  "id",
			wantErr: false,
		},
		{
			name:    "Valid column with underscore",
			column:  "created_at",
			wantErr: false,
		},
		{
			name:    "Invalid column - SQL injection",
			column:  "id; DROP TABLE users; --",
			wantErr: true,
		},
		{
			name:    "Invalid column - starts with number",
			column:  "123column",
			wantErr: true,
		},
		{
			name:    "Invalid column - has space",
			column:  "my column",
			wantErr: true,
		},
		{
			name:    "Invalid column - has special chars",
			column:  "column@name",
			wantErr: true,
		},
		{
			name:    "Empty column name",
			column:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateColumnName(tt.column)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateColumnName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQuoteTableName(t *testing.T) {
	tests := []struct {
		name    string
		table   string
		want    string
		wantErr bool
	}{
		{
			name:    "Valid table",
			table:   "users",
			want:    "\"users\"",
			wantErr: false,
		},
		{
			name:    "Invalid table",
			table:   "hacker; DROP TABLE; --",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QuoteTableName(tt.table)
			if (err != nil) != tt.wantErr {
				t.Errorf("QuoteTableName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("QuoteTableName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuoteColumnName(t *testing.T) {
	tests := []struct {
		name    string
		column  string
		want    string
		wantErr bool
	}{
		{
			name:    "Valid column",
			column:  "id",
			want:    "\"id\"",
			wantErr: false,
		},
		{
			name:    "Valid column with underscore",
			column:  "created_at",
			want:    "\"created_at\"",
			wantErr: false,
		},
		{
			name:    "Invalid column",
			column:  "column; DROP TABLE; --",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QuoteColumnName(tt.column)
			if (err != nil) != tt.wantErr {
				t.Errorf("QuoteColumnName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("QuoteColumnName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSafeSelectQuery(t *testing.T) {
	tests := []struct {
		name       string
		tableName  string
		columns    []string
		whereClause string
		wantErr    bool
		contains   string
	}{
		{
			name:       "Valid query",
			tableName:  "users",
			columns:    []string{"id", "name"},
			whereClause: "WHERE id = ?",
			wantErr:    false,
			contains:   "SELECT \"id\", \"name\" FROM \"users\" WHERE id = ?",
		},
		{
			name:       "Invalid table",
			tableName:  "users; DROP TABLE; --",
			columns:    []string{"id"},
			whereClause: "",
			wantErr:    true,
		},
		{
			name:       "Invalid column",
			tableName:  "users",
			columns:    []string{"id; DROP TABLE; --"},
			whereClause: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSafeSelectQuery(tt.tableName, tt.columns, tt.whereClause)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildSafeSelectQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(got, tt.contains) {
				t.Errorf("BuildSafeSelectQuery() = %v, want to contain %v", got, tt.contains)
			}
		})
	}
}

func TestBuildSafeInsertQuery(t *testing.T) {
	tests := []struct {
		name     string
		tableName string
		columns  []string
		wantErr  bool
		contains string
	}{
		{
			name:     "Valid insert",
			tableName: "users",
			columns:  []string{"id", "name", "email"},
			wantErr:  false,
			contains: "INSERT INTO \"users\" (\"id\", \"name\", \"email\") VALUES (?, ?, ?)",
		},
		{
			name:     "Invalid table",
			tableName: "users; DROP TABLE; --",
			columns:  []string{"id"},
			wantErr:  true,
		},
		{
			name:     "Invalid column",
			tableName: "users",
			columns:  []string{"id; DROP TABLE; --"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSafeInsertQuery(tt.tableName, tt.columns)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildSafeInsertQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(got, tt.contains) {
				t.Errorf("BuildSafeInsertQuery() = %v, want to contain %v", got, tt.contains)
			}
		})
	}
}

func TestGetAllowedTables(t *testing.T) {
	allowed := GetAllowedTables()
	
	if len(allowed) == 0 {
		t.Error("GetAllowedTables() returned empty map")
	}
	
	// Check some expected tables
	expectedTables := []string{"users", "models", "providers", "verification_results"}
	for _, table := range expectedTables {
		if !allowed[table] {
			t.Errorf("GetAllowedTables() missing expected table: %s", table)
		}
	}
}

func TestAddAllowedTable(t *testing.T) {
	// This should fail because the table doesn't match pattern
	err := AddAllowedTable("invalid; DROP TABLE; --")
	if err == nil {
		t.Error("AddAllowedTable() should reject invalid table name")
	}
}
