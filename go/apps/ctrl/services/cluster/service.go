package cluster

import (
	"sync"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type client struct {
	clientID  string
	selectors map[string]string
	buffer    *buffer.Buffer[*ctrlv1.InfraEvent]
	done      chan struct{}
}

type Service struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	db     db.Database
	logger logging.Logger

	// Maps regions to open clients
	clientsMu sync.RWMutex
	// clientID -> stream
	clients map[string]*client

	// static bearer token for authentication
	bearer string
}

type Config struct {
	Database db.Database
	Logger   logging.Logger
	Bearer   string
}

func New(cfg Config) *Service {
	s := &Service{
		UnimplementedClusterServiceHandler: ctrlv1connect.UnimplementedClusterServiceHandler{},
		db:                                 cfg.Database,
		logger:                             cfg.Logger,
		clientsMu:                          sync.RWMutex{},
		clients:                            make(map[string]*client),
		bearer:                             cfg.Bearer,
	}

	return s
}

var _ ctrlv1connect.ClusterServiceHandler = (*Service)(nil)
