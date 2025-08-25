// Package mysql provides a MySQL-backed implementation of the kv.Store interface.
//
// This implementation uses sqlc-generated code for type-safe database operations
// and supports both primary and read-replica database connections for optimal
// performance in production environments.
//
// The store automatically handles:
//   - TTL expiration by deleting expired keys on read
//   - Cursor-based pagination using created_at timestamps
//   - Connection routing (reads to replica, writes to primary)
//   - Auto-incrementing primary keys for efficient storage
//
// Database schema (inspired by GitHub's approach):
//
//	CREATE TABLE kv (
//	    id BIGINT(20) NOT NULL AUTO_INCREMENT,
//	    `key` VARCHAR(255) NOT NULL,
//	    workspace_id VARCHAR(255) NOT NULL,
//	    value BLOB NOT NULL,
//	    ttl BIGINT NULL,
//	    created_at BIGINT NOT NULL,
//
//	    PRIMARY KEY (id),
//	    UNIQUE KEY unique_key (`key`),
//	    INDEX idx_workspace_id (workspace_id),
//	    INDEX idx_ttl (ttl)
//	);
package mysql
