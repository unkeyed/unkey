package db

import (
	"encoding/json"
	"fmt"

	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
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
// even though the underlying value is typically []byte. We perform a type assertion to safely
// extract the bytes before unmarshaling.
//
// Returns zero value if:
//   - data is nil
//   - data is not []byte (logs warning about unexpected type)
//   - byte slice is empty
//   - JSON unmarshaling fails (logs error with details)
//
// This fail-safe behavior prevents panics and allows callers to continue execution with
// empty values when JSON data is malformed or missing.
//
// Example usage:
//
//	roles := UnmarshalNullableJSONTo[[]RoleInfo](row.Roles, logger)
//	meta := UnmarshalNullableJSONTo[map[string]any](row.Meta, logger)
func UnmarshalNullableJSONTo[T any](data any, logger logging.Logger) T {
	var zero T

	if data == nil {
		return zero
	}

	bytes, ok := data.([]byte)
	if !ok {
		if logger != nil {
			logger.Warn("type assertion failed during unmarshal",
				"expected", "[]byte",
				"got", fmt.Sprintf("%T", data),
			)
		}
		return zero
	}
	if len(bytes) == 0 {
		return zero
	}

	var result T
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		if logger != nil {
			logger.Error("failed to unmarshal JSON",
				"error", err,
			)
		}
		return zero
	}

	return result
}
