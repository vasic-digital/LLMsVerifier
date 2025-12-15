package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// CreateUser creates a new user in the database
func (d *Database) CreateUser(user *User) error {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Convert preferences to JSON
	var preferencesJSON []byte
	if user.Preferences != nil {
		preferencesJSON, err = json.Marshal(user.Preferences)
		if err != nil {
			return fmt.Errorf("failed to marshal preferences: %w", err)
		}
	}

	query := `
		INSERT INTO users (
			username, email, password_hash, full_name, role, is_active,
			last_login, preferences
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.conn.Exec(query,
		user.Username,
		user.Email,
		string(hashedPassword),
		user.FullName,
		user.Role,
		user.IsActive,
		user.LastLogin,
		preferencesJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	user.ID = id
	user.PasswordHash = string(hashedPassword)
	return nil
}

// GetUser retrieves a user by ID
func (d *Database) GetUser(id int64) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, role, is_active,
		       last_login, created_at, updated_at, preferences
		FROM users WHERE id = ?
	`

	row := d.db.QueryRow(query, id)
	return d.scanUser(row)
}

// GetUserByUsername retrieves a user by username
func (d *Database) GetUserByUsername(username string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, role, is_active,
		       last_login, created_at, updated_at, preferences
		FROM users WHERE username = ?
	`

	row := d.db.QueryRow(query, username)
	return d.scanUser(row)
}

// GetUserByEmail retrieves a user by email
func (d *Database) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, role, is_active,
		       last_login, created_at, updated_at, preferences
		FROM users WHERE email = ?
	`

	row := d.db.QueryRow(query, email)
	return d.scanUser(row)
}

// UpdateUser updates an existing user
func (d *Database) UpdateUser(user *User) error {
	// If password is being updated, hash it
	var passwordHash string
	if user.PasswordHash != "" && len(user.PasswordHash) < 60 { // Not already hashed
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		passwordHash = string(hashedPassword)
	} else {
		// Get current password hash
		currentUser, err := d.GetUser(user.ID)
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}
		passwordHash = currentUser.PasswordHash
	}

	// Convert preferences to JSON
	var preferencesJSON []byte
	if user.Preferences != nil {
		var err error
		preferencesJSON, err = json.Marshal(user.Preferences)
		if err != nil {
			return fmt.Errorf("failed to marshal preferences: %w", err)
		}
	}

	query := `
		UPDATE users SET
			username = ?,
			email = ?,
			password_hash = ?,
			full_name = ?,
			role = ?,
			is_active = ?,
			last_login = ?,
			preferences = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := d.db.Exec(query,
		user.Username,
		user.Email,
		passwordHash,
		user.FullName,
		user.Role,
		user.IsActive,
		user.LastLogin,
		preferencesJSON,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	user.PasswordHash = passwordHash
	return nil
}

// DeleteUser deletes a user by ID
func (d *Database) DeleteUser(id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := d.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// ListUsers retrieves users with optional filtering
func (d *Database) ListUsers(filters map[string]interface{}) ([]*User, error) {
	query := `
		SELECT id, username, email, password_hash, full_name, role, is_active,
		       last_login, created_at, updated_at, preferences
		FROM users
		WHERE 1=1
	`

	args := []interface{}{}

	// Apply filters
	if role, ok := filters["role"]; ok {
		query += " AND role = ?"
		args = append(args, role)
	}

	if isActive, ok := filters["is_active"]; ok {
		query += " AND is_active = ?"
		args = append(args, isActive)
	}

	if search, ok := filters["search"]; ok {
		query += " AND (username LIKE ? OR email LIKE ? OR full_name LIKE ?)"
		searchPattern := "%" + search.(string) + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	// Add pagination
	limit := 50
	offset := 0
	if l, ok := filters["limit"]; ok {
		limit = l.(int)
	}
	if o, ok := filters["offset"]; ok {
		offset = o.(int)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user, err := d.scanUserFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// UpdateUserLastLogin updates the last login timestamp for a user
func (d *Database) UpdateUserLastLogin(userID int64) error {
	query := `UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// VerifyPassword verifies a password against the stored hash
func (d *Database) VerifyPassword(username, password string) (*User, error) {
	user, err := d.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	return user, nil
}

// scanUser scans a user from a sql.Row
func (d *Database) scanUser(row *sql.Row) (*User, error) {
	var user User
	var preferencesJSON []byte
	var lastLogin sql.NullTime

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.IsActive,
		&lastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
		&preferencesJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	if len(preferencesJSON) > 0 {
		err = json.Unmarshal(preferencesJSON, &user.Preferences)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
		}
	}

	return &user, nil
}

// scanUserFromRows scans a user from sql.Rows
func (d *Database) scanUserFromRows(rows *sql.Rows) (*User, error) {
	var user User
	var preferencesJSON []byte
	var lastLogin sql.NullTime

	err := rows.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.IsActive,
		&lastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
		&preferencesJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	if len(preferencesJSON) > 0 {
		err = json.Unmarshal(preferencesJSON, &user.Preferences)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
		}
	}

	return &user, nil
}
