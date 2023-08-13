package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// connect to a database and ping it to ensure the connection works
func connect(dsn string) (*sql.DB, error) {

	if strings.Contains(dsn, "?") {
		dsn = fmt.Sprintf("%s&parseTime=true", dsn)
	} else {
		dsn = fmt.Sprintf("%s?parseTime=true", dsn)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database %s: %w", dsn, err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}
	return db, nil
}
