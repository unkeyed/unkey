package database

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrNotUnique = errors.New("not unique")
)
