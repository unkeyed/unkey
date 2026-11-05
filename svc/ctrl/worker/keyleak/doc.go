// Package keyleak delivers alerts for matched key leaks without raw tokens.
// Slack and each recipient's email retry independently. Slack delivery is at
// least once; email also uses provider idempotency within its retention window.
// A notification that asks for revocation soft-deletes the key before any alert
// is sent, but only when the notification carries the key's stored hash, so a
// caller that knows only the key ID cannot revoke it. If the key was already
// deleted or the hash does not match, no alerts are sent.
package keyleak
