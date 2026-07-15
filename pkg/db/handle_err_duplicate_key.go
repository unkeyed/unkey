package db

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// IsDuplicateKeyError reports whether err is a MySQL duplicate-entry error
// (error number 1062), traversing wrapped errors via errors.As.
func IsDuplicateKeyError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}

	return false
}
