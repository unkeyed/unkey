package dbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// RegionConfig maps region IDs to replica counts, e.g. {"us-east-1": 3, "eu-central-1": 1}.
// An empty map means 1 replica in all available regions (default behavior).
type RegionConfig map[string]int

// Scan implements sql.Scanner for reading JSON objects from the database.
func (r *RegionConfig) Scan(value interface{}) error {
	if value == nil {
		*r = RegionConfig{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("RegionConfig.Scan: expected []byte or string, got %T", value)
	}

	if len(bytes) == 0 {
		*r = RegionConfig{}
		return nil
	}

	return json.Unmarshal(bytes, r)
}

// Value implements driver.Valuer for writing JSON objects to the database.
func (r RegionConfig) Value() (driver.Value, error) {
	if r == nil {
		return "{}", nil
	}

	bytes, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return string(bytes), nil
}

// MarshalJSON implements json.Marshaler.
func (r RegionConfig) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("{}"), nil
	}

	return json.Marshal(map[string]int(r))
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *RegionConfig) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*r = RegionConfig{}
		return nil
	}

	var m map[string]int
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	*r = m
	return nil
}
