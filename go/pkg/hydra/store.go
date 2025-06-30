package hydra

import (
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/hydra/store/gorm"
	gormDriver "gorm.io/gorm"
)

type Store = store.Store

type StoreFactory = store.StoreFactory

func NewGORMStore(db *gormDriver.DB, clk clock.Clock) Store {
	return gorm.NewGORMStore(db, clk)
}
