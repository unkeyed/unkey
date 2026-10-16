// Package keyleak classifies key leaks and delivers alerts without raw tokens.
// Slack and each recipient's email retry independently. Slack delivery is at
// least once; email also uses provider idempotency within its retention window.
package keyleak
