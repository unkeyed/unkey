package hydra

import (
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/hydra/store/gorm"
	gormDriver "gorm.io/gorm"
)

// Store defines the persistence interface for workflow state and metadata.
// This interface abstracts the underlying storage implementation to allow
// for different database backends while maintaining the same API.
type Store = store.Store

// StoreFactory creates Store instances for testing and dependency injection.
// This is primarily used in testing scenarios where multiple store instances
// may be needed or when using dependency injection frameworks.
type StoreFactory = store.StoreFactory

// NewGORMStore creates a new Store implementation using GORM and the provided database.
//
// This is the primary store implementation for production use. It supports:
// - MySQL, PostgreSQL, and SQLite databases through GORM
// - Automatic schema migration
// - Connection pooling and transaction management
// - Optimized queries with proper indexing
//
// The clock parameter is used for testing with controllable time and should
// typically be clock.New() for production use.
//
// Example:
//
//	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
//	if err != nil {
//	    return err
//	}
//
//	store := hydra.NewGORMStore(db, clock.New())
//	engine := hydra.New(hydra.Config{
//	    Store: store,
//	    // ... other config
//	})
//
// The database connection should be configured with appropriate timeouts,
// connection limits, and retry logic before passing to this function.
func NewGORMStore(db *gormDriver.DB, clk clock.Clock) Store {
	return gorm.NewGORMStore(db, clk)
}
