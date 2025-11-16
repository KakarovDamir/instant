package email

import (
	"time"
)

// EmailEventType represents the type of email to be sent
type EmailEventType string

const (
	// EmailTypeVerificationCode is for authentication verification codes
	EmailTypeVerificationCode EmailEventType = "verification_code"
	// EmailTypeWelcome is for welcome emails (future use)
	EmailTypeWelcome EmailEventType = "welcome"
	// EmailTypePasswordReset is for password reset emails (future use)
	EmailTypePasswordReset EmailEventType = "password_reset"
)

// EmailEvent represents an email event to be published to Kafka
// This matches the schema defined in the plan
type EmailEvent struct {
	// MessageID is a unique identifier for this email event (UUID v4)
	// Used for deduplication to ensure exactly-once delivery
	MessageID string `json:"message_id"`

	// EventType specifies what kind of email to send
	EventType EmailEventType `json:"event_type"`

	// Timestamp when the event was created
	Timestamp time.Time `json:"timestamp"`

	// Recipient is the email address to send to
	Recipient string `json:"recipient"`

	// Data contains type-specific information for the email
	// For verification_code: {"code": "123456", "expires_in": "10m"}
	// For welcome: {"username": "john_doe"}
	// For password_reset: {"reset_link": "https://..."}
	Data map[string]interface{} `json:"data"`
}

// VerificationCodeData represents the data for a verification code email
type VerificationCodeData struct {
	Code      string `json:"code"`
	ExpiresIn string `json:"expires_in"`
}

// EmailMetadata represents metadata stored in Redis for deduplication
type EmailMetadata struct {
	SentAt    time.Time      `json:"sent_at"`
	Recipient string         `json:"recipient"`
	EventType EmailEventType `json:"event_type"`
}
