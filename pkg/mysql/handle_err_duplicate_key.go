package mysql

import (
	"github.com/go-sql-driver/mysql"
)

// IsDuplicateKeyError reports whether err is MySQL error 1062
// (duplicate key / unique constraint violation).
func IsDuplicateKeyError(err error) bool {
	if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
		return true
	}

	return false
}
