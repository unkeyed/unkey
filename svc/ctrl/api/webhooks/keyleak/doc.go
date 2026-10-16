// Package keyleak accepts signed GitHub secret-scanning reports for root keys.
// A 201 response means all matching keys' notifications were accepted by Restate,
// not that delivery completed. Keys are not revoked automatically.
package keyleak
