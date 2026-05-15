// Package email sends transactional emails.
// Falls back to console output when SMTP_HOST is not set — perfect for dev/test.
//
// Configure via environment variables:
//   SMTP_HOST  e.g. smtp.gmail.com
//   SMTP_PORT  default 587
//   SMTP_FROM  e.g. noreply@yourapp.com
//   SMTP_PASS  app password
package email

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// Mailer sends emails via SMTP, or logs them to stdout in dev mode.
type Mailer struct {
	host string
	port string
	from string
	pass string
}

// New creates a Mailer configured from environment variables.
func New() *Mailer {
	return &Mailer{
		host: os.Getenv("SMTP_HOST"),
		port: getEnv("SMTP_PORT", "587"),
		from: os.Getenv("SMTP_FROM"),
		pass: os.Getenv("SMTP_PASS"),
	}
}

// Send dispatches an email. Logs to console when SMTP is not configured.
// Safe to call on a nil Mailer — acts as no-op (useful in tests).
func (m *Mailer) Send(to, subject, body string) error {
	if m == nil {
		return nil // no-op in tests
	}
	if m.host == "" {
		// Dev-mode fallback: print so developer can see the email content.
		sep := strings.Repeat("─", 50)
		fmt.Printf("\n┌%s\n│ 📧 EMAIL (dev mode — set SMTP_HOST to send real emails)\n│ To:      %s\n│ Subject: %s\n│\n│ %s\n└%s\n\n",
			sep, to, subject,
			strings.ReplaceAll(body, "\n", "\n│ "),
			sep)
		return nil
	}
	auth := smtp.PlainAuth("", m.from, m.pass, m.host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, body)
	return smtp.SendMail(m.host+":"+m.port, auth, m.from, []string{to}, []byte(msg))
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
