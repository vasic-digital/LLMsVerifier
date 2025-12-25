package events

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// EmailNotifier handles email notifications
type EmailNotifier struct {
	smtpServer  string
	smtpPort    int
	username    string
	password    string
	fromAddress string
	toAddresses []string
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(smtpServer string, smtpPort int, username, password, fromAddress string, toAddresses []string) *EmailNotifier {
	return &EmailNotifier{
		smtpServer:  smtpServer,
		smtpPort:    smtpPort,
		username:    username,
		password:    password,
		fromAddress: fromAddress,
		toAddresses: toAddresses,
	}
}

// SendNotification sends an email notification
func (en *EmailNotifier) SendNotification(event *Event) error {
	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(string(event.Severity)), event.Title)
	body := fmt.Sprintf("Event: %s\n\n%s\n\nSeverity: %s\nSource: %s\nTimestamp: %s\n",
		event.Title, event.Message, event.Severity, event.Source, event.Timestamp.Format(time.RFC3339))

	if event.ModelID != nil {
		body += fmt.Sprintf("Model ID: %d\n", *event.ModelID)
	}
	if event.ProviderID != nil {
		body += fmt.Sprintf("Provider ID: %d\n", *event.ProviderID)
	}
	if event.ClientID != nil {
		body += fmt.Sprintf("Client ID: %s\n", *event.ClientID)
	}

	return en.sendEmail(subject, body)
}

// sendEmail sends the actual email
func (en *EmailNotifier) sendEmail(subject, body string) error {
	// Construct the email message
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		en.fromAddress,
		strings.Join(en.toAddresses, ","),
		subject,
		body)

	// Set up authentication
	auth := smtp.PlainAuth("", en.username, en.password, en.smtpServer)

	// Send the email
	addr := fmt.Sprintf("%s:%d", en.smtpServer, en.smtpPort)
	err := smtp.SendMail(addr, auth, en.fromAddress, en.toAddresses, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
