package depot

import (
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

type Depot struct {
	ctrlv1connect.UnimplementedBuildServiceHandler
	instanceID  string
	db          db.Database
	storage     storage.Storage
	apiUrl      string
	registryUrl string
	username    string
	accessToken string
	logger      logging.Logger
}

type Config struct {
	InstanceID  string
	DB          db.Database
	Storage     storage.Storage
	APIUrl      string
	RegistryUrl string
	Username    string
	AccessToken string
	Logger      logging.Logger
}

func New(cfg Config) *Depot {
	return &Depot{
		UnimplementedBuildServiceHandler: ctrlv1connect.UnimplementedBuildServiceHandler{},
		instanceID:                       cfg.InstanceID,
		db:                               cfg.DB,
		storage:                          cfg.Storage,
		apiUrl:                           cfg.APIUrl,
		registryUrl:                      cfg.RegistryUrl,
		username:                         cfg.Username,
		accessToken:                      cfg.AccessToken,
		logger:                           cfg.Logger,
	}
}
