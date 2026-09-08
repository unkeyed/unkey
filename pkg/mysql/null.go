package mysql

import (
	"database/sql"
	"time"
)

// FindFirstString returns the first valid string in values. It returns the zero
// value and false when no value is valid.
func FindFirstString(values ...sql.NullString) (string, bool) {
	for _, value := range values {
		if value.Valid {
			return value.String, true
		}
	}

	return "", false
}

// FindFirstBool returns the first valid boolean in values. It returns the zero
// value and false when no value is valid.
func FindFirstBool(values ...sql.NullBool) (bool, bool) {
	for _, value := range values {
		if value.Valid {
			return value.Bool, true
		}
	}

	return false, false
}

// FindFirstInt16 returns the first valid int16 in values. It returns the zero
// value and false when no value is valid.
func FindFirstInt16(values ...sql.NullInt16) (int16, bool) {
	for _, value := range values {
		if value.Valid {
			return value.Int16, true
		}
	}

	return 0, false
}

// FindFirstInt32 returns the first valid int32 in values. It returns the zero
// value and false when no value is valid.
func FindFirstInt32(values ...sql.NullInt32) (int32, bool) {
	for _, value := range values {
		if value.Valid {
			return value.Int32, true
		}
	}

	return 0, false
}

// FindFirstInt64 returns the first valid int64 in values. It returns the zero
// value and false when no value is valid.
func FindFirstInt64(values ...sql.NullInt64) (int64, bool) {
	for _, value := range values {
		if value.Valid {
			return value.Int64, true
		}
	}

	return 0, false
}

// FindFirstTime returns the first valid time in values. It returns the zero
// value and false when no value is valid.
func FindFirstTime(values ...sql.NullTime) (time.Time, bool) {
	for _, value := range values {
		if value.Valid {
			return value.Time, true
		}
	}

	return time.Time{}, false
}
