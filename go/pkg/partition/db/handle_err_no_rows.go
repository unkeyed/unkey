package db

import (
	"database/sql"
	"errors"
)

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
