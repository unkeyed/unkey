// Package keyrefill implements a Restate workflow for refilling API key usage limits.
//
// This service processes keys that need their remaining_requests refilled to refill_amount.
// It runs daily at 00:00 UTC and uses batch queries to efficiently process large numbers of keys.
//
// Keys are refilled if:
//   - refill_day is NULL (daily refill)
//   - refill_day matches today's day of month
//   - refill_day > today's day AND today is the last day of month (catch-up for short months)
//
// The service is keyed by date (e.g., "2026-02-04") which provides:
//   - Natural idempotency (each date can only be processed once)
//   - Resumability via Restate state (tracks processed key IDs)
//   - Clear audit trail of when refills occurred
//
// Performance optimizations:
//   - Batch queries (100 keys per query) to minimize database round trips
//   - Bulk updates and inserts for keys and audit logs
//   - State-based resumability to continue after interruption
//
// Bootstrap:
// To run the daily refill, call RunRefill with today's date:
//
//	POST /KeyRefillService/2026-02-04/RunRefill
//	{}
package keyrefill
