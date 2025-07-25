package dbtype

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

// This is a custom type that represents a nullable string.
type NullString sql.NullString

// MarshalJSON implements the json.Marshaler interface.
func (x *NullString) MarshalJSON() ([]byte, error) {
	if !x.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(x.String)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ns *NullString) UnmarshalJSON(data []byte) error {
	val := string(data)
	if val == "null" {
		ns.Valid = false
		return nil
	}

	ns.Valid = true
	ns.String = val

	return nil
}

// Scan implements the sql.Scanner interface.
func (ns *NullString) Scan(value interface{}) error {
	return (*sql.NullString)(ns).Scan(value)
}

// Value implements the driver.Valuer interface.
func (ns NullString) Value() (driver.Value, error) {
	return sql.NullString(ns).Value()
}
