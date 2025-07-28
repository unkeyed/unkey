package dbtype

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

// This is a custom type that represents a nullable string.
type NullString sql.NullString

// MarshalJSON implements the json.Marshaler interface.
func (ns *NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(ns.String)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ns *NullString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ns.Valid = false
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	ns.Valid = true
	ns.String = s
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
