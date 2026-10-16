// Package keyleak delivers root-key leak alerts without retaining raw tokens.
// Slack and each recipient's email retry independently. Slack delivery is at
// least once; email also uses provider idempotency within its retention window.
package keyleak
