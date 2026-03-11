package db

import "github.com/unkeyed/unkey/pkg/mysql"

// IsNotFound re-exports [mysql.IsNotFound] for callers of this package.
var IsNotFound = mysql.IsNotFound
