package permissions

import (
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
)

type service struct {
	db     db.Database
	logger logging.Logger
	rbac   *rbac.RBAC
}

var _ PermissionService = (*service)(nil)

type Config struct {
	DB     db.Database
	Logger logging.Logger
}

func New(config Config) *service {

	return &service{
		db:     config.DB,
		logger: config.Logger,
		rbac:   rbac.New(),
	}
}
