package db

import "github.com/unkeyed/unkey/pkg/mysql"

// IsNotFound re-exports [mysql.IsNotFound] for callers of this package.
var IsNotFound = mysql.IsNotFound

// IsDuplicateKeyError re-exports [mysql.IsDuplicateKeyError] for callers of this package.
var IsDuplicateKeyError = mysql.IsDuplicateKeyError

// IsTransientError re-exports [mysql.IsTransientError] for callers of this package.
var IsTransientError = mysql.IsTransientError
