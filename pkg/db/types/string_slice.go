package dbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringSlice is a custom type that represents a JSON-encoded []string in the database.
// It implements sql.Scanner and driver.Valuer for automatic serialization.
type StringSlice []string

// Scan implements the sql.Scanner interface.
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	data, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into StringSlice", value)
	}

	// Handle empty, "null", or "[]" - all return nil to not override image entrypoint
	if len(data) == 0 || string(data) == "null" || string(data) == "[]" {
		*s = nil
		return nil
	}

	if err := json.Unmarshal(data, s); err != nil {
		return err
	}

	// If unmarshaled to empty slice, return nil
	if len(*s) == 0 {
		*s = nil
	}

	return nil
}

// Value implements the driver.Valuer interface.
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s)
}

// IsEmpty returns true if the slice is nil or empty.
func (s StringSlice) IsEmpty() bool {
	return len(s) == 0
}
