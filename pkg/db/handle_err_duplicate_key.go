package db

import (
	"github.com/go-sql-driver/mysql"
)

func IsDuplicateKeyError(err error) bool {
	if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
		return true
	}

	return false
}
