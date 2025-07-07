package keys

import (
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
)

type Config struct {
	Logger       logging.Logger
	DB           db.Database
	RateLimiter  ratelimit.Service
	UsageLimiter usagelimiter.Service
	RBAC         *rbac.RBAC
	Clickhouse   clickhouse.ClickHouse

	KeyCache       cache.Cache[string, db.FindKeyForVerificationRow]
	WorkspaceCache cache.Cache[string, db.Workspace]
}

type service struct {
	logger       logging.Logger
	db           db.Database
	raterLimiter ratelimit.Service
	usageLimiter usagelimiter.Service
	rbac         *rbac.RBAC
	clickhouse   clickhouse.ClickHouse

	// hash -> key
	keyCache       cache.Cache[string, db.FindKeyForVerificationRow]
	workspaceCache cache.Cache[string, db.Workspace]
}

func New(config Config) (*service, error) {
	return &service{
		logger:         config.Logger,
		db:             config.DB,
		rbac:           config.RBAC,
		raterLimiter:   config.RateLimiter,
		usageLimiter:   config.UsageLimiter,
		clickhouse:     config.Clickhouse,
		keyCache:       config.KeyCache,
		workspaceCache: config.WorkspaceCache,
	}, nil
}
