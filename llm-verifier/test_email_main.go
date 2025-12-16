package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"llm-verifier/database"
	"llm-verifier/notifications"
)

func main() {
	fmt.Println("Testing Email Notification System...")

	// Create a temporary database for testing
	dbPath := "./test_email_notifications.db"
	defer os.Remove(dbPath)

	// Initialize database
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Initialize notification manager with email configuration
	config := &notifications.NotificationConfig{
		Slack: notifications.SlackConfig{
			Enabled: false, // Disabled for this test
		},
		Email: notifications.EmailConfig{
			Enabled:      true,
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     587,
			Username:     "test@example.com", // Replace with actual email for testing
			Password:     "test-password",    // Replace with app password
			From:         "test@example.com",
			To:           []string{"recipient@example.com"},
			UseTLS:       true,
			SkipVerify:   false,
			TemplatesDir: "./notifications/templates",
		},
		Telegram: notifications.TelegramConfig{
			Enabled: false, // Disabled for this test
		},
	}

	manager := notifications.NewNotificationManager(config, db)

	// Test 1: Email notification with template
	fmt.Println("\n1. Testing email notification with template...")
	notification := notifications.CreateVerificationCompletedNotification(15, 14, 5*time.Minute)
	notification.Channel = notifications.ChannelEmail
	notification.Recipient = "recipient@example.com"

	err = manager.SendNotification(notification)
	if err != nil {
		fmt.Printf("❌ Email notification failed: %v\n", err)
		fmt.Println("Note: This might be expected if SMTP credentials are not configured")
	} else {
		fmt.Printf("✅ Email notification sent successfully\n")
	}

	// Test 2: Email notification for verification failure
	fmt.Println("\n2. Testing email notification for verification failure...")
	notification = notifications.CreateVerificationFailedNotification("gpt-4-test", "Rate limit exceeded")
	notification.Channel = notifications.ChannelEmail
	notification.Recipient = "recipient@example.com"

	err = manager.SendNotification(notification)
	if err != nil {
		fmt.Printf("❌ Email notification failed: %v\n", err)
	} else {
		fmt.Printf("✅ Email notification sent successfully\n")
	}

	// Test 3: Email notification for score change
	fmt.Println("\n3. Testing email notification for score change...")
	notification = notifications.CreateScoreChangedNotification("claude-3-sonnet", 85.5, 92.1)
	notification.Channel = notifications.ChannelEmail
	notification.Recipient = "recipient@example.com"

	err = manager.SendNotification(notification)
	if err != nil {
		fmt.Printf("❌ Email notification failed: %v\n", err)
	} else {
		fmt.Printf("✅ Email notification sent successfully\n")
	}

	// Test 4: Test database retrieval
	fmt.Println("\n4. Testing notification database retrieval...")
	notifications, err := db.GetNotifications(10, 0, nil)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve notifications: %v\n", err)
	} else {
		fmt.Printf("✅ Retrieved %d notifications from database\n", len(notifications))
		for i, n := range notifications {
			fmt.Printf("   %d. %s: %s (%s)\n", i+1, n.Type, n.Title, n.Channel)
		}
	}

	// Test 5: Test notification stats
	fmt.Println("\n5. Testing notification statistics...")
	stats, err := db.GetNotificationStats()
	if err != nil {
		fmt.Printf("❌ Failed to get notification stats: %v\n", err)
	} else {
		fmt.Printf("✅ Notification statistics:\n")
		for key, value := range stats {
			fmt.Printf("   %s: %v\n", key, value)
		}
	}

	fmt.Println("\n🎉 Email notification system test completed!")
	fmt.Println("\nNote: Email sending may fail if SMTP credentials are not properly configured.")
	fmt.Println("For actual email testing, configure real SMTP settings in the notification config.")
}
