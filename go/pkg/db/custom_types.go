package db

import (
	"encoding/json"

	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
)

// These types mirror the database models and support JSON serialization and deserialization.
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

// UnmarshalJSONArrayTo deserializes a JSON byte array into a slice of type T.
// It handles the common pattern of database queries returning aggregated JSON arrays
// that need to be unmarshaled into Go structs.
//
// Returns an empty slice if:
//   - data is nil
//   - data is not []byte
//   - the byte slice is empty
//   - JSON unmarshaling fails
//
// This fail-safe behavior prevents nil pointer panics and simplifies error handling
// at call sites, allowing callers to always work with a valid slice.
//
// Example usage:
//
//	roles := UnmarshalJSONArrayTo[RoleInfo](row.Roles)
//	// roles is guaranteed to be []RoleInfo, never nil
func UnmarshalJSONArrayTo[T any](data any) []T {
	if data == nil {
		return []T{}
	}

	bytes, ok := data.([]byte)
	if !ok || len(bytes) == 0 {
		return []T{}
	}

	var result []T
	if err := json.Unmarshal(bytes, &result); err != nil {
		return []T{}
	}
	return result
}

// UnmarshalNullableJSONTo unmarshals the JSON data and returns the result with an error.
// Returns zero value and nil if the JSON is NULL or invalid.
// Returns zero value and error if JSON unmarshaling fails.
//
// Use this when you want inline assignment with generic type inference:
//
//	myData, err := UnmarshalNullableJSONTo[MyStruct](nullJSON)
//	if err != nil {
//	    return err
//	}
//	// use myData
func UnmarshalNullableJSONTo[T any](data []byte) (T, error) {
	var result T

	// NULL check
	if data == nil {
		return result, nil
	}

	// Empty check
	if len(data) == 0 {
		return result, nil
	}

	err := json.Unmarshal(data, &result)
	return result, err
}
