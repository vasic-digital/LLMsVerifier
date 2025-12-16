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
	fmt.Println("Testing Telegram Notification System...")

	// Create a temporary database for testing
	dbPath := "./test_telegram_notifications.db"
	defer os.Remove(dbPath)

	// Initialize database
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Initialize notification manager with Telegram configuration
	config := &notifications.NotificationConfig{
		Slack: notifications.SlackConfig{
			Enabled: false, // Disabled for this test
		},
		Email: notifications.EmailConfig{
			Enabled: false, // Disabled for this test
		},
		Telegram: notifications.TelegramConfig{
			Enabled:   true,
			BotToken:  "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz", // Fake token for testing
			ChatID:    "123456789",
			ParseMode: "HTML",
			Timeout:   10,
		},
	}

	manager := notifications.NewNotificationManager(config, db)

	// Test 1: Telegram notification for verification completed
	fmt.Println("\n1. Testing Telegram notification for verification completed...")
	notification := notifications.CreateVerificationCompletedNotification(15, 14, 5*time.Minute)
	notification.Channel = notifications.ChannelTelegram
	notification.Recipient = "@test_chat"

	err = manager.SendNotification(notification)
	if err != nil {
		fmt.Printf("❌ Telegram notification failed: %v\n", err)
		fmt.Println("Note: This is expected with fake credentials - testing error handling")
	} else {
		fmt.Printf("✅ Telegram notification sent successfully\n")
	}

	// Test 2: Telegram notification for verification failure
	fmt.Println("\n2. Testing Telegram notification for verification failure...")
	notification = notifications.CreateVerificationFailedNotification("gpt-4-test", "Rate limit exceeded")
	notification.Channel = notifications.ChannelTelegram
	notification.Recipient = "@test_chat"

	err = manager.SendNotification(notification)
	if err != nil {
		fmt.Printf("❌ Telegram notification failed: %v\n", err)
	} else {
		fmt.Printf("✅ Telegram notification sent successfully\n")
	}

	// Test 3: Telegram notification for score change
	fmt.Println("\n3. Testing Telegram notification for score change...")
	notification = notifications.CreateScoreChangedNotification("claude-3-sonnet", 85.5, 92.1)
	notification.Channel = notifications.ChannelTelegram
	notification.Recipient = "@test_chat"

	err = manager.SendNotification(notification)
	if err != nil {
		fmt.Printf("❌ Telegram notification failed: %v\n", err)
	} else {
		fmt.Printf("✅ Telegram notification sent successfully\n")
	}

	// Test 4: Test database retrieval
	fmt.Println("\n4. Testing notification database retrieval...")
	notifications, err := db.GetNotifications(10, 0, nil)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve notifications: %v\n", err)
	} else {
		fmt.Printf("✅ Retrieved %d notifications from database\n", len(notifications))
		for i, n := range notifications {
			fmt.Printf("   %d. %s: %s (%s) - Sent: %t\n", i+1, n.Type, n.Title, n.Channel, n.Sent)
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

	fmt.Println("\n🎉 Telegram notification system test completed!")
	fmt.Println("\nNote: Telegram sending may fail if bot token/chat ID are not properly configured.")
	fmt.Println("For actual Telegram testing, create a bot with @BotFather and get the chat ID.")
}
