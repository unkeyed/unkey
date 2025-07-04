package hydra

import (
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
)

// Store defines the persistence interface for workflow state and metadata.
// This interface abstracts the underlying storage implementation to allow
// for different database backends while maintaining the same API.
type Store = store.Store

// StoreFactory creates Store instances for testing and dependency injection.
// This is primarily used in testing scenarios where multiple store instances
// may be needed or when using dependency injection frameworks.
type StoreFactory = store.StoreFactory
