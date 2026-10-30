package db

import (
	"context"
	"slices"
)

// LockUnkeyRootKeys requires a transaction and locks rows in ID order to avoid
// reciprocal-mutation deadlocks. Duplicate IDs are locked once. Missing keys are
// absent from the result so callers can distinguish missing callers and targets.
func LockUnkeyRootKeys(ctx context.Context, tx DBTX, ids ...string) (map[string]UnkeyRootKey, error) {
	ids = slices.Clone(ids)
	slices.Sort(ids)
	keys := make(map[string]UnkeyRootKey, len(ids))
	for _, id := range slices.Compact(ids) {
		key, err := Query.FindUnkeyRootKeyByIDForUpdate(ctx, tx, id)
		if IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		keys[id] = key
	}
	return keys, nil
}
