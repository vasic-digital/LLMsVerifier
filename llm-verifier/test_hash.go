package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"

	"llm-verifier/database"
)

func main() {
	// Test creating a new hash
	password := "Password123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	fmt.Printf("New hash: %s\n", string(hashedPassword))
	fmt.Printf("New hash length: %d\n", len(hashedPassword))

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

	fmt.Printf("\nStored hash: %s\n", user.PasswordHash)
	fmt.Printf("Stored hash length: %d\n", len(user.PasswordHash))

	// Compare new hash with stored hash
	if string(hashedPassword) == user.PasswordHash {
		fmt.Println("Hashes match!")
	} else {
		fmt.Println("Hashes DON'T match!")
	}

	// Try to compare with bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		fmt.Printf("Bcrypt comparison with stored hash failed: %v\n", err)
	} else {
		fmt.Println("Bcrypt comparison with stored hash succeeded")
	}
}
