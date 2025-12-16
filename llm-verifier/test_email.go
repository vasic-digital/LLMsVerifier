package main

import (
	"fmt"
	"log"
	"os"

	"llm-verifier/config"
	"llm-verifier/database"
	"llm-verifier/events"
	"llm-verifier/notifications"
)

func TestEmailNotifications() {
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

	// Initialize event bus for notifications
	eventBus := events.NewEventBus(db)

	// Initialize notification manager with email configuration
	cfg := &config.Config{
		Notifications: config.NotificationsConfig{
			Email: config.EmailConfig{
				Enabled:          true,
				SMTPHost:         "smtp.gmail.com",
				SMTPPort:         587,
				Username:         "test@example.com",
				Password:         "test-password",
				DefaultRecipient: "recipient@example.com",
			},
		},
	}
	manager := notifications.NewNotificationManager(cfg, eventBus)

	// Test 1: Email notification with template
	fmt.Println("\n1. Testing email notification with template...")
	event := events.NewEvent(events.EventTypeModelVerified, events.EventSeverityInfo,
		"Model verification completed successfully", "test")
	event.WithData("model_id", "gpt-4").WithData("score", 95.2)

	err = manager.SendEventNotification(event, []notifications.NotificationChannel{notifications.ChannelEmail})
	if err != nil {
		fmt.Printf("❌ Email notification failed: %v\n", err)
		fmt.Println("Note: This might be expected if SMTP credentials are not configured")
	} else {
		fmt.Printf("✅ Email notification sent successfully\n")
	}

	// Test 2: Email notification for verification failure
	fmt.Println("\n2. Testing email notification for verification failure...")
	event = events.NewEvent(events.EventTypeModelVerificationFailed, events.EventSeverityError,
		"Model verification failed: Rate limit exceeded", "test")
	event.WithData("model_id", "gpt-4-test").WithData("error", "Rate limit exceeded")

	err = manager.SendEventNotification(event, []notifications.NotificationChannel{notifications.ChannelEmail})
	if err != nil {
		fmt.Printf("❌ Email notification for failure failed: %v\n", err)
	} else {
		fmt.Printf("✅ Email notification for failure sent successfully\n")
	}

	// Test 3: Email notification for score change
	fmt.Println("\n3. Testing email notification for score change...")
	event = events.NewEvent(events.EventTypeScoreChanged, events.EventSeverityInfo,
		"Model score changed from 85.5 to 92.1", "test")
	event.WithData("model_id", "claude-3").WithData("old_score", 85.5).WithData("new_score", 92.1)

	err = manager.SendEventNotification(event, []notifications.NotificationChannel{notifications.ChannelEmail})
	if err != nil {
		fmt.Printf("❌ Email notification for score change failed: %v\n", err)
	} else {
		fmt.Printf("✅ Email notification for score change sent successfully\n")
	}

	// Test 3: Email notification for score change
	fmt.Println("\n3. Testing email notification for score change...")
	event = events.NewEvent(events.EventTypeScoreChanged, events.EventSeverityInfo,
		"Model score changed from 85.5 to 92.1", "test")
	event.WithData("model_id", "claude-3-sonnet").WithData("old_score", 85.5).WithData("new_score", 92.1)

	err = manager.SendEventNotification(event, []notifications.NotificationChannel{notifications.ChannelEmail})
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
