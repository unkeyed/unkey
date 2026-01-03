package db

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

func IsDeadlockError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1213 {
		return true
	}

	return false
}
