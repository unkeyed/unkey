package namespace

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/db"
)

// ParseNamespaceRow converts a raw DB row into a FindRatelimitNamespace with parsed overrides.
func ParseNamespaceRow(row db.FindRatelimitNamespaceRow) db.FindRatelimitNamespace {
	result := db.FindRatelimitNamespace{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		Name:              row.Name,
		CreatedAtM:        row.CreatedAtM,
		UpdatedAtM:        row.UpdatedAtM,
		DeletedAtM:        row.DeletedAtM,
		DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
		WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
	}

	overrides, err := db.UnmarshalNullableJSONTo[[]db.FindRatelimitNamespaceLimitOverride](row.Overrides)
	if err != nil {
		return result
	}

	for _, override := range overrides {
		result.DirectOverrides[override.Identifier] = override
		if strings.Contains(override.Identifier, "*") {
			result.WildcardOverrides = append(result.WildcardOverrides, override)
		}
	}

	return result
}

// RowToNamespace converts a FindManyRatelimitNamespacesRow to FindRatelimitNamespace.
func RowToNamespace(row db.FindManyRatelimitNamespacesRow) db.FindRatelimitNamespace {
	result := db.FindRatelimitNamespace{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		Name:              row.Name,
		CreatedAtM:        row.CreatedAtM,
		UpdatedAtM:        row.UpdatedAtM,
		DeletedAtM:        row.DeletedAtM,
		DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
		WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
	}

	overrides, err := db.UnmarshalNullableJSONTo[[]db.FindRatelimitNamespaceLimitOverride](row.Overrides)
	if err != nil {
		return result
	}

	for _, override := range overrides {
		result.DirectOverrides[override.Identifier] = override
		if strings.Contains(override.Identifier, "*") {
			result.WildcardOverrides = append(result.WildcardOverrides, override)
		}
	}

	return result
}
