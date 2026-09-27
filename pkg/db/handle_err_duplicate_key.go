package db

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// IsDuplicateKeyError reports whether err is a MySQL duplicate-entry error
// (error number 1062), including wrapped errors.
func IsDuplicateKeyError(err error) bool {
	mysqlErr, ok := errors.AsType[*mysql.MySQLError](err)
	return ok && mysqlErr.Number == 1062
}
