// dbtype/null_json.go
package dbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type NullJSON struct {
	Data  json.RawMessage
	Valid bool
}

func (n *NullJSON) Scan(value any) error {
	if value == nil {
		n.Data, n.Valid = nil, false
		return nil
	}

	buf, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T to NullJSON", value)
	}

	if len(buf) == 0 {
		n.Data, n.Valid = nil, false
		return nil
	}

	// CRITICAL: A defensive copy to prevent buffer reuse corruption
	// json.RawMessage holds a reference to the underlying byte slice.
	// When the driver reuses that buffer for the next row, your NullJSON.Data suddenly points to corrupted memory.
	// This consumes more memory compared to copyless version and also guarantees data integrity.
	clone := make([]byte, len(buf))
	copy(clone, buf)

	n.Data, n.Valid = json.RawMessage(clone), true
	return nil
}

func (n NullJSON) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return []byte(n.Data), nil
}

// UnmarshalTo unmarshals the JSON data into the provided variable.
// Returns nil if the JSON is NULL or invalid (no error, just no-op).
// Returns error only if JSON unmarshaling fails.
//
// Use this when you want to declare your own variable and handle the result directly:
//
//	var myData MyStruct
//	if err := nullJSON.UnmarshalTo(&myData); err != nil {
//	    return err
//	}
//	// use myData
func (n *NullJSON) UnmarshalTo(v any) error {
	if !n.Valid {
		return nil
	}
	return json.Unmarshal(n.Data, v)
}
