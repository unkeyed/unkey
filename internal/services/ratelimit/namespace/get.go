package namespace

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
)

// Get looks up a namespace by name or ID within the given workspace.
// It uses the SWR cache with a DB fallback. Returns (zero, false, nil)
// when the namespace does not exist.
func (s *service) Get(ctx context.Context, workspaceID, nameOrID string) (db.FindRatelimitNamespace, bool, error) {
	cacheKey := cache.ScopedKey{WorkspaceID: workspaceID, Key: nameOrID}

	ns, hit, err := s.cache.SWR(ctx, cacheKey, func(ctx context.Context) (db.FindRatelimitNamespace, error) {
		return s.loadFromDB(ctx, workspaceID, nameOrID)
	}, caches.DefaultFindFirstOp)

	if err != nil {
		if db.IsNotFound(err) {
			return db.FindRatelimitNamespace{}, false, nil //nolint:exhaustruct
		}
		return db.FindRatelimitNamespace{}, false, err //nolint:exhaustruct
	}

	if hit == cache.Null {
		return db.FindRatelimitNamespace{}, false, nil //nolint:exhaustruct
	}

	return ns, true, nil
}

// GetMany looks up multiple namespaces by name within the given workspace.
// Uses the batch SWR cache with a DB fallback. Returns the found namespaces
// keyed by name, plus a slice of any names that were not found.
func (s *service) GetMany(ctx context.Context, workspaceID string, names []string) (map[string]db.FindRatelimitNamespace, []string, error) {
	cacheKeys := make([]cache.ScopedKey, len(names))
	for i, name := range names {
		cacheKeys[i] = cache.ScopedKey{WorkspaceID: workspaceID, Key: name}
	}

	loader := func(ctx context.Context, keys []cache.ScopedKey) (map[cache.ScopedKey]db.FindRatelimitNamespace, error) {
		if len(keys) == 0 {
			return map[cache.ScopedKey]db.FindRatelimitNamespace{}, nil
		}

		namespaceNames := make([]string, len(keys))
		for i, key := range keys {
			namespaceNames[i] = key.Key
		}

		rows, err := db.WithRetryContext(ctx, func() ([]db.FindManyRatelimitNamespacesRow, error) {
			return db.Query.FindManyRatelimitNamespaces(ctx, s.db.RO(), db.FindManyRatelimitNamespacesParams{
				WorkspaceID: workspaceID,
				Namespaces:  namespaceNames,
			})
		})
		if err != nil {
			return nil, err
		}

		results := make(map[cache.ScopedKey]db.FindRatelimitNamespace, len(rows)*2)
		for _, row := range rows {
			ns := rowToNamespace(row)

			// Cache by name
			nameKey := cache.ScopedKey{WorkspaceID: workspaceID, Key: row.Name}
			results[nameKey] = ns

			// Also cache by ID for ID-based lookups
			idKey := cache.ScopedKey{WorkspaceID: workspaceID, Key: row.ID}
			results[idKey] = ns
		}
		return results, nil
	}

	namespaces, hits, err := s.cache.SWRMany(ctx, cacheKeys, loader, caches.DefaultFindFirstOp)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[string]db.FindRatelimitNamespace, len(names))
	var missing []string
	for _, key := range cacheKeys {
		if hits[key] == cache.Null {
			missing = append(missing, key.Key)
			continue
		}
		found[key.Key] = namespaces[key]
	}

	return found, missing, nil
}

// loadFromDB fetches a single namespace from the read replica and parses overrides.
func (s *service) loadFromDB(ctx context.Context, workspaceID, nameOrID string) (db.FindRatelimitNamespace, error) {
	row, err := db.WithRetryContext(ctx, func() (db.FindRatelimitNamespaceRow, error) {
		return db.Query.FindRatelimitNamespace(ctx, s.db.RO(), db.FindRatelimitNamespaceParams{
			WorkspaceID: workspaceID,
			Namespace:   nameOrID,
		})
	})
	if err != nil {
		return db.FindRatelimitNamespace{}, err
	}

	return parseNamespaceRow(row), nil
}

// loadFromDBWith fetches a single namespace using the provided connection (e.g. a write
// transaction) instead of the read replica, avoiding replica lag after a duplicate-key race.
func (s *service) loadFromDBWith(ctx context.Context, dbtx db.DBTX, workspaceID, nameOrID string) (db.FindRatelimitNamespace, error) {
	row, err := db.Query.FindRatelimitNamespace(ctx, dbtx, db.FindRatelimitNamespaceParams{
		WorkspaceID: workspaceID,
		Namespace:   nameOrID,
	})
	if err != nil {
		return db.FindRatelimitNamespace{}, err
	}

	return parseNamespaceRow(row), nil
}

// parseNamespaceRow converts a raw DB row into a FindRatelimitNamespace with parsed overrides.
func parseNamespaceRow(row db.FindRatelimitNamespaceRow) db.FindRatelimitNamespace {
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

	overrides := make([]db.FindRatelimitNamespaceLimitOverride, 0)
	if overrideBytes, ok := row.Overrides.([]byte); ok && overrideBytes != nil {
		if unmarshalErr := json.Unmarshal(overrideBytes, &overrides); unmarshalErr != nil {
			return result
		}
	}

	for _, override := range overrides {
		result.DirectOverrides[override.Identifier] = override
		if strings.Contains(override.Identifier, "*") {
			result.WildcardOverrides = append(result.WildcardOverrides, override)
		}
	}

	return result
}

// rowToNamespace converts a FindManyRatelimitNamespacesRow to FindRatelimitNamespace.
func rowToNamespace(row db.FindManyRatelimitNamespacesRow) db.FindRatelimitNamespace {
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
