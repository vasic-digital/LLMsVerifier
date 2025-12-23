package database

import (
	"fmt"
	"regexp"
	"strings"
)

// allowedTables is a whitelist of valid table names
var allowedTables = map[string]bool{
	"providers":               true,
	"models":                   true,
	"verification_results":     true,
	"schedules":                true,
	"schedule_runs":            true,
	"users":                    true,
	"notifications":           true,
	"api_keys":                 true,
	"sessions":                 true,
	"settings":                 true,
	"logs":                     true,
	"migrations":               true,
}

// allowedColumns is a whitelist of valid column patterns
// Using regex for flexibility while maintaining security
var allowedColumnPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateTableName checks if a table name is allowed
func ValidateTableName(tableName string) error {
	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	// Check if table is in whitelist
	if !allowedTables[tableName] {
		return fmt.Errorf("invalid table name: '%s' (not in allowed list)", tableName)
	}

	return nil
}

// ValidateColumnName checks if a column name is valid
func ValidateColumnName(columnName string) error {
	if columnName == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	// Check against pattern (letters, numbers, underscores only)
	if !allowedColumnPattern.MatchString(columnName) {
		return fmt.Errorf("invalid column name: '%s' (must match pattern: ^[a-zA-Z_][a-zA-Z0-9_]*$)", columnName)
	}

	return nil
}

// ValidateColumnNames checks multiple column names
func ValidateColumnNames(columns []string) error {
	for i, column := range columns {
		if err := ValidateColumnName(column); err != nil {
			return fmt.Errorf("column %d: %w", i, err)
		}
	}
	return nil
}

// QuoteTableName safely quotes a table name
func QuoteTableName(tableName string) (string, error) {
	if err := ValidateTableName(tableName); err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%s"`, tableName), nil
}

// QuoteColumnName safely quotes a column name
func QuoteColumnName(columnName string) (string, error) {
	if err := ValidateColumnName(columnName); err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%s"`, columnName), nil
}

// QuoteColumnNames safely quotes multiple column names
func QuoteColumnNames(columns []string) ([]string, error) {
	quoted := make([]string, len(columns))
	for i, column := range columns {
		q, err := QuoteColumnName(column)
		if err != nil {
			return nil, fmt.Errorf("column %d: %w", i, err)
		}
		quoted[i] = q
	}
	return quoted, nil
}

// BuildSafeSelectQuery builds a safe SELECT query
func BuildSafeSelectQuery(tableName string, columns []string, whereClause string) (string, error) {
	quotedTable, err := QuoteTableName(tableName)
	if err != nil {
		return "", err
	}

	quotedColumns, err := QuoteColumnNames(columns)
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(quotedColumns, ", "), quotedTable)
	
	if whereClause != "" {
		query += " " + whereClause
	}

	return query, nil
}

// BuildSafeInsertQuery builds a safe INSERT query
func BuildSafeInsertQuery(tableName string, columns []string) (string, error) {
	quotedTable, err := QuoteTableName(tableName)
	if err != nil {
		return "", err
	}

	quotedColumns, err := QuoteColumnNames(columns)
	if err != nil {
		return "", err
	}

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quotedTable,
		strings.Join(quotedColumns, ", "),
		strings.Join(placeholders, ", "))

	return query, nil
}

// GetAllowedTables returns a copy of the allowed tables map
func GetAllowedTables() map[string]bool {
	allowed := make(map[string]bool, len(allowedTables))
	for k, v := range allowedTables {
		allowed[k] = v
	}
	return allowed
}

// AddAllowedTable adds a table to the allowed list (use with caution!)
func AddAllowedTable(tableName string) error {
	if err := ValidateTableName(tableName); err != nil {
		return fmt.Errorf("cannot add table: %w", err)
	}
	allowedTables[tableName] = true
	return nil
}
