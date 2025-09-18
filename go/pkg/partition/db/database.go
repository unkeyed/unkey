package db

import (
	_ "github.com/go-sql-driver/mysql"

	maindb "github.com/unkeyed/unkey/go/pkg/db"
)

// Type aliases to use main db interfaces
type DBTX = maindb.DBTX
