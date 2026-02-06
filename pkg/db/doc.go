// Package db provides a database access layer for MySQL with support for
// read/write splitting, connection management, and type-safe SQL operations.
//
// The package uses sqlc (https://sqlc.dev/) to generate type-safe Go code from
// SQL queries, providing compile-time verification of database operations.
//
// Key features:
//
// - Primary/replica configuration with automatic routing of reads and writes
// - Support for transactions
// - Type-safe query methods generated from SQL
// - Bulk insert operations for improved performance
// - BulkQuerier interface for type-safe bulk operations
//
// Basic usage:
//
//	// Initialize the database with primary and optional read replica
//	db, err := db.New(db.Config{
//	    PrimaryDSN:  "mysql://user:pass@primary:3306/dbname?parseTime=true",
//	    ReadOnlyDSN: "mysql://user:pass@replica:3306/dbname?parseTime=true",
//	})
//	if err != nil {
//	    return fmt.Errorf("database initialization failed: %w", err)
//	}
//	defer db.Close()
//
//	// Execute a read query using the read replica
//	workspace, err := db.Query.FindWorkspaceByID(ctx, db.RO(), workspaceID)
//	if err != nil {
//	    return fmt.Errorf("failed to find workspace: %w", err)
//	}
//
//	// Execute a write query using the read/write connection
//	err = db.Query.InsertKey(ctx, db.RW(), insertKeyParams)
//	if err != nil {
//	    return fmt.Errorf("failed to insert key: %w", err)
//	}
//
//	// Use bulk operations for efficient batch inserts
//	err = db.Query.BulkInsertKey(ctx, db.RW(), []db.InsertKeyParams{
//	    insertKeyParams1,
//	    insertKeyParams2,
//	    insertKeyParams3,
//	})
//	if err != nil {
//	    return fmt.Errorf("failed to bulk insert keys: %w", err)
//	}
//
//	// Type-safe bulk operations using the BulkQuerier interface
//	var bulkQuerier db.BulkQuerier = db.Query
//	err = bulkQuerier.BulkInsertKey(ctx, db.RW(), keyBatch)
//	if err != nil {
//	    return fmt.Errorf("failed to bulk insert via interface: %w", err)
//	}
//
// This package relies on the standard Go database/sql package and the
// go-sql-driver/mysql driver for the actual database communication.
package db
