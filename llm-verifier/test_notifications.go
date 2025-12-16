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
	fmt.Println("Testing Notification System...")

	// Create a temporary database for testing
	dbPath := "./test_notifications.db"
	defer os.Remove(dbPath)

	// Initialize database
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Initialize notification manager
	config := &notifications.NotificationConfig{
		Slack: notifications.SlackConfig{
			Enabled:    true,
			WebhookURL: "https://hooks.slack.com/test",
			Channel:    "#llm-verifier",
			Username:   "LLM Verifier",
		},
		Email: notifications.EmailConfig{
			Enabled: false, // Disabled for testing
		},
		Telegram: notifications.TelegramConfig{
			Enabled:  true,
			BotToken: "test_token",
			ChatID:   "test_chat",
		},
	}

	manager := notifications.NewNotificationManager(config, db)

	// Test 1: Create and send verification completed notification
	fmt.Println("\n1. Testing verification completed notification...")
	notification := notifications.CreateVerificationCompletedNotification(15, 14, 5*time.Minute)
	err = manager.SendNotification(notification)
	if err != nil {
		fmt.Printf("❌ Failed to send notification: %v\n", err)
	} else {
		fmt.Printf("✅ Sent verification completed notification\n")
	}

	// Test 2: Create and send verification failed notification
	fmt.Println("\n2. Testing verification failed notification...")
	notification = notifications.CreateVerificationFailedNotification("gpt-4-test", "Rate limit exceeded")
	err = manager.SendNotification(notification)
	if err != nil {
		fmt.Printf("❌ Failed to send notification: %v\n", err)
	} else {
		fmt.Printf("✅ Sent verification failed notification\n")
	}

	// Test 3: Create and send score changed notification
	fmt.Println("\n3. Testing score changed notification...")
	notification = notifications.CreateScoreChangedNotification("claude-3-sonnet", 85.5, 92.1)
	err = manager.SendNotification(notification)
	if err != nil {
		fmt.Printf("❌ Failed to send notification: %v\n", err)
	} else {
		fmt.Printf("✅ Sent score changed notification\n")
	}

	// Test 4: Test manual notification creation
	fmt.Println("\n4. Testing manual notification...")
	manualNotification := &notifications.Notification{
		Type:     "test_notification",
		Channel:  "slack",
		Priority: "normal",
		Title:    "Test Notification",
		Message:  "This is a test notification to verify the system is working correctly.",
		Data: map[string]interface{}{
			"test_key":    "test_value",
			"test_number": 42,
		},
		Timestamp: time.Now(),
		Recipient: "#test-channel",
	}

	err = manager.SendNotification(manualNotification)
	if err != nil {
		fmt.Printf("❌ Failed to send manual notification: %v\n", err)
	} else {
		fmt.Printf("✅ Sent manual notification\n")
	}

	// Test 5: Test database retrieval
	fmt.Println("\n5. Testing notification database retrieval...")
	notifications, err := db.GetNotifications(10, 0, nil)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve notifications: %v\n", err)
	} else {
		fmt.Printf("✅ Retrieved %d notifications from database\n", len(notifications))
		for i, n := range notifications {
			fmt.Printf("   %d. %s: %s (%s)\n", i+1, n.Type, n.Title, n.Channel)
		}
	}

	// Test 6: Test notification stats
	fmt.Println("\n6. Testing notification statistics...")
	stats, err := db.GetNotificationStats()
	if err != nil {
		fmt.Printf("❌ Failed to get notification stats: %v\n", err)
	} else {
		fmt.Printf("✅ Notification statistics:\n")
		for key, value := range stats {
			fmt.Printf("   %s: %v\n", key, value)
		}
	}

	fmt.Println("\n🎉 Notification system test completed successfully!")
}
