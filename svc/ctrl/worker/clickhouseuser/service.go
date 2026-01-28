package clickhouseuser

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Service orchestrates ClickHouse user provisioning for workspaces.
//
// Service implements hydrav1.ClickhouseUserServiceServer with [Service.ConfigureUser]
// as the primary handler for creating and updating ClickHouse users. It coordinates
// between MySQL (credential storage), Vault (password encryption), and ClickHouse
// (user creation with permissions).
//
// Not safe for concurrent use on the same workspace. Concurrency control is handled
// by Restate's virtual object model which keys handlers by workspace_id.
type Service struct {
	hydrav1.UnimplementedClickhouseUserServiceServer
	db         db.Database
	vault      vaultv1connect.VaultServiceClient
	clickhouse clickhouse.ClickHouse
	logger     logging.Logger
}

var _ hydrav1.ClickhouseUserServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service] instance.
type Config struct {
	// DB provides database access for storing encrypted credentials.
	DB db.Database

	// Vault encrypts passwords before database storage. Passwords are encrypted using
	// the workspace ID as the keyring identifier.
	Vault vaultv1connect.VaultServiceClient

	// Clickhouse is the admin connection for creating users and managing permissions.
	// Must be connected as a user with CREATE/ALTER/DROP permissions for USER, QUOTA,
	// ROW POLICY, and SETTINGS PROFILE, plus GRANT OPTION on analytics tables.
	Clickhouse clickhouse.ClickHouse

	// Logger receives structured log output from user provisioning operations.
	Logger logging.Logger
}

// New creates a [Service] with the given configuration. The returned service is
// ready to handle user provisioning requests.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedClickhouseUserServiceServer: hydrav1.UnimplementedClickhouseUserServiceServer{},
		db:                                       cfg.DB,
		vault:                                    cfg.Vault,
		clickhouse:                               cfg.Clickhouse,
		logger:                                   cfg.Logger,
	}
}
