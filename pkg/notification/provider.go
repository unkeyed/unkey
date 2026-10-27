package notification

import (
	"context"
	"time"
)

// Channel selects the delivery transport independently of its provider.
type Channel string

const (
	Email   Channel = "email"
	Slack   Channel = "slack"
	Discord Channel = "discord"
	Webhook Channel = "webhook"
)

// Provider sends once, without internal retries.
type Provider interface {
	// Name identifies the provider across all configured channels.
	Name() string
	// Channel identifies the transport accepted by Send.
	Channel() Channel
	// IdempotencyWindow is the guaranteed retention of DeliveryID;
	// zero means ambiguous sends cannot be retried safely.
	IdempotencyWindow() time.Duration
	// Send returns provider acceptance, not end-recipient delivery.
	// Failures use *SendError to distinguish rejection from unknown outcomes.
	Send(context.Context, Request) (Receipt, error)
}

// Target contains plaintext credentials. Never persist or journal it without encryption.
type Target struct {
	Email         string            `json:"email,omitempty"`
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	SigningSecret string            `json:"signingSecret,omitempty"`
}

// Request binds rendered content and a resolved target to a stable delivery identity.
type Request struct {
	DeliveryID string  `json:"deliveryId"`
	Event      Event   `json:"event"`
	Content    Content `json:"content"`
	Target     Target  `json:"target"`
}

// Receipt records provider acceptance. MessageID is empty when the provider supplies none.
type Receipt struct {
	MessageID string `json:"messageId"`
}

// FailureKind determines whether a failed send can be retried safely.
type FailureKind string

const (
	// Rejected indicates a permanent refusal.
	Rejected FailureKind = "rejected"
	// Retryable means the request was not sent or was explicitly not accepted.
	Retryable FailureKind = "retryable"
	// Ambiguous means acceptance is unknown and retrying may duplicate delivery.
	Ambiguous FailureKind = "ambiguous"
)

// SendError must not contain recipient addresses, webhook URLs, credentials, or response bodies.
type SendError struct {
	Kind       FailureKind
	Code       string
	RetryAfter time.Duration
}

// Error exposes only the non-sensitive failure code.
func (e *SendError) Error() string { return e.Code }

// DeliveryID derives a stable identity for one notification, channel, and target key.
func DeliveryID(notificationID string, channel Channel, targetKey string) string {
	return stableID("ndlv", notificationID, string(channel), targetKey)
}
