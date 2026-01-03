package db

import (
	"encoding/json"
	"fmt"

	dbtype "github.com/unkeyed/unkey/pkg/db/types"
)

// RoleInfo  types mirror the database models and support JSON serialization and deserialization.
// They are used to unmarshal aggregated results (e.g., JSON arrays) returned by database queries.
type RoleInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description dbtype.NullString `json:"description"`
}

type PermissionInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description dbtype.NullString `json:"description"`
}

type RatelimitInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	KeyID      dbtype.NullString `json:"key_id"`
	IdentityID dbtype.NullString `json:"identity_id"`
	Limit      int32             `json:"limit"`
	Duration   int64             `json:"duration"`
	AutoApply  bool              `json:"auto_apply"`
}

// UnmarshalNullableJSONTo unmarshals JSON data from database columns into Go types.
// It handles the common pattern where database queries return JSON as []byte that needs
// to be deserialized into structs, slices, or maps.
//
// The function accepts 'any' type because database drivers return interface{} for JSON columns,
// even though the underlying value is typically []byte.
//
// Returns:
//   - (T, nil) on successful unmarshal
//   - (zero, nil) if data is nil or empty []byte (these are valid null/empty states)
//   - (zero, error) if type assertion fails or JSON unmarshal fails
//
// Example usage:
//
//	roles, err := UnmarshalNullableJSONTo[[]RoleInfo](row.Roles)
//	if err != nil {
//	    logger.Error("failed to unmarshal roles", "error", err)
//	    return err
//	}
func UnmarshalNullableJSONTo[T any](data any) (T, error) {
	var zero T
	if data == nil {
		return zero, nil
	}

	var bytes []byte
	switch v := data.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return zero, fmt.Errorf("type assertion failed during unmarshal: expected []byte or string, got %T", data)
	}

	if len(bytes) == 0 {
		return zero, nil
	}

	var result T
	if err := json.Unmarshal(bytes, &result); err != nil {
		return zero, fmt.Errorf("json unmarshal failed: %w", err)
	}

	return result, nil
}
