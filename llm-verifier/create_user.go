package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Open database
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create test user
	username := "admin"
	email := "admin@example.com"
	password := "admin123"
	fullName := "Administrator"
	role := "admin"

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	// Insert user
	query := `
		INSERT OR IGNORE INTO users (
			username, email, password_hash, full_name, role, is_active
		) VALUES (?, ?, ?, ?, ?, 1)
	`

	_, err = db.Exec(query, username, email, string(hashedPassword), fullName, role)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Test user created successfully!")
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Password: %s\n", password)
}
