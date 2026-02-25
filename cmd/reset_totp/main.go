package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/smart-cafeteria/backend/internal/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	email := "admin@cafeteria.com"
	if len(os.Args) > 1 {
		email = os.Args[1]
	}

	result := db.Exec("UPDATE users SET totp_secret = '', totp_enabled = false WHERE email = ?", email)
	if result.Error != nil {
		log.Fatalf("Failed to reset TOTP: %v", result.Error)
	}

	fmt.Printf("TOTP reset for %s. Rows affected: %d\n", email, result.RowsAffected)
	fmt.Println("User will be prompted to set up 2FA again on next login.")
}
