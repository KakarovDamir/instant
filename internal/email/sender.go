// Package email provides email sending functionality for the application.
// It supports both development mode (log-only) and production mode (SMTP).
package email

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
)

// Sender defines the interface for sending emails
type Sender interface {
	SendVerificationCode(email, code string) error
	SendEmailEvent(event EmailEvent) error
}

// Config holds email configuration
type Config struct {
	Mode     string // "log" or "smtp"
	Host     string
	Port     int
	User     string
	Password string
	From     string
	FromName string
}

// NewConfig creates a new email configuration from environment variables
func NewConfig() *Config {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))

	return &Config{
		Mode:     getEnvOrDefault("EMAIL_MODE", "log"),
		Host:     os.Getenv("SMTP_HOST"),
		Port:     port,
		User:     os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     getEnvOrDefault("SMTP_FROM", "noreply@example.com"),
		FromName: getEnvOrDefault("SMTP_FROM_NAME", "Your App"),
	}
}

// NewSender creates a new email sender based on configuration
func NewSender(cfg *Config) Sender {
	if cfg.Mode == "smtp" {
		return &smtpSender{config: cfg}
	}
	return &logSender{}
}

// logSender logs emails to console (development mode)
type logSender struct{}

func (s *logSender) SendVerificationCode(email, code string) error {
	log.Printf("[DEV] Verification code for %s: %s (expires in 10 minutes)", email, code)
	return nil
}

func (s *logSender) SendEmailEvent(event EmailEvent) error {
	switch event.EventType {
	case EmailTypeVerificationCode:
		code, ok := event.Data["code"].(string)
		if !ok {
			return fmt.Errorf("invalid verification code data")
		}
		return s.SendVerificationCode(event.Recipient, code)
	default:
		log.Printf("[DEV] Email event for %s: type=%s, data=%v", event.Recipient, event.EventType, event.Data)
		return nil
	}
}

// smtpSender sends emails via SMTP (production mode)
type smtpSender struct {
	config *Config
}

func (s *smtpSender) SendVerificationCode(email, code string) error {
	// Email subject and body
	subject := "Your Verification Code"
	body := s.buildEmailBody(email, code)

	// Construct email message
	message := fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.From)
	message += fmt.Sprintf("To: %s\r\n", email)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n"
	message += body

	// SMTP authentication
	auth := smtp.PlainAuth("", s.config.User, s.config.Password, s.config.Host)

	// Send email
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	err := smtp.SendMail(addr, auth, s.config.From, []string{email}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Verification code sent to %s via SMTP", email)
	return nil
}

func (s *smtpSender) SendEmailEvent(event EmailEvent) error {
	switch event.EventType {
	case EmailTypeVerificationCode:
		code, ok := event.Data["code"].(string)
		if !ok {
			return fmt.Errorf("invalid verification code data")
		}
		return s.SendVerificationCode(event.Recipient, code)
	default:
		return fmt.Errorf("unsupported email type: %s", event.EventType)
	}
}

func (s *smtpSender) buildEmailBody(email, code string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verification Code</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0;">Verification Code</h1>
    </div>

    <div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px;">
        <p style="font-size: 16px;">Hello,</p>

        <p style="font-size: 16px;">Your verification code is:</p>

        <div style="background: white; border: 2px solid #667eea; border-radius: 8px; padding: 20px; text-align: center; margin: 20px 0;">
            <span style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #667eea;">%s</span>
        </div>

        <p style="font-size: 14px; color: #666;">
            This code will expire in <strong>10 minutes</strong>.
        </p>

        <p style="font-size: 14px; color: #666;">
            If you didn't request this code, you can safely ignore this email.
        </p>

        <hr style="border: none; border-top: 1px solid #ddd; margin: 30px 0;">

        <p style="font-size: 12px; color: #999; text-align: center;">
            This is an automated message, please do not reply to this email.
        </p>
    </div>
</body>
</html>
`, code)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
