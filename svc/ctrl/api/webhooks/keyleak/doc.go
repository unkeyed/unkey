// Package keyleak queues matched keys from signed GitHub secret-scanning reports.
// A 200 response means no keys matched or every batch was accepted by Restate,
// not that keys are disabled, alerts delivered, or API caches refreshed.
// Database and enqueue failures return 503 so GitHub can retry the report.
package keyleak
