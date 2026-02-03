package dbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringSlice is a []string that serializes to/from JSON in the database.
// Unlike a raw []byte, it provides type safety and defaults to an empty array
// rather than null.
type StringSlice []string

// Scan implements sql.Scanner for reading JSON arrays from the database.
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("StringSlice.Scan: expected []byte or string, got %T", value)
	}

	if len(bytes) == 0 {
		*s = StringSlice{}
		return nil
	}

	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer for writing JSON arrays to the database.
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}

	bytes, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return string(bytes), nil
}

// MarshalJSON implements json.Marshaler.
func (s StringSlice) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}

	return json.Marshal([]string(s))
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *StringSlice) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = StringSlice{}
		return nil
	}

	var slice []string
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*s = slice
	return nil
}
