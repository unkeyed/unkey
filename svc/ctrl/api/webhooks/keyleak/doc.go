// Package keyleak filters signed GitHub secret-scanning reports before enqueueing
// notifications. A 200 response means no keys matched or every matching key's
// notification was accepted by Restate, not that delivery completed. Database
// lookup failures return 503 and are not durably retried. Keys are not revoked.
package keyleak
