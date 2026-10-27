// Package keyleak disables keys in signed GitHub secret-scanning reports and
// records the leak in the same transaction before enqueueing notifications.
// A 200 response means no keys matched or every matching key's notification was
// accepted by Restate, not that delivery completed or API caches have refreshed.
// Database and enqueue failures return 503 so GitHub can retry the report.
package keyleak
