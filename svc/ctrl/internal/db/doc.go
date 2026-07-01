// Package db provides the control plane database access layer.
//
// The package owns the SQL queries used by svc/ctrl and delegates connection
// management, tracing, metrics, transactions, and MySQL error classification to
// pkg/mysql. Control plane processes use one read-write MySQL DSN; RO and RW are
// compatibility aliases for that same connection while call sites migrate to a
// narrower query interface.
package db
