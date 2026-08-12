package events

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"digital.vasic.llmsverifier/pkg/helixendpoint"
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

// dialAddress returns the "host:port" this notifier hands to smtp.SendMail.
//
// HXC-268: composed through helixendpoint.DialAddress rather than
// fmt.Sprintf("%s:%d") so an IPv6 literal is bracketed per RFC 3986 §3.2.2 —
// unbracketed, "::1" + ":587" concatenates to "::1:587", which is not the address
// that was configured.
//
// DialAddress, not BaseURL: this value is handed to smtp.SendMail and is never
// URL-parsed, so it must carry no scheme, no RFC 6874 zone encoding, and — above
// all — no HelixAgent placeholder fallback, which would silently redirect the
// mail client to the placeholder HOST while keeping this relay's own port, so it
// would dial the wrong machine instead of failing on a bad host.
func (en *EmailNotifier) dialAddress() string {
	return helixendpoint.DialAddress(en.smtpServer, en.smtpPort)
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
	addr := en.dialAddress()
	err := smtp.SendMail(addr, auth, en.fromAddress, en.toAddresses, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
