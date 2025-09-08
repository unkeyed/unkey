// Package db provides database operations for the partition database.
//
// The partition database is regionally replicated and optimized for high-throughput,
// low-latency data plane access. It provides gateways with routing tables, TLS certificates,
// API key metadata, and VM status information needed for request routing and traffic management.
//
// This package follows the same patterns as the main database package but is focused
// on partition-specific operations and optimized for the data plane access patterns.
//
// The database supports both primary/replica configurations for high availability
// and is designed to operate independently from the control plane database.
package db
