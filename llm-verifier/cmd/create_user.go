package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"

	"llm-verifier/database"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run cmd/create_user.go <username> <password> <email> [full_name]")
		fmt.Println("Example: go run cmd/create_user.go admin Password123! admin@example.com \"Admin User\"")
		os.Exit(1)
	}

	username := os.Args[1]
	password := os.Args[2]
	email := os.Args[3]
	fullName := ""
	if len(os.Args) > 4 {
		fullName = os.Args[4]
	}

	// Initialize database
	db, err := database.New("llm-verifier.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Create user
	user := &database.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		FullName:     fullName,
		Role:         "admin",
		IsActive:     true,
	}

	err = db.CreateUser(user)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("User '%s' created successfully with ID: %d\n", username, user.ID)
	fmt.Println("Role: admin")
	fmt.Println("Status: active")
}
