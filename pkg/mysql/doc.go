// Package mysql provides a minimal shared MySQL abstraction for backend services.
//
// The package intentionally stays small: it owns connection setup, read/write
// replica selection, tracing/metrics wrappers, transaction helpers, and
// MySQL-specific error classification. Query logic stays in caller packages.
// This keeps Bazel cache keys stable for dependents and avoids pulling
// service-specific SQL concerns into a base dependency.
//
// New requires [Config.PrimaryDSN] and enforces `parseTime=true` in DSNs so
// datetime values decode correctly. When [Config.ReadOnlyDSN] is empty, reads
// and writes use the same underlying pool. Use [Database.RW] for writes and
// [Database.RO] for read paths.
//
// Generated query code should depend on [DBTX], so the same query methods can
// run against either a [Replica] or a transaction. Use [Tx] / [TxWithResult]
// for single-attempt transactions, and [TxRetry] / [TxWithResultRetry] when the
// operation is safe to retry on transient failures.
//
// Retry helpers only retry errors classified as transient by [IsTransientError].
// [IsNotFound] and [IsDuplicateKeyError] are treated as terminal and are not
// retried.
package mysql
