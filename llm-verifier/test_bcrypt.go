package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"

	"llm-verifier/database"
)

func main() {
	// Initialize database
	db, err := database.New("llm-verifier.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Get the user we just created
	user, err := db.GetUserByUsername("admin")
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("User: %+v\n", user)
	fmt.Printf("Password hash: %s\n", user.PasswordHash)
	fmt.Printf("Password hash length: %d\n", len(user.PasswordHash))

	// Test bcrypt comparison
	password := "Password123!"
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		fmt.Printf("Bcrypt comparison failed: %v\n", err)
	} else {
		fmt.Println("Bcrypt comparison succeeded")
	}
}
